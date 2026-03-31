//go:build integration

package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
	"github.com/Chuckzera1/event-source-todo-app/internal/infrastructure"
	taskrepo "github.com/Chuckzera1/event-source-todo-app/internal/infrastructure/gorm/gormrepo/task"
	"github.com/Chuckzera1/event-source-todo-app/internal/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateTaskRepositoryImpl_CreateTask_ShouldPersistRow_WhenTaskIsValid(t *testing.T) {
	ctx := context.Background()

	connStr := testutils.PostgresDSN(t, ctx)

	db, err := infrastructure.NewGorm(connStr)
	require.NoError(t, err)

	err = db.WithContext(ctx).AutoMigrate(&taskrepo.TaskModel{})
	require.NoError(t, err)

	repo := taskrepo.NewCreateTaskRepositoryImpl(db)
	title := "integration-" + uuid.New().String()
	task := domain.Task{
		Title:       title,
		Description: "desc",
		Completed:   false,
		Type:        domain.TaskTypeOther,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err = repo.CreateTask(ctx, task)
	require.NoError(t, err)

	var got taskrepo.TaskModel
	err = db.WithContext(ctx).Where("title = ?", title).First(&got).Error
	require.NoError(t, err)
	require.Equal(t, task.Description, got.Description)
	require.Equal(t, task.Completed, got.Completed)
	require.Equal(t, string(task.Type), got.Type)
	require.NotEqual(t, uuid.Nil, got.ID)
}
