package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	kopsapp "k8s-platform-backend/internal/kops/application"
	model "k8s-platform-backend/internal/kops/domain"
)

type WorkloadActionTarget struct {
	Kind      string
	Namespace string
	Name      string
}

type PrepareWorkloadActionRequest struct {
	ClusterID  uint64
	ActionType string
	Target     WorkloadActionTarget
	Payload    model.JSONMap
	Reason     string
}

type PreparedWorkloadAction struct {
	ActionType   string
	Target       WorkloadActionTarget
	RiskLevel    string
	ConfirmLevel string
	Title        string
	Summary      string
	Change       model.JSONMap
	Preview      string
	Diff         string
}

type ExecuteWorkloadActionRequest struct {
	ClusterID     uint64
	ActionType    string
	Target        WorkloadActionTarget
	Change        model.JSONMap
	ProposalTitle string
	CreatedBy     uint64
	CreatedByName string
}

type ExecuteWorkloadActionResult struct {
	Result  model.JSONMap
	Summary string
}

// WorkloadActionService keeps the previous service-facing type available to
// AI callers. All proposal policy and mutation dispatch now live in Kops.
type WorkloadActionService struct {
	k8sSvc      *K8sService
	manifestSvc *ManifestApplyRecordService
	core        *kopsapp.ActionProposalService
}

func NewWorkloadActionService(k8sSvc *K8sService, manifestSvc *ManifestApplyRecordService) *WorkloadActionService {
	service := &WorkloadActionService{k8sSvc: k8sSvc, manifestSvc: manifestSvc}
	service.core = kopsapp.NewActionProposalService(legacyActionProposalRuntime{service: service})
	return service
}

func (s *WorkloadActionService) PrepareProposal(ctx context.Context, req PrepareWorkloadActionRequest) (PreparedWorkloadAction, error) {
	if s == nil || s.core == nil {
		return PreparedWorkloadAction{}, errors.New("workload action service is required")
	}
	prepared, err := s.core.Prepare(ctx, kopsapp.ActionProposalPrepareRequest{
		ClusterID: req.ClusterID, ActionType: req.ActionType,
		Target:  kopsapp.ActionProposalTarget{Kind: req.Target.Kind, Namespace: req.Target.Namespace, Name: req.Target.Name},
		Payload: req.Payload, Reason: req.Reason,
	})
	if err != nil {
		return PreparedWorkloadAction{}, mapKopsActionProposalError(err)
	}
	return PreparedWorkloadAction{
		ActionType: prepared.ActionType,
		Target:     WorkloadActionTarget{Kind: prepared.Target.Kind, Namespace: prepared.Target.Namespace, Name: prepared.Target.Name},
		RiskLevel:  prepared.RiskLevel, ConfirmLevel: prepared.ConfirmLevel, Title: prepared.Title, Summary: prepared.Summary,
		Change: prepared.Change, Preview: prepared.Preview, Diff: prepared.Diff,
	}, nil
}

func (s *WorkloadActionService) ExecuteProposalAction(ctx context.Context, req ExecuteWorkloadActionRequest) (ExecuteWorkloadActionResult, error) {
	if s == nil || s.core == nil {
		return ExecuteWorkloadActionResult{}, errors.New("workload action service is required")
	}
	result, err := s.core.Execute(ctx, kopsapp.ActionProposalExecutionRequest{
		ClusterID: req.ClusterID, ActionType: req.ActionType,
		Target: kopsapp.ActionProposalTarget{Kind: req.Target.Kind, Namespace: req.Target.Namespace, Name: req.Target.Name},
		Change: req.Change, ProposalTitle: req.ProposalTitle, CreatedBy: req.CreatedBy, CreatedByName: req.CreatedByName,
	})
	if err != nil {
		return ExecuteWorkloadActionResult{}, mapKopsActionProposalError(err)
	}
	return ExecuteWorkloadActionResult{Result: result.Result, Summary: result.Summary}, nil
}

type legacyActionProposalRuntime struct{ service *WorkloadActionService }

