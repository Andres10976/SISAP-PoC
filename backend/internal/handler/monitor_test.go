package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/monitor"
)

type mockMonitorService struct {
	startFn     func(ctx context.Context) error
	stopFn      func(ctx context.Context) error
	isRunningFn func() bool
}

func (m *mockMonitorService) Start(ctx context.Context) error { return m.startFn(ctx) }
func (m *mockMonitorService) Stop(ctx context.Context) error  { return m.stopFn(ctx) }
func (m *mockMonitorService) IsRunning() bool                 { return m.isRunningFn() }

type mockMonitorStateStore struct {
	getFn func(ctx context.Context) (*model.MonitorState, error)
}

func (m *mockMonitorStateStore) Get(ctx context.Context) (*model.MonitorState, error) {
	return m.getFn(ctx)
}

func TestMonitorStatus_Success(t *testing.T) {
	now := time.Now()
	h := NewMonitorHandler(
		&mockMonitorService{},
		&mockMonitorStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return &model.MonitorState{
					IsRunning:          true,
					LastProcessedIndex: 500,
					TotalProcessed:     1000,
					UpdatedAt:          now,
				}, nil
			},
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/monitor/status", nil)
	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMonitorStatus_Error(t *testing.T) {
	h := NewMonitorHandler(
		&mockMonitorService{},
		&mockMonitorStateStore{
			getFn: func(ctx context.Context) (*model.MonitorState, error) {
				return nil, errors.New("db error")
			},
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/monitor/status", nil)
	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestMonitorStart_Success(t *testing.T) {
	h := NewMonitorHandler(
		&mockMonitorService{
			startFn: func(ctx context.Context) error { return nil },
		},
		&mockMonitorStateStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/monitor/start", nil)
	rec := httptest.NewRecorder()
	h.Start(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMonitorStart_AlreadyRunning(t *testing.T) {
	h := NewMonitorHandler(
		&mockMonitorService{
			startFn: func(ctx context.Context) error { return monitor.ErrAlreadyRunning },
		},
		&mockMonitorStateStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/monitor/start", nil)
	rec := httptest.NewRecorder()
	h.Start(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestMonitorStart_Error(t *testing.T) {
	h := NewMonitorHandler(
		&mockMonitorService{
			startFn: func(ctx context.Context) error { return errors.New("start failed") },
		},
		&mockMonitorStateStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/monitor/start", nil)
	rec := httptest.NewRecorder()
	h.Start(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestMonitorStop_Success(t *testing.T) {
	h := NewMonitorHandler(
		&mockMonitorService{
			stopFn: func(ctx context.Context) error { return nil },
		},
		&mockMonitorStateStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/monitor/stop", nil)
	rec := httptest.NewRecorder()
	h.Stop(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMonitorStop_NotRunning(t *testing.T) {
	h := NewMonitorHandler(
		&mockMonitorService{
			stopFn: func(ctx context.Context) error { return monitor.ErrNotRunning },
		},
		&mockMonitorStateStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/monitor/stop", nil)
	rec := httptest.NewRecorder()
	h.Stop(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestMonitorStop_Error(t *testing.T) {
	h := NewMonitorHandler(
		&mockMonitorService{
			stopFn: func(ctx context.Context) error { return errors.New("stop failed") },
		},
		&mockMonitorStateStore{},
	)

	req := httptest.NewRequest(http.MethodPost, "/monitor/stop", nil)
	rec := httptest.NewRecorder()
	h.Stop(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
