package models

import (
	"time"
)

type Query struct {
	CreatedAt time.Time `json:"created_at"`
	Protocol  string    `json:"protocol"`
	Query     string    `json:"query"`
	Payload   string    `json:"payload"`
}
