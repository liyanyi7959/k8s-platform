package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestRolloutHistoryRejectsUnsupportedKind(t *testing.T) {
	var svc *K8sService
	_, err := svc.RolloutHistory(context.Background(), 1, "default", "demo", "StatefulSet")
	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("expected ErrInvalidParams, got %v", err)
	}
	msg, ok := UserMessage(err)
	if !ok || !strings.Contains(msg, "Deployment") {
		t.Fatalf("expected deployment-only message, got %q", msg)
	}
}

func TestBuildDeploymentRolloutHistoryFiltersSortsAndMarksCurrent(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "demo",
			Namespace:   "default",
			UID:         types.UID("dep-uid"),
			Annotations: map[string]string{deploymentRevisionAnnotation: "3"},
		},
	}

	history := buildDeploymentRolloutHistory(deployment, []*appsv1.ReplicaSet{
		newReplicaSetForHistory("demo-rs-2", "demo", "dep-uid", 2, 0, time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC), "scale to 3 replicas", []string{"nginx:1.20.0"}, nil),
		newReplicaSetForHistory("demo-rs-3", "demo", "dep-uid", 3, 2, time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC), "update image to v1.2.0", []string{"nginx:1.21.0"}, []string{"busybox:1.36"}),
		newReplicaSetForHistory("demo-rs-other", "other", "other-uid", 9, 1, time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC), "ignore", []string{"redis:7"}, nil),
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "demo-rs-missing-revision",
				Namespace:         "default",
				CreationTimestamp: metav1.NewTime(time.Date(2026, 4, 4, 11, 0, 0, 0, time.UTC)),
				OwnerReferences:   []metav1.OwnerReference{{Kind: "Deployment", Name: "demo", UID: types.UID("dep-uid")}},
			},
		},
	})

	if len(history) != 2 {
		t.Fatalf("expected 2 rollout revisions, got %d", len(history))
	}
	if history[0].Revision != 3 || !history[0].IsCurrent {
		t.Fatalf("expected revision 3 current first, got %+v", history[0])
	}
	if history[1].Revision != 2 || history[1].IsCurrent {
		t.Fatalf("expected revision 2 non-current second, got %+v", history[1])
	}
	if history[0].ChangeCause != "update image to v1.2.0" {
		t.Fatalf("unexpected change cause: %q", history[0].ChangeCause)
	}
	if len(history[0].Images) != 2 || history[0].Images[0] != "busybox:1.36" || history[0].Images[1] != "nginx:1.21.0" {
		t.Fatalf("unexpected images: %#v", history[0].Images)
	}
}

func TestBuildDeploymentRolloutHistoryFallsBackToActiveReplicaSet(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
			UID:       types.UID("dep-uid"),
		},
	}

	history := buildDeploymentRolloutHistory(deployment, []*appsv1.ReplicaSet{
		newReplicaSetForHistory("demo-rs-2", "demo", "dep-uid", 2, 0, time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC), "", []string{"nginx:1.21.0"}, nil),
		newReplicaSetForHistory("demo-rs-1", "demo", "dep-uid", 1, 1, time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC), "", []string{"nginx:1.20.0"}, nil),
	})

	if len(history) != 2 {
		t.Fatalf("expected 2 rollout revisions, got %d", len(history))
	}
	if history[0].Revision != 2 || history[0].IsCurrent {
		t.Fatalf("expected latest revision not current when replicas=0, got %+v", history[0])
	}
	if history[1].Revision != 1 || !history[1].IsCurrent {
		t.Fatalf("expected active replica set to be current fallback, got %+v", history[1])
	}
}

func TestSelectDeploymentRolloutUndoTarget(t *testing.T) {
	sources := []deploymentRolloutRevisionSource{
		{history: WorkloadRolloutRevision{Revision: 3, IsCurrent: true}},
		{history: WorkloadRolloutRevision{Revision: 2, IsCurrent: false}},
		{history: WorkloadRolloutRevision{Revision: 1, IsCurrent: false}},
	}

	target, err := selectDeploymentRolloutUndoTarget(sources, 0)
	if err != nil {
		t.Fatalf("expected previous revision target, got error %v", err)
	}
	if target.history.Revision != 2 {
		t.Fatalf("expected revision 2 as default rollback target, got %+v", target.history)
	}

	_, err = selectDeploymentRolloutUndoTarget(sources, 3)
	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("expected invalid params for current revision rollback, got %v", err)
	}

	target, err = selectDeploymentRolloutUndoTarget(sources, 1)
	if err != nil {
		t.Fatalf("expected explicit revision target, got error %v", err)
	}
	if target.history.Revision != 1 {
		t.Fatalf("expected explicit revision 1 target, got %+v", target.history)
	}
}

