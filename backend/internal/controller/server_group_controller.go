package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type ServerGroupController struct {
	svc *service.ServerGroupService
}

func NewServerGroupController(svc *service.ServerGroupService) *ServerGroupController {
	return &ServerGroupController{svc: svc}
}

// @Summary 服务器分组树
// @Description 返回大区/环境的分组结构
// @Tags 服务器分组接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} resp.Result{data=[]service.ServerGroupRegionItem} "查询成功"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /server-groups [get]
func (sgc *ServerGroupController) List(c *gin.Context) {
	data, err := sgc.svc.ListServerGroups(c.Request.Context())
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type serverGroupNameReq struct {
	Name string `json:"name"`
}

type CreateServerGroupItemResp struct {
	ID uint64 `json:"id"`
}

// @Summary 创建大区
// @Description 创建服务器分组大区，返回新建记录 ID
// @Tags 服务器分组接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body serverGroupNameReq true "名称"
// @Success 200 {object} resp.Result{data=CreateServerGroupItemResp} "创建成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 409 {object} resp.Result "资源冲突"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /server-groups/regions [post]
func (sgc *ServerGroupController) CreateRegion(c *gin.Context) {
	var req serverGroupNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := sgc.svc.CreateRegion(c.Request.Context(), req.Name)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// @Summary 更新大区
// @Description 根据大区 ID 更新名称
// @Tags 服务器分组接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "大区ID" example(1001)
// @Param body body serverGroupNameReq true "名称"
// @Success 200 {object} resp.Result "更新成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "资源不存在"
// @Failure 409 {object} resp.Result "资源冲突"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /server-groups/regions/{id} [patch]
func (sgc *ServerGroupController) PatchRegion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req serverGroupNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := sgc.svc.PatchRegion(c.Request.Context(), id, req.Name); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// @Summary 删除大区
// @Description 根据大区 ID 删除
// @Tags 服务器分组接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "大区ID" example(1001)
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "资源不存在"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /server-groups/regions/{id} [delete]
func (sgc *ServerGroupController) DeleteRegion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := sgc.svc.DeleteRegion(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// @Summary 创建环境
// @Description 在指定大区下创建环境，返回新建记录 ID
// @Tags 服务器分组接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "大区ID" example(1001)
// @Param body body serverGroupNameReq true "名称"
// @Success 200 {object} resp.Result{data=CreateServerGroupItemResp} "创建成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "资源不存在"
// @Failure 409 {object} resp.Result "资源冲突"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /server-groups/regions/{id}/envs [post]
func (sgc *ServerGroupController) CreateEnv(c *gin.Context) {
	regionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || regionID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req serverGroupNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	id, err := sgc.svc.CreateEnv(c.Request.Context(), regionID, req.Name)
	if err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// @Summary 更新环境
// @Description 根据环境 ID 更新名称
// @Tags 服务器分组接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "环境ID" example(1001)
// @Param body body serverGroupNameReq true "名称"
// @Success 200 {object} resp.Result "更新成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "资源不存在"
// @Failure 409 {object} resp.Result "资源冲突"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /server-groups/envs/{id} [patch]
func (sgc *ServerGroupController) PatchEnv(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	var req serverGroupNameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := sgc.svc.PatchEnv(c.Request.Context(), id, req.Name); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// @Summary 删除环境
// @Description 根据环境 ID 删除
// @Tags 服务器分组接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "环境ID" example(1001)
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "资源不存在"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /server-groups/envs/{id} [delete]
func (sgc *ServerGroupController) DeleteEnv(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := sgc.svc.DeleteEnv(c.Request.Context(), id); err != nil {
		WriteServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}
