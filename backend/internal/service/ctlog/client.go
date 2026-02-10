package ctlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// STH represents a Signed Tree Head response (RFC 6962 §4.3).
type STH struct {
	TreeSize  int64  `json:"tree_size"`
	Timestamp int64  `json:"timestamp"`
	RootHash  string `json:"sha256_root_hash"`
}

// RawEntry represents a single entry from get-entries (RFC 6962 §4.6).
type RawEntry struct {
	LeafInput []byte `json:"leaf_input"`
	ExtraData []byte `json:"extra_data"`
}

// Client talks to a Certificate Transparency log over HTTP.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetSTH retrieves the latest Signed Tree Head.
func (c *Client) GetSTH(ctx context.Context) (*STH, error) {
	url := fmt.Sprintf("%s/ct/v1/get-sth", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create STH request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch STH: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("STH returned status %d", resp.StatusCode)
	}

	var sth STH
	if err := json.NewDecoder(resp.Body).Decode(&sth); err != nil {
		return nil, fmt.Errorf("decode STH: %w", err)
	}
	return &sth, nil
}

// GetEntries retrieves log entries in range [start, end] inclusive.
func (c *Client) GetEntries(ctx context.Context, start, end int64) ([]RawEntry, error) {
	url := fmt.Sprintf("%s/ct/v1/get-entries?start=%d&end=%d", c.baseURL, start, end)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create entries request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch entries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get-entries returned status %d", resp.StatusCode)
	}

	var result struct {
		Entries []RawEntry `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode entries: %w", err)
	}
	return result.Entries, nil
}
