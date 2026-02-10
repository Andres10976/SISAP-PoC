package model

import "time"

type MatchedCertificate struct {
	ID            int       `json:"id"`
	SerialNumber  string    `json:"serial_number"`
	CommonName    string    `json:"common_name"`
	SANs          []string  `json:"sans"`
	Issuer        string    `json:"issuer"`
	NotBefore     time.Time `json:"not_before"`
	NotAfter      time.Time `json:"not_after"`
	KeywordID     int       `json:"keyword_id"`
	KeywordValue  string    `json:"keyword_value,omitempty"`
	MatchedDomain string    `json:"matched_domain"`
	CTLogIndex    int64     `json:"ct_log_index"`
	DiscoveredAt  time.Time `json:"discovered_at"`
}
