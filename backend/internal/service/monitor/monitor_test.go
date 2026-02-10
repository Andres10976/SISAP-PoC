package monitor

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
)

// --- mocks ---

type mockCTClient struct {
	getSTHFn     func(ctx context.Context) (*ctlog.STH, error)
	getEntriesFn func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error)
}

func (m *mockCTClient) GetSTH(ctx context.Context) (*ctlog.STH, error) {
	return m.getSTHFn(ctx)
}
func (m *mockCTClient) GetEntries(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
	return m.getEntriesFn(ctx, start, end)
}

type mockKeywordLister struct {
	listFn func(ctx context.Context) ([]model.Keyword, error)
}

func (m *mockKeywordLister) List(ctx context.Context) ([]model.Keyword, error) {
	return m.listFn(ctx)
}

type mockCertCreator struct {
	createFn func(ctx context.Context, cert *model.MatchedCertificate) error
}

func (m *mockCertCreator) Create(ctx context.Context, cert *model.MatchedCertificate) error {
	return m.createFn(ctx, cert)
}

type mockStateStore struct {
	getFn        func(ctx context.Context) (*model.MonitorState, error)
	updateFn     func(ctx context.Context, state *model.MonitorState) error
	setRunningFn func(ctx context.Context, running bool) error
	setErrorFn   func(ctx context.Context, errMsg string) error
}

func (m *mockStateStore) Get(ctx context.Context) (*model.MonitorState, error) {
	return m.getFn(ctx)
}
func (m *mockStateStore) Update(ctx context.Context, state *model.MonitorState) error {
	return m.updateFn(ctx, state)
}
func (m *mockStateStore) SetRunning(ctx context.Context, running bool) error {
	return m.setRunningFn(ctx, running)
}
func (m *mockStateStore) SetError(ctx context.Context, errMsg string) error {
	if m.setErrorFn != nil {
		return m.setErrorFn(ctx, errMsg)
	}
	return nil
}

// --- helpers ---

// buildLeaf constructs a minimal MerkleTreeLeaf blob (x509_entry) for testing.
func buildLeaf(t *testing.T, certDER []byte) []byte {
	t.Helper()
	var buf []byte
	buf = append(buf, 0, 0) // version + leaf type
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(time.Now().UnixMilli()))
	buf = append(buf, ts...)
	buf = append(buf, 0, 0) // entry type 0 = x509_entry
	lenBytes := []byte{
		byte(len(certDER) >> 16),
		byte(len(certDER) >> 8),
		byte(len(certDER)),
	}
	buf = append(buf, lenBytes...)
	buf = append(buf, certDER...)
	return buf
}

func selfSignedDER(t *testing.T, cn string, sans []string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     sans,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return der
}

// --- Start / Stop / IsRunning tests ---

func TestStart_Success(t *testing.T) {
	ss := &mockStateStore{
		setRunningFn: func(ctx context.Context, running bool) error { return nil },
		// processBatch will call these; provide stubs that cause early return
		getFn: func(ctx context.Context) (*model.MonitorState, error) {
			return nil, errors.New("stub")
		},
	}
	ct := &mockCTClient{
		getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
			return nil, errors.New("stub")
		},
	}
	m := New(ct, &mockKeywordLister{}, &mockCertCreator{}, ss, 10, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !m.IsRunning() {
		t.Error("IsRunning() = false after Start")
	}

	// Cleanup
	cancel()
	time.Sleep(10 * time.Millisecond)
}

func TestStart_SurvivesCanceledCallerContext(t *testing.T) {
	ticks := make(chan struct{}, 5)
	ss := &mockStateStore{
		setRunningFn: func(ctx context.Context, running bool) error { return nil },
		getFn: func(ctx context.Context) (*model.MonitorState, error) {
			return nil, errors.New("stub")
		},
	}
	ct := &mockCTClient{
		getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
			ticks <- struct{}{}
			return nil, errors.New("stub")
		},
	}
	m := New(ct, &mockKeywordLister{}, &mockCertCreator{}, ss, 10, 20*time.Millisecond)

	// Start with a context, then immediately cancel it — simulates
	// an HTTP handler returning before the goroutine runs.
	callerCtx, callerCancel := context.WithCancel(context.Background())
	if err := m.Start(callerCtx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	callerCancel()

	// Wait for at least two ticks — proves the goroutine survived the canceled caller ctx.
	for i := 0; i < 2; i++ {
		select {
		case <-ticks:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for tick %d — goroutine likely died", i+1)
		}
	}

	// Cleanup
	m.Stop(context.Background())
}

