package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"k8s-platform-backend/internal/service"
	"k8s-platform-backend/pkg/resp"
)

type ServerController struct {
	// svc 为服务器管理的业务逻辑层（CRUD、加密存储、软删除等）。
	svc *service.ServerService
}

func NewServerController(svc *service.ServerService) *ServerController {
	// NewServerController 负责创建 ServerController。
	// controller 层职责：
	// - 解析/校验 HTTP 参数
	// - 组装 service 请求结构
	// - 将 service 错误映射为统一的业务错误码
	return &ServerController{svc: svc}
}

type ServerListPage struct {
	List     []service.ServerItem `json:"list"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

// @Summary 服务器列表
// @Description 分页列出服务器，支持 keyword/tag/status 过滤与排序
// @Tags 服务器管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param page query int false "页码" example(1)
// @Param page_size query int false "每页条数" example(10)
// @Param keyword query string false "关键字（name/ip 模糊搜索）"
// @Param tag query string false "标签过滤"
// @Param status query string false "状态过滤" example(active)
// @Param sort_by query string false "排序字段" example(created_at)
// @Param order query string false "排序方向" example(desc)
// @Success 200 {object} resp.Result{data=ServerListPage} "查询成功"
// @Failure 200 {object} resp.Result "查询失败"
// @Router /servers [get]
func (sc *ServerController) List(c *gin.Context) {
	// List 分页查询服务器列表。
	// Query：
	// - page/page_size：分页
	// - keyword：按 name/ip 模糊搜索
	// - status：active/disabled
	// - tag：按 tags 中包含某个标签过滤
	// - sort_by=created_at&order=asc|desc：创建时间排序（默认 id desc）
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 10)
	data, err := sc.svc.ListServers(c.Request.Context(), service.ListServersRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
		Tag:      c.Query("tag"),
		Status:   c.Query("status"),
		SortBy:   c.Query("sort_by"),
		Order:    c.Query("order"),
	})
	if err != nil {
		resp.Fail(c, 5000, "内部错误")
		return
	}
	resp.OK(c, data)
}

// @Summary 服务器详情
// @Description 根据服务器 ID 查询详情
// @Tags 服务器管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "服务器ID" example(1001)
// @Success 200 {object} resp.Result{data=service.ServerDetail} "查询成功"
// @Failure 200 {object} resp.Result "查询失败"
// @Router /servers/{id} [get]
func (sc *ServerController) Get(c *gin.Context) {
	// Get 获取单个服务器详情。
	// Path：/servers/:id
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := sc.svc.GetServer(c.Request.Context(), id)
	if err != nil {
		sc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

// @Summary 检查服务器 SSH
// @Description 对指定服务器执行 SSH 连通性检查
// @Tags 服务器管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "服务器ID" example(1001)
// @Success 200 {object} resp.Result{data=service.CheckSSHResult} "检查完成"
// @Failure 200 {object} resp.Result "检查失败"
// @Router /servers/{id}/check-ssh [post]
func (sc *ServerController) CheckSSH(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	data, err := sc.svc.CheckSSH(c.Request.Context(), id)
	if err != nil {
		sc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, data)
}

type createServerReq struct {
	// createServerReq 为创建服务器的请求体。
	// 说明：password/private_key 为明文输入，仅用于传输；后端入库会进行加密存储。
	Name       string   `json:"name"`
	IP         string   `json:"ip"`
	Port       int      `json:"port"`
	AuthType   string   `json:"auth_type"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	PrivateKey string   `json:"private_key"`
	Tags       []string `json:"tags"`
	Status     string   `json:"status"`
}

type CreateServerResp struct {
	ServerID uint64 `json:"server_id"`
}

