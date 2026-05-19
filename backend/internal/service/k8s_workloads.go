package service

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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
	changeCauseAnnotation        = "kubernetes.io/change-cause"
)

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

// ===========================================================================
// Deployments
// ===========================================================================

func (s *K8sService) listDeploymentsCached(ctx context.Context, clusterID uint64, namespace string, sortBy, order string, labelSelector string) ([]any, error) {
	entry, err := s.getOrStartObjCache(ctx, clusterID, "deployments")
	if err != nil {
		return nil, err
	}

	var selector labels.Selector
	if strings.TrimSpace(labelSelector) != "" {
		sel, err := labels.Parse(labelSelector)
		if err != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "label_selector 无效")
		}
		selector = sel
	}

	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && s.podTTL > 0 && clusterID > 0
	if cacheEnabled {
		if out, ok := s.listDeploymentsFromRedisAll(ctx, clusterID, namespace, selector, sortBy, order); ok {
			return out, nil
		}
	}

	if !entry.informer.HasSynced() {
		items, err := s.listDeploymentsDirectDeployments(ctx, clusterID, namespace, labelSelector)
		if err != nil {
			return nil, err
		}
		if cacheEnabled && strings.TrimSpace(namespace) == "" && selector == nil {
			s.setDeploymentsAllToRedis(clusterID, items)
		}
		sortDeployments(items, sortBy, order)
		return deploymentsToAnyList(items), nil
	}

	items := filterDeploymentsFromInformer(entry.informer, namespace, selector)
	if cacheEnabled && strings.TrimSpace(namespace) == "" && selector == nil {
		s.setDeploymentsAllToRedis(clusterID, items)
	}
	sortDeployments(items, sortBy, order)
	return deploymentsToAnyList(items), nil
}

// ---------------------------------------------------------------------------
// Deployment — informer / filter / sort / convert
// ---------------------------------------------------------------------------

func deploymentsFromInformerStore(informer cache.SharedIndexInformer) []*appsv1.Deployment {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*appsv1.Deployment, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*appsv1.Deployment)
		if !ok || it == nil {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func filterDeploymentsFromInformer(informer cache.SharedIndexInformer, namespace string, selector labels.Selector) []*appsv1.Deployment {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*appsv1.Deployment, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*appsv1.Deployment)
		if !ok || it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		if selector != nil && !selector.Matches(labels.Set(it.Labels)) {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func deploymentsToAnyList(items []*appsv1.Deployment) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(it)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func sortDeployments(items []*appsv1.Deployment, sortBy, order string) {
	sb := strings.TrimSpace(sortBy)
	if sb == "" {
		return
	}
	desc := strings.ToLower(strings.TrimSpace(order)) == "desc"
	if sb == "metadata.name" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Name > items[j].Name
			}
			return items[i].Name < items[j].Name
		})
		return
	}
	if sb == "metadata.namespace" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Namespace > items[j].Namespace
			}
			return items[i].Namespace < items[j].Namespace
		})
	}
}

// ---------------------------------------------------------------------------
// Deployment — Redis cache
// ---------------------------------------------------------------------------

func (s *K8sService) listDeploymentsFromRedisAll(ctx context.Context, clusterID uint64, namespace string, selector labels.Selector, sortBy, order string) ([]any, bool) {
	items, ok := s.deploymentsFromRedisKey(ctx, s.objAllCacheKey(clusterID, "deployments"))
	if !ok {
		return nil, false
	}
	filtered := make([]*appsv1.Deployment, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		if selector != nil && !selector.Matches(labels.Set(it.Labels)) {
			continue
		}
		filtered = append(filtered, it.DeepCopy())
	}
	sortDeployments(filtered, sortBy, order)
	return deploymentsToAnyList(filtered), true
}

