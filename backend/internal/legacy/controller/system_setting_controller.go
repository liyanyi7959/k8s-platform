package controller

import (
	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/legacy/service"
	"k8s-platform-backend/pkg/resp"
)

type SystemSettingController struct {
	svc *service.SystemSettingsService
}

func NewSystemSettingController(svc *service.SystemSettingsService) *SystemSettingController {
	return &SystemSettingController{svc: svc}
}

// Get 读取系统设置。
func (ctl *SystemSettingController) Get(c *gin.Context) {
	settings, err := ctl.svc.Get(c.Request.Context())
	if err != nil {
		resp.Fail(c, 5000, "读取系统设置失败")
		return
	}
	resp.OK(c, settings)
}

// Update 保存系统设置。
func (ctl *SystemSettingController) Update(c *gin.Context) {
	var req service.SystemSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := ctl.svc.Update(c.Request.Context(), &req); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK[any](c, nil)
}
