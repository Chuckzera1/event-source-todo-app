package irepositories

import (
	"context"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
)

type ICreateEventRepository interface {
	CreateEvent(ctx context.Context, event domain.Event) error
}

type IEventRepository interface {
	ICreateEventRepository
}