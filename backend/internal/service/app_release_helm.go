package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"k8s-platform-backend/internal/model"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	"gorm.io/gorm"
)

type releaseManifestResourceRef struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

type releaseManifestDocument struct {
	Object     *unstructured.Unstructured
	Ref        releaseManifestResourceRef
	GVR        schema.GroupVersionResource
	Namespaced bool
}

func (s *AppReleaseService) ensureHelmTarget(ctx context.Context, clusterID uint64, clusterName *string) error {
	if s.clusterReg == nil {
		return errors.New("cluster registry is required")
	}
	cluster, err := s.clusterReg.GetCluster(ctx, clusterID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrWithMessage(ErrNotFound, "目标集群不存在")
		}
		return err
	}
	status := strings.TrimSpace(cluster.Status)
	if status != "active" && status != "degraded" {
		return ErrWithMessage(ErrConflict, "目标集群当前不可用，请先检查集群状态")
	}
	if clusterName != nil && strings.TrimSpace(*clusterName) == "" {
		*clusterName = cluster.Name
	}
	return nil
}

func (s *AppReleaseService) buildHelmSnapshotRequest(ctx context.Context, release model.AppRelease, revision int, values map[string]interface{}, operator string, createdBy uint64) (composeRevisionSnapshotRequest, error) {
	detail, err := s.tpl.Get(ctx, release.TemplateID)
	if err != nil {
		return composeRevisionSnapshotRequest{}, err
	}
	mergedValues := mergeReleaseValues(detail.DefaultValues, values)
	renderedManifest, err := renderAppTemplateManifest(detail, mergedValues, templateRenderOptions{
		ReleaseName: release.Name,
		Namespace:   release.Namespace,
		Revision:    revision,
	})
	if err != nil {
		return composeRevisionSnapshotRequest{}, err
	}
	if strings.TrimSpace(renderedManifest) == "" {
		return composeRevisionSnapshotRequest{}, ErrWithMessage(ErrInvalidParams, "Helm 模板渲染结果为空，无法提交部署")
	}
	return composeRevisionSnapshotRequest{
		Revision:        revision,
		TemplateID:      detail.ID,
		TemplateName:    detail.Name,
		TemplateVersion: detail.Version,
		ComposeManifest: renderedManifest,
		Values:          mergedValues,
		Operator:        operator,
		CreatedBy:       createdBy,
	}, nil
}

func (s *AppReleaseService) startHelmTask(release model.AppRelease, snapshot model.AppReleaseRevision, previousSnapshot *model.AppReleaseRevision, action string, createdBy uint64) (uint64, error) {
	if s.taskStore == nil || s.clusterReg == nil {
		return 0, ErrWithMessage(ErrConflict, "Helm 发布运行时未就绪，请检查任务中心与集群配置")
	}
	title := helmTaskTitle(action, release.Name)
	msg := "排队中"
	percent := 0
	task := &Task{
		Type:      "app_release_helm",
		Status:    TaskPending,
		Title:     &title,
		CreatedBy: int64(createdBy),
		Percent:   &percent,
		Message:   &msg,
		Meta: map[string]any{
			"source":           "app_release",
			"action":           action,
			"app_release_id":   release.ID,
			"name":             release.Name,
			"template_id":      snapshot.TemplateID,
			"template_name":    snapshot.TemplateName,
			"template_engine":  release.TemplateEngine,
			"cluster_id":       release.ClusterID,
			"cluster_name":     release.ClusterName,
			"namespace":        release.Namespace,
			"desired_revision": snapshot.Revision,
			"previous_status":  release.Status,
		},
		Steps: []TaskStep{
			{Key: "connect_cluster", Title: "连接集群", Status: StepPending},
			{Key: "apply_manifest", Title: "应用清单", Status: StepPending},
			{Key: "prune_stale", Title: "清理旧资源", Status: StepPending},
		},
	}
	if err := s.taskStore.Put(task); err != nil {
		return 0, err
	}
	execCtx, cancel := context.WithCancel(context.Background())
	s.taskStore.RegisterCancel(task.ID, cancel)
	go s.runHelmTask(execCtx, task, release, snapshot, previousSnapshot)
	return uint64(task.ID), nil
}

