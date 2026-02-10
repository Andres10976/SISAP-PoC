package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/monitor"
)

type monitorService interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
}

type monitorStateStore interface {
	Get(ctx context.Context) (*model.MonitorState, error)
}

type MonitorHandler struct {
	monitor monitorService
	repo    monitorStateStore
}

func NewMonitorHandler(mon monitorService, repo monitorStateStore) *MonitorHandler {
	return &MonitorHandler{monitor: mon, repo: repo}
}

func (h *MonitorHandler) RegisterRoutes(r chi.Router) {
	r.Get("/monitor/status", h.Status)
	r.Post("/monitor/start", h.Start)
	r.Post("/monitor/stop", h.Stop)
}

func (h *MonitorHandler) Status(w http.ResponseWriter, r *http.Request) {
	state, err := h.repo.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get monitor status")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *MonitorHandler) Start(w http.ResponseWriter, r *http.Request) {
	if err := h.monitor.Start(r.Context()); err != nil {
		if errors.Is(err, monitor.ErrAlreadyRunning) {
			writeError(w, http.StatusConflict, "monitor is already running")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to start monitor")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Monitor started"})
}

func (h *MonitorHandler) Stop(w http.ResponseWriter, r *http.Request) {
	if err := h.monitor.Stop(r.Context()); err != nil {
		if errors.Is(err, monitor.ErrNotRunning) {
			writeError(w, http.StatusConflict, "monitor is not running")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to stop monitor")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Monitor stopped"})
}
