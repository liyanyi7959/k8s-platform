package controller

import (
	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"

	"github.com/gin-gonic/gin"
)

type AnsibleController struct {
	svc *service.AnsibleService
}

func NewAnsibleController(svc *service.AnsibleService) *AnsibleController {
	return &AnsibleController{svc: svc}
}

type RunPlaybookRequest struct {
	ServerIDs []uint64 `json:"server_ids" binding:"required,min=1"`
	Playbook  string   `json:"playbook" binding:"required"`
}

// RunPlaybook 执行 Ansible Playbook
// @Summary 执行 Ansible Playbook
// @Description 在指定服务器上执行 Ansible Playbook
// @Tags 运维操作
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body RunPlaybookRequest true "执行参数"
// @Success 200 {object} resp.Result{data=map[string]int64} "任务ID"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /ops/playbook [post]
func (c *AnsibleController) RunPlaybook(ctx *gin.Context) {
	var req RunPlaybookRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp.Fail(ctx, 4000, "参数无效: "+err.Error())
		return
	}

	userID := ctx.GetInt64("user_id") // Assuming auth middleware sets this

	taskID, err := c.svc.RunPlaybook(ctx.Request.Context(), req.ServerIDs, req.Playbook, uint64(userID))
	if err != nil {
		WriteServiceErr(ctx, err)
		return
	}

	resp.OK(ctx, gin.H{"task_id": taskID})
}
