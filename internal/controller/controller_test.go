package controller_test

import (
	"context"
	"testing"

	"github.com/cnap-oss/app/internal/controller"
	"github.com/cnap-oss/app/internal/storage"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestController(t *testing.T) (*controller.Controller, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, storage.AutoMigrate(db))

	repo, err := storage.NewRepository(db)
	require.NoError(t, err)

	ctrl := controller.NewController(zaptest.NewLogger(t), repo)

	cleanup := func() {
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
	}

	return ctrl, cleanup
}

func TestControllerCreateAndGetAgent(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	ctx := context.Background()

	require.NoError(t, ctrl.CreateAgent(ctx, "agent-x", "Test agent", "gpt-4", "Test prompt"))

	info, err := ctrl.GetAgentInfo(ctx, "agent-x")
	require.NoError(t, err)
	require.Equal(t, "agent-x", info.Name)
	require.Equal(t, "Test agent", info.Description)
	require.Equal(t, "gpt-4", info.Model)
	require.Equal(t, "Test prompt", info.Prompt)
	require.Equal(t, storage.AgentStatusActive, info.Status)
}

func TestControllerListAgents(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-a", "Agent A", "gpt-4", "Prompt A"))
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-b", "Agent B", "gpt-3", "Prompt B"))

	agents, err := ctrl.ListAgents(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"agent-a", "agent-b"}, agents)
}