func (s *AppReleaseService) runHelmTask(ctx context.Context, task *Task, release model.AppRelease, snapshot model.AppReleaseRevision, previousSnapshot *model.AppReleaseRevision) {
	defer s.taskStore.UnregisterCancel(task.ID)

	task.Status = TaskRunning
	startMsg := "开始执行 Helm 发布任务"
	task.Message = &startMsg
	_ = task.Update()

	clients, err := s.newHelmTaskClients(ctx, release.ClusterID)
	if err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, err)
		return
	}
	if err := s.finishHelmStep(task, 0, "已成功连接目标集群"); err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, err)
		return
	}

	if err := s.startHelmStep(task, 1, "开始应用 Helm 渲染清单"); err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, err)
		return
	}
	if err := ensureNamespaceExists(ctx, clients.typed, release.Namespace); err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, ErrWithMessage(ErrConflict, fmt.Sprintf("准备命名空间失败：%v", err)))
		return
	}
	targetDocs, err := s.parseReleaseManifestDocuments(snapshot.ComposeManifest, release.Namespace, clients.mapper)
	if err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, err)
		return
	}
	targetRefs := make(map[string]releaseManifestResourceRef, len(targetDocs))
	for _, doc := range targetDocs {
		select {
		case <-ctx.Done():
			s.cancelHelmTask(task, release, snapshot, previousSnapshot)
			return
		default:
		}
		task.AppendLog(fmt.Sprintf("[Helm] apply %s %s/%s", doc.Ref.Kind, firstNonEmpty(doc.Ref.Namespace, "_cluster"), doc.Ref.Name))
		if err := applyManifestDocument(ctx, clients.dynamic, doc); err != nil {
			s.failHelmTask(task, release, snapshot, previousSnapshot, ErrWithMessage(ErrConflict, fmt.Sprintf("应用资源 %s/%s 失败：%v", doc.Ref.Kind, doc.Ref.Name, err)))
			return
		}
		targetRefs[doc.Ref.key()] = doc.Ref
	}
	if err := s.finishHelmStep(task, 1, fmt.Sprintf("已应用 %d 个资源", len(targetDocs))); err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, err)
		return
	}

	if err := s.startHelmStep(task, 2, "开始清理旧资源"); err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, err)
		return
	}
	prunedCount, err := pruneStaleResources(ctx, clients.dynamic, clients.mapper, previousSnapshot, targetRefs, release.Namespace, task)
	if err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, err)
		return
	}
	if err := s.finishHelmStep(task, 2, fmt.Sprintf("已清理 %d 个旧资源", prunedCount)); err != nil {
		s.failHelmTask(task, release, snapshot, previousSnapshot, err)
		return
	}

	task.Status = TaskSuccess
	successMsg := "Helm 发布执行成功"
	task.Message = &successMsg
	percent := 100
	task.Percent = &percent
	_ = task.Update()

	updates := map[string]any{
		"status":           "healthy",
		"current_revision": snapshot.Revision,
		"desired_revision": snapshot.Revision,
		"values":           stringPointerValue(snapshot.Values),
		"template_id":      snapshot.TemplateID,
		"template_name":    snapshot.TemplateName,
		"template_version": snapshot.TemplateVersion,
		"last_event":       fmt.Sprintf("Helm 发布任务 #%d 执行成功", task.ID),
		"health_score":     95,
		"replicas":         summarizeManifestResources(targetDocs),
	}
	_ = s.db.WithContext(context.Background()).Model(&model.AppRelease{}).Where("id = ? AND deleted_at IS NULL", release.ID).Updates(updates).Error
}

func (s *AppReleaseService) failHelmTask(task *Task, release model.AppRelease, snapshot model.AppReleaseRevision, previousSnapshot *model.AppReleaseRevision, err error) {
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		msg = "Helm 发布执行失败"
	}
	task.Status = TaskFailed
	task.Message = &msg
	_ = task.Update()
	task.AppendLog("[Error] " + msg)
	updates := map[string]any{
		"status":       "failed",
		"last_event":   msg,
		"health_score": 30,
	}
	if previousSnapshot != nil && previousSnapshot.Revision > 0 {
		updates["desired_revision"] = previousSnapshot.Revision
	}
	_ = s.db.WithContext(context.Background()).Model(&model.AppRelease{}).Where("id = ? AND deleted_at IS NULL", release.ID).Updates(updates).Error
}

