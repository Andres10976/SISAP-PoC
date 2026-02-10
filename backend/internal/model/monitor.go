package model

import "time"

type MonitorState struct {
	LastProcessedIndex     int64      `json:"last_processed_index"`
	LastTreeSize           int64      `json:"last_tree_size"`
	LastRunAt              *time.Time `json:"last_run_at"`
	TotalProcessed         int64      `json:"total_processed"`
	CertsInLastCycle       int        `json:"certs_in_last_cycle"`
	MatchesInLastCycle     int        `json:"matches_in_last_cycle"`
	ParseErrorsInLastCycle int        `json:"parse_errors_in_last_cycle"`
	IsRunning              bool       `json:"is_running"`
	UpdatedAt              time.Time  `json:"updated_at"`
}
