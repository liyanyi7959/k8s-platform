package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
//  Namespace 相关接口
// ──────────────────────────────────────────────────────────

// ListNamespaces 获取命名空间列表。
// 入参：路径 /clusters/:id；可选 query：sort_by、order（用于前端列表排序）。
// 出参：{"list": [...]}
// @Summary 命名空间列表
// @Description 获取指定集群命名空间列表
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
// @Router /clusters/{id}/namespaces [get]
func (kc *K8sController) ListNamespaces(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrNamespaces(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

type createNamespaceReq struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

type namespaceResourceSummaryItemResp struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type namespaceResourcesSummaryResp struct {
	Namespace string                             `json:"namespace"`
	Total     int                                `json:"total"`
	Items     []namespaceResourceSummaryItemResp `json:"items"`
}

// CreateNamespace 创建命名空间。
// 入参：路径 /clusters/:id，JSON body：{name, labels}。
// 仅做基本非空校验，具体创建由 service 层通过 dynamic client 调用 K8s API 完成。
// @Summary 创建命名空间
// @Description 在指定集群中创建命名空间
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body createNamespaceReq true "创建命名空间参数"
// @Success 200 {object} resp.Result "创建成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/namespaces [post]
func (kc *K8sController) CreateNamespace(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req createNamespaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.CreateNamespace(c.Request.Context(), id, req.Name, req.Labels); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// DeleteNamespace 删除指定命名空间。
// 入参：路径 /clusters/:id/namespaces/:ns
// @Summary 删除命名空间
// @Description 删除指定命名空间
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/namespaces/{ns} [delete]
func (kc *K8sController) DeleteNamespace(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrNamespaces(), "", ns); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// GetNamespaceResourcesSummary 获取命名空间资源摘要。
// @Summary 获取 Namespace 资源摘要
// @Description 并发统计命名空间下常见资源数量，用于删除前风险确认
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Success 200 {object} resp.Result{data=namespaceResourcesSummaryResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/namespaces/{ns}/resources-summary [get]
func (kc *K8sController) GetNamespaceResourcesSummary(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	items, total, err := kc.svc.GetNamespaceResourcesSummary(c.Request.Context(), id, ns)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	respItems := make([]namespaceResourceSummaryItemResp, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, namespaceResourceSummaryItemResp{Key: item.Key, Label: item.Label, Count: item.Count})
	}
	resp.OK(c, namespaceResourcesSummaryResp{Namespace: ns, Total: total, Items: respItems})
}

// GetNamespaceYAML 获取命名空间 YAML（以文本形式返回）。
// 出参：{"text": "---\n...\n"}
// @Summary 获取命名空间 YAML
// @Description 获取指定命名空间的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/namespaces/{ns}/yaml [get]
func (kc *K8sController) GetNamespaceYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrNamespaces(), "", ns)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

func buildInvolvedObjectFieldSelector(kind, name, uid string) string {
	selectors := make([]fields.Selector, 0, 3)
	if v := strings.TrimSpace(kind); v != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("involvedObject.kind", v))
	}
	if v := strings.TrimSpace(name); v != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("involvedObject.name", v))
	}
	if v := strings.TrimSpace(uid); v != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("involvedObject.uid", v))
	}
	if len(selectors) == 0 {
		return ""
	}
	return fields.AndSelectors(selectors...).String()
}

// ListEvents 获取 Event 列表（按命名空间和 involvedObject 过滤）。
// @Summary Event 列表
// @Description 获取指定集群 Event 列表，可按 namespace 与 involvedObject 字段过滤
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param involved_object_kind query string false "涉及对象 Kind，例如 Pod/Deployment"
// @Param involved_object_name query string false "涉及对象名称"
// @Param involved_object_uid query string false "涉及对象 UID"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/events [get]
func (kc *K8sController) ListEvents(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	extraListOptions := map[string]string{}
	if fieldSelector := buildInvolvedObjectFieldSelector(c.Query("involved_object_kind"), c.Query("involved_object_name"), c.Query("involved_object_uid")); fieldSelector != "" {
		extraListOptions["field_selector"] = fieldSelector
	}
	if len(extraListOptions) == 0 {
		extraListOptions = nil
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrEvents(), ns, c.Query("sort_by"), c.Query("order"), extraListOptions)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// ──────────────────────────────────────────────────────────
//  ServiceAccount 相关接口
// ──────────────────────────────────────────────────────────

// ListServiceAccounts 获取 ServiceAccount 列表。
// @Summary ServiceAccount 列表
// @Description 获取指定集群 ServiceAccount 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/serviceaccounts [get]
func (kc *K8sController) ListServiceAccounts(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}, ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetServiceAccountYAML 获取 ServiceAccount YAML。
// @Summary 获取 ServiceAccount YAML
// @Description 获取指定 ServiceAccount 的 YAML 文本
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
// @Router /clusters/{id}/serviceaccounts/{ns}/{name}/yaml [get]
func (kc *K8sController) GetServiceAccountYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}, ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// DeleteServiceAccount 删除 ServiceAccount。
// @Summary 删除 ServiceAccount
// @Description 删除指定 ServiceAccount
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
// @Router /clusters/{id}/serviceaccounts/{ns}/{name} [delete]
func (kc *K8sController) DeleteServiceAccount(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}, ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) EditServiceAccount(c *gin.Context) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}, req.Namespace, req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// ──────────────────────────────────────────────────────────
//  HPA 相关接口
// ──────────────────────────────────────────────────────────

// ListHPAs 获取 HPA 列表。
// @Summary HPA 列表
// @Description 获取指定集群 HPA 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/hpas [get]
func (kc *K8sController) ListHPAs(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}, ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetHPAYAML 获取 HPA YAML。
// @Summary 获取 HPA YAML
// @Description 获取指定 HPA 的 YAML 文本
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
// @Router /clusters/{id}/hpas/{ns}/{name}/yaml [get]
func (kc *K8sController) GetHPAYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}, ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// EditHPA 编辑 HPA。
// @Summary 编辑 HPA
// @Description 更新指定 HPA 的 YAML
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body K8sEditRequest true "YAML内容"
// @Success 200 {object} resp.Result "更新成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/hpas/edit [patch]
func (kc *K8sController) EditHPA(c *gin.Context) {
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
	if err := kc.svc.ApplyYAML(c.Request.Context(), id, schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}, req.Namespace, req.YAML); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// DeleteHPA 删除 HPA。
// @Summary 删除 HPA
// @Description 删除指定 HPA
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
// @Router /clusters/{id}/hpas/{ns}/{name} [delete]
func (kc *K8sController) DeleteHPA(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}, ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
