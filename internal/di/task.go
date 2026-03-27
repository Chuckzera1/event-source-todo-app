package di

import (
	irepositories "github.com/Chuckzera1/event-source-todo-app/internal/application/repositories"
	rtask "github.com/Chuckzera1/event-source-todo-app/internal/infrastructure/gorm/gormrepo/task"
	"gorm.io/gorm"
)

type taskRepositoryDI struct {
	irepositories.ICreateTaskRepository
}

func NewTaskRepositoryDI(db *gorm.DB) irepositories.ITaskRepository {
	return &taskRepositoryDI{
		rtask.NewCreateTaskRepositoryImpl(db),
	}
}
