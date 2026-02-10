package handler

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
)

type certificateStore interface {
	ListPaginated(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error)
	ExportAll(ctx context.Context) ([]model.MatchedCertificate, error)
}

type CertificateHandler struct {
	repo certificateStore
}

func NewCertificateHandler(repo certificateStore) *CertificateHandler {
	return &CertificateHandler{repo: repo}
}

func (h *CertificateHandler) RegisterRoutes(r chi.Router) {
	r.Get("/certificates", h.List)
	r.Get("/certificates/export", h.Export)
}

func (h *CertificateHandler) List(w http.ResponseWriter, r *http.Request) {
	page := 1
	perPage := 20
	keywordID := 0

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}
	if v := r.URL.Query().Get("keyword"); v != "" {
		if kid, err := strconv.Atoi(v); err == nil {
			keywordID = kid
		}
	}

	certs, total, err := h.repo.ListPaginated(r.Context(), page, perPage, keywordID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list certificates")
		return
	}

	if certs == nil {
		certs = []model.MatchedCertificate{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"certificates": certs,
		"total":        total,
		"page":         page,
		"per_page":     perPage,
	})
}

func (h *CertificateHandler) Export(w http.ResponseWriter, r *http.Request) {
	certs, err := h.repo.ExportAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to export certificates")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="matched_certificates.csv"`)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{
		"id", "serial_number", "common_name", "sans", "issuer",
		"not_before", "not_after", "keyword", "matched_domain",
		"ct_log_index", "discovered_at",
	})

	for _, c := range certs {
		writer.Write([]string{
			strconv.Itoa(c.ID),
			c.SerialNumber,
			c.CommonName,
			strings.Join(c.SANs, ";"),
			c.Issuer,
			c.NotBefore.Format(time.RFC3339),
			c.NotAfter.Format(time.RFC3339),
			c.KeywordValue,
			c.MatchedDomain,
			strconv.FormatInt(c.CTLogIndex, 10),
			c.DiscoveredAt.Format(time.RFC3339),
		})
	}
}
