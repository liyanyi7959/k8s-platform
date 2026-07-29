package application

import (
	"context"
	"errors"
	"testing"

	"k8s-platform-backend/internal/kops/domain"
)

type actionProposalRuntimeStub struct {
	inspectedAction string
	inspectedTarget ActionProposalTarget
	object          map[string]any
	scaledTarget    ActionProposalTarget
	scaledReplicas  int
}

func (stub *actionProposalRuntimeStub) Inspect(_ context.Context, _ uint64, action string, target ActionProposalTarget) (map[string]any, error) {
	stub.inspectedAction, stub.inspectedTarget = action, target
	return stub.object, nil
}
func (*actionProposalRuntimeStub) Restart(context.Context, uint64, ActionProposalTarget) error {
	return nil
}
func (stub *actionProposalRuntimeStub) Scale(_ context.Context, _ uint64, target ActionProposalTarget, replicas int) error {
	stub.scaledTarget, stub.scaledReplicas = target, replicas
	return nil
}
func (*actionProposalRuntimeStub) UpdateImage(context.Context, uint64, ActionProposalTarget, string, string) error {
	return nil
}
func (*actionProposalRuntimeStub) Pause(context.Context, uint64, ActionProposalTarget, bool) error {
	return nil
}
func (*actionProposalRuntimeStub) Undo(context.Context, uint64, ActionProposalTarget, int) error {
	return nil
}
func (*actionProposalRuntimeStub) DeleteWorkload(context.Context, uint64, ActionProposalTarget) error {
	return nil
}
func (*actionProposalRuntimeStub) DeleteResource(context.Context, uint64, ActionProposalTarget) error {
	return nil
}
func (*actionProposalRuntimeStub) DeletePod(context.Context, uint64, ActionProposalTarget, bool) error {
	return nil
}
func (*actionProposalRuntimeStub) SetNodeSchedulable(context.Context, uint64, string, bool) error {
	return nil
}
func (*actionProposalRuntimeStub) DrainNode(context.Context, uint64, string, ActionProposalDrainOptions) error {
	return nil
}
func (*actionProposalRuntimeStub) TriggerCronJob(context.Context, uint64, ActionProposalTarget) (string, error) {
	return "job-once", nil
}
func (*actionProposalRuntimeStub) SuspendCronJob(context.Context, uint64, ActionProposalTarget, bool) error {
	return nil
}
func (*actionProposalRuntimeStub) DeleteCompletedJobs(context.Context, uint64, string, int) (int, error) {
	return 0, nil
}
func (*actionProposalRuntimeStub) ApplyManifest(context.Context, ActionProposalManifestRequest) (ActionProposalManifestResult, error) {
	return ActionProposalManifestResult{}, errors.New("unexpected manifest apply")
}

func TestActionProposalPrepareScaleBuildsPreviewFromRuntimeObject(t *testing.T) {
	runtime := &actionProposalRuntimeStub{object: map[string]any{"spec": map[string]any{"replicas": 2}}}
	service := NewActionProposalService(runtime)
	prepared, err := service.Prepare(context.Background(), ActionProposalPrepareRequest{
		ClusterID: 1, ActionType: proposalActionScaleWorkload,
		Target:  ActionProposalTarget{Kind: "deployment", Namespace: "default", Name: "web"},
		Payload: domain.JSONMap{"replicas": 5},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if runtime.inspectedAction != proposalActionScaleWorkload || runtime.inspectedTarget.Kind != "Deployment" {
		t.Fatalf("inspect=%q %#v", runtime.inspectedAction, runtime.inspectedTarget)
	}
	if prepared.Target.Kind != "Deployment" || prepared.Change["current_replicas"] != 2 || prepared.Change["target_replicas"] != 5 {
		t.Fatalf("prepared=%#v", prepared)
	}
	if prepared.Preview != "replicas: 2 -> 5" {
		t.Fatalf("preview=%q", prepared.Preview)
	}
}

func TestActionProposalExecuteScaleDelegatesNormalizedTarget(t *testing.T) {
	runtime := &actionProposalRuntimeStub{}
	service := NewActionProposalService(runtime)
	result, err := service.Execute(context.Background(), ActionProposalExecutionRequest{
		ClusterID: 1, ActionType: proposalActionScaleWorkload,
		Target: ActionProposalTarget{Kind: "statefulset", Namespace: "apps", Name: "database"},
		Change: domain.JSONMap{"payload": map[string]any{"replicas": float64(3)}},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if runtime.scaledTarget.Kind != "StatefulSet" || runtime.scaledReplicas != 3 {
		t.Fatalf("scale=%#v replicas=%d", runtime.scaledTarget, runtime.scaledReplicas)
	}
	if result.Result["action_type"] != proposalActionScaleWorkload || result.Result["target_replicas"] != 3 {
		t.Fatalf("result=%#v", result)
	}
}

func TestActionProposalRejectsUnsupportedAction(t *testing.T) {
	service := NewActionProposalService(&actionProposalRuntimeStub{})
	_, err := service.Prepare(context.Background(), ActionProposalPrepareRequest{ClusterID: 1, ActionType: "unknown"})
	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("error=%v", err)
	}
}
