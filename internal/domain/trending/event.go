package trending

import "time"

type Event struct {
	RequestID string    `json:"request_id"`
	Query     string    `json:"query"`
	Timestamp time.Time `json:"timestamp"`
}
