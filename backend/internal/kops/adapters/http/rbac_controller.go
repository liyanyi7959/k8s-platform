package http

import (
	"strings"

	"github.com/gin-gonic/gin"

	kopsapp "k8s-platform-backend/internal/kops/application"
	"k8s-platform-backend/pkg/resp"
)

// RBACController owns the transport edge for the pure RBAC policy generator.
// It deliberately has no dependency on the legacy permission-audit service.
type RBACController struct{}

func NewRBACController() *RBACController { return &RBACController{} }

func (ctl *RBACController) Default(c *gin.Context) {
	if ctl == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	resp.OK(c, kopsapp.DefaultRBACMatrix(splitNamespaces(c.Query("namespaces"))))
}

func (ctl *RBACController) Build(c *gin.Context) {
	if ctl == nil {
		resp.Fail(c, 5000, "internal error")
		return
	}
	var request kopsapp.RBACMatrixRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "invalid request")
		return
	}
	resp.OK(c, map[string]string{"yaml_content": kopsapp.BuildRBACFromMatrix(request)})
}

func splitNamespaces(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
