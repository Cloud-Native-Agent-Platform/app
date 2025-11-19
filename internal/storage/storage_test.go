package storage_test

import (
	"context"
	"testing"

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

	// 메시지 인덱스 추가
	msg1, err := repo.AppendMessageIndex(ctx, "task-1", storage.MessageRoleUser, "/tmp/msg0.json")
	require.NoError(t, err)
	require.Equal(t, 0, msg1.ConversationIndex)

	msg2, err := repo.AppendMessageIndex(ctx, "task-1", storage.MessageRoleAssistant, "/tmp/msg1.json")
	require.NoError(t, err)
	require.Equal(t, 1, msg2.ConversationIndex)

	indexRows, err := repo.ListMessageIndexByTask(ctx, "task-1")
	require.NoError(t, err)
	require.Len(t, indexRows, 2)
	require.Equal(t, "/tmp/msg0.json", indexRows[0].FilePath)
	require.Equal(t, storage.MessageRoleUser, indexRows[0].Role)
	require.Equal(t, "/tmp/msg1.json", indexRows[1].FilePath)
	require.Equal(t, storage.MessageRoleAssistant, indexRows[1].Role)
}

func TestRepositoryMessageIndexAutoIncrement(t *testing.T) {
	repo, cleanup := newTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Agent와 Task 생성
	require.NoError(t, repo.CreateAgent(ctx, &storage.Agent{
		AgentID: "agent-1",
		Status:  storage.AgentStatusActive,
	}))
	require.NoError(t, repo.CreateTask(ctx, &storage.Task{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Status:  storage.TaskStatusRunning,
	}))

	// 첫 번째 메시지: index는 0부터 시작
	nextIndex, err := repo.GetNextConversationIndex(ctx, "task-1")
	require.NoError(t, err)
	require.Equal(t, 0, nextIndex)

	// AppendMessageIndex로 메시지 추가
	msg1, err := repo.AppendMessageIndex(ctx, "task-1", storage.MessageRoleUser, "/tmp/msg0.json")
	require.NoError(t, err)
	require.Equal(t, 0, msg1.ConversationIndex)
	require.Equal(t, storage.MessageRoleUser, msg1.Role)
	require.Equal(t, "/tmp/msg0.json", msg1.FilePath)

	// 두 번째 메시지
	msg2, err := repo.AppendMessageIndex(ctx, "task-1", storage.MessageRoleAssistant, "/tmp/msg1.json")
	require.NoError(t, err)
	require.Equal(t, 1, msg2.ConversationIndex)
	require.Equal(t, storage.MessageRoleAssistant, msg2.Role)

	// 세 번째 메시지
	msg3, err := repo.AppendMessageIndex(ctx, "task-1", storage.MessageRoleUser, "/tmp/msg2.json")
	require.NoError(t, err)
	require.Equal(t, 2, msg3.ConversationIndex)

	// 다음 인덱스 확인
	nextIndex, err = repo.GetNextConversationIndex(ctx, "task-1")
	require.NoError(t, err)
	require.Equal(t, 3, nextIndex)

	// 전체 메시지 목록 확인
	messages, err := repo.ListMessageIndexByTask(ctx, "task-1")
	require.NoError(t, err)
	require.Len(t, messages, 3)
	require.Equal(t, 0, messages[0].ConversationIndex)
	require.Equal(t, storage.MessageRoleUser, messages[0].Role)
	require.Equal(t, 1, messages[1].ConversationIndex)
	require.Equal(t, storage.MessageRoleAssistant, messages[1].Role)
	require.Equal(t, 2, messages[2].ConversationIndex)
	require.Equal(t, storage.MessageRoleUser, messages[2].Role)
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
