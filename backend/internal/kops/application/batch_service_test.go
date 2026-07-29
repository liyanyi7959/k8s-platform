package application

import (
	"context"
	"errors"
	"testing"
)

type batchRuntimeSpy struct{ job JobEditInput }

func (spy *batchRuntimeSpy) List(context.Context, BatchResource, BatchListQuery) (any, error) {
	return nil, nil
}
func (spy *batchRuntimeSpy) YAML(context.Context, BatchResource, BatchReference) (any, error) {
	return nil, nil
}
func (spy *batchRuntimeSpy) Delete(context.Context, BatchResource, BatchReference) error { return nil }
func (spy *batchRuntimeSpy) EditJob(_ context.Context, input JobEditInput) error {
	spy.job = input
	return nil
}
func (spy *batchRuntimeSpy) DeleteCompletedJobs(context.Context, uint64, string, int) (int, error) {
	return 0, nil
}
func (spy *batchRuntimeSpy) EditCronJob(context.Context, CronJobEditInput) error { return nil }
func (spy *batchRuntimeSpy) TriggerCronJob(context.Context, BatchReference) (*CronJobTriggerResult, error) {
	return nil, nil
}
func (spy *batchRuntimeSpy) SuspendCronJob(context.Context, BatchReference, bool) error { return nil }

func TestBatchServiceOwnsEditValidation(t *testing.T) {
	spy := &batchRuntimeSpy{}
	service := NewBatchService(spy)
	if err := service.EditJob(context.Background(), JobEditInput{ClusterID: 2, Namespace: " ops ", Name: " report "}); err != nil || spy.job.Namespace != "ops" || spy.job.Name != "report" {
		t.Fatalf("EditJob() input=%#v err=%v", spy.job, err)
	}
	negative := int32(-1)
	if err := service.EditJob(context.Background(), JobEditInput{ClusterID: 2, Namespace: "ops", Name: "report", Parallelism: &negative}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid job error=%v", err)
	}
}
