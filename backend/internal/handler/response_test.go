package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"key": "value"})

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["key"] != "value" {
		t.Errorf("body[key] = %q, want %q", body["key"], "value")
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "something went wrong")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "something went wrong" {
		t.Errorf("body[error] = %q, want %q", body["error"], "something went wrong")
	}
}

func TestIsDuplicateKeyError_StringMatch(t *testing.T) {
	err := errors.New("duplicate key value violates unique constraint")
	if !isDuplicateKeyError(err) {
		t.Error("expected true for string containing 'duplicate key'")
	}
}

func TestIsDuplicateKeyError_NoMatch(t *testing.T) {
	err := errors.New("some other error")
	if isDuplicateKeyError(err) {
		t.Error("expected false for unrelated error")
	}
}

func TestIsDuplicateKeyError_PgError23505(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505"}
	if !isDuplicateKeyError(pgErr) {
		t.Error("expected true for PgError code 23505")
	}
}

func TestIsDuplicateKeyError_PgErrorOtherCode(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23503"} // foreign_key_violation
	if isDuplicateKeyError(pgErr) {
		t.Error("expected false for PgError code 23503")
	}
}
