package application

import (
	"strings"
	"time"
)

const (
	cronJobManualNameSuffix       = "-manual-"
	cronJobGeneratedNameMaxLength = 58
)

// JobCompletionState is the transport-neutral completion contract used by
// batch runtimes. It keeps CronJob/Job retention policy independent from the
// Kubernetes client model.
type JobCompletionState struct {
	CompletionTime            *time.Time
	TerminalTransitionTime    *time.Time
	CreatedAt                 time.Time
	Active, Succeeded, Failed int32
	Terminal                  bool
}

// CronJobManualJobNamePrefix returns a valid generated-name prefix for a Job
// instantiated from a CronJob. Kubernetes appends its own unique suffix.
func CronJobManualJobNamePrefix(name string) string {
	name = strings.TrimSpace(name)
	prefix := name + cronJobManualNameSuffix
	if len(prefix) <= cronJobGeneratedNameMaxLength {
		return prefix
	}
	maxNameLength := cronJobGeneratedNameMaxLength - len(cronJobManualNameSuffix)
	if maxNameLength < 1 {
		return prefix[:cronJobGeneratedNameMaxLength]
	}
	return name[:maxNameLength] + cronJobManualNameSuffix
}

// JobFinishedAt determines when a Job became terminal. A completion timestamp
// is preferred, then terminal-condition time, with creation time as the safe
// fallback for older Kubernetes status shapes.
func JobFinishedAt(state JobCompletionState) (time.Time, bool) {
	if state.CompletionTime != nil && !state.CompletionTime.IsZero() {
		return *state.CompletionTime, true
	}
	if state.Terminal {
		if state.TerminalTransitionTime != nil && !state.TerminalTransitionTime.IsZero() {
			return *state.TerminalTransitionTime, true
		}
		return state.CreatedAt, !state.CreatedAt.IsZero()
	}
	if state.Active <= 0 && (state.Succeeded > 0 || state.Failed > 0) {
		return state.CreatedAt, !state.CreatedAt.IsZero()
	}
	return time.Time{}, false
}

// CompletedJobCutoff maps the API retention age to a single timestamp. A
// zero duration means delete all completed jobs and is represented by zero.
func CompletedJobCutoff(olderThanHours int) time.Time {
	if olderThanHours <= 0 {
		return time.Time{}
	}
	return time.Now().Add(-time.Duration(olderThanHours) * time.Hour)
}

// ShouldDeleteCompletedJob applies the retention decision after the runtime
// has identified a terminal Job. A zero cutoff retains the existing "all"
// behavior for a zero-hour cleanup request.
func ShouldDeleteCompletedJob(finishedAt, cutoff time.Time) bool {
	return !finishedAt.IsZero() && (cutoff.IsZero() || !finishedAt.After(cutoff))
}
