package application

import (
	"context"
	"strings"
)

type BatchResource string

const (
	BatchJob     BatchResource = "job"
	BatchCronJob BatchResource = "cronjob"
)

type BatchListQuery struct {
	ClusterID uint64
	Namespace string
	SortBy    string
	Order     string
}

type BatchReference struct {
	ClusterID uint64
	Namespace string
	Name      string
}

type JobEditInput struct {
	ClusterID               uint64            `json:"-"`
	Namespace               string            `json:"namespace"`
	Name                    string            `json:"name"`
	Labels                  map[string]string `json:"labels"`
	Parallelism             *int32            `json:"parallelism"`
	Completions             *int32            `json:"completions"`
	BackoffLimit            *int32            `json:"backoffLimit"`
	TTLSecondsAfterFinished *int32            `json:"ttlSecondsAfterFinished"`
}

type CronJobEditInput struct {
	ClusterID                  uint64            `json:"-"`
	Namespace                  string            `json:"namespace"`
	Name                       string            `json:"name"`
	Labels                     map[string]string `json:"labels"`
	Schedule                   string            `json:"schedule"`
	Suspend                    *bool             `json:"suspend"`
	ConcurrencyPolicy          *string           `json:"concurrencyPolicy"`
	SuccessfulJobsHistoryLimit *int32            `json:"successfulJobsHistoryLimit"`
	FailedJobsHistoryLimit     *int32            `json:"failedJobsHistoryLimit"`
}

type CronJobTriggerResult struct {
	JobName string `json:"job_name"`
}

type BatchRuntime interface {
	List(context.Context, BatchResource, BatchListQuery) (any, error)
	YAML(context.Context, BatchResource, BatchReference) (any, error)
	Delete(context.Context, BatchResource, BatchReference) error
	EditJob(context.Context, JobEditInput) error
	DeleteCompletedJobs(context.Context, uint64, string, int) (int, error)
	EditCronJob(context.Context, CronJobEditInput) error
	TriggerCronJob(context.Context, BatchReference) (*CronJobTriggerResult, error)
	SuspendCronJob(context.Context, BatchReference, bool) error
}

type BatchService struct{ runtime BatchRuntime }

func NewBatchService(runtime BatchRuntime) *BatchService { return &BatchService{runtime: runtime} }

func (s *BatchService) List(ctx context.Context, resource BatchResource, query BatchListQuery) (any, error) {
	if !validBatchResource(resource) || query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.Namespace = strings.TrimSpace(query.Namespace)
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.Order = strings.TrimSpace(query.Order)
	return s.runtime.List(ctx, resource, query)
}

func (s *BatchService) YAML(ctx context.Context, resource BatchResource, ref BatchReference) (any, error) {
	if err := validateBatchReference(resource, ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, resource, normalizeBatchReference(ref))
}

func (s *BatchService) Delete(ctx context.Context, resource BatchResource, ref BatchReference) error {
	if err := validateBatchReference(resource, ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, resource, normalizeBatchReference(ref))
}

func (s *BatchService) EditJob(ctx context.Context, input JobEditInput) error {
	if err := normalizeJobEdit(&input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.EditJob(ctx, input)
}

func (s *BatchService) DeleteCompletedJobs(ctx context.Context, clusterID uint64, namespace string, olderThanHours int) (int, error) {
	if clusterID == 0 || olderThanHours < 0 {
		return 0, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return 0, ErrConflict
	}
	return s.runtime.DeleteCompletedJobs(ctx, clusterID, strings.TrimSpace(namespace), olderThanHours)
}

func (s *BatchService) EditCronJob(ctx context.Context, input CronJobEditInput) error {
	if err := normalizeCronJobEdit(&input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.EditCronJob(ctx, input)
}

func (s *BatchService) TriggerCronJob(ctx context.Context, ref BatchReference) (*CronJobTriggerResult, error) {
	if err := validateBatchReference(BatchCronJob, ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.TriggerCronJob(ctx, normalizeBatchReference(ref))
}

func (s *BatchService) SuspendCronJob(ctx context.Context, ref BatchReference, suspend bool) error {
	if err := validateBatchReference(BatchCronJob, ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.SuspendCronJob(ctx, normalizeBatchReference(ref), suspend)
}

func validBatchResource(resource BatchResource) bool {
	return resource == BatchJob || resource == BatchCronJob
}

func validateBatchReference(resource BatchResource, ref BatchReference) error {
	if !validBatchResource(resource) || ref.ClusterID == 0 || strings.TrimSpace(ref.Namespace) == "" || strings.TrimSpace(ref.Name) == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizeBatchReference(ref BatchReference) BatchReference {
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	ref.Name = strings.TrimSpace(ref.Name)
	return ref
}

func normalizeJobEdit(input *JobEditInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" || negativeInt32(input.Parallelism) || negativeInt32(input.Completions) || negativeInt32(input.BackoffLimit) || negativeInt32(input.TTLSecondsAfterFinished) {
		return ErrInvalidParams
	}
	return nil
}

func normalizeCronJobEdit(input *CronJobEditInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	input.Schedule = strings.TrimSpace(input.Schedule)
	if input.ConcurrencyPolicy != nil {
		value := strings.TrimSpace(*input.ConcurrencyPolicy)
		input.ConcurrencyPolicy = &value
	}
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" || input.Schedule == "" || negativeInt32(input.SuccessfulJobsHistoryLimit) || negativeInt32(input.FailedJobsHistoryLimit) {
		return ErrInvalidParams
	}
	return nil
}

func negativeInt32(value *int32) bool { return value != nil && *value < 0 }