func TestStart_AlreadyRunning(t *testing.T) {
	ss := &mockStateStore{
		setRunningFn: func(ctx context.Context, running bool) error { return nil },
		getFn: func(ctx context.Context) (*model.MonitorState, error) {
			return nil, errors.New("stub")
		},
	}
	ct := &mockCTClient{
		getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
			return nil, errors.New("stub")
		},
	}
	m := New(ct, &mockKeywordLister{}, &mockCertCreator{}, ss, 10, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.Start(ctx)
	defer func() { cancel(); time.Sleep(10 * time.Millisecond) }()

	err := m.Start(ctx)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("Start() error = %v, want ErrAlreadyRunning", err)
	}
}

func TestStart_SetRunningError(t *testing.T) {
	dbErr := errors.New("db down")
	ss := &mockStateStore{
		setRunningFn: func(ctx context.Context, running bool) error { return dbErr },
	}
	m := New(&mockCTClient{}, &mockKeywordLister{}, &mockCertCreator{}, ss, 10, time.Hour)

	err := m.Start(context.Background())
	if !errors.Is(err, dbErr) {
		t.Errorf("Start() error = %v, want %v", err, dbErr)
	}
	if m.IsRunning() {
		t.Error("IsRunning() = true after failed Start")
	}
}

func TestStop_Success(t *testing.T) {
	ss := &mockStateStore{
		setRunningFn: func(ctx context.Context, running bool) error { return nil },
		getFn: func(ctx context.Context) (*model.MonitorState, error) {
			return nil, errors.New("stub")
		},
	}
	ct := &mockCTClient{
		getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
			return nil, errors.New("stub")
		},
	}
	m := New(ct, &mockKeywordLister{}, &mockCertCreator{}, ss, 10, time.Hour)

	ctx := context.Background()
	m.Start(ctx)
	time.Sleep(10 * time.Millisecond) // let goroutine start

	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if m.IsRunning() {
		t.Error("IsRunning() = true after Stop")
	}
}

func TestStop_NotRunning(t *testing.T) {
	m := New(&mockCTClient{}, &mockKeywordLister{}, &mockCertCreator{}, &mockStateStore{}, 10, time.Hour)

	err := m.Stop(context.Background())
	if !errors.Is(err, ErrNotRunning) {
		t.Errorf("Stop() error = %v, want ErrNotRunning", err)
	}
}

func TestIsRunning_DefaultFalse(t *testing.T) {
	m := New(&mockCTClient{}, &mockKeywordLister{}, &mockCertCreator{}, &mockStateStore{}, 10, time.Hour)
	if m.IsRunning() {
		t.Error("IsRunning() = true for new monitor")
	}
}

// --- processBatch tests ---

func TestProcessBatch_Success(t *testing.T) {
	der := selfSignedDER(t, "example.com", []string{"www.example.com"})
	leaf := buildLeaf(t, der)

	var storedCert *model.MatchedCertificate
	var updatedState *model.MonitorState

	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return &ctlog.STH{TreeSize: 200}, nil
			},
			getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
				return []ctlog.RawEntry{{LeafInput: leaf}}, nil
			},
		},
		&mockKeywordLister{
			listFn: func(ctx context.Context) ([]model.Keyword, error) {
				return []model.Keyword{{ID: 1, Value: "example"}}, nil
			},
		},
		&mockCertCreator{
			createFn: func(ctx context.Context, cert *model.MatchedCertificate) error {
				storedCert = cert
				return nil
			},
		},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return &model.MonitorState{LastProcessedIndex: 100}, nil
			},
			updateFn: func(ctx context.Context, state *model.MonitorState) error {
				updatedState = state
				return nil
			},
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if storedCert == nil {
		t.Fatal("expected a certificate to be stored")
	}
	if storedCert.CommonName != "example.com" {
		t.Errorf("storedCert.CommonName = %q, want %q", storedCert.CommonName, "example.com")
	}
	if storedCert.KeywordID != 1 {
		t.Errorf("storedCert.KeywordID = %d, want 1", storedCert.KeywordID)
	}
	if storedCert.MatchedDomain != "example.com" {
		t.Errorf("storedCert.MatchedDomain = %q, want %q", storedCert.MatchedDomain, "example.com")
	}

	if updatedState == nil {
		t.Fatal("expected state to be updated")
	}
	if updatedState.MatchesInLastCycle != 1 {
		t.Errorf("MatchesInLastCycle = %d, want 1", updatedState.MatchesInLastCycle)
	}
	if updatedState.CertsInLastCycle != 1 {
		t.Errorf("CertsInLastCycle = %d, want 1", updatedState.CertsInLastCycle)
	}
}

