package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/cnap-oss/app/internal/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func buildTaskCommands(logger *zap.Logger) *cobra.Command {
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Task 관리 명령어",
		Long:  "Controller를 통한 Task 생성, 조회, 상태 변경 기능을 제공합니다.",
	}

	// task create
	taskCreateCmd := &cobra.Command{
		Use:   "create <agent-name> <task-id>",
		Short: "새로운 Task 생성",
		Long:  "특정 Agent에 새로운 Task를 생성합니다.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskCreate(logger, args[0], args[1])
		},
	}

	// task list
	taskListCmd := &cobra.Command{
		Use:   "list <agent-name>",
		Short: "Task 목록 조회",
		Long:  "특정 Agent의 모든 Task 목록을 조회합니다.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskList(logger, args[0])
		},
	}

	// task view
	taskViewCmd := &cobra.Command{
		Use:   "view <task-id>",
		Short: "Task 상세 정보 조회",
		Long:  "특정 Task의 상세 정보를 조회합니다.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskView(logger, args[0])
		},
	}

	// task update-status
	taskUpdateStatusCmd := &cobra.Command{
		Use:   "update-status <task-id> <status>",
		Short: "Task 상태 변경",
		Long:  "Task의 상태를 변경합니다. (pending, running, completed, failed, canceled)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskUpdateStatus(logger, args[0], args[1])
		},
	}

	// task cancel
	taskCancelCmd := &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "Task 취소",
		Long:  "Task를 취소 상태로 변경합니다.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskUpdateStatus(logger, args[0], storage.TaskStatusCanceled)
		},
	}

	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskViewCmd)
	taskCmd.AddCommand(taskUpdateStatusCmd)
	taskCmd.AddCommand(taskCancelCmd)

	return taskCmd
}

func runTaskCreate(logger *zap.Logger, agentName, taskID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ctrl, cleanup, err := newController(logger)
	if err != nil {
		return fmt.Errorf("컨트롤러 초기화 실패: %w", err)
	}
	defer cleanup()

	if err := ctrl.CreateTask(ctx, agentName, taskID); err != nil {
		return fmt.Errorf("Task 생성 실패: %w", err)
	}

	fmt.Printf("✓ Task '%s' 생성 완료 (Agent: %s)\n", taskID, agentName)
	return nil
}

func runTaskList(logger *zap.Logger, agentName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ctrl, cleanup, err := newController(logger)
	if err != nil {
		return fmt.Errorf("컨트롤러 초기화 실패: %w", err)
	}
	defer cleanup()

	tasks, err := ctrl.ListTasksByAgent(ctx, agentName)
	if err != nil {
		return fmt.Errorf("Task 목록 조회 실패: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Printf("Agent '%s'에 등록된 Task가 없습니다.\n", agentName)
		return nil
	}

	// 테이블 형식 출력
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK ID\tSTATUS\tCREATED\tUPDATED")
	fmt.Fprintln(w, "-------\t------\t-------\t-------")

	for _, task := range tasks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			task.TaskID,
			task.Status,
			task.CreatedAt.Format("2006-01-02 15:04"),
			task.UpdatedAt.Format("2006-01-02 15:04"),
		)
	}
	w.Flush()

	return nil
}

func runTaskView(logger *zap.Logger, taskID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ctrl, cleanup, err := newController(logger)
	if err != nil {
		return fmt.Errorf("컨트롤러 초기화 실패: %w", err)
	}
	defer cleanup()

	task, err := ctrl.GetTaskInfo(ctx, taskID)
	if err != nil {
		return fmt.Errorf("Task 조회 실패: %w", err)
	}

	// 상세 정보 출력
	fmt.Printf("=== Task 정보: %s ===\n\n", task.TaskID)
	fmt.Printf("Task ID:     %s\n", task.TaskID)
	fmt.Printf("Agent ID:    %s\n", task.AgentID)
	fmt.Printf("상태:        %s\n", task.Status)
	fmt.Printf("생성일:      %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("수정일:      %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func runTaskUpdateStatus(logger *zap.Logger, taskID, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ctrl, cleanup, err := newController(logger)
	if err != nil {
		return fmt.Errorf("컨트롤러 초기화 실패: %w", err)
	}
	defer cleanup()

	// 상태 검증
	validStatuses := []string{
		storage.TaskStatusPending,
		storage.TaskStatusRunning,
		storage.TaskStatusCompleted,
		storage.TaskStatusFailed,
		storage.TaskStatusCanceled,
	}

	isValid := false
	for _, s := range validStatuses {
		if status == s {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("유효하지 않은 상태: %s (사용 가능: %v)", status, validStatuses)
	}

	if err := ctrl.UpdateTaskStatus(ctx, taskID, status); err != nil {
		return fmt.Errorf("Task 상태 변경 실패: %w", err)
	}

	fmt.Printf("✓ Task '%s' 상태 변경: %s\n", taskID, status)
	return nil
}
