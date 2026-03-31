package event

import (
	"context"

	"github.com/Chuckzera1/event-source-todo-app/internal/application/repositories"
	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
)

type CreateEventUseCase interface {
	Execute(ctx context.Context, event domain.Event) error
}

type createEventUseCaseImpl struct {
	repo repositories.CreateEventRepository
}

func (uc *createEventUseCaseImpl) Execute(ctx context.Context, event domain.Event) error {
	return uc.repo.CreateEvent(ctx, event)
}

func NewCreateEventUseCaseImpl(repo repositories.CreateEventRepository) *createEventUseCaseImpl {
	return &createEventUseCaseImpl{repo: repo}
}
