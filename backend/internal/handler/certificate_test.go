package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
)

type mockCertificateStore struct {
	listPaginatedFn func(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error)
	exportAllFn     func(ctx context.Context) ([]model.MatchedCertificate, error)
}

func (m *mockCertificateStore) ListPaginated(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
	return m.listPaginatedFn(ctx, page, perPage, keywordID)
}
func (m *mockCertificateStore) ExportAll(ctx context.Context) ([]model.MatchedCertificate, error) {
	return m.exportAllFn(ctx)
}

func sampleCert() model.MatchedCertificate {
	return model.MatchedCertificate{
		ID:            1,
		SerialNumber:  "abc123",
		CommonName:    "example.com",
		SANs:          []string{"www.example.com"},
		Issuer:        "Let's Encrypt",
		NotBefore:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		KeywordID:     1,
		KeywordValue:  "example",
		MatchedDomain: "example.com",
		CTLogIndex:    999,
		DiscoveredAt:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestCertificateList_Defaults(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		listPaginatedFn: func(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
			if page != 1 {
				t.Errorf("page = %d, want 1", page)
			}
			if perPage != 20 {
				t.Errorf("perPage = %d, want 20", perPage)
			}
			if keywordID != 0 {
				t.Errorf("keywordID = %d, want 0", keywordID)
			}
			return []model.MatchedCertificate{sampleCert()}, 1, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(rec.Body).Decode(&body)
	var certs []model.MatchedCertificate
	json.Unmarshal(body["certificates"], &certs)
	if len(certs) != 1 {
		t.Errorf("got %d certs, want 1", len(certs))
	}
}

func TestCertificateList_CustomPagination(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		listPaginatedFn: func(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
			if page != 3 {
				t.Errorf("page = %d, want 3", page)
			}
			if perPage != 50 {
				t.Errorf("perPage = %d, want 50", perPage)
			}
			return nil, 0, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates?page=3&per_page=50", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCertificateList_KeywordFilter(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		listPaginatedFn: func(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
			if keywordID != 5 {
				t.Errorf("keywordID = %d, want 5", keywordID)
			}
			return nil, 0, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates?keyword=5", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCertificateList_InvalidPage(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		listPaginatedFn: func(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
			if page != 1 {
				t.Errorf("page = %d, want default 1 for invalid input", page)
			}
			return nil, 0, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates?page=-1", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCertificateList_PerPageClamp(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		listPaginatedFn: func(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
			if perPage != 20 {
				t.Errorf("perPage = %d, want default 20 for per_page>100", perPage)
			}
			return nil, 0, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates?per_page=200", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCertificateList_NilCerts(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		listPaginatedFn: func(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
			return nil, 0, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(rec.Body).Decode(&body)
	var certs []model.MatchedCertificate
	json.Unmarshal(body["certificates"], &certs)
	if certs == nil {
		t.Error("certificates should be empty array, not null")
	}
}

func TestCertificateList_Error(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		listPaginatedFn: func(ctx context.Context, page, perPage, keywordID int) ([]model.MatchedCertificate, int, error) {
			return nil, 0, errors.New("db error")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestCertificateExport_Success(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		exportAllFn: func(ctx context.Context) ([]model.MatchedCertificate, error) {
			return []model.MatchedCertificate{sampleCert()}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates/export", nil)
	rec := httptest.NewRecorder()
	h.Export(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Content-Type = %q, want text/csv", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "matched_certificates.csv") {
		t.Errorf("Content-Disposition = %q, want filename", cd)
	}

	reader := csv.NewReader(rec.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	// Header + 1 data row
	if len(records) != 2 {
		t.Errorf("got %d CSV rows, want 2 (header + 1 data)", len(records))
	}
}

func TestCertificateExport_Empty(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		exportAllFn: func(ctx context.Context) ([]model.MatchedCertificate, error) {
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates/export", nil)
	rec := httptest.NewRecorder()
	h.Export(rec, req)

	reader := csv.NewReader(rec.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	// Header only
	if len(records) != 1 {
		t.Errorf("got %d CSV rows, want 1 (header only)", len(records))
	}
}

func TestCertificateExport_Error(t *testing.T) {
	h := NewCertificateHandler(&mockCertificateStore{
		exportAllFn: func(ctx context.Context) ([]model.MatchedCertificate, error) {
			return nil, errors.New("db error")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/certificates/export", nil)
	rec := httptest.NewRecorder()
	h.Export(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
