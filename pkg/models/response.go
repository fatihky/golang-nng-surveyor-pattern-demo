package models

import (
	"time"
)

type Response struct {
	CreatedAt      time.Time     `json:"created_at"`
	ResponseTimeMs int64         `json:"response_time_ms"`
	ResponseTimeNs int64         `json:"response_time_ns"`
	ServiceName    string        `json:"service_name"`
	Query          string        `json:"query"`
	Results        []interface{} `json:"results"`
}
