package http

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

type StorageController struct{ service *kopsapp.StorageService }

func NewStorageController(service *kopsapp.StorageService) *StorageController {
	return &StorageController{service: service}
}
func (ctl *StorageController) List(resource kopsapp.StorageResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.List(c.Request.Context(), resource, kopsapp.StorageListQuery{ClusterID: namespaceClusterID(c), Namespace: c.Query("namespace"), SortBy: c.Query("sort_by"), Order: c.Query("order")})
		ctl.respond(c, value, err)
	}
}
func (ctl *StorageController) YAML(resource kopsapp.StorageResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		value, err := ctl.service.YAML(c.Request.Context(), resource, storageReference(c))
		ctl.respond(c, value, err)
	}
}
func (ctl *StorageController) Delete(resource kopsapp.StorageResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ctl.ready(c) {
			return
		}
		if err := ctl.service.Delete(c.Request.Context(), resource, storageReference(c)); err != nil {
			writeKopsApplicationError(c, err)
			return
		}
		resp.OK(c, gin.H{})
	}
}
func (ctl *StorageController) Apply(resource kopsapp.StorageResource) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Namespace string `json:"namespace"`
			YAML      string `json:"yaml"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			resp.Fail(c, 4000, "invalid params")
			return
		}
		if !ctl.ready(c) {
			return
		}
		ctl.empty(c, ctl.service.Apply(c.Request.Context(), resource, kopsapp.StorageEdit{ClusterID: namespaceClusterID(c), Namespace: request.Namespace, YAML: request.YAML}))
	}
}
func (ctl *StorageController) CreatePVC(c *gin.Context) {
	var input kopsapp.CreatePVCInput
	if err := c.ShouldBindJSON(&input); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	if !ctl.ready(c) {
		return
	}
	input.ClusterID = namespaceClusterID(c)
	ctl.emptyObject(c, ctl.service.CreatePVC(c.Request.Context(), input))
}
func (ctl *StorageController) ResourceSupport(c *gin.Context) {
	if !ctl.ready(c) {
		return
	}
	value, err := ctl.service.ResourceSupport(c.Request.Context(), namespaceClusterID(c))
	ctl.respond(c, value, err)
}
func (ctl *StorageController) ready(c *gin.Context) bool {
	if ctl != nil && ctl.service != nil {
		return true
	}
	resp.Fail(c, 5000, "internal error")
	return false
}
func (ctl *StorageController) respond(c *gin.Context, value any, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, value)
}
func (ctl *StorageController) empty(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK[any](c, nil)
}
func (ctl *StorageController) emptyObject(c *gin.Context, err error) {
	if err != nil {
		writeKopsApplicationError(c, err)
		return
	}
	resp.OK(c, gin.H{})
}
func storageReference(c *gin.Context) kopsapp.StorageReference {
	decode := func(value string) string {
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		return strings.TrimSpace(value)
	}
	return kopsapp.StorageReference{ClusterID: namespaceClusterID(c), Namespace: decode(c.Param("ns")), Name: decode(c.Param("name"))}
}
