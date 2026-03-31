package task

import (
	"context"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createTaskRepositoryImpl struct {
	db *gorm.DB
}

func (c *createTaskRepositoryImpl) CreateTask(ctx context.Context, task domain.Task) error {
	return c.db.WithContext(ctx).Create(&TaskModel{
		ID:          uuid.New(),
		Title:       task.Title,
		Description: task.Description,
		Completed:   task.Completed,
		Type:        string(task.Type),
	}).Error
}

func NewCreateTaskRepositoryImpl(db *gorm.DB) *createTaskRepositoryImpl {
	return &createTaskRepositoryImpl{db: db}
}
