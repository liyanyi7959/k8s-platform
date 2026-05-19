package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/middleware"
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type ServerTaskController struct {
	svc *service.ServerTaskService
}

func NewServerTaskController(svc *service.ServerTaskService) *ServerTaskController {
	return &ServerTaskController{svc: svc}
}

type createServerTaskReq struct {
	ServerIDs  []uint64 `json:"server_ids"`
	Command    string   `json:"command"`
	TimeoutSec int      `json:"timeout_sec"`
}

type CreateServerTaskResp struct {
	TaskID int64 `json:"task_id"`
}

func (stc *ServerTaskController) Create(c *gin.Context) {
	var req createServerTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	req.Command = strings.TrimSpace(req.Command)
	if len(req.ServerIDs) == 0 || req.Command == "" {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID := int64(1)
	if claims, ok := middleware.GetClaims(c); ok && claims != nil && claims.UserID > 0 {
		userID = claims.UserID
	}
	id, err := stc.svc.CreateTask(c.Request.Context(), service.CreateServerTaskRequest{
		ServerIDs:  req.ServerIDs,
		Command:    req.Command,
		TimeoutSec: req.TimeoutSec,
		CreatedBy:  userID,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidParams) {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		resp.Fail(c, 5000, "内部错误")
		return
	}
	resp.OK(c, gin.H{"task_id": id})
}

type ServerTaskListItem struct {
	ID         int64  `json:"id"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	CreatedBy  int64  `json:"created_by"`
	Command    string `json:"command"`
	TimeoutSec int    `json:"timeout_sec"`
}

type ServerTaskListPage struct {
	List     []ServerTaskListItem `json:"list"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

func (stc *ServerTaskController) List(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 10)
	req := service.ListServerTasksRequest{
		Page:      page,
		PageSize:  pageSize,
		Status:    c.Query("status"),
		Keyword:   c.Query("keyword"),
		CreatedBy: parseInt64Ptr(c.Query("created_by")),
		SortBy:    c.Query("sort_by"),
		Order:     c.Query("order"),
	}
	data := stc.svc.ListTasks(req)
	items := make([]ServerTaskListItem, 0, len(data.List))
	for _, t := range data.List {
		items = append(items, ServerTaskListItem{
			ID:         t.ID,
			Type:       t.Type,
			Status:     string(t.Status),
			CreatedAt:  t.CreatedAt,
			CreatedBy:  t.CreatedBy,
			Command:    t.Command,
			TimeoutSec: t.TimeoutSec,
		})
	}
	resp.OK(c, ServerTaskListPage{List: items, Total: data.Total, Page: data.Page, PageSize: data.PageSize})
}

type ServerTaskTarget struct {
	ServerID   uint64               `json:"server_id"`
	Status     string               `json:"status"`
	ExitCode   *int                 `json:"exit_code,omitempty"`
	DurationMs *int64               `json:"duration_ms,omitempty"`
	Server     *service.ServerBrief `json:"server,omitempty"`
}

type ServerTaskDetail struct {
	ID         int64              `json:"id"`
	Type       string             `json:"type"`
	Status     string             `json:"status"`
	CreatedAt  string             `json:"created_at"`
	CreatedBy  int64              `json:"created_by"`
	Command    string             `json:"command"`
	TimeoutSec int                `json:"timeout_sec"`
	Targets    []ServerTaskTarget `json:"targets"`
}

func (stc *ServerTaskController) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	t, err := stc.svc.GetTask(id)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParams) {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			resp.Fail(c, 4040, "任务不存在")
			return
		}
		resp.Fail(c, 5000, "内部错误")
		return
	}
	out := ServerTaskDetail{
		ID:         t.ID,
		Type:       t.Type,
		Status:     string(t.Status),
		CreatedAt:  t.CreatedAt,
		CreatedBy:  t.CreatedBy,
		Command:    t.Command,
		TimeoutSec: t.TimeoutSec,
	}
	targets := make([]ServerTaskTarget, 0, len(t.Targets))
	ids := make([]uint64, 0, len(t.Targets))
	for _, tg := range t.Targets {
		if tg.ServerID > 0 {
			ids = append(ids, tg.ServerID)
		}
	}
	serverMap, _ := stc.svc.GetServerBriefs(c.Request.Context(), ids)
	for _, tg := range t.Targets {
		var srv *service.ServerBrief
		if serverMap != nil {
			if brief, ok := serverMap[tg.ServerID]; ok {
				b := brief
				srv = &b
			}
		}
		targets = append(targets, ServerTaskTarget{
			ServerID:   tg.ServerID,
			Status:     string(tg.Status),
			ExitCode:   tg.ExitCode,
			DurationMs: tg.DurationMs,
			Server:     srv,
		})
	}
	out.Targets = targets
	resp.OK(c, out)
}

type ServerTaskLogsResp struct {
	TargetServerID uint64   `json:"target_server_id"`
	Offset         int      `json:"offset"`
	Limit          int      `json:"limit"`
	Lines          []string `json:"lines"`
}

func (stc *ServerTaskController) Logs(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	targetID := parseInt64(c.Query("target_server_id"), 0)
	if targetID <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	offset := parseInt(c.Query("offset"), 0)
	limit := parseInt(c.Query("limit"), 200)
	if limit <= 0 {
		limit = 200
	}
	serverID, lines, err := stc.svc.GetTaskLogs(id, uint64(targetID), offset, limit)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParams) {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			resp.Fail(c, 4040, "任务不存在")
			return
		}
		resp.Fail(c, 5000, "内部错误")
		return
	}
	resp.OK(c, ServerTaskLogsResp{
		TargetServerID: serverID,
		Offset:         offset,
		Limit:          limit,
		Lines:          lines,
	})
}

func (stc *ServerTaskController) Cancel(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := stc.svc.CancelTask(id); err != nil {
		if errors.Is(err, service.ErrInvalidParams) {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			resp.Fail(c, 4040, "任务不存在")
			return
		}
		if errors.Is(err, service.ErrConflict) {
			resp.Fail(c, 4090, "无法取消")
			return
		}
		resp.Fail(c, 5000, "内部错误")
		return
	}
	resp.OK(c, gin.H{})
}
