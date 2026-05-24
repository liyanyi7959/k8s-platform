package controller

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

// ──────────────────────────────────────────────────────────
//  Pod 相关接口
// ──────────────────────────────────────────────────────────

// ListPods 获取 Pod 列表。
// 可选 query：namespace（为空表示 all namespaces）、sort_by、order。
// @Summary Pod 列表
// @Description 获取指定集群 Pod 列表（可按 namespace 过滤）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pods [get]
func (kc *K8sController) ListPods(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := strings.TrimSpace(c.Query("namespace"))
	ls := strings.TrimSpace(c.Query("label_selector"))
	var extra map[string]string
	if ls != "" {
		extra = map[string]string{"label_selector": ls}
	}
	list, err := kc.svc.List(c.Request.Context(), id, gvrPods(), ns, c.Query("sort_by"), c.Query("order"), extra)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// ListPodMetrics 获取 PodMetrics 列表。
// 可选 query：namespace（为空表示 all namespaces）、sort_by、order。
// @Summary PodMetrics 列表
// @Description 获取指定集群 PodMetrics 列表（可按 namespace 过滤）
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param namespace query string false "命名空间（不填表示全部）"
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=K8sListResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/podmetrics [get]
func (kc *K8sController) ListPodMetrics(c *gin.Context) {
	listNamespacedResource(c, kc, gvrPodMetrics())
}

// GetPodYAML 获取指定 Pod 的 YAML。
// @Summary 获取 Pod YAML
// @Description 获取指定 Pod 的 YAML 文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param pod path string true "Pod 名称"
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pods/{ns}/{pod}/yaml [get]
func (kc *K8sController) GetPodYAML(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("pod"))
	text, err := kc.svc.GetYAML(c.Request.Context(), id, gvrPods(), ns, name)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

// GetPodLogs 获取 Pod 日志。
// query：container（可选）、tail_lines（可选，默认 200；传 0 返回全量日志）、previous（可选，传 true 返回上一个实例日志）。
// @Summary 获取 Pod 日志
// @Description 获取指定 Pod 日志文本
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param pod path string true "Pod 名称"
// @Param container query string false "容器名（不填使用默认）"
// @Param tail_lines query int false "返回末尾行数，传 0 返回全量日志" example(200)
// @Param previous query bool false "是否返回上一个容器实例日志" example(false)
// @Success 200 {object} resp.Result{data=K8sTextResp} "查询成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pods/{ns}/{pod}/logs [get]
func (kc *K8sController) GetPodLogs(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("pod"))
	container := c.Query("container")
	tail := parseInt(c.Query("tail_lines"), 200)
	previous := strings.EqualFold(strings.TrimSpace(c.Query("previous")), "true") || strings.TrimSpace(c.Query("previous")) == "1"
	text, err := kc.svc.PodLogs(c.Request.Context(), id, ns, name, container, int64(tail), previous)
	if err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"text": text})
}

type createPodLogSessionReq struct {
	Container *string `json:"container"`
	Follow    *bool   `json:"follow"`
	TailLines *int64  `json:"tail_lines"`
}

