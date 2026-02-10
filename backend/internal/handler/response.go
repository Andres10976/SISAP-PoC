package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func isDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if ok := errors.As(err, &pgErr); ok {
		return pgErr.Code == "23505" // unique_violation
	}
	return strings.Contains(err.Error(), "duplicate key")
}
