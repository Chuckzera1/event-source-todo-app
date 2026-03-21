package di

import (
	irepositories "github.com/Chuckzera1/event-source-todo-app/internal/application/repositories"
	rtask "github.com/Chuckzera1/event-source-todo-app/internal/infrastructure/gorm/gormrepo/task"
	"gorm.io/gorm"
)

type TaskRepositoryDI struct {
	irepositories.ITaskRepository
}

func NewTaskRepositoryDI(db *gorm.DB) TaskRepositoryDI {
	createTaskRepository := rtask.CreateTaskRepositoryImpl{DB: db}
	return TaskRepositoryDI{
		&createTaskRepository,
	}
}