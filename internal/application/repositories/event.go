package repositories

import (
	"context"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
)

type CreateEventRepository interface {
	CreateEvent(ctx context.Context, event domain.Event) error
}

type EventRepository interface {
	CreateEventRepository
}