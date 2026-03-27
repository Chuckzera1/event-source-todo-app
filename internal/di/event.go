package di

import (
	"github.com/Chuckzera1/event-source-todo-app/internal/application/repositories"
	eventrepo "github.com/Chuckzera1/event-source-todo-app/internal/infrastructure/gorm/gormrepo/event"
	"gorm.io/gorm"
)

type eventRepositoryDI struct {
	repositories.CreateEventRepository
}

func NewEventRepositoryDI(db *gorm.DB) repositories.EventRepository {
	return &eventRepositoryDI{
		eventrepo.NewCreateEventRepositoryImpl(db),
	}
}
