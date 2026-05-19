package controller

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// ServerFileController 服务器 SFTP 文件管理 HTTP 处理器。
type ServerFileController struct {
	svc *service.ServerSFTPService
}

// NewServerFileController 创建实例。
func NewServerFileController(svc *service.ServerSFTPService) *ServerFileController {
	return &ServerFileController{svc: svc}
}

// ── 错误辅助 ──

func (fc *ServerFileController) writeErr(c *gin.Context, err error) {
	WriteServiceErr(c, err, SSHErrMappings...)
}

func (fc *ServerFileController) parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp.Fail(c, 4000, "无效的服务器 ID")
		return 0, false
	}
	return id, true
}

// ── 路由处理器 ──

// ListDir 列出远程目录内容。
// @Summary 列出远程目录
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Param path query string false "远程路径（默认 home 目录）"
// @Success 200 {object} resp.Result{data=[]service.FileEntry}
// @Router /servers/{id}/files [get]
func (fc *ServerFileController) ListDir(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	dir := strings.TrimSpace(c.Query("path"))
	if dir == "" {
		// 获取 home 目录
		home, err := fc.svc.GetHomeDir(c.Request.Context(), id)
		if err != nil {
			fc.writeErr(c, err)
			return
		}
		dir = home
	}
	entries, err := fc.svc.ListDir(c.Request.Context(), id, dir)
	if err != nil {
		fc.writeErr(c, err)
		return
	}
	resp.OK(c, entries)
}

// ReadFile 读取远程文本文件内容。
// @Summary 读取远程文件
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Param path query string true "文件路径"
// @Success 200 {object} resp.Result{data=service.FileContent}
// @Router /servers/{id}/files/read [get]
func (fc *ServerFileController) ReadFile(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	filePath := strings.TrimSpace(c.Query("path"))
	if filePath == "" {
		resp.Fail(c, 4000, "缺少 path 参数")
		return
	}
	content, err := fc.svc.ReadFile(c.Request.Context(), id, filePath)
	if err != nil {
		fc.writeErr(c, err)
		return
	}
	resp.OK(c, content)
}

// WriteFile 写入远程文件。
// @Summary 写入远程文件
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Success 200 {object} resp.Result
// @Router /servers/{id}/files/write [put]
func (fc *ServerFileController) WriteFile(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := fc.svc.WriteFile(c.Request.Context(), id, req.Path, req.Content); err != nil {
		fc.writeErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// Upload 上传文件到远程服务器。
// @Summary 上传文件
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Param path formData string true "远程目标路径"
// @Param file formData file true "上传的文件"
// @Success 200 {object} resp.Result
// @Router /servers/{id}/files/upload [post]
func (fc *ServerFileController) Upload(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	remotePath := strings.TrimSpace(c.PostForm("path"))
	if remotePath == "" {
		resp.Fail(c, 4000, "缺少 path 参数")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		resp.Fail(c, 4000, "缺少上传文件")
		return
	}
	defer func() { _ = file.Close() }()

	// 如果 remotePath 以 / 结尾，则拼接文件名
	if strings.HasSuffix(remotePath, "/") {
		remotePath += header.Filename
	}

	if err := fc.svc.Upload(c.Request.Context(), id, remotePath, file, header.Size); err != nil {
		fc.writeErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// Download 下载远程文件。
// @Summary 下载文件
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Param path query string true "文件路径"
// @Router /servers/{id}/files/download [get]
func (fc *ServerFileController) Download(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	filePath := strings.TrimSpace(c.Query("path"))
	if filePath == "" {
		resp.Fail(c, 4000, "缺少 path 参数")
		return
	}
	reader, size, filename, sftpClient, sshClient, releaseSlot, err := fc.svc.Download(c.Request.Context(), id, filePath)
	if err != nil {
		fc.writeErr(c, err)
		return
	}
	defer func() {
		_ = reader.Close()
		_ = sftpClient.Close()
		_ = sshClient.Close()
		if releaseSlot != nil {
			releaseSlot()
		}
	}()

	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", "application/octet-stream")
	if size > 0 {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, reader)
}

// Mkdir 创建远程目录。
// @Summary 创建目录
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Success 200 {object} resp.Result
// @Router /servers/{id}/files/mkdir [post]
func (fc *ServerFileController) Mkdir(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := fc.svc.Mkdir(c.Request.Context(), id, req.Path); err != nil {
		fc.writeErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// Rename 重命名/移动远程文件。
// @Summary 重命名文件
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Success 200 {object} resp.Result
// @Router /servers/{id}/files/rename [post]
func (fc *ServerFileController) Rename(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	var req struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := fc.svc.Rename(c.Request.Context(), id, req.OldPath, req.NewPath); err != nil {
		fc.writeErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// Remove 删除远程文件或目录。
// @Summary 删除文件
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Param path query string true "文件路径"
// @Success 200 {object} resp.Result
// @Router /servers/{id}/files [delete]
func (fc *ServerFileController) Remove(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	filePath := strings.TrimSpace(c.Query("path"))
	if filePath == "" {
		resp.Fail(c, 4000, "缺少 path 参数")
		return
	}
	if err := fc.svc.Remove(c.Request.Context(), id, filePath); err != nil {
		fc.writeErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// Stat 获取文件信息。
// @Summary 文件信息
// @Tags 服务器文件管理
// @Security BearerAuth
// @Param id path int true "服务器ID"
// @Param path query string true "文件路径"
// @Success 200 {object} resp.Result{data=service.FileEntry}
// @Router /servers/{id}/files/stat [get]
func (fc *ServerFileController) Stat(c *gin.Context) {
	id, ok := fc.parseID(c)
	if !ok {
		return
	}
	filePath := strings.TrimSpace(c.Query("path"))
	if filePath == "" {
		resp.Fail(c, 4000, "缺少 path 参数")
		return
	}
	entry, err := fc.svc.Stat(c.Request.Context(), id, filePath)
	if err != nil {
		fc.writeErr(c, err)
		return
	}
	resp.OK(c, entry)
}