func (s *K8sService) deploymentsFromRedisKey(ctx context.Context, key string) ([]*appsv1.Deployment, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 {
		return nil, false
	}
	k := strings.TrimSpace(key)
	if k == "" {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	b, ok, err := s.cache.Get(rctx, k)
	cancel()
	if err != nil || !ok || b == nil {
		return nil, false
	}
	var arr []appsv1.Deployment
	if json.Unmarshal(b, &arr) != nil {
		return nil, false
	}
	out := make([]*appsv1.Deployment, 0, len(arr))
	for i := range arr {
		out = append(out, arr[i].DeepCopy())
	}
	return out, true
}

func (s *K8sService) setDeploymentsAllToRedis(clusterID uint64, items []*appsv1.Deployment) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 || clusterID == 0 {
		return
	}
	arr := make([]appsv1.Deployment, 0, len(items))
	for _, it := range items {
		if it != nil {
			arr = append(arr, *it)
		}
	}
	b, err := json.Marshal(arr)
	if err != nil || len(b) == 0 {
		return
	}
	ttl := s.podTTL
	wctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, "deployments"), b, ttl)
	cancel()
}

// ---------------------------------------------------------------------------
// Deployment — direct API
// ---------------------------------------------------------------------------

func (s *K8sService) listDeploymentsDirectDeployments(ctx context.Context, clusterID uint64, namespace string, labelSelector string) ([]*appsv1.Deployment, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	if ls := strings.TrimSpace(labelSelector); ls != "" {
		opts.LabelSelector = ls
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	items := make([]*appsv1.Deployment, 0, 128)
	for {
		ul, err := cs.AppsV1().Deployments(ns).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range ul.Items {
			items = append(items, ul.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(ul.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return items, nil
}

// RolloutHistory 返回工作负载的版本历史。
// 当前仅支持 Deployment，通过其关联 ReplicaSet 的 revision annotation 分析历史版本。
func (s *K8sService) RolloutHistory(ctx context.Context, clusterID uint64, namespace, name, kind string) ([]WorkloadRolloutRevision, error) {
	if !strings.EqualFold(strings.TrimSpace(kind), "Deployment") {
		return nil, ErrWithMessage(ErrInvalidParams, "当前仅支持 Deployment 的版本历史")
	}
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if namespace == "" || name == "" {
		return nil, ErrWithMessage(ErrInvalidParams, "命名空间和工作负载名称不能为空")
	}

	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	deployment, err := cs.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, normalizeK8sErr(err)
	}

	replicaSets, err := listReplicaSetsDirect(ctx, cs, namespace)
	if err != nil {
		return nil, err
	}

	return buildDeploymentRolloutHistory(deployment, replicaSets), nil
}

// RolloutUndo 将 Deployment 回滚到指定 revision。
// revision=0 时默认回滚到上一历史版本。
func (s *K8sService) RolloutUndo(ctx context.Context, clusterID uint64, namespace, name, kind string, revision int) error {
	if !strings.EqualFold(strings.TrimSpace(kind), "Deployment") {
		return ErrWithMessage(ErrInvalidParams, "当前仅支持 Deployment 回滚")
	}
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if namespace == "" || name == "" {
		return ErrWithMessage(ErrInvalidParams, "命名空间和工作负载名称不能为空")
	}
	if revision < 0 {
		return ErrWithMessage(ErrInvalidParams, "回滚版本不能小于 0")
	}

	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	deployment, err := cs.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return normalizeK8sErr(err)
	}

	replicaSets, err := listReplicaSetsDirect(ctx, cs, namespace)
	if err != nil {
		return err
	}

	sources := buildDeploymentRolloutSources(deployment, replicaSets)
	target, err := selectDeploymentRolloutUndoTarget(sources, revision)
	if err != nil {
		return err
	}

	updated := deployment.DeepCopy()
	updated.Spec.Template = sanitizeRolloutTemplate(target.replicaSet.Spec.Template)
	if updated.Annotations == nil {
		updated.Annotations = make(map[string]string)
	}
	updated.Annotations[changeCauseAnnotation] = buildRolloutUndoChangeCause(target.history)

	if _, err := cs.AppsV1().Deployments(namespace).Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
		return normalizeK8sErr(err)
	}
	return nil
}

// UpdateWorkloadImage 更新工作负载指定容器的镜像。
// 当前支持 Deployment、StatefulSet、DaemonSet，并同步写入 change-cause 注解。
func (s *K8sService) UpdateWorkloadImage(ctx context.Context, clusterID uint64, namespace, name, kind, containerName, image string) error {
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	kind = strings.TrimSpace(kind)
	containerName = strings.TrimSpace(containerName)
	image = strings.TrimSpace(image)
	if namespace == "" || name == "" || containerName == "" || image == "" {
		return ErrWithMessage(ErrInvalidParams, "命名空间、工作负载、容器名称和镜像不能为空")
	}

	gvr, ok := workloadImageGVR(kind)
	if !ok {
		return ErrWithMessage(ErrInvalidParams, "仅支持 Deployment、StatefulSet、DaemonSet 更新镜像")
	}

	obj, err := s.GetObject(ctx, clusterID, gvr, namespace, name)
	if err != nil {
		return err
	}

	patch, err := buildWorkloadImagePatch(obj, containerName, image)
	if err != nil {
		return err
	}

	return s.PatchJSON(ctx, clusterID, gvr, namespace, name, patch)
}

func (s *K8sService) UpdateWorkloadPaused(ctx context.Context, clusterID uint64, namespace, name, kind string, paused bool) error {
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if strings.TrimSpace(kind) != "Deployment" {
		return ErrWithMessage(ErrInvalidParams, "当前仅支持 Deployment 暂停或恢复 Rollout")
	}
	if namespace == "" || name == "" {
		return ErrWithMessage(ErrInvalidParams, "命名空间和工作负载名称不能为空")
	}
	gvr, _ := workloadImageGVR(kind)
	return s.PatchJSON(ctx, clusterID, gvr, namespace, name, map[string]any{
		"spec": map[string]any{
			"paused": paused,
		},
	})
}

func workloadImageGVR(kind string) (schema.GroupVersionResource, bool) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, true
	case "statefulset":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, true
	case "daemonset":
		return schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, true
	default:
		return schema.GroupVersionResource{}, false
	}
}

