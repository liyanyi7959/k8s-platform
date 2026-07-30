package http

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"k8s-platform-backend/internal/fleet/application"
	"k8s-platform-backend/internal/fleet/domain"
	"k8s-platform-backend/internal/fleet/ports"
	"k8s-platform-backend/pkg/resp"
)

type ClusterController struct {
	registry *application.Registry
	runtime  ports.ClusterRuntime
}

func NewClusterController(registry *application.Registry, runtime ports.ClusterRuntime) *ClusterController {
	return &ClusterController{registry: registry, runtime: runtime}
}

func (ctl *ClusterController) List(c *gin.Context) {
	data, err := ctl.registry.List(c.Request.Context(), application.ListRequest{Page: parseInt(c.Query("page"), 1), PageSize: parseInt(c.Query("page_size"), 10), Keyword: c.Query("keyword"), Status: c.Query("status"), Type: c.Query("type"), SortBy: c.Query("sort_by"), Order: c.Query("order")})
	if err != nil {
		writeClusterError(c, err)
		return
	}
	resp.OK(c, data)
}

type importRequest struct {
	Name        string `json:"name"`
	Kubeconfig  string `json:"kubeconfig"`
	Description string `json:"description"`
}

func (ctl *ClusterController) Import(c *gin.Context) {
	var request importRequest
	if err := bindImport(c, &request); err != nil {
		resp.Fail(c, 4000, err.Error())
		return
	}
	kubeconfig, err := ctl.runtime.NormalizeAndValidate(c.Request.Context(), request.Kubeconfig)
	if err != nil {
		writeClusterError(c, err)
		return
	}
	id, err := ctl.registry.Import(c.Request.Context(), request.Name, kubeconfig, request.Description)
	if err != nil {
		writeClusterError(c, err)
		return
	}
	resp.OK(c, gin.H{"cluster_id": id})
}

func (ctl *ClusterController) Get(c *gin.Context) {
	id, ok := clusterID(c)
	if !ok {
		return
	}
	data, err := ctl.registry.Get(c.Request.Context(), id)
	if err != nil {
		writeClusterError(c, err)
		return
	}
	resp.OK(c, data)
}

type healthResponse struct {
	APIOk        bool   `json:"api_ok"`
	NodeReady    int    `json:"node_ready"`
	NodeTotal    int    `json:"node_total"`
	CheckedAt    string `json:"checked_at"`
	Status       string `json:"status"`
	LastHealthAt string `json:"last_health_at,omitempty"`
}

func (ctl *ClusterController) CheckHealth(c *gin.Context) {
	id, ok := clusterID(c)
	if !ok {
		return
	}
	apiOK, ready, total, version, err := ctl.runtime.CheckHealth(c.Request.Context(), id)
	if err != nil {
		_ = ctl.registry.UpdateHealth(c.Request.Context(), id, false, 0, "")
		writeClusterError(c, err)
		return
	}
	_ = ctl.registry.UpdateHealth(c.Request.Context(), id, apiOK, total, version)
	now := time.Now().UTC().Format(time.RFC3339)
	status, last := "active", now
	if !apiOK {
		status = "degraded"
	}
	if detail, getErr := ctl.registry.Get(c.Request.Context(), id); getErr == nil {
		status = detail.Status
		if detail.LastHealthAt != "" {
			last = detail.LastHealthAt
		}
	}
	resp.OK(c, healthResponse{APIOk: apiOK, NodeReady: ready, NodeTotal: total, CheckedAt: now, Status: status, LastHealthAt: last})
}

func (ctl *ClusterController) Patch(c *gin.Context) {
	id, ok := clusterID(c)
	if !ok {
		return
	}
	var request application.PatchRequest
	if c.ShouldBindJSON(&request) != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if request.Kubeconfig != nil {
		normalized, err := ctl.runtime.NormalizeAndValidate(c.Request.Context(), *request.Kubeconfig)
		if err != nil {
			writeClusterError(c, err)
			return
		}
		request.Kubeconfig = &normalized
	}
	if err := ctl.registry.Patch(c.Request.Context(), id, request); err != nil {
		writeClusterError(c, err)
		return
	}
	if request.Kubeconfig != nil {
		ctl.runtime.StopCaches(id)
	}
	resp.OK(c, gin.H{})
}

func (ctl *ClusterController) Delete(c *gin.Context) {
	id, ok := clusterID(c)
	if !ok {
		return
	}
	if err := ctl.registry.Delete(c.Request.Context(), id); err != nil {
		writeClusterError(c, err)
		return
	}
	ctl.runtime.StopCaches(id)
	resp.OK(c, gin.H{})
}

const maxKubeconfigSize = 1024 * 1024

var allowedExtensions = map[string]struct{}{".yaml": {}, ".yml": {}, ".json": {}, ".txt": {}, ".kubeconfig": {}, ".config": {}}

func bindImport(c *gin.Context, request *importRequest) error {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		if c.ShouldBindJSON(request) != nil {
			return fmt.Errorf("请求参数格式错误")
		}
		return nil
	}
	request.Name, request.Description, request.Kubeconfig = c.PostForm("name"), c.PostForm("description"), c.PostForm("kubeconfig")
	file, err := c.FormFile("file")
	if err == nil {
		request.Kubeconfig, err = readKubeconfig(file)
		return err
	}
	if strings.TrimSpace(request.Kubeconfig) == "" {
		return fmt.Errorf("请上传或填写 kubeconfig")
	}
	return nil
}
func readKubeconfig(file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", fmt.Errorf("未上传 kubeconfig 文件")
	}
	if file.Size > maxKubeconfigSize {
		return "", fmt.Errorf("kubeconfig 文件不能超过 1MB")
	}
	if _, ok := allowedExtensions[strings.ToLower(filepath.Ext(file.Filename))]; !ok {
		return "", fmt.Errorf("不支持的 kubeconfig 文件类型")
	}
	source, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("读取 kubeconfig 文件失败")
	}
	defer source.Close()
	content, err := io.ReadAll(io.LimitReader(source, maxKubeconfigSize+1))
	if err != nil {
		return "", fmt.Errorf("读取 kubeconfig 文件失败")
	}
	if len(content) > maxKubeconfigSize {
		return "", fmt.Errorf("kubeconfig 文件不能超过 1MB")
	}
	return string(content), nil
}
func clusterID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return 0, false
	}
	return id, true
}
func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
func writeClusterError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		resp.Fail(c, 4000, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		resp.Fail(c, 4040, "集群不存在")
	case errors.Is(err, domain.ErrConflict):
		resp.Fail(c, 4090, "集群名称已存在")
	case errors.Is(err, domain.ErrCrypto):
		resp.Fail(c, 5000, "集群凭证处理失败")
	case errors.Is(err, domain.ErrRuntimeUnauthorized):
		resp.Fail(c, resp.CodeClusterCredentialInvalid, "凭据无效或已过期")
	case errors.Is(err, domain.ErrRuntimeForbidden):
		resp.Fail(c, 1003, "权限不足")
	case errors.Is(err, domain.ErrRuntimeNetwork):
		resp.Fail(c, 5000, "网络连接失败")
	case errors.Is(err, domain.ErrRuntimeTimeout):
		resp.Fail(c, 5000, "连接超时")
	case errors.Is(err, domain.ErrRuntimeTLS):
		resp.Fail(c, 5000, "证书校验失败")
	case errors.Is(err, domain.ErrRuntime):
		resp.Fail(c, 5000, "集群访问失败")
	default:
		resp.Fail(c, 5000, "内部错误")
	}
}