func (runtime legacyActionProposalRuntime) Inspect(ctx context.Context, clusterID uint64, actionType string, target kopsapp.ActionProposalTarget) (map[string]any, error) {
	if runtime.service == nil || runtime.service.k8sSvc == nil {
		return nil, errors.New("kubernetes runtime is required")
	}
	gvr, namespace, err := legacyActionProposalGVR(actionType, target)
	if err != nil {
		return nil, err
	}
	return runtime.service.k8sSvc.GetObject(ctx, clusterID, gvr, namespace, target.Name)
}

func (runtime legacyActionProposalRuntime) Restart(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget) error {
	gvr, ok := aiActionWorkloadGVR(target.Kind)
	if !ok {
		return ErrWithMessage(ErrInvalidParams, "restart proposal resource kind is not supported")
	}
	return runtime.service.k8sSvc.PatchJSON(ctx, clusterID, gvr, target.Namespace, target.Name, map[string]any{
		"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]any{
			"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339),
		}}}},
	})
}

func (runtime legacyActionProposalRuntime) Scale(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, replicas int) error {
	gvr, ok := aiActionWorkloadGVR(target.Kind)
	if !ok {
		return ErrWithMessage(ErrInvalidParams, "scale proposal resource kind is not supported")
	}
	return runtime.service.k8sSvc.PatchJSON(ctx, clusterID, gvr, target.Namespace, target.Name, map[string]any{"spec": map[string]any{"replicas": replicas}})
}

func (runtime legacyActionProposalRuntime) UpdateImage(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, container, image string) error {
	return runtime.service.k8sSvc.UpdateWorkloadImage(ctx, clusterID, target.Namespace, target.Name, target.Kind, container, image)
}

func (runtime legacyActionProposalRuntime) Pause(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, paused bool) error {
	return runtime.service.k8sSvc.UpdateWorkloadPaused(ctx, clusterID, target.Namespace, target.Name, target.Kind, paused)
}

func (runtime legacyActionProposalRuntime) Undo(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, revision int) error {
	return runtime.service.k8sSvc.RolloutUndo(ctx, clusterID, target.Namespace, target.Name, target.Kind, revision)
}

func (runtime legacyActionProposalRuntime) DeleteWorkload(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget) error {
	gvr, ok := aiActionWorkloadGVR(target.Kind)
	if !ok {
		return ErrWithMessage(ErrInvalidParams, "delete workload proposal resource kind is not supported")
	}
	return runtime.service.k8sSvc.Delete(ctx, clusterID, gvr, target.Namespace, target.Name)
}

func (runtime legacyActionProposalRuntime) DeleteResource(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget) error {
	gvr, namespaced, ok := aiSupportedResourceGVR(target.Kind)
	if !ok {
		return ErrWithMessage(ErrInvalidParams, "delete resource proposal kind is not supported")
	}
	namespace := target.Namespace
	if !namespaced {
		namespace = ""
	}
	return runtime.service.k8sSvc.Delete(ctx, clusterID, gvr, namespace, target.Name)
}

func (runtime legacyActionProposalRuntime) DeletePod(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, force bool) error {
	return runtime.service.k8sSvc.DeletePod(ctx, clusterID, target.Namespace, target.Name, force)
}

func (runtime legacyActionProposalRuntime) SetNodeSchedulable(ctx context.Context, clusterID uint64, name string, unschedulable bool) error {
	return runtime.service.k8sSvc.UpdateNodeSchedulable(ctx, clusterID, name, unschedulable)
}

func (runtime legacyActionProposalRuntime) DrainNode(ctx context.Context, clusterID uint64, name string, options kopsapp.ActionProposalDrainOptions) error {
	return runtime.service.k8sSvc.DrainNode(ctx, clusterID, name, DrainNodeOptions{TimeoutSeconds: options.TimeoutSeconds, Force: options.Force, IgnoreDaemonSets: options.IgnoreDaemonSets})
}

