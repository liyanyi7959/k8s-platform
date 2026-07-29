package http

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/fleet/ports"
	"k8s-platform-backend/pkg/resp"
)

type DashboardController struct{ reader ports.DashboardReader }

func NewDashboardController(reader ports.DashboardReader) *DashboardController {
	return &DashboardController{reader: reader}
}
func (ctl *DashboardController) GetClusterOverview(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.reader.GetClusterOverview(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, 5000, "查询集群概览失败")
		return
	}
	resp.OK(c, data)
}
func (ctl *DashboardController) GetClusterCertificateRisks(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := ctl.reader.GetClusterCertificateRisks(c.Request.Context(), id)
	if err != nil {
		resp.Fail(c, 5000, "查询证书风险失败")
		return
	}
	resp.OK(c, data)
}
