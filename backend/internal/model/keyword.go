package model

import "time"

type Keyword struct {
	ID        int       `json:"id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
