package provisioning

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	platformapp "k8s-platform-backend/internal/platform/application"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
)

// ansibleLogWriter projects streamed remote Ansible output onto the platform
// task. It belongs to the Provisioning runner, alongside the transport that
// produces the stream, rather than to a separate service coordinator.
type ansibleLogWriter struct {
	task            *platformapp.Task
	store           *platformapp.TaskStore
	mu              sync.Mutex
	buf             []byte
	stepIndex       int
	failedStepIndex int
}

func newAnsibleLogWriter(task *platformapp.Task, store *platformapp.TaskStore) *ansibleLogWriter {
	return &ansibleLogWriter{task: task, store: store, stepIndex: -1, failedStepIndex: -1}
}

func (w *ansibleLogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.task == nil {
		return len(p), nil
	}
	w.buf = append(w.buf, p...)
	for {
		index := bytes.IndexByte(w.buf, '\n')
		if index < 0 {
			break
		}
		line := string(w.buf[:index])
		w.buf = w.buf[index+1:]
		w.parseStepProgress(line)
		w.task.AppendLog(line, w.currentStepKey())
	}
	w.putTask()
	return len(p), nil
}

func (w *ansibleLogWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.task == nil || len(w.buf) == 0 {
		return
	}
	w.task.AppendLog(string(w.buf), w.currentStepKey())
	w.buf = nil
	w.putTask()
}

func (w *ansibleLogWriter) putTask() {
	if w != nil && w.store != nil && w.task != nil {
		_ = w.store.Put(w.task)
	}
}

func (w *ansibleLogWriter) currentStepKey() string {
	if w == nil || w.task == nil {
		return ""
	}
	if w.failedStepIndex >= 0 && w.failedStepIndex < len(w.task.Steps) {
		return w.task.Steps[w.failedStepIndex].Key
	}
	if w.stepIndex >= 0 && w.stepIndex < len(w.task.Steps) {
		return w.task.Steps[w.stepIndex].Key
	}
	return activeTaskStepKey(w.task)
}

func (w *ansibleLogWriter) parseStepProgress(line string) {
	if w == nil || w.task == nil {
		return
	}
	trimmed := strings.TrimSpace(line)
	now := time.Now().UTC()
	if strings.HasPrefix(trimmed, "PLAY [") {
		if w.failedStepIndex >= 0 {
			return
		}
		playName := provisionapp.ExtractAnsiblePlayName(trimmed)
		for _, step := range provisionapp.DefaultAnsibleSteps() {
			if !strings.Contains(playName, step.PlayName) {
				continue
			}
			taskStepIndex := findTaskStepIndex(w.task, step.Key)
			if taskStepIndex < 0 {
				w.stepIndex = -1
				break
			}
			for index := 0; index < taskStepIndex; index++ {
				if w.task.Steps[index].Status == platformapp.StepRunning {
					w.task.Steps[index].Status = platformapp.StepSuccess
					w.task.Steps[index].FinishedAt = &now
					w.finishRunningSubStep(index, now)
				}
			}
			if w.task.Steps[taskStepIndex].Status != platformapp.StepSuccess {
				w.task.Steps[taskStepIndex].Status = platformapp.StepRunning
				if w.task.Steps[taskStepIndex].StartedAt == nil {
					w.task.Steps[taskStepIndex].StartedAt = &now
				}
			}
			percent := 0
			if len(w.task.Steps) > 0 {
				percent = taskStepIndex * 100 / len(w.task.Steps)
			}
			w.task.Percent = &percent
			w.stepIndex = taskStepIndex
			break
		}
	}

	if strings.HasPrefix(trimmed, "TASK [") {
		taskName := provisionapp.ExtractAnsiblePlayName(trimmed)
		if taskName != "" && w.stepIndex >= 0 && w.stepIndex < len(w.task.Steps) {
			step := &w.task.Steps[w.stepIndex]
			w.finishRunningSubStep(w.stepIndex, now)
			found := false
			for index := range step.SubSteps {
				if step.SubSteps[index].Title == taskName {
					step.SubSteps[index].Status = platformapp.StepRunning
					step.SubSteps[index].StartedAt = &now
					step.SubSteps[index].FinishedAt = nil
					found = true
					break
				}
			}
			if !found {
				step.SubSteps = append(step.SubSteps, platformapp.TaskSubStep{
					Key: fmt.Sprintf("%s-%d", step.Key, len(step.SubSteps)), Title: taskName,
					Status: platformapp.StepRunning, StartedAt: &now,
				})
			}
		}
	}

	if strings.Contains(trimmed, "FAILED!") || strings.Contains(trimmed, "UNREACHABLE!") {
		w.markCurrentStepFailed(now)
		return
	}
	if strings.HasPrefix(trimmed, "PLAY RECAP") {
		if w.failedStepIndex >= 0 {
			return
		}
		for index := range w.task.Steps {
			if w.task.Steps[index].Status == platformapp.StepRunning {
				w.task.Steps[index].Status = platformapp.StepSuccess
				w.task.Steps[index].FinishedAt = &now
				w.finishRunningSubStep(index, now)
			}
		}
		return
	}
	if provisionapp.HasAnsibleRecapFailure(trimmed) {
		w.markCurrentStepFailed(now)
	}
}

func findTaskStepIndex(task *platformapp.Task, stepKey string) int {
	if task == nil {
		return -1
	}
	for index, step := range task.Steps {
		if step.Key == stepKey {
			return index
		}
	}
	if task.Type == "install_cluster_addons" && len(task.Steps) == 1 && task.Steps[0].Key == "install_addons" {
		for _, enabledStep := range taskMetaStringSlice(task.Meta, "enabled_steps") {
			if enabledStep == stepKey {
				return 0
			}
		}
	}
	return -1
}

func (w *ansibleLogWriter) markCurrentStepFailed(now time.Time) {
	if w.failedStepIndex >= 0 || w.stepIndex < 0 || w.stepIndex >= len(w.task.Steps) {
		return
	}
	w.failedStepIndex = w.stepIndex
	w.task.Steps[w.stepIndex].Status = platformapp.StepFailed
	w.task.Steps[w.stepIndex].FinishedAt = &now
	w.finishRunningSubStep(w.stepIndex, now)
}

func (w *ansibleLogWriter) finishRunningSubStep(stepIndex int, finishedAt time.Time) {
	if stepIndex < 0 || stepIndex >= len(w.task.Steps) {
		return
	}
	step := &w.task.Steps[stepIndex]
	for index := range step.SubSteps {
		if step.SubSteps[index].Status != platformapp.StepRunning {
			continue
		}
		step.SubSteps[index].FinishedAt = &finishedAt
		if step.Status == platformapp.StepFailed {
			step.SubSteps[index].Status = platformapp.StepFailed
		} else {
			step.SubSteps[index].Status = platformapp.StepSuccess
		}
	}
}
