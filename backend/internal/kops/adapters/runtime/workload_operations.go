package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	kopsapp "k8s-platform-backend/internal/kops/application"
)

const (
	deploymentRevisionAnnotation         = "deployment.kubernetes.io/revision"
	workloadChangeCauseAnnotation        = "kubernetes.io/change-cause"
	workloadOperationListPageLimit int64 = 500
)

// WorkloadOperations owns direct Kubernetes interactions for rollout history,
// rollback, image replacement and Deployment pause state. Client creation is
// injected through the same narrow typed/dynamic transport as node operations.
type WorkloadOperations struct{ transport NodeOperationsTransport }

func NewWorkloadOperations(transport NodeOperationsTransport) *WorkloadOperations {
	return &WorkloadOperations{transport: transport}
}

type WorkloadRolloutRevision struct {
	Revision    int       `json:"revision"`
	ChangeCause string    `json:"change_cause"`
	Images      []string  `json:"images"`
	CreatedAt   time.Time `json:"created_at"`
	IsCurrent   bool      `json:"is_current"`
}

type deploymentRolloutRevisionSource struct {
	history    WorkloadRolloutRevision
	replicaSet *appsv1.ReplicaSet
}

func (o *WorkloadOperations) RolloutHistory(ctx context.Context, clusterID uint64, namespace, name, kind string) ([]WorkloadRolloutRevision, error) {
	if !strings.EqualFold(strings.TrimSpace(kind), "Deployment") {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "当前仅支持 Deployment 的版本历史")
	}
	namespace, name = strings.TrimSpace(namespace), strings.TrimSpace(name)
	if namespace == "" || name == "" {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "命名空间和工作负载名称不能为空")
	}
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, nodeOperationError(err)
	}
	replicaSets, err := listReplicaSets(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	return buildDeploymentRolloutHistory(deployment, replicaSets), nil
}

func (o *WorkloadOperations) RolloutUndo(ctx context.Context, clusterID uint64, namespace, name, kind string, revision int) error {
	if !strings.EqualFold(strings.TrimSpace(kind), "Deployment") {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "当前仅支持 Deployment 回滚")
	}
	namespace, name = strings.TrimSpace(namespace), strings.TrimSpace(name)
	if namespace == "" || name == "" {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "命名空间和工作负载名称不能为空")
	}
	if revision < 0 {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "回滚版本不能小于 0")
	}
	client, err := o.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}
	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nodeOperationError(err)
	}
	replicaSets, err := listReplicaSets(ctx, client, namespace)
	if err != nil {
		return err
	}
	target, err := selectDeploymentRolloutUndoTarget(buildDeploymentRolloutSources(deployment, replicaSets), revision)
	if err != nil {
		return err
	}
	updated := deployment.DeepCopy()
	updated.Spec.Template = sanitizeRolloutTemplate(target.replicaSet.Spec.Template)
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	updated.Annotations[workloadChangeCauseAnnotation] = buildRolloutUndoChangeCause(target.history)
	_, err = client.AppsV1().Deployments(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	return nodeOperationError(err)
}

func (o *WorkloadOperations) UpdateWorkloadImage(ctx context.Context, clusterID uint64, namespace, name, kind, containerName, image string) error {
	namespace, name = strings.TrimSpace(namespace), strings.TrimSpace(name)
	containerName, image = strings.TrimSpace(containerName), strings.TrimSpace(image)
	if namespace == "" || name == "" || containerName == "" || image == "" {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "命名空间、工作负载、容器名称和镜像不能为空")
	}
	gvr, ok := workloadGVR(kind)
	if !ok {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "仅支持 Deployment、StatefulSet、DaemonSet 更新镜像")
	}
	client, err := o.dynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	resource := client.Resource(gvr).Namespace(namespace)
	object, err := resource.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nodeOperationError(err)
	}
	patch, err := buildWorkloadImagePatch(object.Object, containerName, image)
	if err != nil {
		return err
	}
	return patchWorkload(ctx, resource, name, patch)
}

func (o *WorkloadOperations) UpdateWorkloadPaused(ctx context.Context, clusterID uint64, namespace, name, kind string, paused bool) error {
	if !strings.EqualFold(strings.TrimSpace(kind), "Deployment") {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "当前仅支持 Deployment 暂停或恢复 Rollout")
	}
	namespace, name = strings.TrimSpace(namespace), strings.TrimSpace(name)
	if namespace == "" || name == "" {
		return kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "命名空间和工作负载名称不能为空")
	}
	client, err := o.dynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	return patchWorkload(ctx, client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}).Namespace(namespace), name, map[string]any{
		"spec": map[string]any{"paused": paused},
	})
}

func (o *WorkloadOperations) typedClient(ctx context.Context, clusterID uint64) (*kubernetes.Clientset, error) {
	if o == nil || o.transport == nil {
		return nil, kopsapp.ErrConflict
	}
	return o.transport.TypedClient(ctx, clusterID)
}