func (runtime legacyActionProposalRuntime) TriggerCronJob(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget) (string, error) {
	result, err := runtime.service.k8sSvc.TriggerCronJob(ctx, clusterID, target.Namespace, target.Name)
	if err != nil {
		return "", err
	}
	return result.JobName, nil
}

func (runtime legacyActionProposalRuntime) SuspendCronJob(ctx context.Context, clusterID uint64, target kopsapp.ActionProposalTarget, suspend bool) error {
	return runtime.service.k8sSvc.SuspendCronJob(ctx, clusterID, target.Namespace, target.Name, suspend)
}

func (runtime legacyActionProposalRuntime) DeleteCompletedJobs(ctx context.Context, clusterID uint64, namespace string, olderThanHours int) (int, error) {
	return runtime.service.k8sSvc.DeleteCompletedJobs(ctx, clusterID, namespace, olderThanHours)
}

func (runtime legacyActionProposalRuntime) ApplyManifest(ctx context.Context, input kopsapp.ActionProposalManifestRequest) (kopsapp.ActionProposalManifestResult, error) {
	if runtime.service == nil || runtime.service.manifestSvc == nil {
		return kopsapp.ActionProposalManifestResult{}, ErrWithMessage(ErrConflict, "Manifest 执行服务未初始化")
	}
	result, err := runtime.service.manifestSvc.Execute(ctx, ManifestApplyExecuteRequest{
		ClusterID: input.ClusterID, YAML: input.YAML, DefaultNamespace: input.DefaultNamespace, DryRun: false,
		SourceLabel: input.SourceLabel, SourceResource: input.SourceResource, CreatedBy: input.CreatedBy, CreatedByName: input.CreatedByName,
	})
	if err != nil {
		return kopsapp.ActionProposalManifestResult{}, err
	}
	return kopsapp.ActionProposalManifestResult{RecordID: result.RecordID, Status: result.Status, Summary: result.Summary}, nil
}

func legacyActionProposalGVR(actionType string, target kopsapp.ActionProposalTarget) (schema.GroupVersionResource, string, error) {
	switch strings.TrimSpace(actionType) {
	case aiActionTypeRestartWorkload, aiActionTypeScaleWorkload, aiActionTypeUpdateWorkloadImage, aiActionTypePauseWorkloadRollout, aiActionTypeRolloutUndo, aiActionTypeDeleteWorkload:
		gvr, ok := aiActionWorkloadGVR(target.Kind)
		if !ok {
			return schema.GroupVersionResource{}, "", ErrWithMessage(ErrInvalidParams, "workload resource kind is not supported")
		}
		return gvr, target.Namespace, nil
	case aiActionTypeDeleteResource:
		gvr, namespaced, ok := aiSupportedResourceGVR(target.Kind)
		if !ok {
			return schema.GroupVersionResource{}, "", ErrWithMessage(ErrInvalidParams, "resource kind is not supported")
		}
		if !namespaced {
			return gvr, "", nil
		}
		return gvr, target.Namespace, nil
	case aiActionTypeDeletePod:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, target.Namespace, nil
	case aiActionTypeCordonNode, aiActionTypeUncordonNode, aiActionTypeDrainNode:
		return schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}, "", nil
	case aiActionTypeTriggerCronJob, aiActionTypeSuspendCronJob:
		return schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, target.Namespace, nil
	default:
		return schema.GroupVersionResource{}, "", ErrWithMessage(ErrInvalidParams, "action inspection is not supported")
	}
}

func mapKopsActionProposalError(err error) error {
	if err == nil {
		return nil
	}
	for _, candidate := range []struct{ application, legacy error }{
		{kopsapp.ErrInvalidParams, ErrInvalidParams}, {kopsapp.ErrNotFound, ErrNotFound}, {kopsapp.ErrConflict, ErrConflict},
	} {
		if errors.Is(err, candidate.application) {
			if message, ok := UserMessage(err); ok {
				return ErrWithMessage(candidate.legacy, message)
			}
			return candidate.legacy
		}
	}
	return err
}

var _ kopsapp.ActionProposalRuntime = legacyActionProposalRuntime{}
