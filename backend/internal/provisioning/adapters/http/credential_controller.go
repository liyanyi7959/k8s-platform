package http

import (
	"github.com/gin-gonic/gin"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	"k8s-platform-backend/pkg/resp"
)

// CredentialController exposes SSH credential metadata and commands without
// ever serializing the encrypted or plaintext secret value.
type CredentialController struct {
	svc *provisionapp.CredentialService
}

func NewCredentialController(svc *provisionapp.CredentialService) *CredentialController {
	return &CredentialController{svc: svc}
}

func (cc *CredentialController) List(c *gin.Context) {
	data, err := cc.svc.List(c.Request.Context(), provisionapp.ListCredentialsRequest{
		Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 20),
		Keyword: c.Query("keyword"), AuthType: c.Query("auth_type"),
	})
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (cc *CredentialController) Create(c *gin.Context) {
	var req provisionapp.CreateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := cc.svc.Create(c.Request.Context(), req)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (cc *CredentialController) Get(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	data, err := cc.svc.Get(c.Request.Context(), id)
	if err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK(c, data)
}

func (cc *CredentialController) Update(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	var req provisionapp.UpdateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := cc.svc.Update(c.Request.Context(), id, req); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (cc *CredentialController) Delete(c *gin.Context) {
	id, ok := provisioningResourceID(c)
	if !ok {
		return
	}
	if err := cc.svc.Delete(c.Request.Context(), id); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}

func (cc *CredentialController) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		resp.Fail(c, 4000, "请选择要删除的凭据")
		return
	}
	if err := cc.svc.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		writeProvisioningError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
