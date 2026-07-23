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
