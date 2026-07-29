package application

import (
	"strings"
	"testing"
	"time"
)

func TestCronJobManualJobNamePrefix(t *testing.T) {
	if got := CronJobManualJobNamePrefix(" report "); got != "report-manual-" {
		t.Fatalf("CronJobManualJobNamePrefix() = %q", got)
	}
	longName := strings.Repeat("a", 80)
	got := CronJobManualJobNamePrefix(longName)
	if len(got) != cronJobGeneratedNameMaxLength || !strings.HasSuffix(got, cronJobManualNameSuffix) {
		t.Fatalf("long prefix = %q", got)
	}
}

func TestJobFinishedAt(t *testing.T) {
	created := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)
	completed := created.Add(time.Hour)
	if got, ok := JobFinishedAt(JobCompletionState{CreatedAt: created, CompletionTime: &completed}); !ok || !got.Equal(completed) {
		t.Fatalf("completion timestamp = %v, %t", got, ok)
	}
	if got, ok := JobFinishedAt(JobCompletionState{CreatedAt: created, Terminal: true}); !ok || !got.Equal(created) {
		t.Fatalf("terminal fallback = %v, %t", got, ok)
	}
	if got, ok := JobFinishedAt(JobCompletionState{CreatedAt: created, Active: 0, Succeeded: 1}); !ok || !got.Equal(created) {
		t.Fatalf("legacy completed fallback = %v, %t", got, ok)
	}
	if _, ok := JobFinishedAt(JobCompletionState{CreatedAt: created, Active: 1}); ok {
		t.Fatal("active job must not be completed")
	}
}

func TestShouldDeleteCompletedJob(t *testing.T) {
	finished := time.Date(2026, time.January, 1, 9, 0, 0, 0, time.UTC)
	if !ShouldDeleteCompletedJob(finished, time.Time{}) {
		t.Fatal("zero cutoff should delete every completed job")
	}
	if !ShouldDeleteCompletedJob(finished, finished.Add(time.Minute)) {
		t.Fatal("job before cutoff should be deleted")
	}
	if ShouldDeleteCompletedJob(finished, finished.Add(-time.Minute)) {
		t.Fatal("job after cutoff should be retained")
	}
}
