package mocks_test

import (
	"context"
	"testing"

	taskrunner "github.com/cnap-oss/app/internal/runner"
	"github.com/cnap-oss/app/internal/testutil/mocks"
	"github.com/stretchr/testify/require"
)

func TestMockRunner_Run(t *testing.T) {
	m := mocks.NewMockRunner()
	m.SetResponse("task-001", "Hello from mock!")

	ctx := context.Background()
	req := &taskrunner.RunRequest{
		TaskID:       "task-001",
		Model:        "gpt-4",
		SystemPrompt: "You are a helpful assistant",
		Messages: []taskrunner.ChatMessage{
			{Role: "user", Content: "Hi"},
		},
	}

	result, err := m.Run(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "Hello from mock!", result.Output)
	require.True(t, result.Success)
	require.Equal(t, 1, m.GetCallCount())
}

func TestMockRunner_DefaultResponse(t *testing.T) {
	m := mocks.NewMockRunner()
	m.DefaultResponse = "Default response"

	ctx := context.Background()
	req := &taskrunner.RunRequest{
		TaskID: "unknown-task",
		Model:  "gpt-4",
	}

	result, err := m.Run(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "Default response", result.Output)
}

func TestMockRunner_Error(t *testing.T) {
	m := mocks.NewMockRunner()
	m.SetErrorMessage("task-fail", "API error")

	ctx := context.Background()
	req := &taskrunner.RunRequest{
		TaskID: "task-fail",
		Model:  "gpt-4",
	}

	result, err := m.Run(ctx, req)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "API error")
}

func TestMockRunner_CallHistory(t *testing.T) {
	m := mocks.NewMockRunner()

	ctx := context.Background()

	_, _ = m.Run(ctx, &taskrunner.RunRequest{TaskID: "task-1"})
	_, _ = m.Run(ctx, &taskrunner.RunRequest{TaskID: "task-2"})
	_, _ = m.Run(ctx, &taskrunner.RunRequest{TaskID: "task-3"})

	require.Equal(t, 3, m.GetCallCount())
	require.Equal(t, "task-3", m.GetLastCall().TaskID)

	m.Reset()
	require.Equal(t, 0, m.GetCallCount())
	require.Nil(t, m.GetLastCall())
}
