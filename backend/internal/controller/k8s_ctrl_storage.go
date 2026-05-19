package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type createPVCReq struct {
	Namespace    string   `json:"namespace"`
	Name         string   `json:"name"`
	StorageClass string   `json:"storage_class"`
	AccessModes  []string `json:"access_modes"`
	Capacity     string   `json:"capacity"`
}

type K8sResourceSupportResp struct {
	ReplicaSets                     *bool `json:"replicasets,omitempty"`
	ClusterRoles                    *bool `json:"clusterroles,omitempty"`
	Endpoints                       *bool `json:"endpoints,omitempty"`
	EndpointSlices                  *bool `json:"endpointslices,omitempty"`
	NetworkPolicies                 *bool `json:"networkpolicies,omitempty"`
	VolumeAttachments               *bool `json:"volumeattachments,omitempty"`
	ResourceQuotas                  *bool `json:"resourcequotas,omitempty"`
	LimitRanges                     *bool `json:"limitranges,omitempty"`
	CustomResourceDefinitions       *bool `json:"customresourcedefinitions,omitempty"`
	APIServices                     *bool `json:"apiservices,omitempty"`
	PriorityClasses                 *bool `json:"priorityclasses,omitempty"`
	ValidatingWebhookConfigurations *bool `json:"validatingwebhookconfigurations,omitempty"`
	MutatingWebhookConfigurations   *bool `json:"mutatingwebhookconfigurations,omitempty"`
	Leases                          *bool `json:"leases,omitempty"`
	VolumeSnapshots                 *bool `json:"volumesnapshots,omitempty"`
	VolumeSnapshotClasses           *bool `json:"volumesnapshotclasses,omitempty"`
	VolumeSnapshotContents          *bool `json:"volumesnapshotcontents,omitempty"`
}

func boolPtr(v bool) *bool {
	return &v
}

// ──────────────────────────────────────────────────────────
//  PVC 相关接口
// ──────────────────────────────────────────────────────────