func buildWorkloadImagePatch(obj map[string]any, containerName, image string) (map[string]any, error) {
	spec, _ := obj["spec"].(map[string]any)
	template, _ := spec["template"].(map[string]any)
	templateSpec, _ := template["spec"].(map[string]any)
	if spec == nil || template == nil || templateSpec == nil {
		return nil, ErrWithMessage(ErrInvalidParams, "工作负载模板异常，无法更新镜像")
	}

	containers, _ := templateSpec["containers"].([]any)
	if len(containers) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "工作负载未配置容器，无法更新镜像")
	}

	updatedContainers, found, changed := updateContainerImageList(containers, containerName, image)
	if found {
		if !changed {
			return nil, ErrWithMessage(ErrInvalidParams, "新镜像与当前镜像一致")
		}
		return map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					changeCauseAnnotation: buildWorkloadImageChangeCause(containerName, image),
				},
			},
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"containers": updatedContainers,
					},
				},
			},
		}, nil
	}

	initContainers, _ := templateSpec["initContainers"].([]any)
	updatedInitContainers, found, changed := updateContainerImageList(initContainers, containerName, image)
	if found {
		if !changed {
			return nil, ErrWithMessage(ErrInvalidParams, "新镜像与当前镜像一致")
		}
		return map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					changeCauseAnnotation: buildWorkloadImageChangeCause(containerName, image),
				},
			},
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"initContainers": updatedInitContainers,
					},
				},
			},
		}, nil
	}

	return nil, ErrWithMessage(ErrNotFound, "未找到指定容器")
}

