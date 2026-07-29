package service

import (
	"testing"
	"time"

	platformapp "k8s-platform-backend/internal/platform/application"
)

func TestFinishUnresolvedSubStepsUsesParentTerminalStatus(t *testing.T) {
	finishedAt := time.Date(2026, time.July, 23, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		parent   platformapp.TaskStepStatus
		expected platformapp.TaskStepStatus
	}{
		{name: "successful parent completes final task", parent: platformapp.StepSuccess, expected: platformapp.StepSuccess},
		{name: "failed parent fails final task", parent: platformapp.StepFailed, expected: platformapp.StepFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := platformapp.TaskStep{
				Status:   tt.parent,
				SubSteps: []platformapp.TaskSubStep{{Key: "pre_check-14", Status: platformapp.StepRunning}},
			}

			finishUnresolvedSubSteps(&step, finishedAt)

			subStep := step.SubSteps[0]
			if subStep.Status != tt.expected {
				t.Fatalf("status = %q, want %q", subStep.Status, tt.expected)
			}
			if subStep.FinishedAt == nil || !subStep.FinishedAt.Equal(finishedAt) {
				t.Fatalf("finished_at = %v, want %v", subStep.FinishedAt, finishedAt)
			}
		})
	}
}

func TestActiveDeployStepKey(t *testing.T) {
	tests := []struct {
		name string
		task *platformapp.Task
		want string
	}{
		{name: "nil task", task: nil, want: ""},
		{
			name: "uses running step",
			task: &platformapp.Task{Steps: []platformapp.TaskStep{
				{Key: "pre_check", Status: platformapp.StepSuccess},
				{Key: "bootstrap", Status: platformapp.StepRunning},
				{Key: "container_runtime", Status: platformapp.StepPending},
			}},
			want: "bootstrap",
		},
		{
			name: "falls back to first unresolved step before runner starts",
			task: &platformapp.Task{Steps: []platformapp.TaskStep{
				{Key: "pre_check", Status: platformapp.StepSuccess},
				{Key: "bootstrap", Status: platformapp.StepPending},
			}},
			want: "bootstrap",
		},
		{
			name: "returns empty after all steps succeed",
			task: &platformapp.Task{Steps: []platformapp.TaskStep{
				{Key: "pre_check", Status: platformapp.StepSuccess},
				{Key: "bootstrap", Status: platformapp.StepSuccess},
			}},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := activeDeployStepKey(tt.task); got != tt.want {
				t.Fatalf("activeDeployStepKey() = %q, want %q", got, tt.want)
			}
		})
	}
}
