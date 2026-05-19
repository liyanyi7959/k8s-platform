package controller

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type ClusterManageController struct {
	svc    *service.ClusterRegistryService
	k8sSvc *service.K8sService
}

func NewClusterManageController(svc *service.ClusterRegistryService, k8sSvc *service.K8sService) *ClusterManageController {
	return &ClusterManageController{svc: svc, k8sSvc: k8sSvc}
}

type ClusterListPage struct {
	List     []service.ClusterItem `json:"list"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

// @Summary 集群列表
// @Description 分页列出集群，支持 keyword/status/type 过滤与排序
// @Tags 集群管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "页码" example(1)
// @Param page_size query int false "每页条数" example(10)
// @Param keyword query string false "关键字（按名称模糊搜索）"
// @Param status query string false "状态过滤" example(active)
// @Param type query string false "类型过滤" example(imported)
// @Param sort_by query string false "排序字段" example(created_at)
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=ClusterListPage} "查询成功"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters [get]
func (cc *ClusterManageController) List(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 10)
	data, err := cc.svc.ListClusters(c.Request.Context(), service.ListClustersRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		Type:     c.Query("type"),
		SortBy:   c.Query("sort_by"),
		Order:    c.Query("order"),
	})
	if err != nil {
		cc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type importClusterReq struct {
	Name       string `json:"name"`
	Kubeconfig string `json:"kubeconfig"`
}

type ImportClusterResp struct {
	ClusterID uint64 `json:"cluster_id"`
}

// @Summary 导入集群
// @Description 传入集群名称与 kubeconfig，保存后返回集群 ID
// @Tags 集群管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body importClusterReq true "导入参数"
// @Success 200 {object} resp.Result{data=ImportClusterResp} "导入成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 409 {object} resp.Result "资源冲突"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/import [post]
func (cc *ClusterManageController) Import(c *gin.Context) {
	var req importClusterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := cc.k8sSvc.ValidateKubeconfig(c.Request.Context(), req.Kubeconfig); err != nil {
		cc.writeServiceErr(c, err)
		return
	}
	id, err := cc.svc.ImportCluster(c.Request.Context(), req.Name, req.Kubeconfig)
	if err != nil {
		cc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"cluster_id": id})
}

// @Summary 查询集群详情
// @Description 根据集群 ID 查询集群基本信息与健康信息
// @Tags 集群管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Success 200 {object} resp.Result{data=service.ClusterDetail} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "集群不存在"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id} [get]
func (cc *ClusterManageController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := cc.svc.GetCluster(c.Request.Context(), uint64(id))
	if err != nil {
		cc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type CheckClusterHealthResp struct {
	APIOk        bool   `json:"api_ok"`
	NodeReady    int    `json:"node_ready"`
	NodeTotal    int    `json:"node_total"`
	CheckedAt    string `json:"checked_at"`
	Status       string `json:"status"`
	LastHealthAt string `json:"last_health_at,omitempty"`
}

// @Summary 检查集群健康
// @Description 对指定集群执行健康检查，并更新健康状态
// @Tags 集群管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Success 200 {object} resp.Result{data=CheckClusterHealthResp} "检查完成"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/check-health [post]
func (cc *ClusterManageController) CheckHealth(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	apiOK, ready, total, k8sVer, err := cc.k8sSvc.CheckHealth(c.Request.Context(), uint64(id))
	if err != nil {
		apiOK = false
		ready = 0
		total = 0
		_ = cc.svc.UpdateClusterHealth(c.Request.Context(), uint64(id), apiOK, ready, total, "")
		cc.writeServiceErr(c, err)
		return
	}
	_ = cc.svc.UpdateClusterHealth(c.Request.Context(), uint64(id), apiOK, ready, total, k8sVer)

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	status := "active"
	if !apiOK {
		status = "degraded"
	}
	lastHealthAt := ""
	if d, err := cc.svc.GetCluster(c.Request.Context(), uint64(id)); err == nil {
		status = d.Status
		if d.LastHealthAt != nil {
			lastHealthAt = *d.LastHealthAt
		}
	}
	if lastHealthAt == "" {
		lastHealthAt = nowStr
	}
	resp.OK(c, CheckClusterHealthResp{
		APIOk:        apiOK,
		NodeReady:    ready,
		NodeTotal:    total,
		CheckedAt:    nowStr,
		Status:       status,
		LastHealthAt: lastHealthAt,
	})
}

type patchClusterReq struct {
	Name       *string `json:"name"`
	Kubeconfig *string `json:"kubeconfig"`
}

// @Summary 更新集群
// @Description 根据集群 ID 更新集群信息（支持更新名称；导入集群支持更新 kubeconfig）
// @Tags 集群管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param body body patchClusterReq true "更新参数"
// @Success 200 {object} resp.Result "更新成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "集群不存在"
// @Failure 409 {object} resp.Result "资源冲突"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id} [patch]
func (cc *ClusterManageController) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req patchClusterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	if req.Kubeconfig != nil {
		if err := cc.k8sSvc.ValidateKubeconfigFormat(c.Request.Context(), *req.Kubeconfig); err != nil {
			cc.writeServiceErr(c, err)
			return
		}
	}

	if err := cc.svc.PatchCluster(c.Request.Context(), id, service.PatchClusterRequest{Name: req.Name, Kubeconfig: req.Kubeconfig}); err != nil {
		cc.writeServiceErr(c, err)
		return
	}
	if req.Kubeconfig != nil {
		cc.k8sSvc.StopClusterCaches(uint64(id))
	}
	resp.OK(c, gin.H{})
}

// @Summary 删除集群
// @Description 根据集群 ID 软删除
// @Tags 集群管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "集群不存在"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id} [delete]
func (cc *ClusterManageController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := cc.svc.DeleteCluster(c.Request.Context(), id); err != nil {
		cc.writeServiceErr(c, err)
		return
	}
	cc.k8sSvc.StopClusterCaches(uint64(id))
	resp.OK(c, gin.H{})
}

// writeServiceErr 委托给共享 WriteServiceErr，追加 K8s 领域特定映射。
func (cc *ClusterManageController) writeServiceErr(c *gin.Context, err error) {
	WriteServiceErr(c, err, K8sErrMappings...)
}
