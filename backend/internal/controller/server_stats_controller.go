package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// ServerStatsController 服务器监控/系统信息 HTTP 处理器。
type ServerStatsController struct {
	svc *service.ServerStatsService
}

// NewServerStatsController 创建实例。
func NewServerStatsController(svc *service.ServerStatsService) *ServerStatsController {
	return &ServerStatsController{svc: svc}
}

func (sc *ServerStatsController) writeErr(c *gin.Context, err error) {
	WriteServiceErr(c, err, SSHErrMappings...)
}

func (sc *ServerStatsController) parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Fail(c, 4000, "无效的服务器 ID")
		return 0, false
	}
	return id, true
}

// GetStats 获取服务器实时监控指标。
// @Summary 服务器实时指标
// @Tags 服务器监控
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Success 200 {object} resp.Result{data=service.ServerStats}
// @Router /servers/{id}/stats [get]
func (sc *ServerStatsController) GetStats(c *gin.Context) {
	id, ok := sc.parseID(c)
	if !ok {
		return
	}
	stats, err := sc.svc.GetStats(c.Request.Context(), id)
	if err != nil {
		sc.writeErr(c, err)
		return
	}
	resp.OK(c, stats)
}

// GetSysInfo 获取服务器系统信息。
// @Summary 服务器系统信息
// @Tags 服务器监控
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Success 200 {object} resp.Result{data=service.ServerSysInfo}
// @Router /servers/{id}/sysinfo [get]
func (sc *ServerStatsController) GetSysInfo(c *gin.Context) {
	id, ok := sc.parseID(c)
	if !ok {
		return
	}
	info, err := sc.svc.GetSysInfo(c.Request.Context(), id)
	if err != nil {
		sc.writeErr(c, err)
		return
	}
	resp.OK(c, info)
}
