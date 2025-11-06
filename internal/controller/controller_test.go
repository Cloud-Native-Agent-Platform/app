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

	require.NoError(t, ctrl.CreateAgent(ctx, "agent-x"))

	info, err := ctrl.GetAgentInfo(ctx, "agent-x")
	require.NoError(t, err)
	require.Equal(t, "agent-x", info.Name)
	require.Equal(t, storage.AgentStatusActive, info.Status)
}

func TestControllerListAgents(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-a"))
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-b"))

	agents, err := ctrl.ListAgents(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"agent-a", "agent-b"}, agents)
}

func TestControllerCreateAndGetTask(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	ctx := context.Background()

	// Agent 먼저 생성
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-1"))

	// Task 생성
	require.NoError(t, ctrl.CreateTask(ctx, "agent-1", "task-1"))

	// Task 조회
	task, err := ctrl.GetTask(ctx, "task-1")
	require.NoError(t, err)
	require.Equal(t, "task-1", task.TaskID)
	require.Equal(t, "agent-1", task.AgentID)
	require.Equal(t, storage.TaskStatusPending, task.Status)
}

func TestControllerCreateTaskWithoutAgent(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	ctx := context.Background()

	// Agent 없이 Task 생성 시도
	err := ctrl.CreateTask(ctx, "non-existent-agent", "task-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "agent not found")
}

func TestControllerUpdateTaskStatus(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	ctx := context.Background()

	// Agent 및 Task 생성
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-1"))
	require.NoError(t, ctrl.CreateTask(ctx, "agent-1", "task-1"))

	// 상태 업데이트
	require.NoError(t, ctrl.UpdateTaskStatus(ctx, "task-1", storage.TaskStatusRunning))

	// 확인
	task, err := ctrl.GetTask(ctx, "task-1")
	require.NoError(t, err)
	require.Equal(t, storage.TaskStatusRunning, task.Status)

	// 다시 업데이트
	require.NoError(t, ctrl.UpdateTaskStatus(ctx, "task-1", storage.TaskStatusCompleted))

	task, err = ctrl.GetTask(ctx, "task-1")
	require.NoError(t, err)
	require.Equal(t, storage.TaskStatusCompleted, task.Status)
}

func TestControllerListTasksByAgent(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	ctx := context.Background()

	// Agent 생성
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-1"))
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-2"))

	// Task 생성
	require.NoError(t, ctrl.CreateTask(ctx, "agent-1", "task-1"))
	require.NoError(t, ctrl.CreateTask(ctx, "agent-1", "task-2"))
	require.NoError(t, ctrl.CreateTask(ctx, "agent-2", "task-3"))

	// agent-1의 Task 조회
	tasks, err := ctrl.ListTasksByAgent(ctx, "agent-1")
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	taskIDs := make([]string, len(tasks))
	for i, task := range tasks {
		taskIDs[i] = task.TaskID
	}
	require.ElementsMatch(t, []string{"task-1", "task-2"}, taskIDs)

	// agent-2의 Task 조회
	tasks, err = ctrl.ListTasksByAgent(ctx, "agent-2")
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "task-3", tasks[0].TaskID)
}

func TestControllerGetTaskInfo(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	ctx := context.Background()

	// Agent 및 Task 생성
	require.NoError(t, ctrl.CreateAgent(ctx, "agent-1"))
	require.NoError(t, ctrl.CreateTask(ctx, "agent-1", "task-1"))

	// TaskInfo 조회
	info, err := ctrl.GetTaskInfo(ctx, "task-1")
	require.NoError(t, err)
	require.Equal(t, "task-1", info.TaskID)
	require.Equal(t, "agent-1", info.AgentID)
	require.Equal(t, storage.TaskStatusPending, info.Status)
	require.False(t, info.CreatedAt.IsZero())
	require.False(t, info.UpdatedAt.IsZero())
}

func TestControllerValidateTask(t *testing.T) {
	ctrl, cleanup := newTestController(t)
	defer cleanup()

	// 빈 Task ID
	err := ctrl.ValidateTask("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be empty")

	// 너무 긴 Task ID
	longTaskID := string(make([]byte, 65))
	err = ctrl.ValidateTask(longTaskID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too long")

	// 정상 Task ID
	err = ctrl.ValidateTask("valid-task-id")
	require.NoError(t, err)
}
