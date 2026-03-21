package domain

import "time"

type Event struct {
	ID        string
	Aggregate string
	Version   int
	Data      any
	Timestamp time.Time
}
