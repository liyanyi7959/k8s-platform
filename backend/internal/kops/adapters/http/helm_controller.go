package http

import (
	"github.com/gin-gonic/gin"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
	"strings"
)

type HelmController struct{ service *kopsapp.HelmService }

func NewHelmController(service *kopsapp.HelmService) *HelmController {
	return &HelmController{service: service}
}
func (c *HelmController) Preflight(ctx *gin.Context) {
	if !c.ready(ctx) {
		return
	}
	value, err := c.service.Preflight(ctx.Request.Context(), namespaceClusterID(ctx))
	c.respond(ctx, value, err)
}
func (c *HelmController) Releases(ctx *gin.Context) {
	if !c.ready(ctx) {
		return
	}
	value, err := c.service.ListReleases(ctx.Request.Context(), namespaceClusterID(ctx), ctx.Query("namespace"))
	c.respond(ctx, value, err)
}
func (c *HelmController) ReleaseDetail(ctx *gin.Context) {
	if !c.ready(ctx) {
		return
	}
	namespace, name := strings.TrimSpace(ctx.Query("namespace")), strings.TrimSpace(ctx.Query("name"))
	if namespace == "" {
		namespace = ctx.Param("ns")
	}
	if name == "" {
		name = ctx.Param("name")
	}
	value, err := c.service.ReleaseDetail(ctx.Request.Context(), namespaceClusterID(ctx), namespace, name)
	c.respond(ctx, value, err)
}
func (c *HelmController) Uninstall(ctx *gin.Context) {
	if !c.ready(ctx) {
		return
	}
	output, err := c.service.Uninstall(ctx.Request.Context(), namespaceClusterID(ctx), ctx.Param("ns"), ctx.Param("name"))
	if err != nil {
		writeKopsApplicationError(ctx, err)
		return
	}
	resp.OK(ctx, gin.H{"output": output})
}
func (c *HelmController) Upgrade(ctx *gin.Context) {
	var input kopsapp.HelmUpgradeInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Fail(ctx, 4000, "invalid params")
		return
	}
	if !c.ready(ctx) {
		return
	}
	input.ClusterID = namespaceClusterID(ctx)
	input.Namespace = ctx.Param("ns")
	input.Name = ctx.Param("name")
	output, err := c.service.Upgrade(ctx.Request.Context(), input)
	if err != nil {
		writeKopsApplicationError(ctx, err)
		return
	}
	resp.OK(ctx, gin.H{"output": output})
}
func (c *HelmController) Rollback(ctx *gin.Context) {
	var input kopsapp.HelmRollbackInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Fail(ctx, 4000, "invalid params")
		return
	}
	if !c.ready(ctx) {
		return
	}
	input.ClusterID = namespaceClusterID(ctx)
	input.Namespace = ctx.Param("ns")
	input.Name = ctx.Param("name")
	output, err := c.service.Rollback(ctx.Request.Context(), input)
	if err != nil {
		writeKopsApplicationError(ctx, err)
		return
	}
	resp.OK(ctx, gin.H{"output": output})
}
func (c *HelmController) Install(ctx *gin.Context) {
	var input kopsapp.HelmInstallInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Fail(ctx, 4000, "invalid params")
		return
	}
	if !c.ready(ctx) {
		return
	}
	input.ClusterID = namespaceClusterID(ctx)
	value, err := c.service.Install(ctx.Request.Context(), input)
	c.respond(ctx, value, err)
}
func (c *HelmController) Repositories(ctx *gin.Context) {
	if !c.ready(ctx) {
		return
	}
	value, err := c.service.ListRepositories(ctx.Request.Context(), namespaceClusterID(ctx))
	c.respond(ctx, gin.H{"list": value}, err)
}
func (c *HelmController) AddRepository(ctx *gin.Context) {
	var input kopsapp.HelmRepositoryInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp.Fail(ctx, 4000, "invalid params")
		return
	}
	if !c.ready(ctx) {
		return
	}
	input.ClusterID = namespaceClusterID(ctx)
	err := c.service.AddRepository(ctx.Request.Context(), input)
	if err != nil {
		writeKopsApplicationError(ctx, err)
		return
	}
	resp.OK(ctx, gin.H{"name": input.Name, "url": input.URL})
}
func (c *HelmController) DeleteRepository(ctx *gin.Context) {
	if !c.ready(ctx) {
		return
	}
	name, err := c.service.DeleteRepository(ctx.Request.Context(), namespaceClusterID(ctx), ctx.Param("name"))
	if err != nil {
		writeKopsApplicationError(ctx, err)
		return
	}
	resp.OK(ctx, gin.H{"name": name})
}
func (c *HelmController) Search(ctx *gin.Context) {
	if !c.ready(ctx) {
		return
	}
	value, err := c.service.Search(ctx.Request.Context(), namespaceClusterID(ctx), strings.TrimSpace(ctx.Query("keyword")))
	c.respond(ctx, gin.H{"list": value}, err)
}
func (c *HelmController) ready(ctx *gin.Context) bool {
	if c != nil && c.service != nil {
		return true
	}
	resp.Fail(ctx, 5000, "internal error")
	return false
}
func (c *HelmController) respond(ctx *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(ctx, err)
		return
	}
	resp.OK(ctx, value)
}
