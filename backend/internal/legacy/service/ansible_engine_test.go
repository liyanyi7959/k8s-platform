package service

import (
	"testing"
	"time"
)

func TestFinishUnresolvedSubStepsUsesParentTerminalStatus(t *testing.T) {
	finishedAt := time.Date(2026, time.July, 23, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		parent   TaskStepStatus
		expected TaskStepStatus
	}{
		{name: "successful parent completes final task", parent: StepSuccess, expected: StepSuccess},
		{name: "failed parent fails final task", parent: StepFailed, expected: StepFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := TaskStep{
				Status:   tt.parent,
				SubSteps: []TaskSubStep{{Key: "pre_check-14", Status: StepRunning}},
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
		task *Task
		want string
	}{
		{name: "nil task", task: nil, want: ""},
		{
			name: "uses running step",
			task: &Task{Steps: []TaskStep{
				{Key: "pre_check", Status: StepSuccess},
				{Key: "bootstrap", Status: StepRunning},
				{Key: "container_runtime", Status: StepPending},
			}},
			want: "bootstrap",
		},
		{
			name: "falls back to first unresolved step before runner starts",
			task: &Task{Steps: []TaskStep{
				{Key: "pre_check", Status: StepSuccess},
				{Key: "bootstrap", Status: StepPending},
			}},
			want: "bootstrap",
		},
		{
			name: "returns empty after all steps succeed",
			task: &Task{Steps: []TaskStep{
				{Key: "pre_check", Status: StepSuccess},
				{Key: "bootstrap", Status: StepSuccess},
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
