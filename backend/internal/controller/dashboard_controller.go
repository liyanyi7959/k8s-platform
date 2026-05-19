package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type DashboardController struct {
	svc *service.DashboardService
}

func NewDashboardController(svc *service.DashboardService) *DashboardController {
	return &DashboardController{svc: svc}
}

type AnyMap map[string]interface{}

// @Summary 集群概览
// @Description 根据集群 ID 获取集群概览统计信息（节点/Pod/工作负载/资源使用率等）
// @Tags 仪表盘接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Success 200 {object} resp.Result{data=AnyMap} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /dashboard/clusters/{id}/overview [get]
func (dc *DashboardController) GetClusterOverview(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := dc.svc.GetClusterOverview(c.Request.Context(), uint64(id))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// @Summary 集群证书风险
// @Description 根据集群 ID 获取集群证书风险信息，不阻塞概览首屏加载
// @Tags 仪表盘接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Success 200 {object} resp.Result{data=[]AnyMap} "查询成功"
// @Failure default {object} resp.Result "失败响应"
// @Router /dashboard/clusters/{id}/certificate-risks [get]
func (dc *DashboardController) GetClusterCertificateRisks(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := dc.svc.GetClusterCertificateRisks(c.Request.Context(), uint64(id))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}
