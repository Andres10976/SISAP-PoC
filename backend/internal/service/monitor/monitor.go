package monitor

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/matcher"
)

var (
	ErrAlreadyRunning = errors.New("monitor already running")
	ErrNotRunning     = errors.New("monitor not running")
)

type ctClient interface {
	GetSTH(ctx context.Context) (*ctlog.STH, error)
	GetEntries(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error)
}

type keywordLister interface {
	List(ctx context.Context) ([]model.Keyword, error)
}

type certCreator interface {
	Create(ctx context.Context, cert *model.MatchedCertificate) error
}

type stateStore interface {
	Get(ctx context.Context) (*model.MonitorState, error)
	Update(ctx context.Context, state *model.MonitorState) error
	SetRunning(ctx context.Context, running bool) error
}

type Monitor struct {
	ctClient  ctClient
	keywords  keywordLister
	certs     certCreator
	state     stateStore
	batchSize int
	interval  time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
}

func New(
	ct ctClient,
	kw keywordLister,
	cert certCreator,
	st stateStore,
	batchSize int,
	interval time.Duration,
) *Monitor {
	return &Monitor{
		ctClient:  ct,
		keywords:  kw,
		certs:     cert,
		state:     st,
		batchSize: batchSize,
		interval:  interval,
	}
}

// Start launches the background monitoring loop.
func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		return ErrAlreadyRunning
	}

	monCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	if err := m.state.SetRunning(ctx, true); err != nil {
		cancel()
		m.cancel = nil
		return err
	}

	go m.run(monCtx)
	return nil
}

// Stop halts the monitoring loop.
func (m *Monitor) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel == nil {
		return ErrNotRunning
	}

	m.cancel()
	m.cancel = nil

	return m.state.SetRunning(ctx, false)
}

// IsRunning returns whether the monitor loop is active.
func (m *Monitor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancel != nil
}

func (m *Monitor) run(ctx context.Context) {
	m.processBatch(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.processBatch(ctx)
		}
	}
}

func (m *Monitor) processBatch(ctx context.Context) {
	logger := slog.Default()

	// 1. Get current Signed Tree Head
	sth, err := m.ctClient.GetSTH(ctx)
	if err != nil {
		logger.Error("failed to get STH", "error", err)
		return
	}

	// 2. Load current monitor state
	state, err := m.state.Get(ctx)
	if err != nil {
		logger.Error("failed to get monitor state", "error", err)
		return
	}

	// 3. Calculate batch range
	start := state.LastProcessedIndex
	if start == 0 {
		start = max(0, sth.TreeSize-int64(m.batchSize))
	}
	end := min(start+int64(m.batchSize)-1, sth.TreeSize-1)

	if start > end {
		logger.Info("no new entries to process")
		return
	}

	logger.Info("fetching CT log entries",
		"start", start, "end", end, "tree_size", sth.TreeSize)

	// 4. Fetch entries
	entries, err := m.ctClient.GetEntries(ctx, start, end)
	if err != nil {
		logger.Error("failed to fetch entries", "error", err)
		return
	}

	// 5. Load keywords
	keywords, err := m.keywords.List(ctx)
	if err != nil {
		logger.Error("failed to load keywords", "error", err)
		return
	}

	if len(keywords) == 0 {
		logger.Info("no keywords configured, skipping matching")
		m.updateState(ctx, state, end, sth.TreeSize, len(entries), 0)
		return
	}

	// 6. Parse and match
	matchCount := 0
	parseErrors := 0
	for i, entry := range entries {
		cert, err := ctlog.ParseLeafInput(entry.LeafInput)
		if err != nil {
			parseErrors++
			continue
		}

		matches := matcher.Match(cert, keywords)
		for _, match := range matches {
			err := m.certs.Create(ctx, &model.MatchedCertificate{
				SerialNumber:  cert.Serial,
				CommonName:    cert.CommonName,
				SANs:          cert.SANs,
				Issuer:        cert.Issuer,
				NotBefore:     cert.NotBefore,
				NotAfter:      cert.NotAfter,
				KeywordID:     match.KeywordID,
				MatchedDomain: match.MatchedDomain,
				CTLogIndex:    start + int64(i),
			})
			if err != nil {
				logger.Error("failed to store match", "error", err, "domain", match.MatchedDomain)
				continue
			}
			matchCount++
		}
	}

	logger.Info("batch processed",
		"entries", len(entries),
		"parse_errors", parseErrors,
		"matches", matchCount,
	)

	// 7. Update state
	m.updateState(ctx, state, end, sth.TreeSize, len(entries), matchCount)
}

func (m *Monitor) updateState(
	ctx context.Context,
	prev *model.MonitorState,
	endIndex, treeSize int64,
	processed, matches int,
) {
	err := m.state.Update(ctx, &model.MonitorState{
		LastProcessedIndex: endIndex + 1,
		LastTreeSize:       treeSize,
		TotalProcessed:     prev.TotalProcessed + int64(processed),
		CertsInLastCycle:   processed,
		MatchesInLastCycle: matches,
		IsRunning:          true,
	})
	if err != nil {
		slog.Error("failed to update monitor state", "error", err)
	}
}