func (s *AppReleaseService) cancelHelmTask(task *Task, release model.AppRelease, snapshot model.AppReleaseRevision, previousSnapshot *model.AppReleaseRevision) {
	msg := "Helm 发布任务已取消"
	task.Status = TaskCanceled
	task.Message = &msg
	_ = task.Update()
	updates := map[string]any{
		"status":       release.Status,
		"last_event":   msg,
		"health_score": 50,
	}
	if previousSnapshot != nil && previousSnapshot.Revision > 0 {
		updates["desired_revision"] = previousSnapshot.Revision
	}
	_ = s.db.WithContext(context.Background()).Model(&model.AppRelease{}).Where("id = ? AND deleted_at IS NULL", release.ID).Updates(updates).Error
}

func (s *AppReleaseService) startHelmStep(task *Task, stepIdx int, message string) error {
	if stepIdx >= len(task.Steps) {
		return nil
	}
	now := time.Now().UTC()
	task.Steps[stepIdx].Status = StepRunning
	task.Steps[stepIdx].StartedAt = &now
	if strings.TrimSpace(message) != "" {
		task.AppendLog("[Helm] " + strings.TrimSpace(message))
	}
	return task.Update()
}

func (s *AppReleaseService) finishHelmStep(task *Task, stepIdx int, message string) error {
	if stepIdx >= len(task.Steps) {
		return nil
	}
	now := time.Now().UTC()
	task.Steps[stepIdx].Status = StepSuccess
	task.Steps[stepIdx].FinishedAt = &now
	progress := int(float64(stepIdx+1) / float64(len(task.Steps)) * 100)
	task.Percent = &progress
	if strings.TrimSpace(message) != "" {
		task.AppendLog("[Helm] " + strings.TrimSpace(message))
		task.Message = &message
	}
	return task.Update()
}

type helmTaskClients struct {
	dynamic dynamic.Interface
	typed   kubernetes.Interface
	mapper  apimeta.RESTMapper
}

func (s *AppReleaseService) newHelmTaskClients(ctx context.Context, clusterID uint64) (*helmTaskClients, error) {
	if s.clusterReg == nil {
		return nil, errors.New("cluster registry is required")
	}
	kubeconfig, err := s.clusterReg.GetKubeconfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, ErrWithMessage(ErrInvalidParams, "目标集群 kubeconfig 无效")
	}
	cfg.Timeout = 60 * time.Second
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, ErrWithMessage(ErrK8s, fmt.Sprintf("初始化动态客户端失败：%v", err))
	}
	typed, err := kubernetes.NewForConfig(rest.CopyConfig(cfg))
	if err != nil {
		return nil, ErrWithMessage(ErrK8s, fmt.Sprintf("初始化集群客户端失败：%v", err))
	}
	disc, err := discovery.NewDiscoveryClientForConfig(rest.CopyConfig(cfg))
	if err != nil {
		return nil, ErrWithMessage(ErrK8s, fmt.Sprintf("初始化集群发现客户端失败：%v", err))
	}
	resources, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		return nil, ErrWithMessage(ErrK8s, fmt.Sprintf("读取集群资源映射失败：%v", err))
	}
	mapper := restmapper.NewDiscoveryRESTMapper(resources)
	return &helmTaskClients{dynamic: dyn, typed: typed, mapper: mapper}, nil
}