func (o *WorkloadOperations) dynamicClient(ctx context.Context, clusterID uint64) (*dynamic.DynamicClient, error) {
	if o == nil || o.transport == nil {
		return nil, kopsapp.ErrConflict
	}
	return o.transport.DynamicClient(ctx, clusterID)
}

func workloadGVR(kind string) (schema.GroupVersionResource, bool) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment", "deployments":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, true
	case "statefulset", "statefulsets":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, true
	case "daemonset", "daemonsets":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, true
	default:
		return schema.GroupVersionResource{}, false
	}
}

func patchWorkload(ctx context.Context, resource dynamic.ResourceInterface, name string, patch map[string]any) error {
	payload, err := json.Marshal(patch)
	if err != nil {
		return kopsapp.ErrInvalidParams
	}
	_, err = resource.Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return nodeOperationError(err)
}

func buildWorkloadImagePatch(object map[string]any, containerName, image string) (map[string]any, error) {
	spec, _ := object["spec"].(map[string]any)
	template, _ := spec["template"].(map[string]any)
	templateSpec, _ := template["spec"].(map[string]any)
	if spec == nil || template == nil || templateSpec == nil {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "工作负载模板异常，无法更新镜像")
	}
	containers, _ := templateSpec["containers"].([]any)
	if len(containers) == 0 {
		return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "工作负载未配置容器，无法更新镜像")
	}
	if updated, found, changed := updateContainerImageList(containers, containerName, image); found {
		if !changed {
			return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "新镜像与当前镜像一致")
		}
		return workloadImagePatch("containers", updated, containerName, image), nil
	}
	initContainers, _ := templateSpec["initContainers"].([]any)
	if updated, found, changed := updateContainerImageList(initContainers, containerName, image); found {
		if !changed {
			return nil, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "新镜像与当前镜像一致")
		}
		return workloadImagePatch("initContainers", updated, containerName, image), nil
	}
	return nil, kopsapp.ErrWithMessage(kopsapp.ErrNotFound, "未找到指定容器")
}

func workloadImagePatch(containerKey string, containers []any, containerName, image string) map[string]any {
	return map[string]any{
		"metadata": map[string]any{"annotations": map[string]any{workloadChangeCauseAnnotation: buildWorkloadImageChangeCause(containerName, image)}},
		"spec":     map[string]any{"template": map[string]any{"spec": map[string]any{containerKey: containers}}},
	}
}

func updateContainerImageList(items []any, containerName, image string) ([]any, bool, bool) {
	updated := cloneMapSlice(items)
	for index, raw := range updated {
		item, _ := raw.(map[string]any)
		if item == nil || strings.TrimSpace(fmt.Sprint(item["name"])) != containerName {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(item["image"])) == image {
			return updated, true, false
		}
		item["image"] = image
		updated[index] = item
		return updated, true, true
	}
	return updated, false, false
}

func cloneMapSlice(items []any) []any {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]any, 0, len(items))
	for _, raw := range items {
		if item, ok := raw.(map[string]any); ok && item != nil {
			copy := make(map[string]any, len(item))
			for key, value := range item {
				copy[key] = value
			}
			cloned = append(cloned, copy)
			continue
		}
		cloned = append(cloned, raw)
	}
	return cloned
}

func buildWorkloadImageChangeCause(containerName, image string) string {
	return fmt.Sprintf("update image %s to %s", strings.TrimSpace(containerName), strings.TrimSpace(image))
}

func listReplicaSets(ctx context.Context, client *kubernetes.Clientset, namespace string) ([]*appsv1.ReplicaSet, error) {
	if client == nil {
		return nil, kopsapp.ErrRuntime
	}
	options := metav1.ListOptions{Limit: workloadOperationListPageLimit}
	items := make([]*appsv1.ReplicaSet, 0, 64)
	for {
		list, err := client.AppsV1().ReplicaSets(namespace).List(ctx, options)
		if err != nil {
			return nil, nodeOperationError(err)
		}
		for index := range list.Items {
			items = append(items, list.Items[index].DeepCopy())
		}
		if strings.TrimSpace(list.Continue) == "" {
			return items, nil
		}
		options.Continue = list.Continue
	}
}

func buildDeploymentRolloutHistory(deployment *appsv1.Deployment, replicaSets []*appsv1.ReplicaSet) []WorkloadRolloutRevision {
	sources := buildDeploymentRolloutSources(deployment, replicaSets)
	items := make([]WorkloadRolloutRevision, 0, len(sources))
	for _, source := range sources {
		items = append(items, source.history)
	}
	return items
}