// ListPVCs 获取 PVC 列表。
// @Summary PVC 列表
// @Description 获取指定集群 PVC 列表（可按 namespace 过滤）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pvcs [get]
func (kc *K8sController) ListPVCs(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrPVCs(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// CreatePVC 创建 PVC。
// @Summary 创建 PVC
// @Description 在指定集群中创建 PVC
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body createPVCReq true "创建 PVC 参数"
// @Success 200 {object} resp.Result "创建成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pvcs [post]
func (kc *K8sController) CreatePVC(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req createPVCReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.CreatePVC(c.Request.Context(), id, service.CreatePVCInput{
		Namespace:    req.Namespace,
		Name:         req.Name,
		StorageClass: req.StorageClass,
		AccessModes:  req.AccessModes,
		Capacity:     req.Capacity,
	}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// GetPVCYAML 获取 PVC YAML。
// @Summary 获取 PVC YAML
// @Description 获取指定 PVC 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pvcs/{ns}/{name}/yaml [get]
func (kc *K8sController) GetPVCYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrPVCs(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// DeletePVC 删除 PVC。
// @Summary 删除 PVC
// @Description 删除指定 PVC
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pvcs/{ns}/{name} [delete]
func (kc *K8sController) DeletePVC(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrPVCs(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// ──────────────────────────────────────────────────────────
//  PV 相关接口
// ──────────────────────────────────────────────────────────

// ListPVs 获取 PV 列表。
// @Summary PV 列表
// @Description 获取指定集群 PV 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pvs [get]
func (kc *K8sController) ListPVs(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrPVs(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetPVYAML 获取 PV YAML。
// @Summary 获取 PV YAML
// @Description 获取指定 PV 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pvs/{name}/yaml [get]
func (kc *K8sController) GetPVYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrPVs(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// DeletePV 删除 PV。
// @Summary 删除 PV
// @Description 删除指定 PV
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pvs/{name} [delete]
func (kc *K8sController) DeletePV(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrPVs(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// ──────────────────────────────────────────────────────────
//  StorageClass 相关接口
// ──────────────────────────────────────────────────────────

// ListStorageClasses 获取 StorageClass 列表。
// @Summary StorageClass 列表
// @Description 获取指定集群 StorageClass 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/storageclasses [get]
func (kc *K8sController) ListStorageClasses(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrStorageClasses(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetStorageClassYAML 获取 StorageClass YAML。
// @Summary 获取 StorageClass YAML
// @Description 获取指定 StorageClass 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/storageclasses/{name}/yaml [get]
func (kc *K8sController) GetStorageClassYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrStorageClasses(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func (kc *K8sController) EditStorageClass(c *gin.Context) {
	var req K8sEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrStorageClasses(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteStorageClass 删除 StorageClass。
// @Summary 删除 StorageClass
// @Description 删除指定 StorageClass
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/storageclasses/{name} [delete]
func (kc *K8sController) DeleteStorageClass(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrStorageClasses(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// ──────────────────────────────────────────────────────────
//  VolumeSnapshot 相关接口
// ──────────────────────────────────────────────────────────

func (kc *K8sController) buildResourceSupportResp(c *gin.Context, id uint64) (K8sResourceSupportResp, error) {
	ctx := c.Request.Context()
	checks := []struct {
		set func(*K8sResourceSupportResp, *bool)
		gvr schema.GroupVersionResource
	}{
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.ReplicaSets = ok }, gvrReplicaSets()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.ClusterRoles = ok }, gvrClusterRoles()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.Endpoints = ok }, gvrEndpoints()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.EndpointSlices = ok }, gvrEndpointSlices()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.NetworkPolicies = ok }, gvrNetworkPolicies()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.VolumeAttachments = ok }, gvrVolumeAttachments()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.ResourceQuotas = ok }, gvrResourceQuotas()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.LimitRanges = ok }, gvrLimitRanges()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.CustomResourceDefinitions = ok }, gvrCustomResourceDefinitions()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.APIServices = ok }, gvrAPIServices()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.PriorityClasses = ok }, gvrPriorityClasses()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.ValidatingWebhookConfigurations = ok }, gvrValidatingWebhookConfigurations()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.MutatingWebhookConfigurations = ok }, gvrMutatingWebhookConfigurations()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.Leases = ok }, gvrLeases()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.VolumeSnapshots = ok }, gvrVolumeSnapshots()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.VolumeSnapshotClasses = ok }, gvrVolumeSnapshotClasses()},
		{func(resp *K8sResourceSupportResp, ok *bool) { resp.VolumeSnapshotContents = ok }, gvrVolumeSnapshotContents()},
	}

	var out K8sResourceSupportResp
	for _, item := range checks {
		ok, err := kc.svc.SupportsCompatibleGVR(ctx, id, item.gvr)
		if err != nil {
			continue
		}
		item.set(&out, boolPtr(ok))
	}
	return out, nil
}

// GetResourceSupport 获取资源支持情况。
// @Summary 获取资源支持情况
// @Description 返回当前集群对关键 K8s 资源的支持情况，前端可据此隐藏不支持的入口并避免页面 404
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Success 200 {object} resp.Result{data=K8sResourceSupportResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/resource-support [get]
func (kc *K8sController) GetResourceSupport(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	data, err := kc.buildResourceSupportResp(c, id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// GetStorageSnapshotSupport 获取存储快照资源支持情况。
// @Summary 获取存储快照资源支持情况
// @Description 兼容旧前端调用，返回的字段已扩展为通用资源支持矩阵
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Success 200 {object} resp.Result{data=K8sResourceSupportResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/storage-snapshot-support [get]
func (kc *K8sController) GetStorageSnapshotSupport(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	data, err := kc.buildResourceSupportResp(c, id)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// ListVolumeSnapshots 获取 VolumeSnapshot 列表。
// @Summary VolumeSnapshot 列表
// @Description 获取指定集群 VolumeSnapshot 列表（可按 namespace 过滤）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshots [get]
func (kc *K8sController) ListVolumeSnapshots(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrVolumeSnapshots(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetVolumeSnapshotYAML 获取 VolumeSnapshot YAML。
// @Summary 获取 VolumeSnapshot YAML
// @Description 获取指定 VolumeSnapshot 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshots/{ns}/{name}/yaml [get]
func (kc *K8sController) GetVolumeSnapshotYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrVolumeSnapshots(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// EditVolumeSnapshot 编辑 VolumeSnapshot YAML。
// @Summary 编辑 VolumeSnapshot YAML
// @Description 通过 YAML 更新指定集群中的 VolumeSnapshot
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param payload body K8sNamespacedEditRequest true "YAML 编辑内容"
// @Success 200 {object} resp.Result "更新成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshots/edit [patch]
func (kc *K8sController) EditVolumeSnapshot(c *gin.Context) {
	var req K8sNamespacedEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrVolumeSnapshots(), strings.TrimSpace(req.Namespace), req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteVolumeSnapshot 删除 VolumeSnapshot。
// @Summary 删除 VolumeSnapshot
// @Description 删除指定 VolumeSnapshot
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshots/{ns}/{name} [delete]
func (kc *K8sController) DeleteVolumeSnapshot(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrVolumeSnapshots(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// ──────────────────────────────────────────────────────────
//  VolumeSnapshotClass 相关接口
// ──────────────────────────────────────────────────────────

// ListVolumeSnapshotClasses 获取 VolumeSnapshotClass 列表。
// @Summary VolumeSnapshotClass 列表
// @Description 获取指定集群 VolumeSnapshotClass 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshotclasses [get]
func (kc *K8sController) ListVolumeSnapshotClasses(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrVolumeSnapshotClasses(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetVolumeSnapshotClassYAML 获取 VolumeSnapshotClass YAML。
// @Summary 获取 VolumeSnapshotClass YAML
// @Description 获取指定 VolumeSnapshotClass 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshotclasses/{name}/yaml [get]
func (kc *K8sController) GetVolumeSnapshotClassYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrVolumeSnapshotClasses(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// EditVolumeSnapshotClass 编辑 VolumeSnapshotClass YAML。
// @Summary 编辑 VolumeSnapshotClass YAML
// @Description 通过 YAML 更新指定集群中的 VolumeSnapshotClass
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param payload body K8sEditRequest true "YAML 编辑内容"
// @Success 200 {object} resp.Result "更新成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshotclasses/edit [patch]
func (kc *K8sController) EditVolumeSnapshotClass(c *gin.Context) {
	var req K8sEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrVolumeSnapshotClasses(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteVolumeSnapshotClass 删除 VolumeSnapshotClass。
// @Summary 删除 VolumeSnapshotClass
// @Description 删除指定 VolumeSnapshotClass
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshotclasses/{name} [delete]
func (kc *K8sController) DeleteVolumeSnapshotClass(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrVolumeSnapshotClasses(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// ──────────────────────────────────────────────────────────
//  VolumeSnapshotContent 相关接口
// ──────────────────────────────────────────────────────────

// ListVolumeSnapshotContents 获取 VolumeSnapshotContent 列表。
// @Summary VolumeSnapshotContent 列表
// @Description 获取指定集群 VolumeSnapshotContent 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshotcontents [get]
func (kc *K8sController) ListVolumeSnapshotContents(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrVolumeSnapshotContents(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetVolumeSnapshotContentYAML 获取 VolumeSnapshotContent YAML。
// @Summary 获取 VolumeSnapshotContent YAML
// @Description 获取指定 VolumeSnapshotContent 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshotcontents/{name}/yaml [get]
func (kc *K8sController) GetVolumeSnapshotContentYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrVolumeSnapshotContents(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// EditVolumeSnapshotContent 编辑 VolumeSnapshotContent YAML。
// @Summary 编辑 VolumeSnapshotContent YAML
// @Description 通过 YAML 更新指定集群中的 VolumeSnapshotContent
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param payload body K8sEditRequest true "YAML 编辑内容"
// @Success 200 {object} resp.Result "更新成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshotcontents/edit [patch]
func (kc *K8sController) EditVolumeSnapshotContent(c *gin.Context) {
	var req K8sEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, gvrVolumeSnapshotContents(), "", req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteVolumeSnapshotContent 删除 VolumeSnapshotContent。
// @Summary 删除 VolumeSnapshotContent
// @Description 删除指定 VolumeSnapshotContent
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/volumesnapshotcontents/{name} [delete]
func (kc *K8sController) DeleteVolumeSnapshotContent(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrVolumeSnapshotContents(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}
