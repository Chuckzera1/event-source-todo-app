//go:build integration

package rtask

import (
	"context"
	"testing"
	"time"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
	"github.com/Chuckzera1/event-source-todo-app/internal/infrastructure"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func integrationPostgresDSN(t *testing.T, ctx context.Context) string {
	t.Helper()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("todo_test"),
		postgres.WithUsername("user"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		termCtx := context.Background()
		if termErr := pgContainer.Terminate(termCtx); termErr != nil {
			t.Logf("terminate postgres container: %v", termErr)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	return connStr
}

func TestCreateTaskRepositoryImpl_CreateTask_ShouldPersistRow_WhenTaskIsValid(t *testing.T) {
	ctx := context.Background()

	connStr := integrationPostgresDSN(t, ctx)

	db, err := infrastructure.NewGorm(connStr)
	require.NoError(t, err)

	err = db.WithContext(ctx).AutoMigrate(&TaskModel{})
	require.NoError(t, err)

	repo := &CreateTaskRepositoryImpl{DB: db}
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

	var got TaskModel
	err = db.WithContext(ctx).Where("title = ?", title).First(&got).Error
	require.NoError(t, err)
	require.Equal(t, task.Description, got.Description)
	require.Equal(t, task.Completed, got.Completed)
	require.Equal(t, string(task.Type), got.Type)
	require.NotEqual(t, uuid.Nil, got.ID)
}
