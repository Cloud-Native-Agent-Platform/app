package mocks_test

import (
	"context"
	"testing"

	"github.com/cnap-oss/app/internal/runner"
	"github.com/cnap-oss/app/internal/testutil/mocks"
	"github.com/stretchr/testify/require"
)

func TestMockRunner_Run(t *testing.T) {
	mock := mocks.NewMockRunner()
	mock.SetResponse("task-001", "Hello from mock!")

	ctx := context.Background()
	req := &runner.RunRequest{
		TaskID:       "task-001",
		Model:        "gpt-4",
		SystemPrompt: "You are a helpful assistant",
		Messages: []runner.ChatMessage{
			{Role: "user", Content: "Hi"},
		},
	}

	result, err := mock.Run(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "Hello from mock!", result.Output)
	require.True(t, result.Success)
	require.Equal(t, 1, mock.GetCallCount())
}

func TestMockRunner_DefaultResponse(t *testing.T) {
	mock := mocks.NewMockRunner()
	mock.DefaultResponse = "Default response"

	ctx := context.Background()
	req := &runner.RunRequest{
		TaskID: "unknown-task",
		Model:  "gpt-4",
	}

	result, err := mock.Run(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "Default response", result.Output)
}

func TestMockRunner_Error(t *testing.T) {
	mock := mocks.NewMockRunner()
	mock.SetErrorMessage("task-fail", "API error")

	ctx := context.Background()
	req := &runner.RunRequest{
		TaskID: "task-fail",
		Model:  "gpt-4",
	}

	result, err := mock.Run(ctx, req)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "API error")
}

func TestMockRunner_CallHistory(t *testing.T) {
	mock := mocks.NewMockRunner()

	ctx := context.Background()

	// 여러 번 호출
	_, _ = mock.Run(ctx, &runner.RunRequest{TaskID: "task-1"})
	_, _ = mock.Run(ctx, &runner.RunRequest{TaskID: "task-2"})
	_, _ = mock.Run(ctx, &runner.RunRequest{TaskID: "task-3"})

	require.Equal(t, 3, mock.GetCallCount())
	require.Equal(t, "task-3", mock.GetLastCall().TaskID)

	// Reset
	mock.Reset()
	require.Equal(t, 0, mock.GetCallCount())
	require.Nil(t, mock.GetLastCall())
}
