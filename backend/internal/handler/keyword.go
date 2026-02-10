package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/repository"
)

type keywordStore interface {
	List(ctx context.Context) ([]model.Keyword, error)
	Create(ctx context.Context, value string) (*model.Keyword, error)
	Delete(ctx context.Context, id int) error
}

type KeywordHandler struct {
	repo keywordStore
}

func NewKeywordHandler(repo keywordStore) *KeywordHandler {
	return &KeywordHandler{repo: repo}
}

func (h *KeywordHandler) RegisterRoutes(r chi.Router) {
	r.Get("/keywords", h.List)
	r.Post("/keywords", h.Create)
	r.Delete("/keywords/{id}", h.Delete)
}

func (h *KeywordHandler) List(w http.ResponseWriter, r *http.Request) {
	keywords, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list keywords")
		return
	}
	if keywords == nil {
		keywords = []model.Keyword{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"keywords": keywords})
}

func (h *KeywordHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB

	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	value := strings.TrimSpace(req.Value)
	if value == "" {
		writeError(w, http.StatusBadRequest, "keyword value cannot be empty")
		return
	}
	if len(value) < 3 {
		writeError(w, http.StatusBadRequest, "keyword must be at least 3 characters")
		return
	}

	kw, err := h.repo.Create(r.Context(), value)
	if err != nil {
		if isDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "keyword already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create keyword")
		return
	}

	writeJSON(w, http.StatusCreated, kw)
}

func (h *KeywordHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid keyword id")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "keyword not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete keyword")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
