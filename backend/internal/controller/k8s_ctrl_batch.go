package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
//  Job 相关接口
// ──────────────────────────────────────────────────────────

// ListJobs 获取 Job 列表。
// @Summary Job 列表
// @Description 获取指定集群 Job 列表（可按 namespace 过滤）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/jobs [get]
func (kc *K8sController) ListJobs(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrJobs(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetJobYAML 获取 Job YAML。
// @Summary 获取 Job YAML
// @Description 获取指定 Job 的 YAML 文本
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
// @Router /clusters/{id}/jobs/{ns}/{name}/yaml [get]
func (kc *K8sController) GetJobYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrJobs(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// DeleteJob 删除 Job。
// @Summary 删除 Job
// @Description 删除指定 Job
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
// @Router /clusters/{id}/jobs/{ns}/{name} [delete]
func (kc *K8sController) DeleteJob(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrJobs(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

type editJobReq struct {
	Namespace               string            `json:"namespace"`
	Name                    string            `json:"name"`
	Labels                  map[string]string `json:"labels"`
	Parallelism             *int32            `json:"parallelism"`
	Completions             *int32            `json:"completions"`
	BackoffLimit            *int32            `json:"backoffLimit"`
	TTLSecondsAfterFinished *int32            `json:"ttlSecondsAfterFinished"`
}

type triggerCronJobResp struct {
	JobName string `json:"job_name"`
}

type suspendCronJobReq struct {
	Suspend *bool `json:"suspend"`
}

type suspendCronJobResp struct {
	Suspend bool `json:"suspend"`
}

type deleteCompletedJobsResp struct {
	DeletedCount int `json:"deleted_count"`
}

func (kc *K8sController) EditJob(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Name = strings.TrimSpace(req.Name)
	if req.Namespace == "" || req.Name == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.Parallelism != nil && *req.Parallelism < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.Completions != nil && *req.Completions < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.BackoffLimit != nil && *req.BackoffLimit < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.TTLSecondsAfterFinished != nil && *req.TTLSecondsAfterFinished < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	patch := map[string]any{}
	if req.Labels != nil {
		patch["metadata"] = map[string]any{"labels": req.Labels}
	}
	spec := map[string]any{}
	if req.Parallelism != nil {
		spec["parallelism"] = *req.Parallelism
	}
	if req.Completions != nil {
		spec["completions"] = *req.Completions
	}
	if req.BackoffLimit != nil {
		spec["backoffLimit"] = *req.BackoffLimit
	}
	if req.TTLSecondsAfterFinished != nil {
		spec["ttlSecondsAfterFinished"] = *req.TTLSecondsAfterFinished
	}
	if len(spec) > 0 {
		patch["spec"] = spec
	}
	if len(patch) == 0 {
		resp.OK(c, gin.H{"ok": true})
		return
	}
	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvrJobs(), req.Namespace, req.Name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// DeleteCompletedJobs 批量清理已完成 Job。
// @Summary 批量清理已完成 Job
// @Description 按命名空间和 older_than_hours 条件批量删除已完成 Job
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param older_than_hours query int false "仅删除早于该小时数的已完成 Job" example(24)
// @Success 200 {object} resp.Result{data=deleteCompletedJobsResp} "清理成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/jobs/completed [delete]
func (kc *K8sController) DeleteCompletedJobs(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	olderThanHours := 24
	if raw := strings.TrimSpace(c.Query("older_than_hours")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			resp.Fail(c, 4000, "invalid params")
			return
		}
		olderThanHours = parsed
	}
	deletedCount, err := kc.svc.DeleteCompletedJobs(c.Request.Context(), id, strings.TrimSpace(c.Query("namespace")), olderThanHours)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, deleteCompletedJobsResp{DeletedCount: deletedCount})
}

// ──────────────────────────────────────────────────────────
//  CronJob 相关接口
// ──────────────────────────────────────────────────────────

// ListCronJobs 获取 CronJob 列表。
// @Summary CronJob 列表
// @Description 获取指定集群 CronJob 列表（可按 namespace 过滤）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/cronjobs [get]
func (kc *K8sController) ListCronJobs(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	list, err := kc.svc.List(c.Request.Context(), id, gvrCronJobs(), ns, c.Query("sort_by"), c.Query("order"), nil)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// GetCronJobYAML 获取 CronJob YAML。
// @Summary 获取 CronJob YAML
// @Description 获取指定 CronJob 的 YAML 文本
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
// @Router /clusters/{id}/cronjobs/{ns}/{name}/yaml [get]
func (kc *K8sController) GetCronJobYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrCronJobs(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// DeleteCronJob 删除 CronJob。
// @Summary 删除 CronJob
// @Description 删除指定 CronJob
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
// @Router /clusters/{id}/cronjobs/{ns}/{name} [delete]
func (kc *K8sController) DeleteCronJob(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("name"))
	if err := kc.svc.Delete(c.Request.Context(), id, gvrCronJobs(), ns, name); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// TriggerCronJob 立即触发 CronJob 一次执行。
// @Summary 立即执行 CronJob
// @Description 根据 CronJob 的 jobTemplate 立即创建一个 Job
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Success 200 {object} resp.Result{data=triggerCronJobResp} "触发成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/cronjobs/{ns}/{name}/trigger [post]
func (kc *K8sController) TriggerCronJob(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	result, err := kc.svc.TriggerCronJob(c.Request.Context(), id, decodePathParam(c.Param("ns")), decodePathParam(c.Param("name")))
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, triggerCronJobResp{JobName: result.JobName})
}

// SuspendCronJob 暂停或恢复 CronJob 调度。
// @Summary 暂停或恢复 CronJob
// @Description patch spec.suspend 字段，控制 CronJob 调度状态
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param name path string true "资源名称"
// @Param body body suspendCronJobReq true "暂停状态"
// @Success 200 {object} resp.Result{data=suspendCronJobResp} "更新成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/cronjobs/{ns}/{name}/suspend [patch]
func (kc *K8sController) SuspendCronJob(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req suspendCronJobReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Suspend == nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if err := kc.svc.SuspendCronJob(c.Request.Context(), id, decodePathParam(c.Param("ns")), decodePathParam(c.Param("name")), *req.Suspend); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, suspendCronJobResp{Suspend: *req.Suspend})
}

type editCronJobReq struct {
	Namespace                  string            `json:"namespace"`
	Name                       string            `json:"name"`
	Labels                     map[string]string `json:"labels"`
	Schedule                   string            `json:"schedule"`
	Suspend                    *bool             `json:"suspend"`
	ConcurrencyPolicy          *string           `json:"concurrencyPolicy"`
	SuccessfulJobsHistoryLimit *int32            `json:"successfulJobsHistoryLimit"`
	FailedJobsHistoryLimit     *int32            `json:"failedJobsHistoryLimit"`
}

func (kc *K8sController) EditCronJob(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req editCronJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Name = strings.TrimSpace(req.Name)
	req.Schedule = strings.TrimSpace(req.Schedule)
	if req.Namespace == "" || req.Name == "" || req.Schedule == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.SuccessfulJobsHistoryLimit != nil && *req.SuccessfulJobsHistoryLimit < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if req.FailedJobsHistoryLimit != nil && *req.FailedJobsHistoryLimit < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}

	patch := map[string]any{}
	if req.Labels != nil {
		patch["metadata"] = map[string]any{"labels": req.Labels}
	}
	spec := map[string]any{"schedule": req.Schedule}
	if req.Suspend != nil {
		spec["suspend"] = *req.Suspend
	}
	if req.ConcurrencyPolicy != nil && strings.TrimSpace(*req.ConcurrencyPolicy) != "" {
		spec["concurrencyPolicy"] = strings.TrimSpace(*req.ConcurrencyPolicy)
	}
	if req.SuccessfulJobsHistoryLimit != nil {
		spec["successfulJobsHistoryLimit"] = *req.SuccessfulJobsHistoryLimit
	}
	if req.FailedJobsHistoryLimit != nil {
		spec["failedJobsHistoryLimit"] = *req.FailedJobsHistoryLimit
	}
	patch["spec"] = spec

	if err := kc.svc.PatchJSON(c.Request.Context(), id, gvrCronJobs(), req.Namespace, req.Name, patch); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}
