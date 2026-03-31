package di

import (
	"github.com/Chuckzera1/event-source-todo-app/internal/application/repositories"
	taskrepo "github.com/Chuckzera1/event-source-todo-app/internal/infrastructure/gorm/gormrepo/task"
	"gorm.io/gorm"
)

type taskRepositoryDI struct {
	repositories.CreateTaskRepository
}

func NewTaskRepositoryDI(db *gorm.DB) repositories.TaskRepository {
	return &taskRepositoryDI{
		taskrepo.NewCreateTaskRepositoryImpl(db),
	}
}
