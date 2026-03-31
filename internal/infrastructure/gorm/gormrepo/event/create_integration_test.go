package event_test

import (
	"context"
	"testing"

	"github.com/Chuckzera1/event-source-todo-app/internal/domain"
	"github.com/Chuckzera1/event-source-todo-app/internal/infrastructure"
	eventrepo "github.com/Chuckzera1/event-source-todo-app/internal/infrastructure/gorm/gormrepo/event"
	"github.com/Chuckzera1/event-source-todo-app/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestCreateEventRepositoryImpl_CreateEvent_ShouldPersistRow_WhenEventIsValid(t *testing.T) {
	ctx := context.Background()

	connStr := testutils.PostgresDSN(t, ctx)

	db, err := infrastructure.NewGorm(connStr)
	require.NoError(t, err)

	err = db.WithContext(ctx).AutoMigrate(&eventrepo.EventModel{})
	require.NoError(t, err)

	repo := eventrepo.NewCreateEventRepositoryImpl(db)
	event := domain.Event{
		Aggregate: "test",
		Version:   1,
		Data:      "test",
	}

	err = repo.CreateEvent(ctx, event)
	require.NoError(t, err)

	var got eventrepo.EventModel
	err = db.WithContext(ctx).Where("aggregate = ?", event.Aggregate).First(&got).Error
	require.NoError(t, err)
	require.Equal(t, event.Aggregate, got.Aggregate)
	require.Equal(t, event.Version, got.Version)
	require.Equal(t, event.Data, got.Data)
	require.False(t, got.Timestamp.IsZero())
}