func TestProcessBatch_STHError(t *testing.T) {
	stateCalled := false
	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return nil, errors.New("network error")
			},
		},
		&mockKeywordLister{},
		&mockCertCreator{},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				stateCalled = true
				return nil, nil
			},
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if stateCalled {
		t.Error("state.Get should not be called when STH fails")
	}
}

func TestProcessBatch_StateGetError(t *testing.T) {
	entriesCalled := false
	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return &ctlog.STH{TreeSize: 200}, nil
			},
			getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
				entriesCalled = true
				return nil, nil
			},
		},
		&mockKeywordLister{},
		&mockCertCreator{},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return nil, errors.New("db error")
			},
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if entriesCalled {
		t.Error("GetEntries should not be called when state.Get fails")
	}
}

func TestProcessBatch_NoNewEntries(t *testing.T) {
	entriesCalled := false
	var updatedState *model.MonitorState
	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return &ctlog.STH{TreeSize: 100}, nil
			},
			getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
				entriesCalled = true
				return nil, nil
			},
		},
		&mockKeywordLister{},
		&mockCertCreator{},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				// Already processed up to tree size
				return &model.MonitorState{LastProcessedIndex: 100}, nil
			},
			updateFn: func(ctx context.Context, state *model.MonitorState) error {
				updatedState = state
				return nil
			},
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if entriesCalled {
		t.Error("GetEntries should not be called when start > end")
	}
	if updatedState == nil {
		t.Fatal("state should still be updated when no new entries (to bump updated_at)")
	}
	if updatedState.LastProcessedIndex != 100 {
		t.Errorf("LastProcessedIndex = %d, want 100 (unchanged)", updatedState.LastProcessedIndex)
	}
}

func TestProcessBatch_NoKeywords(t *testing.T) {
	var updatedState *model.MonitorState
	certCreated := false

	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return &ctlog.STH{TreeSize: 200}, nil
			},
			getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
				return []ctlog.RawEntry{{LeafInput: []byte("dummy")}}, nil
			},
		},
		&mockKeywordLister{
			listFn: func(ctx context.Context) ([]model.Keyword, error) {
				return nil, nil // no keywords
			},
		},
		&mockCertCreator{
			createFn: func(ctx context.Context, cert *model.MatchedCertificate) error {
				certCreated = true
				return nil
			},
		},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return &model.MonitorState{LastProcessedIndex: 100}, nil
			},
			updateFn: func(ctx context.Context, state *model.MonitorState) error {
				updatedState = state
				return nil
			},
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if certCreated {
		t.Error("no certs should be stored when there are no keywords")
	}
	if updatedState == nil {
		t.Fatal("state should still be updated when no keywords")
	}
	if updatedState.MatchesInLastCycle != 0 {
		t.Errorf("MatchesInLastCycle = %d, want 0", updatedState.MatchesInLastCycle)
	}
}

func TestProcessBatch_ParseErrorSkipped(t *testing.T) {
	der := selfSignedDER(t, "example.com", nil)
	goodLeaf := buildLeaf(t, der)
	badLeaf := buildLeaf(t, []byte{0xDE, 0xAD}) // invalid DER

	createCount := 0
	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return &ctlog.STH{TreeSize: 200}, nil
			},
			getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
				return []ctlog.RawEntry{
					{LeafInput: badLeaf},
					{LeafInput: goodLeaf},
				}, nil
			},
		},
		&mockKeywordLister{
			listFn: func(ctx context.Context) ([]model.Keyword, error) {
				return []model.Keyword{{ID: 1, Value: "example"}}, nil
			},
		},
		&mockCertCreator{
			createFn: func(ctx context.Context, cert *model.MatchedCertificate) error {
				createCount++
				return nil
			},
		},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return &model.MonitorState{LastProcessedIndex: 100}, nil
			},
			updateFn: func(ctx context.Context, state *model.MonitorState) error { return nil },
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if createCount != 1 {
		t.Errorf("createCount = %d, want 1 (bad entry should be skipped)", createCount)
	}
}

