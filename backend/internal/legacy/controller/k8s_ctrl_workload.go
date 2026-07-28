package controller

import (
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
//  Workload（Deployment / StatefulSet / DaemonSet）相关接口
// ──────────────────────────────────────────────────────────

// ListWorkloads 获取工作负载列表（Deployment/StatefulSet/DaemonSet）。
// query：kind（必填）、namespace（可选）、label_selector（可选）、sort_by、order。
// @Summary 工作负载列表
// @Description 获取指定集群工作负载列表（Deployment/StatefulSet/DaemonSet）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param kind query string true "工作负载类型" Enums(Deployment,StatefulSet,DaemonSet)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param label_selector query string false "LabelSelector"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads [get]
func (kc *K8sController) ListWorkloads(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	kind := strings.TrimSpace(c.Query("kind"))
	ns := strings.TrimSpace(c.Query("namespace"))
	ls := strings.TrimSpace(c.Query("label_selector"))
	sortBy := c.Query("sort_by")
	order := c.Query("order")
	extra := map[string]string{"label_selector": ls}

	if kind == "" {
		merged := make([]any, 0, 128)
		for _, k := range []string{"Deployment", "StatefulSet", "DaemonSet"} {
			gvr, ok2 := gvrWorkloadKind(k)
			if !ok2 {
				continue
			}
			list, err := kc.svc.List(c.Request.Context(), id, gvr, ns, sortBy, order, extra)
			if err != nil {
				kc.writeServiceErr(c, err)
				return
			}
			merged = append(merged, list...)
		}
		resp.OK(c, gin.H{"list": merged})
		return
	}

	gvr, ok2 := gvrWorkloadKind(kind)
	if !ok2 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvr, ns, sortBy, order, extra)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

type rolloutUndoReq struct {
	Kind     string `json:"kind"`
	Revision int    `json:"revision"`
}

// GetRolloutHistory 获取工作负载版本历史。
// 当前仅支持 Deployment，通过关联 ReplicaSet 的 revision annotation 分析历史。
// @Summary 获取工作负载版本历史
// @Description 获取指定工作负载的 rollout history（当前仅支持 Deployment）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "工作负载名称"
// @Success 200 {object} resp.Result{data=map[string]any} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/deployments/{ns}/{name}/rollout-history [get]
func (kc *K8sController) GetRolloutHistory(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	namespace := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if namespace == "" || name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	history, err := kc.svc.RolloutHistory(c.Request.Context(), id, namespace, name, "Deployment")
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"history": history})
}

// RolloutUndo 回滚工作负载到指定历史版本。
// @Summary 回滚工作负载
// @Description 回滚指定工作负载到历史 revision（当前仅支持 Deployment）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "工作负载名称"
// @Param body body rolloutUndoReq true "回滚参数"
// @Success 200 {object} resp.Result "回滚成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/deployments/{ns}/{name}/rollout-undo [post]
func (kc *K8sController) RolloutUndo(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	namespace := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	var req rolloutUndoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if namespace == "" || name == "" || req.Revision < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.RolloutUndo(c.Request.Context(), id, namespace, name, "Deployment", req.Revision); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

type scaleWorkloadReq struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Replicas  int    `json:"replicas"`
}

// ScaleWorkload 调整工作负载副本数。
// 仅支持可设置 replicas 的资源（Deployment/StatefulSet），DaemonSet 直接拒绝。
// @Summary 调整副本数
// @Description 调整工作负载副本数（Deployment/StatefulSet）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body scaleWorkloadReq true "缩放参数"
// @Success 200 {object} resp.Result{data=K8sOKResp} "操作成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/scale [patch]
func (kc *K8sController) ScaleWorkload(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req scaleWorkloadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	gvr, ok2 := gvrWorkloadKind(req.Kind)
	if !ok2 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.Namespace == "" || req.Name == "" || req.Replicas < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.Kind == "DaemonSet" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	patch := map[string]any{"spec": map[string]any{"replicas": req.Replicas}}
	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvr, req.Namespace, req.Name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

type restartWorkloadReq struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type updateWorkloadImageReq struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Container string `json:"container"`
	Image     string `json:"image"`
}

type pauseWorkloadReq struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Paused    bool   `json:"paused"`
}

