package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
//  Node 相关接口
// ──────────────────────────────────────────────────────────

// ListNodes 获取节点列表（cluster scope）。
// @Summary 节点列表
// @Description 获取指定集群节点列表
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
// @Router /clusters/{id}/nodes [get]
func (kc *K8sController) ListNodes(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrNodes(), "", c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetNodeDetail 获取 Node 详情（原始对象）。
// @Summary 获取 Node 详情
// @Description 获取指定 Node 的详细信息（原始对象）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "节点名称"
// @Success 200 {object} resp.Result{data=map[string]any} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/nodes/{name}/detail [get]
func (kc *K8sController) GetNodeDetail(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	obj, err := kc.svc.GetObject(c.Request.Context(), id, gvrNodes(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"obj": obj})
}

// GetNodeYAML 获取 Node YAML。
// @Summary 获取 Node YAML
// @Description 获取指定 Node 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "节点名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/nodes/{name}/yaml [get]
func (kc *K8sController) GetNodeYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrNodes(), "", name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// ListNodePods 获取指定 Node 上的 Pod 列表（跨命名空间）。
// 可选 query：sort_by、order。
// @Summary Node Pods 列表
// @Description 获取指定节点上的 Pod 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "节点名称"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/nodes/{name}/pods [get]
func (kc *K8sController) ListNodePods(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	list, err := kc.svc.ListPodsOnNode(c.Request.Context(), id, name, c.Query("sort_by"), c.Query("order"))
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// ListNodeEvents 获取指定 Node 的事件列表（跨命名空间）。
// @Summary Node Events 列表
// @Description 获取指定节点相关的 Event 列表
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "节点名称"
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/nodes/{name}/events [get]
func (kc *K8sController) ListNodeEvents(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	list, err := kc.svc.ListNodeEvents(c.Request.Context(), id, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// CordonNode 停止节点调度（Cordon）。
// @Summary 停止节点调度
// @Description 将节点标记为不可调度
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "节点名称"
// @Success 200 {object} resp.Result "操作成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/nodes/{name}/cordon [post]
func (kc *K8sController) CordonNode(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.UpdateNodeSchedulable(c.Request.Context(), id, name, true); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// UncordonNode 恢复节点调度（Uncordon）。
// @Summary 恢复节点调度
// @Description 将节点标记为可调度
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param name path string true "节点名称"
// @Success 200 {object} resp.Result "操作成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/nodes/{name}/uncordon [post]
func (kc *K8sController) UncordonNode(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.UpdateNodeSchedulable(c.Request.Context(), id, name, false); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DrainNode(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))

	force := false
	if v := strings.TrimSpace(c.Query("force")); v != "" {
		v2 := strings.ToLower(v)
		force = v2 == "1" || v2 == "true" || v2 == "yes"
	}
	timeoutSeconds := 0
	if v := strings.TrimSpace(c.Query("timeout_seconds")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			resp.Fail(c, 4000, "invalid params")
			return
		}
		timeoutSeconds = n
	}
	ignoreDaemonSets := true
	if v := strings.TrimSpace(c.Query("ignore_daemonsets")); v != "" {
		v2 := strings.ToLower(v)
		ignoreDaemonSets = v2 == "1" || v2 == "true" || v2 == "yes"
	}

	if err := kc.svc.DrainNode(c.Request.Context(), id, name, service.DrainNodeOptions{
		TimeoutSeconds:   timeoutSeconds,
		Force:            force,
		IgnoreDaemonSets: ignoreDaemonSets,
	}); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (kc *K8sController) DeleteNode(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrNodes(), "", name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