func buildDeploymentRolloutSources(deployment *appsv1.Deployment, replicaSets []*appsv1.ReplicaSet) []deploymentRolloutRevisionSource {
	if deployment == nil || len(replicaSets) == 0 {
		return []deploymentRolloutRevisionSource{}
	}
	currentRevision, hasCurrentRevision := parseRevisionAnnotation(deployment.Annotations)
	items := make([]deploymentRolloutRevisionSource, 0, len(replicaSets))
	for _, replicaSet := range replicaSets {
		if !replicaSetOwnedByDeployment(replicaSet, deployment) {
			continue
		}
		revision, ok := parseRevisionAnnotation(replicaSet.Annotations)
		if !ok || revision <= 0 {
			continue
		}
		isCurrent := hasCurrentRevision && revision == currentRevision
		if !hasCurrentRevision && replicaSet.Spec.Replicas != nil && *replicaSet.Spec.Replicas > 0 {
			isCurrent = true
		}
		items = append(items, deploymentRolloutRevisionSource{
			history: WorkloadRolloutRevision{
				Revision: revision, ChangeCause: strings.TrimSpace(replicaSet.Annotations[workloadChangeCauseAnnotation]),
				Images: extractReplicaSetImages(replicaSet), CreatedAt: replicaSet.CreationTimestamp.UTC(), IsCurrent: isCurrent,
			},
			replicaSet: replicaSet,
		})
	}
	sort.SliceStable(items, func(left, right int) bool {
		if items[left].history.Revision == items[right].history.Revision {
			return items[left].history.CreatedAt.After(items[right].history.CreatedAt)
		}
		return items[left].history.Revision > items[right].history.Revision
	})
	return items
}

func selectDeploymentRolloutUndoTarget(sources []deploymentRolloutRevisionSource, revision int) (deploymentRolloutRevisionSource, error) {
	if len(sources) == 0 {
		return deploymentRolloutRevisionSource{}, kopsapp.ErrWithMessage(kopsapp.ErrNotFound, "未找到该 Deployment 的历史版本")
	}
	if revision == 0 {
		for _, source := range sources {
			if !source.history.IsCurrent {
				return source, nil
			}
		}
		return deploymentRolloutRevisionSource{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "已是最早版本，无法继续回滚")
	}
	for _, source := range sources {
		if source.history.Revision != revision {
			continue
		}
		if source.history.IsCurrent {
			return deploymentRolloutRevisionSource{}, kopsapp.ErrWithMessage(kopsapp.ErrInvalidParams, "请选择当前版本之前的历史版本进行回滚")
		}
		return source, nil
	}
	return deploymentRolloutRevisionSource{}, kopsapp.ErrWithMessage(kopsapp.ErrNotFound, fmt.Sprintf("目标回滚版本 r%d 不存在", revision))
}

func sanitizeRolloutTemplate(template corev1.PodTemplateSpec) corev1.PodTemplateSpec {
	sanitized := *template.DeepCopy()
	sanitized.ResourceVersion, sanitized.UID = "", ""
	sanitized.CreationTimestamp = metav1.Time{}
	sanitized.ManagedFields = nil
	if len(sanitized.Labels) > 0 {
		delete(sanitized.Labels, "pod-template-hash")
		if len(sanitized.Labels) == 0 {
			sanitized.Labels = nil
		}
	}
	return sanitized
}

func buildRolloutUndoChangeCause(target WorkloadRolloutRevision) string {
	base := fmt.Sprintf("rollback to revision %d", target.Revision)
	if target.ChangeCause == "" {
		return base
	}
	return base + ": " + target.ChangeCause
}

func replicaSetOwnedByDeployment(replicaSet *appsv1.ReplicaSet, deployment *appsv1.Deployment) bool {
	if replicaSet == nil || deployment == nil {
		return false
	}
	for _, reference := range replicaSet.OwnerReferences {
		if reference.Kind == "Deployment" && reference.Name == deployment.Name && (reference.UID == "" || deployment.UID == "" || reference.UID == deployment.UID) {
			return true
		}
	}
	return false
}

func parseRevisionAnnotation(annotations map[string]string) (int, bool) {
	revision, err := strconv.Atoi(strings.TrimSpace(annotations[deploymentRevisionAnnotation]))
	return revision, err == nil && revision > 0
}

func extractReplicaSetImages(replicaSet *appsv1.ReplicaSet) []string {
	if replicaSet == nil {
		return []string{}
	}
	items := append(append([]corev1.Container(nil), replicaSet.Spec.Template.Spec.InitContainers...), replicaSet.Spec.Template.Spec.Containers...)
	images, seen := make([]string, 0, len(items)), map[string]struct{}{}
	for _, container := range items {
		image := strings.TrimSpace(container.Image)
		if image == "" {
			continue
		}
		if _, exists := seen[image]; exists {
			continue
		}
		seen[image] = struct{}{}
		images = append(images, image)
	}
	return images
}