type podLogWSFrame struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// CreatePodLogSession 创建一次性 Pod 日志 WebSocket 会话。
// @Summary 创建 Pod 日志会话
// @Description 创建一次性 Pod 日志会话，返回 session_id 与 ws_url
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param pod path string true "Pod 名称"
// @Param body body createPodLogSessionReq false "日志会话参数"
// @Success 200 {object} resp.Result{data=K8sLogSessionResp} "创建成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pods/{ns}/{pod}/logs/session [post]
func (kc *K8sController) CreatePodLogSession(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req createPodLogSessionReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	follow := true
	if req.Follow != nil {
		follow = *req.Follow
	}
	tailLines := int64(200)
	if req.TailLines != nil {
		tailLines = *req.TailLines
	}
	if tailLines < 0 {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	sessionID := kc.logSessions.NewSessionID()
	wsURL := "/api/v1/ws/pod-log?session_id=" + url.QueryEscape(sessionID)
	kc.logSessions.Put(sessionID, service.PodLogSession{
		ClusterID: id,
		Namespace: decodePathParam(c.Param("ns")),
		Pod:       decodePathParam(c.Param("pod")),
		Container: req.Container,
		Follow:    follow,
		TailLines: tailLines,
		CreatedAt: time.Now().UTC(),
	})
	resp.OK(c, gin.H{"session_id": sessionID, "ws_url": wsURL})
}

// PodLogWS 通过 WebSocket 推送 Pod 日志。
// @Summary Pod 日志 WebSocket
// @Description 通过 WebSocket 推送 Pod 日志（携带 session_id）
// @Tags WebSocket 接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param session_id query string true "日志会话ID"
// @Success 200 {object} resp.Result "升级 WebSocket"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 404 {object} resp.Result "会话不存在"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /ws/pod-log [get]
func (kc *K8sController) PodLogWS(c *gin.Context) {
	sid := strings.TrimSpace(c.Query("session_id"))
	if sid == "" {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	sess, ok := kc.logSessions.Take(sid)
	if !ok {
		resp.Fail(c, 4040, "not found")
		return
	}

	up := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				return true
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			host := strings.TrimSpace(r.Host)
			if xf := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0]); xf != "" {
				host = xf
			}
			if strings.EqualFold(u.Host, host) {
				return true
			}
			h, err := url.Parse("http://" + host)
			if err == nil && strings.EqualFold(u.Hostname(), h.Hostname()) {
				return true
			}
			return false
		},
	}
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	var wmu sync.Mutex
	writeFrame := func(frame podLogWSFrame) error {
		wmu.Lock()
		defer wmu.Unlock()
		return conn.WriteJSON(frame)
	}

	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	stream, err := kc.svc.PodLogStream(ctx, sess.ClusterID, sess.Namespace, sess.Pod, derefString(sess.Container), sess.Follow, sess.TailLines, false)
	if err != nil {
		_ = writeFrame(podLogWSFrame{Type: "error", Message: err.Error()})
		return
	}
	defer func() { _ = stream.Close() }()

	buf := make([]byte, 4096)
	for {
		n, readErr := stream.Read(buf)
		if n > 0 {
			if err := writeFrame(podLogWSFrame{Type: "chunk", Data: string(buf[:n])}); err != nil {
				return
			}
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			_ = writeFrame(podLogWSFrame{Type: "eof"})
			return
		}
		if errors.Is(readErr, context.Canceled) {
			return
		}
		_ = writeFrame(podLogWSFrame{Type: "error", Message: readErr.Error()})
		return
	}
}

// DeletePod 删除指定 Pod。
// @Summary 删除 Pod
// @Description 删除指定 Pod
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param pod path string true "Pod 名称"
// @Param force query bool false "是否强制删除（gracePeriodSeconds=0）"
// @Success 200 {object} resp.Result "删除成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pods/{ns}/{pod} [delete]
func (kc *K8sController) DeletePod(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	ns := decodePathParam(c.Param("ns"))
	name := decodePathParam(c.Param("pod"))
	force := strings.EqualFold(strings.TrimSpace(c.Query("force")), "true") || strings.TrimSpace(c.Query("force")) == "1"
	if err := kc.svc.DeletePod(c.Request.Context(), id, ns, name, force); err != nil {
		kc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

type createPodExecSessionReq struct {
	Container *string  `json:"container"`
	Command   []string `json:"command"`
	TTY       *bool    `json:"tty"`
}

// CreatePodExecSession 创建一次性 Pod Exec 会话。
// 流程：
// 1) 生成 session_id，并将 exec 目标信息暂存于内存（带 TTL）；
// 2) 返回前端用于建立 WebSocket 的 ws_url；
// 3) 前端随后以 /ws/pod-exec?session_id=... 建立 WS，服务端会"取走并删除"该会话，避免复用。
// @Summary 创建 Pod Exec 会话
// @Description 创建一次性 exec 会话，返回 session_id 与 ws_url
// @Tags K8s 资源接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "集群ID" example(1001)
// @Param ns path string true "命名空间"
// @Param pod path string true "Pod 名称"
// @Param body body createPodExecSessionReq true "Exec 会话参数"
// @Success 200 {object} resp.Result{data=K8sExecSessionResp} "创建成功"
// @Failure 400 {object} resp.Result "参数错误"
// @Failure 500 {object} resp.Result "内部错误"
// @Router /clusters/{id}/pods/{ns}/{pod}/exec [post]
func (kc *K8sController) CreatePodExecSession(c *gin.Context) {
	id, ok := parseClusterID(c)
	if !ok {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	var req createPodExecSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "invalid params")
		return
	}
	sessionID := kc.execSessions.NewSessionID()
	wsURL := "/api/v1/ws/pod-exec?session_id=" + url.QueryEscape(sessionID)
	kc.execSessions.Put(sessionID, service.ExecSession{
		ClusterID: id,
		Namespace: decodePathParam(c.Param("ns")),
		Pod:       decodePathParam(c.Param("pod")),
		Container: req.Container,
		Command:   req.Command,
		TTY:       req.TTY,
		CreatedAt: time.Now().UTC(),
	})
	resp.OK(c, gin.H{"session_id": sessionID, "ws_url": wsURL})
}
