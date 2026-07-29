package kops

import (
	"errors"
	"testing"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

func TestBuildWorkloadNormalizesDefaultsAndResources(t *testing.T) {
	namespace, name, replicas, labels, containers, err := buildWorkload(kopsapp.WorkloadCreateInput{
		Namespace: " ops ", Name: " api ", Replicas: 0,
		Labels:     map[string]string{"team": " platform ", " ": "ignored"},
		Containers: []kopsapp.WorkloadContainer{{Name: "api", Image: "nginx:stable", CPU: "100m", Memory: "128Mi", Command: "sh -c run"}},
	})
	if err != nil {
		t.Fatalf("buildWorkload() error = %v", err)
	}
	if namespace != "ops" || name != "api" || replicas != 1 || labels["app"] != "api" || labels["team"] != "platform" {
		t.Fatalf("unexpected workload metadata: %q %q %d %#v", namespace, name, replicas, labels)
	}
	if len(containers) != 1 || len(containers[0].Command) != 3 || containers[0].Resources.Requests.Cpu().String() != "100m" || containers[0].Resources.Limits.Memory().String() != "128Mi" {
		t.Fatalf("unexpected containers: %#v", containers)
	}
}

func TestBuildWorkloadRejectsInvalidContainerAndQuantity(t *testing.T) {
	_, _, _, _, _, err := buildWorkload(kopsapp.WorkloadCreateInput{Namespace: "ops", Name: "api", Containers: []kopsapp.WorkloadContainer{{Name: "", Image: "nginx"}}})
	if !errors.Is(err, kopsapp.ErrInvalidParams) {
		t.Fatalf("missing container name error = %v", err)
	}
	_, _, _, _, _, err = buildWorkload(kopsapp.WorkloadCreateInput{Namespace: "ops", Name: "api", Containers: []kopsapp.WorkloadContainer{{Name: "api", Image: "nginx", CPU: "invalid"}}})
	if !errors.Is(err, kopsapp.ErrInvalidParams) {
		t.Fatalf("invalid CPU error = %v", err)
	}
}