func ensureNamespaceExists(ctx context.Context, client kubernetes.Interface, namespace string) error {
	ns := strings.TrimSpace(namespace)
	if ns == "" || ns == metav1.NamespaceAll || client == nil {
		return nil
	}
	_, err := client.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	namespaceObj := unstructuredToNamespace(ns)
	_, err = client.CoreV1().Namespaces().Create(ctx, &namespaceObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return err
}

func unstructuredToNamespace(namespace string) corev1.Namespace {
	return corev1.Namespace{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
}

func (s *AppReleaseService) parseReleaseManifestDocuments(manifest, defaultNamespace string, mapper apimeta.RESTMapper) ([]releaseManifestDocument, error) {
	decoder := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	documents := make([]releaseManifestDocument, 0)
	for {
		var raw map[string]interface{}
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("Helm 清单解析失败：%v", err))
		}
		if len(raw) == 0 {
			continue
		}
		objects, err := expandManifestObjects(raw)
		if err != nil {
			return nil, err
		}
		for _, object := range objects {
			gvk := object.GroupVersionKind()
			if gvk.Empty() {
				return nil, ErrWithMessage(ErrInvalidParams, "渲染结果缺少 apiVersion 或 kind")
			}
			mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return nil, ErrWithMessage(ErrK8s, fmt.Sprintf("无法解析资源 %s：%v", gvk.String(), err))
			}
			namespace := strings.TrimSpace(object.GetNamespace())
			if mapping.Scope.Name() == apimeta.RESTScopeNameNamespace {
				if namespace == "" {
					namespace = strings.TrimSpace(defaultNamespace)
					object.SetNamespace(namespace)
				}
			} else {
				namespace = ""
				object.SetNamespace("")
			}
			name := strings.TrimSpace(object.GetName())
			if name == "" {
				return nil, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("资源 %s 缺少 metadata.name", gvk.String()))
			}
			documents = append(documents, releaseManifestDocument{
				Object: object,
				Ref: releaseManifestResourceRef{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Namespace:  namespace,
					Name:       name,
				},
				GVR:        mapping.Resource,
				Namespaced: mapping.Scope.Name() == apimeta.RESTScopeNameNamespace,
			})
		}
	}
	if len(documents) == 0 {
		return nil, ErrWithMessage(ErrInvalidParams, "Helm 渲染结果没有可应用的 Kubernetes 资源")
	}
	return documents, nil
}

func expandManifestObjects(raw map[string]interface{}) ([]*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{Object: raw}
	if strings.EqualFold(obj.GetKind(), "List") {
		items, ok := raw["items"].([]interface{})
		if !ok || len(items) == 0 {
			return nil, nil
		}
		out := make([]*unstructured.Unstructured, 0, len(items))
		for _, item := range items {
			child, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			out = append(out, &unstructured.Unstructured{Object: child})
		}
		return out, nil
	}
	return []*unstructured.Unstructured{obj}, nil
}

func applyManifestDocument(ctx context.Context, dynamicClient dynamic.Interface, doc releaseManifestDocument) error {
	var resource dynamic.ResourceInterface
	if doc.Namespaced {
		resource = dynamicClient.Resource(doc.GVR).Namespace(doc.Ref.Namespace)
	} else {
		resource = dynamicClient.Resource(doc.GVR)
	}
	existing, err := resource.Get(ctx, doc.Ref.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = resource.Create(ctx, doc.Object, metav1.CreateOptions{})
			return err
		}
		return err
	}
	doc.Object.SetResourceVersion(existing.GetResourceVersion())
	_, err = resource.Update(ctx, doc.Object, metav1.UpdateOptions{})
	return err
}

func pruneStaleResources(ctx context.Context, dynamicClient dynamic.Interface, mapper apimeta.RESTMapper, previousSnapshot *model.AppReleaseRevision, targetRefs map[string]releaseManifestResourceRef, defaultNamespace string, task *Task) (int, error) {
	if previousSnapshot == nil || strings.TrimSpace(previousSnapshot.ComposeManifest) == "" {
		return 0, nil
	}
	decoder := utilyaml.NewYAMLOrJSONDecoder(strings.NewReader(previousSnapshot.ComposeManifest), 4096)
	pruned := 0
	for {
		var raw map[string]interface{}
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return pruned, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("旧版本清单解析失败：%v", err))
		}
		if len(raw) == 0 {
			continue
		}
		objects, err := expandManifestObjects(raw)
		if err != nil {
			return pruned, err
		}
		for _, object := range objects {
			gvk := object.GroupVersionKind()
			mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return pruned, ErrWithMessage(ErrK8s, fmt.Sprintf("无法解析旧资源 %s：%v", gvk.String(), err))
			}
			namespace := strings.TrimSpace(object.GetNamespace())
			if mapping.Scope.Name() == apimeta.RESTScopeNameNamespace {
				if namespace == "" {
					namespace = strings.TrimSpace(defaultNamespace)
				}
			} else {
				namespace = ""
			}
			ref := releaseManifestResourceRef{
				APIVersion: gvk.GroupVersion().String(),
				Kind:       gvk.Kind,
				Namespace:  namespace,
				Name:       strings.TrimSpace(object.GetName()),
			}
			if ref.Name == "" {
				continue
			}
			if _, ok := targetRefs[ref.key()]; ok {
				continue
			}
			var resource dynamic.ResourceInterface
			if namespace != "" {
				resource = dynamicClient.Resource(mapping.Resource).Namespace(namespace)
			} else {
				resource = dynamicClient.Resource(mapping.Resource)
			}
			if task != nil {
				task.AppendLog(fmt.Sprintf("[Helm] delete %s %s/%s", ref.Kind, firstNonEmpty(ref.Namespace, "_cluster"), ref.Name))
			}
			if err := resource.Delete(ctx, ref.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return pruned, ErrWithMessage(ErrK8s, fmt.Sprintf("清理旧资源 %s/%s 失败：%v", ref.Kind, ref.Name, err))
			}
			pruned++
		}
	}
	return pruned, nil
}

