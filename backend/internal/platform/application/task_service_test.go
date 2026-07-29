package application

import "testing"

func TestFilterTasksAppliesScopeKeywordAndSort(t *testing.T) {
	firstTitle := "Deploy API"
	secondTitle := "Deploy Web"
	items := []*Task{
		{ID: 2, Type: "deploy", Status: TaskRunning, Title: &firstTitle, CreatedBy: 7, CreatedAt: "2026-07-28T10:00:00Z"},
		{ID: 4, Type: "deploy", Status: TaskPending, Title: &secondTitle, CreatedBy: 7, CreatedAt: "2026-07-29T10:00:00Z", Meta: map[string]any{"name": "frontend"}},
		{ID: 8, Type: "audit", Status: TaskRunning, CreatedBy: 8, CreatedAt: "2026-07-30T10:00:00Z"},
	}

	filtered := filterTasks(items, ListTasksRequest{Type: "deploy", CreatedBy: int64Ptr(7), Keyword: "front", SortBy: "created_at", Order: "asc"})
	if len(filtered) != 1 || filtered[0].ID != 4 {
		t.Fatalf("filtered tasks = %#v", filtered)
	}

	ordered := filterTasks(items, ListTasksRequest{Type: "deploy"})
	if len(ordered) != 2 || ordered[0].ID != 4 || ordered[1].ID != 2 {
		t.Fatalf("default order = %#v", ordered)
	}
}

func TestTaskCanCancelOnlyWhilePendingOrRunning(t *testing.T) {
	for _, test := range []struct {
		status TaskStatus
		want   bool
	}{
		{TaskPending, true}, {TaskRunning, true}, {TaskSuccess, false}, {TaskFailed, false}, {TaskCanceled, false},
	} {
		if got := (&Task{Status: test.status}).CanCancel(); got != test.want {
			t.Fatalf("CanCancel(%q) = %v, want %v", test.status, got, test.want)
		}
	}
}

func int64Ptr(value int64) *int64 { return &value }
