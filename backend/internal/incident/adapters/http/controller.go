package http

import (
	"errors"
	stdhttp "net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/incident/application"
	"k8s-platform-backend/internal/incident/domain"
	"k8s-platform-backend/internal/incident/ports"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/problem"
)

type Controller struct{ service *application.Service }

func NewController(service *application.Service) *Controller { return &Controller{service: service} }

func (ctl *Controller) List(c *gin.Context) {
	page, err := ctl.service.List(c.Request.Context(), ports.ListFilter{Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20), Status: domain.Status(strings.TrimSpace(c.Query("status")))})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(stdhttp.StatusOK, page)
}

func (ctl *Controller) Get(c *gin.Context) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	detail, err := ctl.service.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(stdhttp.StatusOK, detail)
}

func (ctl *Controller) Acknowledge(c *gin.Context)     { ctl.execute(c, domain.CommandAcknowledge) }
func (ctl *Controller) Diagnose(c *gin.Context)        { ctl.execute(c, domain.CommandDiagnose) }
func (ctl *Controller) RequestApproval(c *gin.Context) { ctl.execute(c, domain.CommandRequestApproval) }
func (ctl *Controller) StartExecution(c *gin.Context)  { ctl.execute(c, domain.CommandStartExecution) }
func (ctl *Controller) StartVerification(c *gin.Context) {
	ctl.execute(c, domain.CommandStartVerification)
}
func (ctl *Controller) Resolve(c *gin.Context) { ctl.execute(c, domain.CommandResolve) }

func (ctl *Controller) execute(c *gin.Context, command domain.Command) {
	id, ok := incidentID(c)
	if !ok {
		return
	}
	var request application.CommandRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		problem.Write(c, stdhttp.StatusBadRequest, "https://aiops.local/problems/invalid-request", "请求参数错误", "请求体必须包含 expected_version")
		return
	}
	claims, _ := middleware.GetClaims(c)
	actor := domain.Actor{}
	if claims != nil {
		actor.ID, actor.Name = uint64(claims.UserID), strings.TrimSpace(claims.Username)
	}
	incident, err := ctl.service.Execute(c.Request.Context(), id, command, request, actor)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(stdhttp.StatusOK, incident)
}

func incidentID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		problem.Write(c, stdhttp.StatusBadRequest, "https://aiops.local/problems/invalid-request", "请求参数错误", "incident id 必须是正整数")
		return 0, false
	}
	return id, true
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		problem.Write(c, stdhttp.StatusNotFound, "https://aiops.local/problems/incident-not-found", "事件不存在", "指定的事件不存在或已被删除")
	case errors.Is(err, domain.ErrVersionConflict):
		problem.Write(c, stdhttp.StatusConflict, "https://aiops.local/problems/version-conflict", "事件已被其他用户更新", "请刷新事件详情后重试")
	case errors.Is(err, domain.ErrInvalidTransition):
		problem.Write(c, stdhttp.StatusConflict, "https://aiops.local/problems/invalid-incident-transition", "当前状态不允许此操作", err.Error())
	case errors.Is(err, domain.ErrValidation):
		problem.Write(c, stdhttp.StatusUnprocessableEntity, "https://aiops.local/problems/domain-validation", "事件处置校验失败", err.Error())
	default:
		problem.Write(c, stdhttp.StatusInternalServerError, "https://aiops.local/problems/internal", "内部错误", "事件服务暂时不可用")
	}
}
