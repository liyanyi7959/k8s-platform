package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type CredentialController struct {
	svc *service.CredentialService
}

func NewCredentialController(svc *service.CredentialService) *CredentialController {
	return &CredentialController{svc: svc}
}

type CredentialListPage struct {
	List     []service.CredentialItem `json:"list"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

func (cc *CredentialController) List(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 10)
	provider := c.Query("provider")
	if provider == "" {
		provider = c.Query("type")
	}
	data, err := cc.svc.ListCredentials(c.Request.Context(), service.ListCredentialsRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
		Provider: provider,
		AuthType: c.Query("auth_type"),
		SortBy:   c.Query("sort_by"),
		Order:    c.Query("order"),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

func (cc *CredentialController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := cc.svc.GetCredential(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type createCredentialReq struct {
	Name     string         `json:"name"`
	Provider string         `json:"provider"`
	AuthType string         `json:"auth_type"`
	Type     string         `json:"type"`
	Desc     *string        `json:"desc"`
	Data     map[string]any `json:"data"`
}

func (cc *CredentialController) Create(c *gin.Context) {
	var req createCredentialReq
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

	provider := req.Provider
	if provider == "" {
		provider = req.Type
	}
	id, err := cc.svc.CreateCredential(c.Request.Context(), service.CreateCredentialRequest{
		Name:      req.Name,
		Provider:  provider,
		AuthType:  req.AuthType,
		Desc:      req.Desc,
		Data:      req.Data,
		CreatedBy: uint64(userID),
	})
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

type patchCredentialReq struct {
	Name     *string         `json:"name"`
	Provider *string         `json:"provider"`
	AuthType *string         `json:"auth_type"`
	Type     *string         `json:"type"`
	Desc     **string        `json:"desc"`
	Data     *map[string]any `json:"data"`
}

func (cc *CredentialController) Patch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req patchCredentialReq
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

	provider := req.Provider
	if provider == nil && req.Type != nil {
		provider = req.Type
	}
	if err := cc.svc.PatchCredential(c.Request.Context(), id, service.PatchCredentialRequest{
		Name:      req.Name,
		Provider:  provider,
		AuthType:  req.AuthType,
		Desc:      req.Desc,
		Data:      req.Data,
		UpdatedBy: uint64(userID),
	}); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (cc *CredentialController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := cc.svc.DeleteCredential(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func (cc *CredentialController) GetData(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := cc.svc.GetCredentialData(c.Request.Context(), id)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"data": data})
}
