package ctlog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetSTH_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ct/v1/get-sth" {
			t.Errorf("path = %q, want /ct/v1/get-sth", r.URL.Path)
		}
		json.NewEncoder(w).Encode(STH{TreeSize: 1000, Timestamp: 123456})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	sth, err := client.GetSTH(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sth.TreeSize != 1000 {
		t.Errorf("TreeSize = %d, want 1000", sth.TreeSize)
	}
	if sth.Timestamp != 123456 {
		t.Errorf("Timestamp = %d, want 123456", sth.Timestamp)
	}
}

func TestGetSTH_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	_, err := client.GetSTH(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %q, want mention of status 500", err.Error())
	}
}

func TestGetSTH_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	_, err := client.GetSTH(context.Background())
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestGetSTH_CanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(STH{TreeSize: 1})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewClient(srv.URL)
	_, err := client.GetSTH(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestGetEntries_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ct/v1/get-entries" {
			t.Errorf("path = %q, want /ct/v1/get-entries", r.URL.Path)
		}
		resp := struct {
			Entries []RawEntry `json:"entries"`
		}{
			Entries: []RawEntry{
				{LeafInput: []byte("leaf1"), ExtraData: []byte("extra1")},
				{LeafInput: []byte("leaf2"), ExtraData: []byte("extra2")},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	entries, err := client.GetEntries(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func TestGetEntries_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string][]RawEntry{"entries": {}})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	entries, err := client.GetEntries(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestGetEntries_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("start"); got != "10" {
			t.Errorf("start = %q, want %q", got, "10")
		}
		if got := r.URL.Query().Get("end"); got != "20" {
			t.Errorf("end = %q, want %q", got, "20")
		}
		json.NewEncoder(w).Encode(map[string][]RawEntry{"entries": {}})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	_, err := client.GetEntries(context.Background(), 10, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetEntries_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	_, err := client.GetEntries(context.Background(), 0, 10)
	if err == nil {
		t.Fatal("expected error for 502 response")
	}
}
