package kops

import (
	"context"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	client, err := r.service.TypedClient(ctx, clusterID)
	if err != nil {
		return 0, translateKopsRuntimeError(err)
	}
	jobs, err := client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, translateBatchAPIError(err)
	}

	cutoff := kopsapp.CompletedJobCutoff(older)
	deleted := 0
	for i := range jobs.Items {
		job := &jobs.Items[i]
		finishedAt, completed := kopsapp.JobFinishedAt(batchJobCompletion(job))
		if !completed || !kopsapp.ShouldDeleteCompletedJob(finishedAt, cutoff) {
			continue
		}
		if err := client.BatchV1().Jobs(job.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return deleted, kopsapp.ErrWithMessage(translateBatchAPIError(err), fmt.Sprintf("已删除 %d 个 Job，但清理 %s/%s 失败", deleted, job.Namespace, job.Name))
		}
		deleted++
	}
	return deleted, nil
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
	client, err := r.service.TypedClient(ctx, ref.ClusterID)
	if err != nil {
		return nil, translateKopsRuntimeError(err)
	}
	cronJob, err := client.BatchV1().CronJobs(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, translateBatchAPIError(err)
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       ref.Namespace,
			GenerateName:    kopsapp.CronJobManualJobNamePrefix(cronJob.Name),
			Labels:          mergeBatchStringMaps(cronJob.Labels, cronJob.Spec.JobTemplate.Labels),
			Annotations:     mergeBatchStringMaps(cronJob.Annotations, cronJob.Spec.JobTemplate.Annotations),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cronJob, batchv1.SchemeGroupVersion.WithKind("CronJob"))},
		},
		Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
	}
	if job.Labels == nil {
		job.Labels = map[string]string{}
	}
	job.Labels["cronjob.kubernetes.io/instantiate"] = "manual"
	created, err := client.BatchV1().Jobs(ref.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, translateBatchAPIError(err)
	}
	return &kopsapp.CronJobTriggerResult{JobName: created.Name}, nil
}
func (r *BatchRuntime) SuspendCronJob(ctx context.Context, ref kopsapp.BatchReference, suspend bool) error {
	if r == nil || r.service == nil {
		return kopsapp.ErrConflict
	}
	return translateKopsRuntimeError(r.service.PatchJSON(ctx, ref.ClusterID, batchCronJobGVR, ref.Namespace, ref.Name, map[string]any{
		"spec": map[string]any{"suspend": suspend},
	}))
}

func translateBatchAPIError(err error) error {
	return translateKopsRuntimeError(service.NormalizeKubernetesError(err))
}

func batchJobCompletion(job *batchv1.Job) kopsapp.JobCompletionState {
	if job == nil {
		return kopsapp.JobCompletionState{}
	}
	state := kopsapp.JobCompletionState{
		CreatedAt: job.CreationTimestamp.Time,
		Active:    job.Status.Active,
		Succeeded: job.Status.Succeeded,
		Failed:    job.Status.Failed,
	}
	if job.Status.CompletionTime != nil && !job.Status.CompletionTime.IsZero() {
		finishedAt := job.Status.CompletionTime.Time
		state.CompletionTime = &finishedAt
	}
	for _, condition := range job.Status.Conditions {
		if condition.Status != corev1.ConditionTrue || (condition.Type != batchv1.JobComplete && condition.Type != batchv1.JobFailed) {
			continue
		}
		state.Terminal = true
		if !condition.LastTransitionTime.IsZero() {
			finishedAt := condition.LastTransitionTime.Time
			state.TerminalTransitionTime = &finishedAt
		}
		break
	}
	return state
}

func mergeBatchStringMaps(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	merged := make(map[string]string, len(base)+len(override))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range override {
		merged[key] = value
	}
	return merged
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