func summarizeManifestResources(documents []releaseManifestDocument) string {
	if len(documents) == 0 {
		return "0 resources"
	}
	workloads := 0
	for _, doc := range documents {
		switch doc.Ref.Kind {
		case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
			workloads++
		}
	}
	if workloads > 0 {
		return fmt.Sprintf("%d workloads", workloads)
	}
	return fmt.Sprintf("%d resources", len(documents))
}

func helmTaskTitle(action, releaseName string) string {
	name := composeFirstNonEmpty(releaseName, "未命名发布")
	switch action {
	case composeTaskActionRollback:
		return "Helm 应用回滚：" + name
	default:
		return "Helm 应用发布：" + name
	}
}

func applyReleaseTaskStates(ctx context.Context, db *gorm.DB, rows []model.AppRelease) []model.AppRelease {
	if len(rows) == 0 || db == nil {
		return rows
	}
	taskIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if row.LastTaskID != nil && *row.LastTaskID > 0 {
			taskIDs = append(taskIDs, *row.LastTaskID)
		}
	}
	if len(taskIDs) == 0 {
		return rows
	}
	var tasks []model.Task
	if err := db.WithContext(ctx).Where("id IN ?", taskIDs).Find(&tasks).Error; err != nil {
		return rows
	}
	taskMap := make(map[uint64]model.Task, len(tasks))
	for _, task := range tasks {
		taskMap[task.ID] = task
	}
	for idx := range rows {
		if rows[idx].LastTaskID == nil {
			continue
		}
		if task, ok := taskMap[*rows[idx].LastTaskID]; ok {
			rows[idx] = applyReleaseTaskState(rows[idx], &task)
		}
	}
	return rows
}

func applyReleaseTaskState(row model.AppRelease, task *model.Task) model.AppRelease {
	if row.TemplateEngine == "yaml" {
		return applyComposeTaskState(row, task)
	}
	if task == nil {
		return row
	}
	action := composeTaskAction(task)
	previousStatus := composeTaskPreviousStatus(task, row.Status)
	switch task.Status {
	case model.TaskPending, model.TaskRunning:
		row.Status = "progressing"
		if action == composeTaskActionRollback {
			row.LastEvent = fmt.Sprintf("Helm 回滚任务 #%d 执行中", task.ID)
		} else {
			row.LastEvent = fmt.Sprintf("Helm 发布任务 #%d 执行中", task.ID)
		}
	case model.TaskSuccess:
		row.Status = "healthy"
		row.CurrentRevision = row.DesiredRevision
		if action == composeTaskActionRollback {
			row.LastEvent = fmt.Sprintf("Helm 回滚任务 #%d 执行成功", task.ID)
		} else {
			row.LastEvent = fmt.Sprintf("Helm 发布任务 #%d 执行成功", task.ID)
		}
		if row.HealthScore < 90 {
			row.HealthScore = 95
		}
	case model.TaskCanceled:
		row.Status = previousStatus
		row.LastEvent = fmt.Sprintf("Helm 任务 #%d 已取消", task.ID)
	case model.TaskFailed, model.TaskTimeout:
		row.Status = "failed"
		if msg := strings.TrimSpace(task.Message); msg != "" {
			row.LastEvent = msg
		} else if action == composeTaskActionRollback {
			row.LastEvent = fmt.Sprintf("Helm 回滚任务 #%d 执行失败", task.ID)
		} else {
			row.LastEvent = fmt.Sprintf("Helm 发布任务 #%d 执行失败", task.ID)
		}
		row.HealthScore = 30
	}
	return row
}

func (ref releaseManifestResourceRef) key() string {
	return strings.ToLower(strings.Join([]string{ref.APIVersion, ref.Kind, ref.Namespace, ref.Name}, "|"))
}