// RestartWorkload 触发工作负载滚动重启。
// 通过 patch 写入模板注解 kubectl.kubernetes.io/restartedAt，让控制器重新创建 Pod。
// @Summary 重启工作负载
// @Description 触发工作负载滚动重启
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body restartWorkloadReq true "重启参数"
// @Success 200 {object} resp.Result{data=K8sOKResp} "操作成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/restart [patch]
func (kc *K8sController) RestartWorkload(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req restartWorkloadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	gvr, ok2 := gvrWorkloadKind(req.Kind)
	if !ok2 || req.Namespace == "" || req.Name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	now := timeRFC3339()
	patch := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/restartedAt": now,
					},
				},
			},
		},
	}
	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvr, req.Namespace, req.Name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// UpdateImage 更新工作负载中指定容器的镜像。
// 仅修改目标容器的 image 字段，并记录 change-cause 便于后续查看 rollout history。
// @Summary 更新工作负载镜像
// @Description 更新指定工作负载中某个容器的镜像
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body updateWorkloadImageReq true "镜像更新参数"
// @Success 200 {object} resp.Result{data=K8sOKResp} "操作成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/image [patch]
func (kc *K8sController) UpdateImage(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req updateWorkloadImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.Kind = strings.TrimSpace(req.Kind)
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Name = strings.TrimSpace(req.Name)
	req.Container = strings.TrimSpace(req.Container)
	req.Image = strings.TrimSpace(req.Image)
	if _, ok := gvrWorkloadKind(req.Kind); !ok || req.Namespace == "" || req.Name == "" || req.Container == "" || req.Image == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.UpdateWorkloadImage(c.Request.Context(), id, req.Namespace, req.Name, req.Kind, req.Container, req.Image); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// UpdateWorkloadPaused 更新 Deployment 的 rollout 暂停状态。
// @Summary 暂停或恢复 Deployment Rollout
// @Description 更新 Deployment 的 spec.paused 状态
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body pauseWorkloadReq true "暂停参数"
// @Success 200 {object} resp.Result{data=K8sOKResp} "操作成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/rollout-pause [patch]
func (kc *K8sController) UpdateWorkloadPaused(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req pauseWorkloadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.Kind = strings.TrimSpace(req.Kind)
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Name = strings.TrimSpace(req.Name)
	if req.Kind != "Deployment" || req.Namespace == "" || req.Name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.UpdateWorkloadPaused(c.Request.Context(), id, req.Namespace, req.Name, req.Kind, req.Paused); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ──────────────────── 编辑工作负载请求体与辅助类型 ────────────────────

type editDeploymentReq struct {
	Namespace      string                    `json:"namespace"`
	Name           string                    `json:"name"`
	Replicas       *int                      `json:"replicas"`
	Labels         map[string]string         `json:"labels"`
	Tolerations    []editToleration          `json:"tolerations"`
	Containers     []editDeploymentContainer `json:"containers"`
	InitContainers []editDeploymentContainer `json:"initContainers"`
	Strategy       *editStrategy             `json:"strategy"`
	Volumes        []map[string]any          `json:"volumes"`
}

type editStrategy struct {
	Type           string `json:"type"`
	MaxSurge       string `json:"maxSurge"`
	MaxUnavailable string `json:"maxUnavailable"`
}

type editToleration struct {
	Key               *string `json:"key"`
	Operator          *string `json:"operator"`
	Value             *string `json:"value"`
	Effect            *string `json:"effect"`
	TolerationSeconds *int64  `json:"tolerationSeconds"`
}

type editDeploymentContainer struct {
	Name            string                         `json:"name"`
	Image           *string                        `json:"image"`
	ImagePullPolicy *string                        `json:"imagePullPolicy"`
	Resources       *editDeploymentResources       `json:"resources"`
	Probes          *editDeploymentContainerProbes `json:"probes"`
	Env             []map[string]any               `json:"env"`
	EnvFrom         []map[string]any               `json:"envFrom"`
	VolumeMounts    []map[string]any               `json:"volumeMounts"`
}

type editDeploymentResources struct {
	Requests map[string]string `json:"requests"`
	Limits   map[string]string `json:"limits"`
}

type editDeploymentContainerProbes struct {
	Liveness  *editProbeTiming `json:"liveness"`
	Readiness *editProbeTiming `json:"readiness"`
	Startup   *editProbeTiming `json:"startup"`
}

type editProbeTiming struct {
	InitialDelaySeconds *int32 `json:"initialDelaySeconds"`
	TimeoutSeconds      *int32 `json:"timeoutSeconds"`
	PeriodSeconds       *int32 `json:"periodSeconds"`
	SuccessThreshold    *int32 `json:"successThreshold"`
	FailureThreshold    *int32 `json:"failureThreshold"`
}

// ──────────────────── EditDeployment ────────────────────

func (kc *K8sController) EditDeployment(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editDeploymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Name = strings.TrimSpace(req.Name)
	if req.Namespace == "" || req.Name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.Replicas != nil && *req.Replicas < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	gvr, _ := gvrWorkloadKind("Deployment")
	obj, err := kc.svc.GetObject(c.Request.Context(), id, gvr, req.Namespace, req.Name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	spec, _ := obj["spec"].(map[string]any)
	if spec == nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	selectorLabels := map[string]string{}
	if selector, _ := spec["selector"].(map[string]any); selector != nil {
		if ml, _ := selector["matchLabels"].(map[string]any); ml != nil {
			for k, v := range ml {
				kk := strings.TrimSpace(toString(k))
				if kk == "" {
					continue
				}
				selectorLabels[kk] = toString(v)
			}
		}
	}
	tpl, _ := spec["template"].(map[string]any)
	tplSpec, _ := tpl["spec"].(map[string]any)
	rawContainers, _ := tplSpec["containers"].([]any)
	if len(rawContainers) == 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	rawInitContainers, _ := tplSpec["initContainers"].([]any)
	if len(req.InitContainers) > 0 && len(rawInitContainers) == 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	applyContainerUpdates(rawContainers, req.Containers, c)
	if c.IsAborted() {
		return
	}
	applyContainerUpdates(rawInitContainers, req.InitContainers, c)
	if c.IsAborted() {
		return
	}

	patchSpec := buildWorkloadPatchSpec(req.Replicas, req.Labels, req.Tolerations, selectorLabels, rawContainers, req.Containers, rawInitContainers, req.InitContainers, req.Strategy, "strategy", req.Volumes, c)
	if c.IsAborted() {
		return
	}
	if len(patchSpec) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}

	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvr, req.Namespace, req.Name, map[string]any{"spec": patchSpec}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ──────────────────── EditStatefulSet ────────────────────

func (kc *K8sController) EditStatefulSet(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editDeploymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Name = strings.TrimSpace(req.Name)
	if req.Namespace == "" || req.Name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.Replicas != nil && *req.Replicas < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	gvr, _ := gvrWorkloadKind("StatefulSet")
	obj, err := kc.svc.GetObject(c.Request.Context(), id, gvr, req.Namespace, req.Name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	spec, _ := obj["spec"].(map[string]any)
	if spec == nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	selectorLabels := map[string]string{}
	if selector, _ := spec["selector"].(map[string]any); selector != nil {
		if ml, _ := selector["matchLabels"].(map[string]any); ml != nil {
			for k, v := range ml {
				kk := strings.TrimSpace(toString(k))
				if kk == "" {
					continue
				}
				selectorLabels[kk] = toString(v)
			}
		}
	}
	tpl, _ := spec["template"].(map[string]any)
	tplSpec, _ := tpl["spec"].(map[string]any)
	rawContainers, _ := tplSpec["containers"].([]any)
	if len(rawContainers) == 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	rawInitContainers, _ := tplSpec["initContainers"].([]any)
	if len(req.InitContainers) > 0 && len(rawInitContainers) == 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	applyContainerUpdates(rawContainers, req.Containers, c)
	if c.IsAborted() {
		return
	}
	applyContainerUpdates(rawInitContainers, req.InitContainers, c)
	if c.IsAborted() {
		return
	}

	patchSpec := buildWorkloadPatchSpec(req.Replicas, req.Labels, req.Tolerations, selectorLabels, rawContainers, req.Containers, rawInitContainers, req.InitContainers, req.Strategy, "updateStrategy", req.Volumes, c)
	if c.IsAborted() {
		return
	}
	if len(patchSpec) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}

	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvr, req.Namespace, req.Name, map[string]any{"spec": patchSpec}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ──────────────────── EditDaemonSet ────────────────────

func (kc *K8sController) EditDaemonSet(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editDeploymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Name = strings.TrimSpace(req.Name)
	if req.Namespace == "" || req.Name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.Replicas != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	gvr, _ := gvrWorkloadKind("DaemonSet")
	obj, err := kc.svc.GetObject(c.Request.Context(), id, gvr, req.Namespace, req.Name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}

	spec, _ := obj["spec"].(map[string]any)
	if spec == nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	selectorLabels := map[string]string{}
	if selector, _ := spec["selector"].(map[string]any); selector != nil {
		if ml, _ := selector["matchLabels"].(map[string]any); ml != nil {
			for k, v := range ml {
				kk := strings.TrimSpace(toString(k))
				if kk == "" {
					continue
				}
				selectorLabels[kk] = toString(v)
			}
		}
	}
	tpl, _ := spec["template"].(map[string]any)
	tplSpec, _ := tpl["spec"].(map[string]any)
	rawContainers, _ := tplSpec["containers"].([]any)
	if len(rawContainers) == 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	rawInitContainers, _ := tplSpec["initContainers"].([]any)
	if len(req.InitContainers) > 0 && len(rawInitContainers) == 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	applyContainerUpdates(rawContainers, req.Containers, c)
	if c.IsAborted() {
		return
	}
	applyContainerUpdates(rawInitContainers, req.InitContainers, c)
	if c.IsAborted() {
		return
	}

	patchSpec := buildWorkloadPatchSpec(nil, req.Labels, req.Tolerations, selectorLabels, rawContainers, req.Containers, rawInitContainers, req.InitContainers, req.Strategy, "updateStrategy", req.Volumes, c)
	if c.IsAborted() {
		return
	}
	if len(patchSpec) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}

	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvr, req.Namespace, req.Name, map[string]any{"spec": patchSpec}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ──────────────────── 工作负载编辑公共辅助 ────────────────────

// applyContainerUpdates 将 upd 列表中的字段更新到 rawContainers 对应元素上。
// 如果遇到校验错误，会通过 c.Abort() 终止请求。
func applyContainerUpdates(rawContainers []any, updates []editDeploymentContainer, c *gin.Context) {
	type containerRef struct {
		idx int
		obj map[string]any
	}
	byName := map[string]containerRef{}
	for i := range rawContainers {
		m, _ := rawContainers[i].(map[string]any)
		if m == nil {
			continue
		}
		name := strings.TrimSpace(toString(m["name"]))
		if name == "" {
			continue
		}
		byName[name] = containerRef{idx: i, obj: m}
	}

	for _, upd := range updates {
		cn := strings.TrimSpace(upd.Name)
		if cn == "" {
			resp.Fail(c, 4000, "invalid params")
			c.Abort()
			return
		}
		ref, ok := byName[cn]
		if !ok || ref.obj == nil {
			resp.Fail(c, 4000, "invalid params")
			c.Abort()
			return
		}
		if upd.Image != nil {
			ref.obj["image"] = strings.TrimSpace(*upd.Image)
		}
		if upd.ImagePullPolicy != nil {
			v := strings.TrimSpace(*upd.ImagePullPolicy)
			if v == "" {
				delete(ref.obj, "imagePullPolicy")
			} else {
				ref.obj["imagePullPolicy"] = v
			}
		}
		if upd.Resources != nil {
			ensureResourceMaps(ref.obj, upd.Resources)
		}
		if upd.Probes != nil {
			applyProbeTiming(ref.obj, "livenessProbe", upd.Probes.Liveness)
			applyProbeTiming(ref.obj, "readinessProbe", upd.Probes.Readiness)
			applyProbeTiming(ref.obj, "startupProbe", upd.Probes.Startup)
		}
		if upd.Env != nil {
			envList := make([]any, len(upd.Env))
			for i, e := range upd.Env {
				envList[i] = e
			}
			ref.obj["env"] = envList
		}
		if upd.EnvFrom != nil {
			envFromList := make([]any, len(upd.EnvFrom))
			for i, e := range upd.EnvFrom {
				envFromList[i] = e
			}
			ref.obj["envFrom"] = envFromList
		}
		if upd.VolumeMounts != nil {
			vmList := make([]any, len(upd.VolumeMounts))
			for i, e := range upd.VolumeMounts {
				vmList[i] = e
			}
			ref.obj["volumeMounts"] = vmList
		}
		rawContainers[ref.idx] = ref.obj
	}
}

// buildWorkloadPatchSpec 构造通用工作负载 spec 补丁。
func buildWorkloadPatchSpec(
	replicas *int,
	labels map[string]string,
	tolerations []editToleration,
	selectorLabels map[string]string,
	rawContainers []any,
	containers []editDeploymentContainer,
	rawInitContainers []any,
	initContainers []editDeploymentContainer,
	strategy *editStrategy,
	strategyKey string,
	volumes []map[string]any,
	c *gin.Context,
) map[string]any {
	patchSpec := map[string]any{}
	if replicas != nil {
		patchSpec["replicas"] = *replicas
	}
	if labels != nil {
		nextLabels := map[string]any{}
		for k, v := range labels {
			kk := strings.TrimSpace(k)
			if kk == "" {
				continue
			}
			nextLabels[kk] = strings.TrimSpace(v)
		}
		for k, v := range selectorLabels {
			if toString(nextLabels[k]) != v {
				resp.Fail(c, 4000, "invalid params")
				c.Abort()
				return nil
			}
		}
		patchSpec["template"] = mergeMap(patchSpec["template"], map[string]any{"metadata": map[string]any{"labels": nextLabels}})
	}
	if tolerations != nil {
		out := []any{}
		for _, t := range tolerations {
			m := map[string]any{}
			if t.Key != nil {
				m["key"] = strings.TrimSpace(*t.Key)
			}
			if t.Operator != nil {
				m["operator"] = strings.TrimSpace(*t.Operator)
			}
			if t.Value != nil {
				m["value"] = strings.TrimSpace(*t.Value)
			}
			if t.Effect != nil {
				m["effect"] = strings.TrimSpace(*t.Effect)
			}
			if t.TolerationSeconds != nil {
				if *t.TolerationSeconds < 0 {
					resp.Fail(c, 4000, "invalid params")
					c.Abort()
					return nil
				}
				m["tolerationSeconds"] = *t.TolerationSeconds
			}
			out = append(out, m)
		}
		patchSpec["template"] = mergeMap(patchSpec["template"], map[string]any{"spec": map[string]any{"tolerations": out}})
	}
	if len(containers) > 0 {
		patchSpec["template"] = mergeMap(patchSpec["template"], map[string]any{"spec": map[string]any{"containers": rawContainers}})
	}
	if len(initContainers) > 0 {
		patchSpec["template"] = mergeMap(patchSpec["template"], map[string]any{"spec": map[string]any{"initContainers": rawInitContainers}})
	}
	if strategy != nil {
		st := map[string]any{"type": strategy.Type}
		if strategy.Type == "RollingUpdate" {
			ru := map[string]any{}
			if strategy.MaxSurge != "" {
				ru["maxSurge"] = strategy.MaxSurge
			}
			if strategy.MaxUnavailable != "" {
				ru["maxUnavailable"] = strategy.MaxUnavailable
			}
			if len(ru) > 0 {
				st["rollingUpdate"] = ru
			}
		}
		if strategyKey == "" {
			strategyKey = "strategy"
		}
		patchSpec[strategyKey] = st
	}
	if volumes != nil {
		volList := make([]any, len(volumes))
		for i, v := range volumes {
			volList[i] = v
		}
		patchSpec["template"] = mergeMap(patchSpec["template"], map[string]any{"spec": map[string]any{"volumes": volList}})
	}
	return patchSpec
}

// ──────────────────── EditWorkloadYAML ────────────────────

type editWorkloadYAMLReq struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	YAML      string `json:"yaml"`
}

// EditWorkloadYAML 通过 YAML 更新工作负载。
// @Summary 通过 YAML 编辑工作负载
// @Description 通过 YAML 更新 Deployment/StatefulSet/DaemonSet
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body editWorkloadYAMLReq true "YAML 内容"
// @Success 200 {object} resp.Result{data=K8sOKResp} "操作成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/yaml/edit [patch]
func (kc *K8sController) EditWorkloadYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editWorkloadYAMLReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.Kind = strings.TrimSpace(req.Kind)
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.YAML = strings.TrimSpace(req.YAML)
	if req.YAML == "" {
		resp.Fail(c, 4000, "yaml is required")
		return
	}
	gvr, ok2 := gvrWorkloadKind(req.Kind)
	if !ok2 {
		resp.Fail(c, 4000, "invalid kind")
		return
	}
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvr, req.Namespace, req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ──────────────────── DeleteWorkload / GetWorkloadYAML ────────────────────

// DeleteWorkload 删除指定工作负载。
// @Summary 删除工作负载
// @Description 删除指定工作负载
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param kind path string true "工作负载类型" Enums(Deployment,StatefulSet,DaemonSet)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/{kind}/{ns}/{name} [delete]
func (kc *K8sController) DeleteWorkload(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	kind := decodePathParam(c.Param("kind"))
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	gvr, ok2 := gvrWorkloadKind(kind)
	if !ok2 || ns == "" || name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.Delete(c.Request.Context(), id, gvr, ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// GetWorkloadYAML 获取工作负载 YAML。
// @Summary 获取工作负载 YAML
// @Description 获取指定工作负载的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param kind path string true "工作负载类型" Enums(Deployment,StatefulSet,DaemonSet)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/workloads/{kind}/{ns}/{name}/yaml [get]
func (kc *K8sController) GetWorkloadYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	kind := decodePathParam(c.Param("kind"))
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	gvr, ok2 := gvrWorkloadKind(kind)
	if !ok2 || ns == "" || name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvr, ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}
