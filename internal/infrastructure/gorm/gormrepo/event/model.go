package event

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventModel struct {
	gorm.Model
	ID        uuid.UUID `gorm:"primaryKey"`
	Aggregate string    `gorm:"not null"`
	Version   int       `gorm:"not null"`
	Data      any       `gorm:"not null"`
	Timestamp time.Time `gorm:"not null"`
}

func (EventModel) TableName() string {
	return "events"
}