func TestProcessBatch_CertStoreError(t *testing.T) {
	der := selfSignedDER(t, "example.com", nil)
	leaf := buildLeaf(t, der)

	var updatedState *model.MonitorState
	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return &ctlog.STH{TreeSize: 200}, nil
			},
			getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
				return []ctlog.RawEntry{{LeafInput: leaf}}, nil
			},
		},
		&mockKeywordLister{
			listFn: func(ctx context.Context) ([]model.Keyword, error) {
				return []model.Keyword{{ID: 1, Value: "example"}}, nil
			},
		},
		&mockCertCreator{
			createFn: func(ctx context.Context, cert *model.MatchedCertificate) error {
				return errors.New("insert failed")
			},
		},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return &model.MonitorState{LastProcessedIndex: 100}, nil
			},
			updateFn: func(ctx context.Context, state *model.MonitorState) error {
				updatedState = state
				return nil
			},
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if updatedState == nil {
		t.Fatal("state should still be updated even when cert store fails")
	}
	if updatedState.MatchesInLastCycle != 0 {
		t.Errorf("MatchesInLastCycle = %d, want 0 (store failed)", updatedState.MatchesInLastCycle)
	}
}

func TestProcessBatch_FirstBatch_StartsNearTreeSize(t *testing.T) {
	var requestedStart int64

	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return &ctlog.STH{TreeSize: 1000}, nil
			},
			getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
				requestedStart = start
				return nil, nil
			},
		},
		&mockKeywordLister{
			listFn: func(ctx context.Context) ([]model.Keyword, error) {
				return nil, nil
			},
		},
		&mockCertCreator{},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return &model.MonitorState{LastProcessedIndex: 0}, nil // fresh start
			},
			updateFn: func(ctx context.Context, state *model.MonitorState) error { return nil },
		},
		50, time.Hour,
	)

	m.processBatch(context.Background())

	// When LastProcessedIndex is 0, start = max(0, TreeSize - batchSize) = 950
	if requestedStart != 950 {
		t.Errorf("start = %d, want 950 (TreeSize 1000 - batchSize 50)", requestedStart)
	}
}

// --- error persistence tests ---

func TestProcessBatch_STHError_PersistsError(t *testing.T) {
	var lastError string
	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return nil, errors.New("network error")
			},
		},
		&mockKeywordLister{},
		&mockCertCreator{},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return nil, nil
			},
			setErrorFn: func(ctx context.Context, errMsg string) error {
				lastError = errMsg
				return nil
			},
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if lastError == "" {
		t.Error("expected SetError to be called with non-empty error")
	}
	if lastError != "failed to get STH: network error" {
		t.Errorf("lastError = %q, want %q", lastError, "failed to get STH: network error")
	}
}

func TestProcessBatch_Success_ClearsError(t *testing.T) {
	der := selfSignedDER(t, "example.com", []string{"www.example.com"})
	leaf := buildLeaf(t, der)

	var lastError string
	setErrorCalled := false
	m := New(
		&mockCTClient{
			getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
				return &ctlog.STH{TreeSize: 200}, nil
			},
			getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
				return []ctlog.RawEntry{{LeafInput: leaf}}, nil
			},
		},
		&mockKeywordLister{
			listFn: func(ctx context.Context) ([]model.Keyword, error) {
				return []model.Keyword{{ID: 1, Value: "example"}}, nil
			},
		},
		&mockCertCreator{
			createFn: func(ctx context.Context, cert *model.MatchedCertificate) error {
				return nil
			},
		},
		&mockStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return &model.MonitorState{LastProcessedIndex: 100}, nil
			},
			updateFn: func(ctx context.Context, state *model.MonitorState) error {
				return nil
			},
			setErrorFn: func(ctx context.Context, errMsg string) error {
				setErrorCalled = true
				lastError = errMsg
				return nil
			},
		},
		10, time.Hour,
	)

	m.processBatch(context.Background())

	if !setErrorCalled {
		t.Error("expected SetError to be called to clear error")
	}
	if lastError != "" {
		t.Errorf("lastError = %q, want empty string (error should be cleared)", lastError)
	}
}

