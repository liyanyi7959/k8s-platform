package http

import (
	"errors"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/platform/application"
	"k8s-platform-backend/internal/platform/domain"
	"k8s-platform-backend/pkg/resp"
)

type SettingsController struct{ service *application.Service }

func NewSettingsController(service *application.Service) *SettingsController {
	return &SettingsController{service: service}
}

func (ctl *SettingsController) Get(c *gin.Context) {
	settings, err := ctl.service.Get(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "读取系统设置失败")
		return
	}
	resp.OK(c, settings)
}

func (ctl *SettingsController) Update(c *gin.Context) {
	var request application.SettingsDTO
	if err := c.ShouldBindJSON(&request); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.service.Update(c.Request.Context(), request); err != nil {
		if errors.Is(err, domain.ErrValidation) {
			resp.Fail(c, 4000, err.Error())
			return
		}
		resp.Fail(c, 5000, "保存系统设置失败")
		return
	}
	resp.OK[any](c, nil)
}
