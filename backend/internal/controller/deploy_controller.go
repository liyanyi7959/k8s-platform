package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type DeployController struct {
	svc *service.DeployService
}

func NewDeployController(svc *service.DeployService) *DeployController {
	return &DeployController{svc: svc}
}

func (dc *DeployController) ListServers(c *gin.Context) {
	data, err := dc.svc.ListServers(c.Request.Context(), service.ListDeployServersRequest{Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20), Keyword: c.Query("keyword"), Status: c.Query("status")})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) CreateServer(c *gin.Context) {
	var req service.CreateDeployServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := dc.svc.CreateServer(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (dc *DeployController) GetServer(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.GetServer(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) UpdateServer(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateDeployServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := dc.svc.UpdateServer(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) DeleteServer(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := dc.svc.DeleteServer(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
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

func (dc *DeployController) ListCredentials(c *gin.Context) {
	data, err := dc.svc.ListCredentials(c.Request.Context(), service.ListCredentialsRequest{Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20), Keyword: c.Query("keyword"), AuthType: c.Query("auth_type")})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) CreateCredential(c *gin.Context) {
	var req service.CreateSSHCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := dc.svc.CreateCredential(c.Request.Context(), req)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (dc *DeployController) DeleteCredential(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := dc.svc.DeleteCredential(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) BatchDeleteCredentials(c *gin.Context) {
	var req struct {
		Ids []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		resp.Fail(c, 4000, "请选择要删除的凭据")
		return
	}
	if err := dc.svc.BatchDeleteCredentials(c.Request.Context(), req.Ids); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) GetCredential(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.GetCredential(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) UpdateCredential(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateSSHCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := dc.svc.UpdateCredential(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) ListPlans(c *gin.Context) {
	data, err := dc.svc.ListPlans(c.Request.Context(), service.ListDeployPlansRequest{Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20), Keyword: c.Query("keyword"), Status: c.Query("status")})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) CreatePlan(c *gin.Context) {
	var req service.CreateDeployPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := dc.svc.CreatePlan(c.Request.Context(), req, currentUserID(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (dc *DeployController) GetPlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.GetPlan(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (dc *DeployController) UpdatePlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req service.UpdateDeployPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := dc.svc.UpdatePlan(c.Request.Context(), id, req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) DryRunPlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.DryRunPlan(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// PreflightPlan 对控制端、拓扑和目标主机执行部署就绪检查。
func (dc *DeployController) PreflightPlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	data, err := dc.svc.PreflightPlan(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type setPreflightIgnoreRequest struct {
	Key     string `json:"key" binding:"required"`
	Ignored bool   `json:"ignored"`
}

func (dc *DeployController) SetPreflightIgnore(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	var req setPreflightIgnoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := dc.svc.SetPreflightIgnore(c.Request.Context(), id, req.Key, req.Ignored); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) DeletePlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := dc.svc.DeletePlan(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

// GetPlanAnsibleConfig 获取部署计划对应的 Ansible 执行配置。
func (dc *DeployController) GetPlanAnsibleConfig(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	cfg, err := dc.svc.GetPlanAnsibleConfig(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, cfg)
}

func (dc *DeployController) ExecutePlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	taskID, err := dc.svc.ExecutePlan(c.Request.Context(), id, currentUserID(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

func (dc *DeployController) CancelPlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	if err := dc.svc.CancelPlan(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (dc *DeployController) RetryPlan(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	taskID, err := dc.svc.RetryPlan(c.Request.Context(), id, currentUserID(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

// RetryDeployStep 从指定步骤开始重试部署。
func (dc *DeployController) RetryDeployStep(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	stepKey := c.Param("stepKey")
	if stepKey == "" {
		resp.Fail(c, 4000, "步骤 key 不能为空")
		return
	}
	taskID, err := dc.svc.RetryStep(c.Request.Context(), id, stepKey, currentUserID(c))
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

// GetDeployTask 获取部署任务详情
func (dc *DeployController) GetDeployTask(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil || taskID <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	task, ok := dc.svc.GetTaskStore().Get(taskID)
	if !ok {
		resp.Fail(c, 4004, "任务不存在")
		return
	}
	resp.OK(c, task)
}

// GetDeployTaskLogs 获取部署任务日志，支持按 step_key 过滤。
func (dc *DeployController) GetDeployTaskLogs(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil || taskID <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	task, ok := dc.svc.GetTaskStore().Get(taskID)
	if !ok {
		resp.Fail(c, 4004, "任务不存在")
		return
	}
	offset := parseInt(c.Query("offset"), 0)
	limit := parseInt(c.Query("limit"), 200)
	stepKey := c.Query("step_key")
	logs := task.Logs(offset, limit, stepKey)
	resp.OK(c, gin.H{"task_id": taskID, "logs": logs, "total": len(logs), "step_key": stepKey})
}

// GetDeployTaskLogsSSE SSE 实时日志推送，支持 step_key 过滤。
func (dc *DeployController) GetDeployTaskLogsSSE(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("taskId"), 10, 64)
	if err != nil || taskID <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	stepKey := c.Query("step_key")

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 获取初始偏移量
	offset := 0
	if lastEventID := c.GetHeader("Last-Event-ID"); lastEventID != "" {
		fmt.Sscanf(lastEventID, "%d", &offset)
	}

	// 发送心跳和日志
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// 先发送已有日志
	dc.sendLogs(c, taskID, stepKey, &offset)

	// 持续推送新日志
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			// 检查任务状态
			task, ok := dc.svc.GetTaskStore().Get(taskID)
			if !ok {
				// 任务不存在，发送完成事件
				fmt.Fprintf(c.Writer, "event: done\ndata: {\"status\":\"not_found\"}\n\n")
				c.Writer.Flush()
				return
			}

			// 发送新日志
			dc.sendLogs(c, taskID, stepKey, &offset)

			// 检查任务是否完成
			if task.Status == "success" || task.Status == "failed" || task.Status == "canceled" || task.Status == "timeout" {
				// 发送任务状态事件
				fmt.Fprintf(c.Writer, "event: done\ndata: {\"status\":\"%s\",\"message\":\"%s\"}\n\n", task.Status, getMessage(task.Message))
				c.Writer.Flush()
				return
			}

			// 发送心跳
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		}
	}
}

// sendLogs 发送日志到 SSE 流，支持按 stepKey 过滤。
func (dc *DeployController) sendLogs(c *gin.Context, taskID int64, stepKey string, offset *int) {
	task, ok := dc.svc.GetTaskStore().Get(taskID)
	if !ok {
		return
	}

	logs := task.Logs(*offset, 100, stepKey)
	for _, log := range logs {
		// 转义 JSON 特殊字符
		escaped := strings.ReplaceAll(log, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		escaped = strings.ReplaceAll(escaped, "\r", "\\r")
		fmt.Fprintf(c.Writer, "data: {\"log\":\"%s\"}\n\n", escaped)
	}
	*offset += len(logs)
	c.Writer.Flush()
}

// getMessage 获取消息字符串
func getMessage(msg *string) string {
	if msg == nil {
		return ""
	}
	return strings.ReplaceAll(*msg, "\"", "\\\"")
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
