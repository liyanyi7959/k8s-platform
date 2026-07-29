package kops

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
)

type BatchRuntime struct{ service *service.K8sService }

func NewBatchRuntime(service *service.K8sService) *BatchRuntime {
	return &BatchRuntime{service: service}
}
func (r *BatchRuntime) List(ctx context.Context, resource kopsapp.BatchResource, query kopsapp.BatchListQuery) (any, error) {
	gvr, err := r.gvr(resource)
	if err != nil {
		return nil, err
	}
	value, err := r.service.List(ctx, query.ClusterID, gvr, query.Namespace, query.SortBy, query.Order, nil)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]any{"list": value}, nil
}
func (r *BatchRuntime) YAML(ctx context.Context, resource kopsapp.BatchResource, ref kopsapp.BatchReference) (any, error) {
	gvr, err := r.gvr(resource)
	if err != nil {
		return nil, err
	}
	value, err := r.service.GetYAML(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return map[string]string{"text": value}, nil
}
func (r *BatchRuntime) Delete(ctx context.Context, resource kopsapp.BatchResource, ref kopsapp.BatchReference) error {
	gvr, err := r.gvr(resource)
	if err != nil {
		return err
	}
	return translateKopsRuntimeError(r.service.Delete(ctx, ref.ClusterID, gvr, ref.Namespace, ref.Name))
}
func (r *BatchRuntime) EditJob(ctx context.Context, input kopsapp.JobEditInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	patch := map[string]any{}
	if input.Labels != nil {
		patch["metadata"] = map[string]any{"labels": input.Labels}
	}
	spec := map[string]any{}
	if input.Parallelism != nil {
		spec["parallelism"] = *input.Parallelism
	}
	if input.Completions != nil {
		spec["completions"] = *input.Completions
	}
	if input.BackoffLimit != nil {
		spec["backoffLimit"] = *input.BackoffLimit
	}
	if input.TTLSecondsAfterFinished != nil {
		spec["ttlSecondsAfterFinished"] = *input.TTLSecondsAfterFinished
	}
	if len(spec) > 0 {
		patch["spec"] = spec
	}
	if len(patch) == 0 {
		return nil
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, input.ClusterID, batchJobGVR, input.Namespace, input.Name, patch))
}
func (r *BatchRuntime) DeleteCompletedJobs(ctx context.Context, clusterID uint64, namespace string, older int) (int, error) {
	if r == nil || r.service == nil {
		return 0, kopsapp.ErrConflict
	}
	value, err := r.service.DeleteCompletedJobs(ctx, clusterID, namespace, older)
	return value, translateKopsRuntimeError(err)
}
func (r *BatchRuntime) EditCronJob(ctx context.Context, input kopsapp.CronJobEditInput) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	patch := map[string]any{}
	if input.Labels != nil {
		patch["metadata"] = map[string]any{"labels": input.Labels}
	}
	spec := map[string]any{"schedule": input.Schedule}
	if input.Suspend != nil {
		spec["suspend"] = *input.Suspend
	}
	if input.ConcurrencyPolicy != nil && strings.TrimSpace(*input.ConcurrencyPolicy) != "" {
		spec["concurrencyPolicy"] = strings.TrimSpace(*input.ConcurrencyPolicy)
	}
	if input.SuccessfulJobsHistoryLimit != nil {
		spec["successfulJobsHistoryLimit"] = *input.SuccessfulJobsHistoryLimit
	}
	if input.FailedJobsHistoryLimit != nil {
		spec["failedJobsHistoryLimit"] = *input.FailedJobsHistoryLimit
	}
	patch["spec"] = spec
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, input.ClusterID, batchCronJobGVR, input.Namespace, input.Name, patch))
}
func (r *BatchRuntime) TriggerCronJob(ctx context.Context, ref kopsapp.BatchReference) (*kopsapp.CronJobTriggerResult, error) {
	if r == nil || r.service == nil {
		return nil, kopsapp.ErrConflict
	}
	value, err := r.service.TriggerCronJob(ctx, ref.ClusterID, ref.Namespace, ref.Name)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	return &kopsapp.CronJobTriggerResult{JobName: value.JobName}, nil
}
func (r *BatchRuntime) SuspendCronJob(ctx context.Context, ref kopsapp.BatchReference, suspend bool) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.SuspendCronJob(ctx, ref.ClusterID, ref.Namespace, ref.Name, suspend))
}
func (r *BatchRuntime) gvr(resource kopsapp.BatchResource) (schema.GroupVersionResource, error) {
	if r == nil || r.service == nil {
		return schema.GroupVersionResource{}, kopsapp.ErrConflict
	}
	if resource == kopsapp.BatchJob {
		return batchJobGVR, nil
	}
	if resource == kopsapp.BatchCronJob {
		return batchCronJobGVR, nil
	}
	return schema.GroupVersionResource{}, kopsapp.ErrInvalidParams
}

var batchJobGVR = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
var batchCronJobGVR = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