func updateContainerImageList(items []any, containerName, image string) ([]any, bool, bool) {
	if len(items) == 0 {
		return nil, false, false
	}
	updated := cloneMapSlice(items)
	for index := range updated {
		item, _ := updated[index].(map[string]any)
		if item == nil {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(item["name"])) != containerName {
			continue
		}
		currentImage := strings.TrimSpace(fmt.Sprint(item["image"]))
		if currentImage == image {
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
	for _, item := range items {
		if m, ok := item.(map[string]any); ok && m != nil {
			dup := make(map[string]any, len(m))
			for key, value := range m {
				dup[key] = value
			}
			cloned = append(cloned, dup)
			continue
		}
		cloned = append(cloned, item)
	}
	return cloned
}

func buildWorkloadImageChangeCause(containerName, image string) string {
	return fmt.Sprintf("update image %s to %s", strings.TrimSpace(containerName), strings.TrimSpace(image))
}

func listReplicaSetsDirect(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]*appsv1.ReplicaSet, error) {
	if cs == nil {
		return nil, ErrWithMessage(ErrK8s, "集群客户端未初始化")
	}
	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	items := make([]*appsv1.ReplicaSet, 0, 64)
	for {
		list, err := cs.AppsV1().ReplicaSets(namespace).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range list.Items {
			items = append(items, list.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(list.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return items, nil
}

func buildDeploymentRolloutHistory(deployment *appsv1.Deployment, replicaSets []*appsv1.ReplicaSet) []WorkloadRolloutRevision {
	sources := buildDeploymentRolloutSources(deployment, replicaSets)
	out := make([]WorkloadRolloutRevision, 0, len(sources))
	for _, source := range sources {
		out = append(out, source.history)
	}
	return out
}

func buildDeploymentRolloutSources(deployment *appsv1.Deployment, replicaSets []*appsv1.ReplicaSet) []deploymentRolloutRevisionSource {
	if deployment == nil || len(replicaSets) == 0 {
		return []deploymentRolloutRevisionSource{}
	}

	currentRevision, hasCurrentRevision := parseRevisionAnnotation(deployment.Annotations)
	out := make([]deploymentRolloutRevisionSource, 0, len(replicaSets))
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

		out = append(out, deploymentRolloutRevisionSource{
			history: WorkloadRolloutRevision{
				Revision:    revision,
				ChangeCause: strings.TrimSpace(replicaSet.Annotations[changeCauseAnnotation]),
				Images:      extractReplicaSetImages(replicaSet),
				CreatedAt:   replicaSet.CreationTimestamp.UTC(),
				IsCurrent:   isCurrent,
			},
			replicaSet: replicaSet,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].history.Revision == out[j].history.Revision {
			return out[i].history.CreatedAt.After(out[j].history.CreatedAt)
		}
		return out[i].history.Revision > out[j].history.Revision
	})
	return out
}

func selectDeploymentRolloutUndoTarget(sources []deploymentRolloutRevisionSource, revision int) (deploymentRolloutRevisionSource, error) {
	if len(sources) == 0 {
		return deploymentRolloutRevisionSource{}, ErrWithMessage(ErrNotFound, "未找到该 Deployment 的历史版本")
	}
	if revision == 0 {
		for _, source := range sources {
			if source.history.IsCurrent {
				continue
			}
			return source, nil
		}
		return deploymentRolloutRevisionSource{}, ErrWithMessage(ErrInvalidParams, "已是最早版本，无法继续回滚")
	}
	for _, source := range sources {
		if source.history.Revision != revision {
			continue
		}
		if source.history.IsCurrent {
			return deploymentRolloutRevisionSource{}, ErrWithMessage(ErrInvalidParams, "请选择当前版本之前的历史版本进行回滚")
		}
		return source, nil
	}
	return deploymentRolloutRevisionSource{}, ErrWithMessage(ErrNotFound, fmt.Sprintf("目标回滚版本 r%d 不存在", revision))
}

func sanitizeRolloutTemplate(template corev1.PodTemplateSpec) corev1.PodTemplateSpec {
	sanitized := *template.DeepCopy()
	sanitized.ResourceVersion = ""
	sanitized.UID = ""
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
	for _, ref := range replicaSet.OwnerReferences {
		if ref.Kind != "Deployment" || ref.Name != deployment.Name {
			continue
		}
		if ref.UID == "" || deployment.UID == "" || ref.UID == deployment.UID {
			return true
		}
	}
	return false
}

func parseRevisionAnnotation(annotations map[string]string) (int, bool) {
	if len(annotations) == 0 {
		return 0, false
	}
	revision, err := strconv.Atoi(strings.TrimSpace(annotations[deploymentRevisionAnnotation]))
	if err != nil || revision <= 0 {
		return 0, false
	}
	return revision, true
}

func extractReplicaSetImages(replicaSet *appsv1.ReplicaSet) []string {
	if replicaSet == nil {
		return []string{}
	}
	images := make([]string, 0, len(replicaSet.Spec.Template.Spec.InitContainers)+len(replicaSet.Spec.Template.Spec.Containers))
	seen := make(map[string]struct{}, cap(images))
	for _, container := range replicaSet.Spec.Template.Spec.InitContainers {
		image := strings.TrimSpace(container.Image)
		if image == "" {
			continue
		}
		if _, ok := seen[image]; ok {
			continue
		}
		seen[image] = struct{}{}
		images = append(images, image)
	}
	for _, container := range replicaSet.Spec.Template.Spec.Containers {
		image := strings.TrimSpace(container.Image)
		if image == "" {
			continue
		}
		if _, ok := seen[image]; ok {
			continue
		}
		seen[image] = struct{}{}
		images = append(images, image)
	}
	return images
}

// ===========================================================================
// StatefulSets
// ===========================================================================

func (s *K8sService) listStatefulSetsCached(ctx context.Context, clusterID uint64, namespace string, sortBy, order string, labelSelector string) ([]any, error) {
	entry, err := s.getOrStartObjCache(ctx, clusterID, "statefulsets")
	if err != nil {
		return nil, err
	}

	var selector labels.Selector
	if strings.TrimSpace(labelSelector) != "" {
		sel, err := labels.Parse(labelSelector)
		if err != nil {
			return nil, ErrWithMessage(ErrInvalidParams, "label_selector 无效")
		}
		selector = sel
	}

	cacheEnabled := s != nil && s.cache != nil && s.cache.Enabled() && s.podTTL > 0 && clusterID > 0
	if cacheEnabled {
		if out, ok := s.listStatefulSetsFromRedisAll(ctx, clusterID, namespace, selector, sortBy, order); ok {
			return out, nil
		}
	}

	if !entry.informer.HasSynced() {
		items, err := s.listStatefulSetsDirectStatefulSets(ctx, clusterID, namespace, labelSelector)
		if err != nil {
			return nil, err
		}
		if cacheEnabled && strings.TrimSpace(namespace) == "" && selector == nil {
			s.setStatefulSetsAllToRedis(clusterID, items)
		}
		sortStatefulSets(items, sortBy, order)
		return statefulSetsToAnyList(items), nil
	}

	items := filterStatefulSetsFromInformer(entry.informer, namespace, selector)
	if cacheEnabled && strings.TrimSpace(namespace) == "" && selector == nil {
		s.setStatefulSetsAllToRedis(clusterID, items)
	}
	sortStatefulSets(items, sortBy, order)
	return statefulSetsToAnyList(items), nil
}

// ---------------------------------------------------------------------------
// StatefulSet — informer / filter / sort / convert
// ---------------------------------------------------------------------------

func statefulSetsFromInformerStore(informer cache.SharedIndexInformer) []*appsv1.StatefulSet {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*appsv1.StatefulSet, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*appsv1.StatefulSet)
		if !ok || it == nil {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func filterStatefulSetsFromInformer(informer cache.SharedIndexInformer, namespace string, selector labels.Selector) []*appsv1.StatefulSet {
	if informer == nil {
		return nil
	}
	objs := informer.GetStore().List()
	out := make([]*appsv1.StatefulSet, 0, len(objs))
	for _, obj := range objs {
		it, ok := obj.(*appsv1.StatefulSet)
		if !ok || it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		if selector != nil && !selector.Matches(labels.Set(it.Labels)) {
			continue
		}
		out = append(out, it.DeepCopy())
	}
	return out
}

func statefulSetsToAnyList(items []*appsv1.StatefulSet) []any {
	out := make([]any, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(it)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out
}

func sortStatefulSets(items []*appsv1.StatefulSet, sortBy, order string) {
	sb := strings.TrimSpace(sortBy)
	if sb == "" {
		return
	}
	desc := strings.ToLower(strings.TrimSpace(order)) == "desc"
	if sb == "metadata.name" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Name > items[j].Name
			}
			return items[i].Name < items[j].Name
		})
		return
	}
	if sb == "metadata.namespace" {
		sort.SliceStable(items, func(i, j int) bool {
			if desc {
				return items[i].Namespace > items[j].Namespace
			}
			return items[i].Namespace < items[j].Namespace
		})
	}
}

// ---------------------------------------------------------------------------
// StatefulSet — Redis cache
// ---------------------------------------------------------------------------

func (s *K8sService) listStatefulSetsFromRedisAll(ctx context.Context, clusterID uint64, namespace string, selector labels.Selector, sortBy, order string) ([]any, bool) {
	items, ok := s.statefulSetsFromRedisKey(ctx, s.objAllCacheKey(clusterID, "statefulsets"))
	if !ok {
		return nil, false
	}
	filtered := make([]*appsv1.StatefulSet, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		if namespace != "" && it.Namespace != namespace {
			continue
		}
		if selector != nil && !selector.Matches(labels.Set(it.Labels)) {
			continue
		}
		filtered = append(filtered, it.DeepCopy())
	}
	sortStatefulSets(filtered, sortBy, order)
	return statefulSetsToAnyList(filtered), true
}

func (s *K8sService) statefulSetsFromRedisKey(ctx context.Context, key string) ([]*appsv1.StatefulSet, bool) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 {
		return nil, false
	}
	k := strings.TrimSpace(key)
	if k == "" {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	b, ok, err := s.cache.Get(rctx, k)
	cancel()
	if err != nil || !ok || b == nil {
		return nil, false
	}
	var arr []appsv1.StatefulSet
	if json.Unmarshal(b, &arr) != nil {
		return nil, false
	}
	out := make([]*appsv1.StatefulSet, 0, len(arr))
	for i := range arr {
		out = append(out, arr[i].DeepCopy())
	}
	return out, true
}

func (s *K8sService) setStatefulSetsAllToRedis(clusterID uint64, items []*appsv1.StatefulSet) {
	if s == nil || s.cache == nil || !s.cache.Enabled() || s.podTTL <= 0 || clusterID == 0 {
		return
	}
	arr := make([]appsv1.StatefulSet, 0, len(items))
	for _, it := range items {
		if it != nil {
			arr = append(arr, *it)
		}
	}
	b, err := json.Marshal(arr)
	if err != nil || len(b) == 0 {
		return
	}
	ttl := s.podTTL
	wctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_ = s.cache.Set(wctx, s.objAllCacheKey(clusterID, "statefulsets"), b, ttl)
	cancel()
}

// ---------------------------------------------------------------------------
// StatefulSet — direct API
// ---------------------------------------------------------------------------

func (s *K8sService) listStatefulSetsDirectStatefulSets(ctx context.Context, clusterID uint64, namespace string, labelSelector string) ([]*appsv1.StatefulSet, error) {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{Limit: k8sListPageLimit}
	if ls := strings.TrimSpace(labelSelector); ls != "" {
		opts.LabelSelector = ls
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	items := make([]*appsv1.StatefulSet, 0, 128)
	for {
		ul, err := cs.AppsV1().StatefulSets(ns).List(ctx, opts)
		if err != nil {
			return nil, normalizeK8sErr(err)
		}
		for i := range ul.Items {
			items = append(items, ul.Items[i].DeepCopy())
		}
		token := strings.TrimSpace(ul.Continue)
		if token == "" {
			break
		}
		opts.Continue = token
	}
	return items, nil
}
