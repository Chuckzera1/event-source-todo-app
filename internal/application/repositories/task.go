package repositories

import (
	"context"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
)

type CreateTaskRepository interface {
	CreateTask(ctx context.Context, task domain.Task) error
}

type TaskRepository interface {
	CreateTaskRepository
}
