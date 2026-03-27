package di

import (
	irepositories "github.com/Chuckzera1/event-source-todo-app/internal/application/repositories"
	revent "github.com/Chuckzera1/event-source-todo-app/internal/infrastructure/gorm/gormrepo/event"
	"gorm.io/gorm"
)

type eventRepositoryDI struct {
	irepositories.ICreateEventRepository
}

func NewEventRepositoryDI(db *gorm.DB) irepositories.IEventRepository {
	createEventRepository := revent.NewCreateEventRepositoryImpl(db)
	return &eventRepositoryDI{
		createEventRepository,
	}
}
