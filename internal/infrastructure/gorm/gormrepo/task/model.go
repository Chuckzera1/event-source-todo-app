package rtask

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskModel struct {
	gorm.Model
	ID          uuid.UUID `gorm:"primaryKey"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"not null"`
	Completed   bool      `gorm:"not null"`
	Type        string    `gorm:"not null"`
}

func (TaskModel) TableName() string {
	return "tasks"
}
