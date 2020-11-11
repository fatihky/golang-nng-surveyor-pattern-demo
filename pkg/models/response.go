package models

import (
	"time"
)

type Response struct {
	CreatedAt   time.Time     `json:"created_at"`
	ServiceName string        `json:"service_name"`
	Query       string        `json:"query"`
	Results     []interface{} `json:"results"`
}
