package rtask

import (
	"context"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateTaskRepositoryImpl struct {
	DB *gorm.DB
}

func (c *CreateTaskRepositoryImpl) CreateTask(ctx context.Context, task domain.Task) error {
	return c.DB.WithContext(ctx).Create(&TaskModel{
		ID:          uuid.New(),
		Title:       task.Title,
		Description: task.Description,
		Completed:   task.Completed,
		Type:        string(task.Type),
	}).Error
}
