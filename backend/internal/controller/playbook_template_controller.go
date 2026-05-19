package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type PlaybookTemplateController struct {
	svc     *service.PlaybookTemplateService
	ansible *service.AnsibleService
}

func NewPlaybookTemplateController(svc *service.PlaybookTemplateService, ansible *service.AnsibleService) *PlaybookTemplateController {
	return &PlaybookTemplateController{svc: svc, ansible: ansible}
}

func (pc *PlaybookTemplateController) List(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 10)
	data, err := pc.svc.List(c.Request.Context(), service.ListPlaybookTemplatesRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
		Scenario: c.Query("scenario"),
		SortBy:   c.Query("sort_by"),
		Order:    c.Query("order"),
	})
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (pc *PlaybookTemplateController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := pc.svc.Get(c.Request.Context(), id)
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type createPlaybookTemplateReq struct {
	Name         string         `json:"name"`
	Scenario     string         `json:"scenario"`
	Description  *string        `json:"description"`
	Version      string         `json:"version"`
	Source       map[string]any `json:"source"`
	ParamsSchema map[string]any `json:"params_schema"`
	Defaults     map[string]any `json:"defaults"`
}

func (pc *PlaybookTemplateController) Create(c *gin.Context) {
	var req createPlaybookTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID := int64(1)
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(int64); ok2 && id > 0 {
			userID = id
		}
	}
	id, ver, err := pc.svc.Create(c.Request.Context(), service.CreatePlaybookTemplateRequest{
		Name:         req.Name,
		Scenario:     req.Scenario,
		Description:  req.Description,
		Version:      req.Version,
		Source:       req.Source,
		ParamsSchema: req.ParamsSchema,
		Defaults:     req.Defaults,
		CreatedBy:    uint64(userID),
	})
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "current_version": ver})
}

type patchPlaybookTemplateReq struct {
	Name        *string  `json:"name"`
	Scenario    *string  `json:"scenario"`
	Description **string `json:"description"`
}

func (pc *PlaybookTemplateController) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req patchPlaybookTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID := int64(1)
	if v, ok := c.Get("user_id"); ok {
		if id2, ok2 := v.(int64); ok2 && id2 > 0 {
			userID = id2
		}
	}
	if err := pc.svc.Patch(c.Request.Context(), id, service.PatchPlaybookTemplateRequest{
		Name:        req.Name,
		Scenario:    req.Scenario,
		Description: req.Description,
		UpdatedBy:   uint64(userID),
	}); err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (pc *PlaybookTemplateController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := pc.svc.Delete(c.Request.Context(), id); err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (pc *PlaybookTemplateController) ListVersions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := pc.svc.ListVersions(c.Request.Context(), id)
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": data})
}

type createVersionReq struct {
	Version      string         `json:"version"`
	Source       map[string]any `json:"source"`
	ParamsSchema map[string]any `json:"params_schema"`
	Defaults     map[string]any `json:"defaults"`
}

func (pc *PlaybookTemplateController) CreateVersion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req createVersionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID := int64(1)
	if v, ok := c.Get("user_id"); ok {
		if id2, ok2 := v.(int64); ok2 && id2 > 0 {
			userID = id2
		}
	}
	ver, err := pc.svc.CreateVersion(c.Request.Context(), id, service.CreatePlaybookTemplateVersionRequest{
		Version:      req.Version,
		Source:       req.Source,
		ParamsSchema: req.ParamsSchema,
		Defaults:     req.Defaults,
		CreatedBy:    uint64(userID),
	})
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"version": ver})
}

type rollbackReq struct {
	Version string `json:"version"`
}

func (pc *PlaybookTemplateController) Rollback(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req rollbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	userID := int64(1)
	if v, ok := c.Get("user_id"); ok {
		if id2, ok2 := v.(int64); ok2 && id2 > 0 {
			userID = id2
		}
	}
	if err := pc.svc.Rollback(c.Request.Context(), id, service.RollbackPlaybookTemplateRequest{Version: req.Version, UpdatedBy: uint64(userID)}); err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (pc *PlaybookTemplateController) UploadPlaybook(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	f, err := fh.Open()
	if err != nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	defer f.Close()
	data, err := pc.svc.SaveUploadedPlaybook(c.Request.Context(), fh.Filename, f, fh.Size)
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type runTemplateReq struct {
	ServerIDs []uint64       `json:"server_ids"`
	Version   string         `json:"version"`
	Params    map[string]any `json:"params"`
}

func (pc *PlaybookTemplateController) Run(c *gin.Context) {
	if pc.ansible == nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req runTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if len(req.ServerIDs) == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	tpl, err := pc.svc.Get(c.Request.Context(), id)
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = tpl.CurrentVersion
	}
	versions, err := pc.svc.ListVersions(c.Request.Context(), id)
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	var src map[string]any
	for _, v := range versions {
		if v.Version == version {
			src = v.Source
			break
		}
	}
	if src == nil {
		resp.Fail(c, 4040, "版本不存在")
		return
	}
	content, err := pc.svc.ResolvePlaybookContent(c.Request.Context(), src)
	if err != nil {
		pc.writeServiceErr(c, err)
		return
	}
	userID := int64(1)
	if v, ok := c.Get("user_id"); ok {
		if id2, ok2 := v.(int64); ok2 && id2 > 0 {
			userID = id2
		}
	}
	taskID, err := pc.ansible.RunPlaybookWithVars(c.Request.Context(), req.ServerIDs, content, req.Params, uint64(userID))
	if err != nil {
		resp.Fail(c, 5000, "启动任务失败")
		return
	}
	resp.OK(c, gin.H{"task_id": taskID})
}

// writeServiceErr 委托给共享 WriteServiceErr（仅通用映射）。
func (pc *PlaybookTemplateController) writeServiceErr(c *gin.Context, err error) {
	WriteServiceErr(c, err)
}