// --- panic recovery tests ---

func TestRun_PanicRecovery(t *testing.T) {
	setRunningCalled := make(chan bool, 1)
	var panicError string

	ss := &mockStateStore{
		setRunningFn: func(ctx context.Context, running bool) error {
			if !running {
				setRunningCalled <- running
			}
			return nil
		},
		setErrorFn: func(ctx context.Context, errMsg string) error {
			panicError = errMsg
			return nil
		},
	}

	ct := &mockCTClient{
		getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
			panic("test panic in processBatch")
		},
	}

	m := New(ct, &mockKeywordLister{}, &mockCertCreator{}, ss, 10, time.Hour)
	// Manually set cancel so we can verify it gets cleared
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.cancel = cancel

	go m.run(ctx)

	select {
	case running := <-setRunningCalled:
		if running {
			t.Error("expected SetRunning(false) after panic")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for panic recovery to call SetRunning(false)")
	}

	// Verify cancel was cleared
	m.mu.Lock()
	cancelNil := m.cancel == nil
	m.mu.Unlock()
	if !cancelNil {
		t.Error("expected m.cancel to be nil after panic recovery")
	}

	if panicError == "" {
		t.Error("expected SetError to be called with panic message")
	}
	if panicError != "panic: test panic in processBatch" {
		t.Errorf("panicError = %q, want %q", panicError, "panic: test panic in processBatch")
	}
}

// --- cache re-matching tests ---

func TestProcessBatch_CacheReMatchOnNewKeyword(t *testing.T) {
	der := selfSignedDER(t, "example.com", []string{"www.example.com"})
	leaf := buildLeaf(t, der)

	callCount := 0
	var storedCerts []*model.MatchedCertificate

	ct := &mockCTClient{
		getSTHFn: func(ctx context.Context) (*ctlog.STH, error) {
			return &ctlog.STH{TreeSize: 200}, nil
		},
		getEntriesFn: func(ctx context.Context, start, end int64) ([]ctlog.RawEntry, error) {
			return []ctlog.RawEntry{{LeafInput: leaf}}, nil
		},
	}

	keywords := []model.Keyword{}
	kw := &mockKeywordLister{
		listFn: func(ctx context.Context) ([]model.Keyword, error) {
			return keywords, nil
		},
	}

	cc := &mockCertCreator{
		createFn: func(ctx context.Context, cert *model.MatchedCertificate) error {
			storedCerts = append(storedCerts, cert)
			return nil
		},
	}

	var updatedState *model.MonitorState
	ss := &mockStateStore{
		getFn: func(ctx context.Context) (*model.MonitorState, error) {
			if callCount == 0 {
				return &model.MonitorState{LastProcessedIndex: 100}, nil
			}
			// Second call: caught up to tree size — forces cache path
			return &model.MonitorState{
				LastProcessedIndex: 200,
				CertsInLastCycle:   1,
			}, nil
		},
		updateFn: func(ctx context.Context, state *model.MonitorState) error {
			updatedState = state
			callCount++
			return nil
		},
	}

	m := New(ct, kw, cc, ss, 10, time.Hour)

	// First batch: no keywords, entries are fetched and cached
	m.processBatch(context.Background())

	if len(storedCerts) != 0 {
		t.Errorf("first batch: expected 0 stored certs (no keywords), got %d", len(storedCerts))
	}

	// Now add a keyword
	keywords = []model.Keyword{{ID: 1, Value: "example"}}

	// Second batch: no new entries, but cached entries get re-matched
	m.processBatch(context.Background())

	if len(storedCerts) != 1 {
		t.Fatalf("second batch: expected 1 stored cert (re-match), got %d", len(storedCerts))
	}
	if storedCerts[0].KeywordID != 1 {
		t.Errorf("storedCerts[0].KeywordID = %d, want 1", storedCerts[0].KeywordID)
	}
	if updatedState == nil {
		t.Fatal("state should be updated on re-match")
	}
	if updatedState.MatchesInLastCycle != 1 {
		t.Errorf("MatchesInLastCycle = %d, want 1", updatedState.MatchesInLastCycle)
	}
}
