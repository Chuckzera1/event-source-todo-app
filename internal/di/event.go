package di

import (
	eventapi "github.com/Chuckzera1/event-source-todo-app/internal/api/http/event"
	"github.com/Chuckzera1/event-source-todo-app/internal/application/repositories"
	"github.com/Chuckzera1/event-source-todo-app/internal/application/usecases/event"
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

func NewCreateEventUseCaseDI(db *gorm.DB) event.CreateEventUseCase {
	repo := NewEventRepositoryDI(db)
	return event.NewCreateEventUseCaseImpl(repo)
}

func NewCreateEventHandlerDI(db *gorm.DB) eventapi.CreateEventHandler {
	useCase := NewCreateEventUseCaseDI(db)
	return *eventapi.NewCreateEventHandler(useCase)
}
