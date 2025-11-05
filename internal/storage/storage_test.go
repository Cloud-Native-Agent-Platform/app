package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/cnap-oss/app/internal/storage"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestRepository(t *testing.T) (*storage.Repository, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, storage.AutoMigrate(db))

	repo, err := storage.NewRepository(db)
	require.NoError(t, err)

	cleanup := func() {
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())
	}
	return repo, cleanup
}

func TestRepositoryAgentLifecycle(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// 에이전트 생성
	err := repo.CreateAgent(ctx, &storage.Agent{
		AgentID: "agent-test",
		Status:  storage.AgentStatusActive,
	})
	require.NoError(t, err)

	// 에이전트 조회
	agent, err := repo.GetAgent(ctx, "agent-test")
	require.NoError(t, err)
	require.Equal(t, storage.AgentStatusActive, agent.Status)

	// 상태 upsert
	err = repo.UpsertAgentStatus(ctx, "agent-test", storage.AgentStatusBusy)
	require.NoError(t, err)

	updated, err := repo.GetAgent(ctx, "agent-test")
	require.NoError(t, err)
	require.Equal(t, storage.AgentStatusBusy, updated.Status)

	// 에이전트 목록 확인
	agents, err := repo.ListAgents(ctx)
	require.NoError(t, err)
	require.Len(t, agents, 1)
	require.Equal(t, "agent-test", agents[0].AgentID)
}

func TestRepositoryTaskAndMessages(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	require.NoError(t, repo.CreateAgent(ctx, &storage.Agent{
		AgentID: "agent-1",
		Status:  storage.AgentStatusActive,
	}))

	task := &storage.Task{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Status:  storage.TaskStatusPending,
	}
	require.NoError(t, repo.CreateTask(ctx, task))

	require.NoError(t, repo.UpsertTaskStatus(ctx, "task-1", "agent-1", storage.TaskStatusRunning))

	fetchedTask, err := repo.GetTask(ctx, "task-1")
	require.NoError(t, err)
	require.Equal(t, storage.TaskStatusRunning, fetchedTask.Status)

	// 메시지 인덱스 upsert
	idxPayload := &storage.MessageIndex{
		TaskID:            "task-1",
		ConversationIndex: 1,
		FilePath:          "/tmp/msg1.json",
		UpdatedAt:         time.Now(),
	}
	require.NoError(t, repo.UpsertMessageIndex(ctx, idxPayload))

	idxPayload.FilePath = "/tmp/msg1-updated.json"
	require.NoError(t, repo.UpsertMessageIndex(ctx, idxPayload))

	indexRows, err := repo.ListMessageIndexByTask(ctx, "task-1")
	require.NoError(t, err)
	require.Len(t, indexRows, 1)
	require.Equal(t, "/tmp/msg1-updated.json", indexRows[0].FilePath)
}

func TestRepositoryRunStepsAndCheckpoints(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	require.NoError(t, repo.CreateAgent(ctx, &storage.Agent{
		AgentID: "agent-1",
		Status:  storage.AgentStatusActive,
	}))
	require.NoError(t, repo.CreateTask(ctx, &storage.Task{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Status:  storage.TaskStatusRunning,
	}))

	step := &storage.RunStep{
		TaskID: "task-1",
		StepNo: 1,
		Type:   storage.RunStepTypeModel,
		Status: storage.RunStepStatusRunning,
	}
	require.NoError(t, repo.UpsertRunStep(ctx, step))

	step.Status = storage.RunStepStatusCompleted
	require.NoError(t, repo.UpsertRunStep(ctx, step))

	steps, err := repo.ListRunSteps(ctx, "task-1")
	require.NoError(t, err)
	require.Len(t, steps, 1)
	require.Equal(t, storage.RunStepStatusCompleted, steps[0].Status)

	// 체크포인트
	chk := &storage.Checkpoint{
		TaskID:  "task-1",
		GitHash: "abc123",
	}
	require.NoError(t, repo.CreateCheckpoint(ctx, chk))
	require.NoError(t, repo.CreateCheckpoint(ctx, chk)) // no duplicates

	checkpoints, err := repo.ListCheckpoints(ctx, "task-1")
	require.NoError(t, err)
	require.Len(t, checkpoints, 1)
	require.Equal(t, "abc123", checkpoints[0].GitHash)
}