// @Summary 创建服务器
// @Description 创建服务器资产（敏感认证信息入库前会加密）
// @Tags 服务器管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createServerReq true "创建参数"
// @Success 200 {object} resp.Result{data=CreateServerResp} "创建成功"
// @Failure 200 {object} resp.Result "创建失败"
// @Router /servers [post]
func (sc *ServerController) Create(c *gin.Context) {
	// Create 创建服务器。
	// Body：createServerReq
	// 返回：{ "server_id": <id> }
	var req createServerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	// created_by 从鉴权中间件写入的 Context 获取。
	// 兼容：若未取到 user_id，则回退为 1（便于本地/开发场景）。
	userID := int64(1)
	if v, ok := c.Get("user_id"); ok {
		if id, ok2 := v.(int64); ok2 && id > 0 {
			userID = id
		}
	}

	id, err := sc.svc.CreateServer(c.Request.Context(), service.CreateServerRequest{
		Name:       req.Name,
		IP:         req.IP,
		Port:       req.Port,
		AuthType:   req.AuthType,
		Username:   req.Username,
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Tags:       req.Tags,
		Status:     req.Status,
		CreatedBy:  uint64(userID),
	})
	if err != nil {
		sc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{"server_id": id})
}

// @Summary 更新服务器
// @Description 根据服务器 ID 进行部分字段更新
// @Tags 服务器管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "服务器ID" example(1001)
// @Success 200 {object} resp.Result "更新成功"
// @Failure 200 {object} resp.Result "更新失败"
// @Router /servers/{id} [patch]
func (sc *ServerController) Patch(c *gin.Context) {
	// Patch 部分更新服务器。
	// Path：/servers/:id
	// Body：一个 JSON 对象，字段按需提供。
	// 约定：
	// - password/private_key 字段为三态：
	//   - 不传：不修改
	//   - 传字符串：更新为新值（入库前加密）
	//   - 传 null：清空对应密钥/密码
	// - tags 传 null 视为清空标签（等价于 []）
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		resp.Fail(c, 4000, "参数错误")
		return
	}

	var req service.PatchServerRequest
	if b, ok := raw["name"]; ok {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		req.Name = &v
	}
	if b, ok := raw["ip"]; ok {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		req.IP = &v
	}
	if b, ok := raw["port"]; ok {
		var v int
		if err := json.Unmarshal(b, &v); err != nil {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		req.Port = &v
	}
	if b, ok := raw["auth_type"]; ok {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		req.AuthType = &v
	}
	if b, ok := raw["username"]; ok {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		req.Username = &v
	}
	if b, ok := raw["status"]; ok {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		req.Status = &v
	}
	if b, ok := raw["password"]; ok {
		if string(b) == "null" {
			var p *string
			req.Password = &p
		} else {
			var v string
			if err := json.Unmarshal(b, &v); err != nil {
				resp.Fail(c, 4000, "参数错误")
				return
			}
			p := &v
			req.Password = &p
		}
	}
	if b, ok := raw["private_key"]; ok {
		if string(b) == "null" {
			var p *string
			req.PrivateKey = &p
		} else {
			var v string
			if err := json.Unmarshal(b, &v); err != nil {
				resp.Fail(c, 4000, "参数错误")
				return
			}
			p := &v
			req.PrivateKey = &p
		}
	}
	if b, ok := raw["tags"]; ok {
		if string(b) == "null" {
			empty := []string{}
			req.Tags = &empty
		} else {
			var v []string
			if err := json.Unmarshal(b, &v); err != nil {
				resp.Fail(c, 4000, "参数错误")
				return
			}
			req.Tags = &v
		}
	}

	if err := sc.svc.PatchServer(c.Request.Context(), id, req); err != nil {
		sc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// @Summary 删除服务器
// @Description 根据服务器 ID 软删除
// @Tags 服务器管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "服务器ID" example(1001)
// @Success 200 {object} resp.Result "删除成功"
// @Failure 200 {object} resp.Result "删除失败"
// @Router /servers/{id} [delete]
func (sc *ServerController) Delete(c *gin.Context) {
	// Delete 删除服务器（软删除）。
	// Path：/servers/:id
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	if err := sc.svc.DeleteServer(c.Request.Context(), id); err != nil {
		sc.writeServiceErr(c, err)
		return
	}
	resp.OK(c, gin.H{})
}

// @Summary SSH WebSocket
// @Description 建立 SSH 终端 WebSocket 连接（需要 server_id、cols、rows）
// @Tags 服务器管理接口
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param server_id query int true "服务器ID" example(1001)
// @Param cols query int false "终端列数" example(120)
// @Param rows query int false "终端行数" example(36)
// @Success 200 {object} resp.Result "升级 WebSocket"
// @Router /ws/ssh [get]
func (sc *ServerController) SSHWS(c *gin.Context) {
	serverID, err := strconv.ParseUint(strings.TrimSpace(c.Query("server_id")), 10, 64)
	if err != nil || serverID == 0 {
		resp.Fail(c, 4000, "参数错误")
		return
	}
	cols := parseInt(c.Query("cols"), 120)
	rows := parseInt(c.Query("rows"), 36)
	if cols < 20 {
		cols = 20
	}
	if rows < 5 {
		rows = 5
	}

	info, err := sc.svc.GetServerSSHAuth(c.Request.Context(), serverID)
	if err != nil {
		sc.writeServiceErr(c, err)
		return
	}
	if strings.TrimSpace(info.Username) == "" || strings.TrimSpace(info.Addr) == "" {
		resp.Fail(c, 5000, "内部错误")
		return
	}

	var authMethod ssh.AuthMethod
	switch info.AuthType {
	case "password":
		if strings.TrimSpace(info.Password) == "" {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		authMethod = ssh.Password(info.Password)
	case "key":
		if strings.TrimSpace(info.PrivateKey) == "" {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		signer, err := ssh.ParsePrivateKey([]byte(info.PrivateKey))
		if err != nil {
			resp.Fail(c, 4000, "参数错误")
			return
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		resp.Fail(c, 4000, "参数错误")
		return
	}

	sshCfg := &ssh.ClientConfig{
		User:            info.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	sshClient, err := ssh.Dial("tcp", info.Addr, sshCfg)
	if err != nil {
		sc.writeServiceErr(c, service.ErrWithMessage(service.ErrSSHNetwork, "SSH 连接失败"))
		return
	}
	defer func() { _ = sshClient.Close() }()

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

	sess, err := sshClient.NewSession()
	if err != nil {
		return
	}
	defer func() { _ = sess.Close() }()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		return
	}

	_ = sess.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{})
	if err := sess.Shell(); err != nil {
		return
	}

	const writeWait = 10 * time.Second
	const pongWait = 25 * time.Second
	const pingPeriod = 20 * time.Second

	var writeMu sync.Mutex
	writeWS := func(payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		return conn.WriteMessage(websocket.BinaryMessage, payload)
	}

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() { close(done) })
	}
	defer closeDone()

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				writeMu.Lock()
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				writeMu.Unlock()
				if err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()

	var wg sync.WaitGroup
	pump := func(r io.Reader) {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				if err2 := writeWS(buf[:n]); err2 != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}

	wg.Add(2)
	go pump(stdout)
	go pump(stderr)

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
			continue
		}
		if mt == websocket.TextMessage && len(msg) > 0 && msg[0] == '{' {
			var ctl struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if err := json.Unmarshal(msg, &ctl); err == nil && ctl.Type == "resize" {
				if ctl.Cols > 0 && ctl.Rows > 0 {
					_ = sess.WindowChange(ctl.Rows, ctl.Cols)
				}
				continue
			}
		}
		if len(msg) == 0 {
			continue
		}
		if _, err := stdin.Write(msg); err != nil {
			break
		}
	}

	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(writeWait))
	closeDone()
	_ = stdin.Close()
	_ = sess.Close()
	_ = sshClient.Close()
	wg.Wait()
}

// writeServiceErr 委托给共享 WriteServiceErr，追加 SSH 领域特定映射。
func (sc *ServerController) writeServiceErr(c *gin.Context, err error) {
	WriteServiceErr(c, err, SSHErrMappings...)
}
