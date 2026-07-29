package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

type DeployController struct {
	svc              *service.DeployService
	terminalSessions *kopsapp.ExecSessionStore
}

func NewDeployController(svc *service.DeployService, terminalSessions ...*kopsapp.ExecSessionStore) *DeployController {
	var sessions *kopsapp.ExecSessionStore
	if len(terminalSessions) > 0 {
		sessions = terminalSessions[0]
	}
	return &DeployController{svc: svc, terminalSessions: sessions}
}

func (dc *DeployController) TestSSH(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.ProbeServerSSH(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func parseUintParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}

func currentUserID(c *gin.Context) uint64 {
	claims, ok := middleware.GetClaims(c)
	if !ok || claims == nil || claims.UserID <= 0 {
		return 0
	}
	return uint64(claims.UserID)
}
