package event

import (
	"context"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createEventRepositoryImpl struct {
	db *gorm.DB
}

func (r *createEventRepositoryImpl) CreateEvent(ctx context.Context, event domain.Event) error {
	return r.db.WithContext(ctx).Create(&EventModel{
		ID:        uuid.New(),
		Aggregate: event.Aggregate,
		Version:   event.Version,
		Data:      event.Data,
		Timestamp: event.Timestamp,
	}).Error
}

func NewCreateEventRepositoryImpl(db *gorm.DB) *createEventRepositoryImpl {
	return &createEventRepositoryImpl{db: db}
}