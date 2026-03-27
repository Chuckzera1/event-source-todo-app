package irepositories

import (
	"context"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
)

type ICreateTaskRepository interface {
	CreateTask(ctx context.Context, task domain.Task) error
}

type ITaskRepository interface {
	ICreateTaskRepository
}
