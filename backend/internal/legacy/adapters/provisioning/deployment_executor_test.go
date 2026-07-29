package provisioning

import (
	"context"
	"errors"
	"testing"
	"time"

	platformapp "k8s-platform-backend/internal/platform/application"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
)

func TestDeploymentExecutorRequiresAllExecutionDependencies(t *testing.T) {
	executor := NewDeploymentExecutor(nil, nil, nil, nil, nil)
	if _, err := executor.Execute(context.Background(), 1, 1); !errors.Is(err, provisionapp.ErrConflict) {
		t.Fatalf("Execute without runtime dependencies error = %v, want conflict", err)
	}
}

func TestPreflightFailureMessageUsesFailingChecks(t *testing.T) {
	message := preflightFailureMessage(provisionapp.PreflightResult{Checks: []provisionapp.PreflightCheck{
		{Status: provisionapp.PreflightWarning, Message: "ignored"},
		{Status: provisionapp.PreflightError, Message: "SSH unreachable"},
	}})
	if message == "" || message == "部署预检未通过" {
		t.Fatalf("preflight failure message = %q, want check detail", message)
	}
}

func TestFinishUnresolvedSubStepsUsesParentTerminalStatus(t *testing.T) {
	finishedAt := time.Date(2026, time.July, 23, 10, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name     string
		parent   platformapp.TaskStepStatus
		expected platformapp.TaskStepStatus
	}{
		{name: "successful parent completes final task", parent: platformapp.StepSuccess, expected: platformapp.StepSuccess},
		{name: "failed parent fails final task", parent: platformapp.StepFailed, expected: platformapp.StepFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			step := platformapp.TaskStep{Status: test.parent, SubSteps: []platformapp.TaskSubStep{{Key: "pre_check-14", Status: platformapp.StepRunning}}}
			finishUnresolvedSubSteps(&step, finishedAt)
			if got := step.SubSteps[0].Status; got != test.expected {
				t.Fatalf("status = %q, want %q", got, test.expected)
			}
			if step.SubSteps[0].FinishedAt == nil || !step.SubSteps[0].FinishedAt.Equal(finishedAt) {
				t.Fatalf("finished_at = %v, want %v", step.SubSteps[0].FinishedAt, finishedAt)
			}
		})
	}
}

func TestActiveTaskStepKey(t *testing.T) {
	for _, test := range []struct {
		name string
		task *platformapp.Task
		want string
	}{
		{name: "nil task", want: ""},
		{name: "uses running step", task: &platformapp.Task{Steps: []platformapp.TaskStep{{Key: "pre_check", Status: platformapp.StepSuccess}, {Key: "bootstrap", Status: platformapp.StepRunning}, {Key: "container_runtime", Status: platformapp.StepPending}}}, want: "bootstrap"},
		{name: "falls back to first unresolved step", task: &platformapp.Task{Steps: []platformapp.TaskStep{{Key: "pre_check", Status: platformapp.StepSuccess}, {Key: "bootstrap", Status: platformapp.StepPending}}}, want: "bootstrap"},
		{name: "returns empty after success", task: &platformapp.Task{Steps: []platformapp.TaskStep{{Key: "pre_check", Status: platformapp.StepSuccess}, {Key: "bootstrap", Status: platformapp.StepSuccess}}}, want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := activeTaskStepKey(test.task); got != test.want {
				t.Fatalf("activeTaskStepKey() = %q, want %q", got, test.want)
			}
		})
	}
}
