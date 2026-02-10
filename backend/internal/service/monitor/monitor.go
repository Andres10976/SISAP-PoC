package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
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
	SetError(ctx context.Context, errMsg string) error
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

	// Cache of the last fetched batch for re-matching when keywords change.
	// Only accessed from the monitor goroutine — no mutex needed.
	cachedEntries []ctlog.RawEntry
	cachedStart   int64
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
// The goroutine uses a context derived from context.Background so it
// survives after the calling HTTP request completes.
func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		return ErrAlreadyRunning
	}

	monCtx, cancel := context.WithCancel(context.Background())
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
// Uses a background context for the DB update so it succeeds even if
// the HTTP request context is already canceled.
func (m *Monitor) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel == nil {
		return ErrNotRunning
	}

	m.cancel()
	m.cancel = nil

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()
	return m.state.SetRunning(dbCtx, false)
}

// IsRunning returns whether the monitor loop is active.
func (m *Monitor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancel != nil
}

func (m *Monitor) run(ctx context.Context) {
	slog.Info("monitor goroutine started", "batch_size", m.batchSize, "interval", m.interval)

	defer func() {
		if r := recover(); r != nil {
			slog.Error("monitor goroutine panicked", "error", r, "stack", string(debug.Stack()))
			m.mu.Lock()
			m.cancel = nil
			m.mu.Unlock()
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			m.state.SetRunning(cleanupCtx, false)
			m.state.SetError(cleanupCtx, fmt.Sprintf("panic: %v", r))
		}
	}()

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
		m.state.SetError(ctx, fmt.Sprintf("failed to get STH: %v", err))
		return
	}

	// 2. Load current monitor state
	state, err := m.state.Get(ctx)
	if err != nil {
		logger.Error("failed to get monitor state", "error", err)
		m.state.SetError(ctx, fmt.Sprintf("failed to get monitor state: %v", err))
		return
	}

	// 3. Calculate batch range
	start := state.LastProcessedIndex
	if start == 0 {
		start = max(0, sth.TreeSize-int64(m.batchSize))
	}
	end := min(start+int64(m.batchSize)-1, sth.TreeSize-1)

	// 4. Get entries — either fresh from the CT log or from cache
	var entries []ctlog.RawEntry
	var batchStart int64
	newEntries := start <= end

	if newEntries {
		logger.Info("fetching CT log entries",
			"start", start, "end", end, "tree_size", sth.TreeSize)

		entries, err = m.ctClient.GetEntries(ctx, start, end)
		if err != nil {
			logger.Error("failed to fetch entries", "error", err)
			m.state.SetError(ctx, fmt.Sprintf("failed to fetch entries: %v", err))
			return
		}
		batchStart = start

		// Cache for re-matching on idle ticks
		m.cachedEntries = entries
		m.cachedStart = batchStart
	} else {
		entries = m.cachedEntries
		batchStart = m.cachedStart
		if len(entries) == 0 {
			logger.Info("no new entries and no cached data", "tree_size", sth.TreeSize)
			m.state.Update(ctx, &model.MonitorState{
				LastProcessedIndex:     state.LastProcessedIndex,
				LastTreeSize:           sth.TreeSize,
				TotalProcessed:         state.TotalProcessed,
				CertsInLastCycle:       state.CertsInLastCycle,
				MatchesInLastCycle:     state.MatchesInLastCycle,
				ParseErrorsInLastCycle: state.ParseErrorsInLastCycle,
				IsRunning:              true,
			})
			m.state.SetError(ctx, "")
			return
		}
		logger.Info("re-matching cached entries against current keywords",
			"cached_entries", len(entries), "tree_size", sth.TreeSize)
	}

	// 5. Load keywords
	keywords, err := m.keywords.List(ctx)
	if err != nil {
		logger.Error("failed to load keywords", "error", err)
		m.state.SetError(ctx, fmt.Sprintf("failed to load keywords: %v", err))
		return
	}

	if len(keywords) == 0 {
		logger.Info("no keywords configured, skipping matching")
		if newEntries {
			m.updateState(ctx, state, end, sth.TreeSize, len(entries), 0, 0)
		} else {
			m.state.Update(ctx, &model.MonitorState{
				LastProcessedIndex:     state.LastProcessedIndex,
				LastTreeSize:           sth.TreeSize,
				TotalProcessed:         state.TotalProcessed,
				CertsInLastCycle:       state.CertsInLastCycle,
				MatchesInLastCycle:     0,
				ParseErrorsInLastCycle: state.ParseErrorsInLastCycle,
				IsRunning:              true,
			})
		}
		m.state.SetError(ctx, "")
		return
	}

	// 6. Parse and match
	matchCount, parseErrors := m.matchEntries(ctx, entries, batchStart, keywords)

	logger.Info("batch processed",
		"entries", len(entries),
		"parse_errors", parseErrors,
		"matches", matchCount,
		"cached", !newEntries,
	)

	// 7. Update state and clear any previous error
	if newEntries {
		m.updateState(ctx, state, end, sth.TreeSize, len(entries), matchCount, parseErrors)
	} else {
		m.state.Update(ctx, &model.MonitorState{
			LastProcessedIndex:     state.LastProcessedIndex,
			LastTreeSize:           sth.TreeSize,
			TotalProcessed:         state.TotalProcessed,
			CertsInLastCycle:       state.CertsInLastCycle,
			MatchesInLastCycle:     matchCount,
			ParseErrorsInLastCycle: state.ParseErrorsInLastCycle,
			IsRunning:              true,
		})
	}
	m.state.SetError(ctx, "")
}

func (m *Monitor) matchEntries(
	ctx context.Context,
	entries []ctlog.RawEntry,
	batchStart int64,
	keywords []model.Keyword,
) (matchCount, parseErrors int) {
	for i, entry := range entries {
		cert, err := ctlog.ParseLeafInput(entry.LeafInput, entry.ExtraData)
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
				CTLogIndex:    batchStart + int64(i),
			})
			if err != nil {
				slog.Error("failed to store match", "error", err, "domain", match.MatchedDomain)
				continue
			}
			matchCount++
		}
	}
	return
}

func (m *Monitor) updateState(
	ctx context.Context,
	prev *model.MonitorState,
	endIndex, treeSize int64,
	processed, matches, parseErrors int,
) {
	err := m.state.Update(ctx, &model.MonitorState{
		LastProcessedIndex:     endIndex + 1,
		LastTreeSize:           treeSize,
		TotalProcessed:         prev.TotalProcessed + int64(processed),
		CertsInLastCycle:       processed,
		MatchesInLastCycle:     matches,
		ParseErrorsInLastCycle: parseErrors,
		IsRunning:              true,
	})
	if err != nil {
		slog.Error("failed to update monitor state", "error", err)
	}
}