func TestSanitizeRolloutTemplateRemovesReplicaSetHash(t *testing.T) {
	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app":               "demo",
				"pod-template-hash": "abc123",
			},
			ResourceVersion:   "10",
			UID:               types.UID("template-uid"),
			CreationTimestamp: metav1.NewTime(time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)),
		},
	}

	sanitized := sanitizeRolloutTemplate(template)
	if sanitized.Labels["pod-template-hash"] != "" {
		t.Fatalf("expected pod-template-hash to be removed, got %#v", sanitized.Labels)
	}
	if sanitized.Labels["app"] != "demo" {
		t.Fatalf("expected app label to be preserved, got %#v", sanitized.Labels)
	}
	if sanitized.ResourceVersion != "" || sanitized.UID != "" || !sanitized.CreationTimestamp.IsZero() {
		t.Fatalf("expected controller managed metadata to be cleared, got %#v", sanitized.ObjectMeta)
	}
}

func TestBuildWorkloadImagePatchUpdatesContainerAndChangeCause(t *testing.T) {
	patch, err := buildWorkloadImagePatch(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{"name": "app", "image": "nginx:1.26.0"},
						map[string]any{"name": "sidecar", "image": "busybox:1.36"},
					},
				},
			},
		},
	}, "app", "nginx:1.27.0")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	metadata, _ := patch["metadata"].(map[string]any)
	annotations, _ := metadata["annotations"].(map[string]any)
	if annotations[changeCauseAnnotation] != "update image app to nginx:1.27.0" {
		t.Fatalf("unexpected change cause: %#v", annotations)
	}
	spec, _ := patch["spec"].(map[string]any)
	template, _ := spec["template"].(map[string]any)
	templateSpec, _ := template["spec"].(map[string]any)
	containers, _ := templateSpec["containers"].([]any)
	first, _ := containers[0].(map[string]any)
	if first["image"] != "nginx:1.27.0" {
		t.Fatalf("expected first container image updated, got %#v", first)
	}
	second, _ := containers[1].(map[string]any)
	if second["image"] != "busybox:1.36" {
		t.Fatalf("expected sibling container unchanged, got %#v", second)
	}
}

func TestBuildWorkloadImagePatchRejectsUnchangedImage(t *testing.T) {
	_, err := buildWorkloadImagePatch(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{map[string]any{"name": "app", "image": "nginx:1.27.0"}},
				},
			},
		},
	}, "app", "nginx:1.27.0")
	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("expected ErrInvalidParams, got %v", err)
	}
	msg, ok := UserMessage(err)
	if !ok || !strings.Contains(msg, "一致") {
		t.Fatalf("expected unchanged-image message, got %q", msg)
	}
}

func TestBuildWorkloadImagePatchSupportsInitContainer(t *testing.T) {
	patch, err := buildWorkloadImagePatch(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers":     []any{map[string]any{"name": "app", "image": "nginx:1.26.0"}},
					"initContainers": []any{map[string]any{"name": "init-db", "image": "busybox:1.35"}},
				},
			},
		},
	}, "init-db", "busybox:1.36")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	spec, _ := patch["spec"].(map[string]any)
	template, _ := spec["template"].(map[string]any)
	templateSpec, _ := template["spec"].(map[string]any)
	initContainers, _ := templateSpec["initContainers"].([]any)
	first, _ := initContainers[0].(map[string]any)
	if first["image"] != "busybox:1.36" {
		t.Fatalf("expected init container image updated, got %#v", first)
	}
	if _, exists := templateSpec["containers"]; exists {
		t.Fatalf("expected regular containers not to be patched when init container changes")
	}
}

func TestBuildWorkloadImagePatchRejectsMissingContainer(t *testing.T) {
	_, err := buildWorkloadImagePatch(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{map[string]any{"name": "app", "image": "nginx:1.26.0"}},
				},
			},
		},
	}, "worker", "nginx:1.27.0")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func newReplicaSetForHistory(name, ownerName, ownerUID string, revision int, replicas int32, createdAt time.Time, changeCause string, images []string, initImages []string) *appsv1.ReplicaSet {
	containers := make([]corev1.Container, 0, len(images))
	for index, image := range images {
		containers = append(containers, corev1.Container{Name: "container-" + strconv.Itoa(index), Image: image})
	}
	initContainers := make([]corev1.Container, 0, len(initImages))
	for index, image := range initImages {
		initContainers = append(initContainers, corev1.Container{Name: "init-" + strconv.Itoa(index), Image: image})
	}
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(createdAt),
			Annotations: map[string]string{
				deploymentRevisionAnnotation: strconv.Itoa(revision),
				changeCauseAnnotation:        changeCause,
			},
			OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: ownerName, UID: types.UID(ownerUID)}},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: initContainers,
					Containers:     containers,
				},
			},
		},
	}
}
