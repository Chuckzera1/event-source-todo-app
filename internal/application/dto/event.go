package dto

import "time"

type CreateEventRequest struct {
	Aggregate string `json:"aggregate"`
	Version   int    `json:"version"`
	Data      any    `json:"data"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}
