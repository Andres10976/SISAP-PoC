package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/repository"
)

// mockKeywordStore implements keywordStore for testing.
type mockKeywordStore struct {
	listFn   func(ctx context.Context) ([]model.Keyword, error)
	createFn func(ctx context.Context, value string) (*model.Keyword, error)
	deleteFn func(ctx context.Context, id int) error
}

func (m *mockKeywordStore) List(ctx context.Context) ([]model.Keyword, error) {
	return m.listFn(ctx)
}
func (m *mockKeywordStore) Create(ctx context.Context, value string) (*model.Keyword, error) {
	return m.createFn(ctx, value)
}
func (m *mockKeywordStore) Delete(ctx context.Context, id int) error {
	return m.deleteFn(ctx, id)
}

func TestKeywordList_Success(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		listFn: func(ctx context.Context) ([]model.Keyword, error) {
			return []model.Keyword{
				{ID: 1, Value: "example", CreatedAt: time.Now()},
			}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/keywords", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(rec.Body).Decode(&body)
	var keywords []model.Keyword
	json.Unmarshal(body["keywords"], &keywords)
	if len(keywords) != 1 {
		t.Errorf("got %d keywords, want 1", len(keywords))
	}
}

func TestKeywordList_Empty(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		listFn: func(ctx context.Context) ([]model.Keyword, error) {
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/keywords", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(rec.Body).Decode(&body)
	var keywords []model.Keyword
	json.Unmarshal(body["keywords"], &keywords)
	if len(keywords) != 0 {
		t.Errorf("got %d keywords, want 0", len(keywords))
	}
}

func TestKeywordList_Error(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		listFn: func(ctx context.Context) ([]model.Keyword, error) {
			return nil, errors.New("db error")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/keywords", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestKeywordCreate_Success(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		createFn: func(ctx context.Context, value string) (*model.Keyword, error) {
			return &model.Keyword{ID: 1, Value: value, CreatedAt: time.Now()}, nil
		},
	})

	body := strings.NewReader(`{"value":"example"}`)
	req := httptest.NewRequest(http.MethodPost, "/keywords", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var kw model.Keyword
	json.NewDecoder(rec.Body).Decode(&kw)
	if kw.Value != "example" {
		t.Errorf("Value = %q, want %q", kw.Value, "example")
	}
}

func TestKeywordCreate_EmptyValue(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{})

	body := strings.NewReader(`{"value":"   "}`)
	req := httptest.NewRequest(http.MethodPost, "/keywords", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestKeywordCreate_TooShort(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{})

	body := strings.NewReader(`{"value":"ab"}`)
	req := httptest.NewRequest(http.MethodPost, "/keywords", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestKeywordCreate_InvalidJSON(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{})

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/keywords", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestKeywordCreate_Duplicate(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		createFn: func(ctx context.Context, value string) (*model.Keyword, error) {
			return nil, errors.New("duplicate key value violates unique constraint")
		},
	})

	body := strings.NewReader(`{"value":"example"}`)
	req := httptest.NewRequest(http.MethodPost, "/keywords", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestKeywordCreate_Error(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		createFn: func(ctx context.Context, value string) (*model.Keyword, error) {
			return nil, errors.New("db error")
		},
	})

	body := strings.NewReader(`{"value":"example"}`)
	req := httptest.NewRequest(http.MethodPost, "/keywords", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

// chiRequest creates an http.Request with chi URL params set.
func chiRequest(method, target string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestKeywordDelete_Success(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		deleteFn: func(ctx context.Context, id int) error {
			if id != 42 {
				t.Errorf("id = %d, want 42", id)
			}
			return nil
		},
	})

	req := chiRequest(http.MethodDelete, "/keywords/42", map[string]string{"id": "42"})
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestKeywordDelete_InvalidID(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{})

	req := chiRequest(http.MethodDelete, "/keywords/abc", map[string]string{"id": "abc"})
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestKeywordDelete_NotFound(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		deleteFn: func(ctx context.Context, id int) error {
			return repository.ErrNotFound
		},
	})

	req := chiRequest(http.MethodDelete, "/keywords/1", map[string]string{"id": "1"})
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestKeywordDelete_Error(t *testing.T) {
	h := NewKeywordHandler(&mockKeywordStore{
		deleteFn: func(ctx context.Context, id int) error {
			return errors.New("db error")
		},
	})

	req := chiRequest(http.MethodDelete, "/keywords/1", map[string]string{"id": "1"})
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
