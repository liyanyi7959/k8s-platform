package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const cronJobManualNameSuffix = "-manual-"
const cronJobGeneratedNameMaxLength = 58

var cronJobsGVR = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}

type TriggerCronJobResult struct {
	JobName string `json:"job_name"`
}

func (s *K8sService) TriggerCronJob(ctx context.Context, clusterID uint64, namespace, name string) (TriggerCronJobResult, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return TriggerCronJobResult{}, err
	}

	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if namespace == "" || name == "" {
		return TriggerCronJobResult{}, ErrWithMessage(ErrInvalidParams, "命名空间和 CronJob 名称不能为空")
	}

	cronJob, err := cs.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return TriggerCronJobResult{}, normalizeK8sErr(err)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       namespace,
			GenerateName:    cronJobGenerateNamePrefix(cronJob.Name),
			Labels:          mergeStringMaps(cronJob.Labels, cronJob.Spec.JobTemplate.Labels),
			Annotations:     mergeStringMaps(cronJob.Annotations, cronJob.Spec.JobTemplate.Annotations),
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cronJob, batchv1.SchemeGroupVersion.WithKind("CronJob"))},
		},
		Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
	}

	if job.Labels == nil {
		job.Labels = map[string]string{}
	}
	job.Labels["cronjob.kubernetes.io/instantiate"] = "manual"

	created, err := cs.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return TriggerCronJobResult{}, normalizeK8sErr(err)
	}

	return TriggerCronJobResult{JobName: created.Name}, nil
}

func (s *K8sService) SuspendCronJob(ctx context.Context, clusterID uint64, namespace, name string, suspend bool) error {
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if namespace == "" || name == "" {
		return ErrWithMessage(ErrInvalidParams, "命名空间和 CronJob 名称不能为空")
	}
	return s.PatchJSON(ctx, clusterID, cronJobsGVR, namespace, name, map[string]any{
		"spec": map[string]any{
			"suspend": suspend,
		},
	})
}

func (s *K8sService) DeleteCompletedJobs(ctx context.Context, clusterID uint64, namespace string, olderThanHours int) (int, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return 0, err
	}

	namespace = strings.TrimSpace(namespace)
	if olderThanHours < 0 {
		return 0, ErrWithMessage(ErrInvalidParams, "older_than_hours 不能小于 0")
	}

	jobs, err := cs.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, normalizeK8sErr(err)
	}

	cutoff := time.Now().Add(-time.Duration(olderThanHours) * time.Hour)
	deleted := 0
	for i := range jobs.Items {
		job := &jobs.Items[i]
		finishedAt, ok := jobFinishedTime(job)
		if !ok {
			continue
		}
		if olderThanHours > 0 && finishedAt.After(cutoff) {
			continue
		}
		if err := cs.BatchV1().Jobs(job.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return deleted, ErrWithMessage(normalizeK8sErr(err), fmt.Sprintf("已删除 %d 个 Job，但清理 %s/%s 失败", deleted, job.Namespace, job.Name))
		}
		deleted++
	}

	return deleted, nil
}

func jobFinishedTime(job *batchv1.Job) (time.Time, bool) {
	if job == nil {
		return time.Time{}, false
	}
	if job.Status.CompletionTime != nil && !job.Status.CompletionTime.IsZero() {
		return job.Status.CompletionTime.Time, true
	}
	for _, cond := range job.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		if cond.Type == batchv1.JobComplete || cond.Type == batchv1.JobFailed {
			if !cond.LastTransitionTime.IsZero() {
				return cond.LastTransitionTime.Time, true
			}
			return job.CreationTimestamp.Time, true
		}
	}
	if job.Status.Active <= 0 && (job.Status.Succeeded > 0 || job.Status.Failed > 0) {
		return job.CreationTimestamp.Time, true
	}
	return time.Time{}, false
}

func cronJobGenerateNamePrefix(name string) string {
	prefix := strings.TrimSpace(name) + cronJobManualNameSuffix
	if len(prefix) <= cronJobGeneratedNameMaxLength {
		return prefix
	}
	maxNameLen := cronJobGeneratedNameMaxLength - len(cronJobManualNameSuffix)
	if maxNameLen < 1 {
		return prefix[:cronJobGeneratedNameMaxLength]
	}
	return strings.TrimSpace(name)[:maxNameLen] + cronJobManualNameSuffix
}

func mergeStringMaps(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}
