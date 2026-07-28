package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"k8s-platform-backend/internal/incident/domain"
	"k8s-platform-backend/internal/incident/ports"
)

type fakeRepository struct {
	incident domain.Incident
	saved    bool
	expected uint64
	entry    domain.TimelineEntry
}

func (f *fakeRepository) List(context.Context, ports.ListFilter) ([]domain.Incident, int64, error) {
	return []domain.Incident{f.incident}, 1, nil
}
func (f *fakeRepository) Get(context.Context, uint64) (domain.Incident, []domain.TimelineEntry, error) {
	return f.incident, nil, nil
}
func (f *fakeRepository) SaveTransition(_ context.Context, incident domain.Incident, expected uint64, entry domain.TimelineEntry) error {
	f.incident, f.saved, f.expected, f.entry = incident, true, expected, entry
	return nil
}

type fakeClusters struct{}

func (fakeClusters) Names(context.Context, []uint64) (map[uint64]string, error) {
	return map[uint64]string{3: "production"}, nil
}

type failingClusters struct{}

func (failingClusters) Names(context.Context, []uint64) (map[uint64]string, error) {
	return nil, errors.New("cluster projection unavailable")
}

func TestExecutePersistsAggregateAndTimelineThroughOnePort(t *testing.T) {
	repository := &fakeRepository{incident: domain.Incident{ID: 8, ClusterID: 3, Status: domain.StatusOpen, Version: 4}}
	service := NewService(repository, fakeClusters{})
	service.now = func() time.Time { return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC) }

	result, err := service.Execute(context.Background(), 8, domain.CommandAcknowledge, CommandRequest{ExpectedVersion: 4}, domain.Actor{ID: 2, Name: "alice"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !repository.saved || repository.expected != 4 {
		t.Fatalf("transition was not persisted with expected version")
	}
	if repository.entry.Type != "acknowledge" || result.Version != 5 || result.ClusterName != "production" {
		t.Fatalf("unexpected result: %+v, entry=%+v", result, repository.entry)
	}
}

func TestExecuteDoesNotFailAfterTransitionWasPersisted(t *testing.T) {
	repository := &fakeRepository{incident: domain.Incident{ID: 8, ClusterID: 3, Status: domain.StatusOpen, Version: 1}}
	service := NewService(repository, failingClusters{})

	result, err := service.Execute(context.Background(), 8, domain.CommandAcknowledge, CommandRequest{ExpectedVersion: 1}, domain.Actor{})
	if err != nil {
		t.Fatalf("persisted transition must remain successful: %v", err)
	}
	if !repository.saved || result.Version != 2 || result.ClusterName != "" {
		t.Fatalf("unexpected degraded result: saved=%v result=%+v", repository.saved, result)
	}
}

func TestListRejectsUnknownStatus(t *testing.T) {
	service := NewService(&fakeRepository{}, fakeClusters{})
	if _, err := service.List(context.Background(), ports.ListFilter{Status: "made_up"}); err != domain.ErrValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}
