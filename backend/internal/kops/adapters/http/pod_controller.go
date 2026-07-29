package http

import (
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/pkg/resp"
)

type PodController struct{ service *kopsapp.PodService }

func NewPodController(service *kopsapp.PodService) *PodController {
	return &PodController{service: service}
}
func (ctl *PodController) List(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.List(c.Request.Context(), kopsapp.PodListQuery{ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order"), LabelSelector: c.Query("label_selector")})
	ctl.respond(c, value, err)
}
func (ctl *PodController) Metrics(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.Metrics(c.Request.Context(), kopsapp.PodListQuery{ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order")})
	ctl.respond(c, value, err)
}
func (ctl *PodController) YAML(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.YAML(c.Request.Context(), podReference(c))
	ctl.respond(c, value, err)
}
func (ctl *PodController) Logs(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	tail := podTailLines(c.Query("tail_lines"))
	value, err := ctl.service.Logs(c.Request.Context(), kopsapp.PodLogsInput{PodReference: podReference(c), Container: c.Query("container"), TailLines: tail, Previous: podBool(c.Query("previous"))})
	ctl.respond(c, value, err)
}
func (ctl *PodController) CreateLogSession(c *gin.Context) {
	var request struct {
		Container *string `json:"container"`
		Follow    *bool   `json:"follow"`
		TailLines *int64  `json:"tail_lines"`
		Previous  bool    `json:"previous"`
	}
	if err := c.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if !ctl.ready(c) {
		return
	}
	result, err := ctl.service.CreateLogSession(kopsapp.PodLogSessionInput{PodReference: podReference(c), Container: request.Container, Follow: request.Follow, TailLines: request.TailLines, Previous: request.Previous})
	ctl.respond(c, result, err)
}
func (ctl *PodController) CreateExecSession(c *gin.Context) {
	var request struct {
		Container *string  `json:"container"`
		Command   []string `json:"command"`
		TTY       *bool    `json:"tty"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if !ctl.ready(c) {
		return
	}
	var userID uint64
	if claims, ok := middleware.GetClaims(c); ok && claims != nil && claims.UserID > 0 {
		userID = uint64(claims.UserID)
	}
	result, err := ctl.service.CreateExecSession(kopsapp.PodExecSessionInput{PodReference: podReference(c), UserID: userID, Container: request.Container, Command: request.Command, TTY: request.TTY})
	ctl.respond(c, result, err)
}
func (ctl *PodController) Delete(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	err := ctl.service.Delete(c.Request.Context(), podReference(c), podBool(c.Query("force")))
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{})
}
func (ctl *PodController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}
func (ctl *PodController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}
func podReference(c *gin.Context) kopsapp.PodReference {
	decode := func(value string) string {
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		return strings.TrimSpace(value)
	}
	return kopsapp.PodReference{ClusterID: namespaceClusterID(c), Namespace: decode(c.Param("ns")), Name: decode(c.Param("pod"))}
}
func podTailLines(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 200
	}
	return parsed
}
func podBool(value string) bool {
	value = strings.TrimSpace(value)
	return strings.EqualFold(value, "true") || value == "1"
}
