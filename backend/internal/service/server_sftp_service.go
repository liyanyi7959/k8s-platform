// server_sftp_service.go 提供基于 SSH/SFTP 的远程文件管理能力。
//
// 设计要点：
// - 复用 ServerService.GetServerSSHAuth 获取解密凭据并建立 SSH 连接
// - 使用 github.com/pkg/sftp 封装 SFTP 客户端
// - 所有路径均做安全校验（禁止路径穿越、限制文件大小）
// - 文件读取限制 2MB，上传限制 100MB
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ──────────────────────────────────────────────────────────
//  Types
// ──────────────────────────────────────────────────────────

// FileEntry 表示远程文件系统中的一个条目。
type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`     // e.g. "-rwxr-xr-x"
	ModTime string `json:"mod_time"` // RFC3339
	Owner   string `json:"owner,omitempty"`
	User    string `json:"user,omitempty"`
	Group   string `json:"group,omitempty"`
}

// FileContent 表示读取到的文件内容（仅文本文件）。
type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

const (
	maxReadSize        = 2 * 1024 * 1024   // 2MB 文本读取上限
	maxUploadSize      = 100 * 1024 * 1024 // 100MB 上传上限
	defaultDialTimeout = 15 * time.Second
	listOpTimeout      = 20 * time.Second
	writeOpTimeout     = 45 * time.Second
	transferOpTimeout  = 5 * time.Minute
	globalSFTPLimit    = 24
	perServerSFTPLimit = 4
)

// ──────────────────────────────────────────────────────────
//  Service
// ──────────────────────────────────────────────────────────

// ServerSFTPService 封装 SFTP 文件管理操作。
type ServerSFTPService struct {
	serverSvc *ServerService
	globalSem chan struct{}
	mu        sync.Mutex
	serverSem map[uint64]chan struct{}
}

// NewServerSFTPService 创建 SFTP 服务。
func NewServerSFTPService(serverSvc *ServerService) *ServerSFTPService {
	return &ServerSFTPService{
		serverSvc: serverSvc,
		globalSem: make(chan struct{}, globalSFTPLimit),
		serverSem: make(map[uint64]chan struct{}),
	}
}

func (s *ServerSFTPService) getServerSemaphore(serverID uint64) chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sem, ok := s.serverSem[serverID]; ok {
		return sem
	}
	sem := make(chan struct{}, perServerSFTPLimit)
	s.serverSem[serverID] = sem
	return sem
}

func acquireSemaphore(ctx context.Context, sem chan struct{}) error {
	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ErrWithMessage(ErrSSHNetwork, "SFTP 会话繁忙，请稍后重试")
	}
}

func releaseSemaphore(sem chan struct{}) {
	select {
	case <-sem:
	default:
	}
}

func (s *ServerSFTPService) acquireSlot(ctx context.Context, serverID uint64) (func(), error) {
	if err := acquireSemaphore(ctx, s.globalSem); err != nil {
		return nil, err
	}
	serverSem := s.getServerSemaphore(serverID)
	if err := acquireSemaphore(ctx, serverSem); err != nil {
		releaseSemaphore(s.globalSem)
		return nil, err
	}
	return func() {
		releaseSemaphore(serverSem)
		releaseSemaphore(s.globalSem)
	}, nil
}

func resolveDialTimeout(ctx context.Context, fallback time.Duration) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fallback
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return time.Second
	}
	if remaining < fallback {
		return remaining
	}
	return fallback
}

// dialSFTP 建立 SSH+SFTP 连接。调用方需负责关闭返回的 client 和 sshClient。
func (s *ServerSFTPService) dialSFTP(ctx context.Context, serverID uint64) (*sftp.Client, *ssh.Client, error) {
	info, err := s.serverSvc.GetServerSSHAuth(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}

	var authMethod ssh.AuthMethod
	switch info.AuthType {
	case "password":
		authMethod = ssh.Password(info.Password)
	case "key":
		signer, err := ssh.ParsePrivateKey([]byte(info.PrivateKey))
		if err != nil {
			return nil, nil, ErrWithMessage(ErrInvalidParams, "私钥格式错误")
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		return nil, nil, ErrInvalidParams
	}

	sshCfg := &ssh.ClientConfig{
		User:            info.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         resolveDialTimeout(ctx, defaultDialTimeout),
	}
	sshClient, err := ssh.Dial("tcp", info.Addr, sshCfg)
	if err != nil {
		return nil, nil, normalizeSSHErr(err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, ErrWithMessage(ErrSSHNetwork, "SFTP 连接失败")
	}

	return sftpClient, sshClient, nil
}

// safePath 清洗路径，防止路径穿越。
func safePath(p string) string {
	cleaned := path.Clean("/" + p)
	if cleaned == "." {
		return "/"
	}
	return cleaned
}

// ──────────────────────────────────────────────────────────
//  ListDir
// ──────────────────────────────────────────────────────────

// ListDir 列出远程目录内容。
func (s *ServerSFTPService) ListDir(ctx context.Context, serverID uint64, dirPath string) ([]FileEntry, error) {
	opCtx, cancel := context.WithTimeout(ctx, listOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return nil, err
	}
	defer release()
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()

	cleanPath := safePath(dirPath)
	entries, err := sftpClient.ReadDir(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrWithMessage(ErrNotFound, "目录不存在: "+cleanPath)
		}
		if os.IsPermission(err) {
			return nil, ErrWithMessage(ErrInvalidParams, "无权限读取: "+cleanPath)
		}
		return nil, ErrWithMessage(ErrSSHNetwork, "读取目录失败")
	}

	type ownerGroupNamer interface {
		Owner() string
		Group() string
	}

	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		user := ""
		group := ""
		if og, ok := any(e).(ownerGroupNamer); ok {
			user = strings.TrimSpace(og.Owner())
			group = strings.TrimSpace(og.Group())
		}
		owner := strings.TrimSpace(strings.Trim(strings.Join([]string{user, group}, ":"), ":"))
		result = append(result, FileEntry{
			Name:    e.Name(),
			Path:    path.Join(cleanPath, e.Name()),
			IsDir:   e.IsDir(),
			Size:    e.Size(),
			Mode:    e.Mode().String(),
			ModTime: e.ModTime().UTC().Format(time.RFC3339),
			Owner:   owner,
			User:    user,
			Group:   group,
		})
	}

	// 目录在前，文件在后；各自按名称排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

// ──────────────────────────────────────────────────────────
//  ReadFile
// ──────────────────────────────────────────────────────────

// ReadFile 读取远程文本文件内容（限制 2MB）。
func (s *ServerSFTPService) ReadFile(ctx context.Context, serverID uint64, filePath string) (FileContent, error) {
	opCtx, cancel := context.WithTimeout(ctx, listOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return FileContent{}, err
	}
	defer release()
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		return FileContent{}, err
	}
	defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()

	cleanPath := safePath(filePath)
	info, err := sftpClient.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileContent{}, ErrWithMessage(ErrNotFound, "文件不存在: "+cleanPath)
		}
		return FileContent{}, ErrWithMessage(ErrSSHNetwork, "获取文件信息失败")
	}
	if info.IsDir() {
		return FileContent{}, ErrWithMessage(ErrInvalidParams, "目标是目录，无法读取")
	}
	if info.Size() > maxReadSize {
		return FileContent{}, ErrWithMessage(ErrInvalidParams, fmt.Sprintf("文件过大（%d 字节），超过 2MB 限制", info.Size()))
	}

	f, err := sftpClient.Open(cleanPath)
	if err != nil {
		return FileContent{}, ErrWithMessage(ErrSSHNetwork, "打开文件失败")
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, maxReadSize+1))
	if err != nil {
		return FileContent{}, ErrWithMessage(ErrSSHNetwork, "读取文件失败")
	}

	return FileContent{
		Path:    cleanPath,
		Content: string(data),
		Size:    info.Size(),
	}, nil
}

// ──────────────────────────────────────────────────────────
//  WriteFile
// ──────────────────────────────────────────────────────────

// WriteFile 写入远程文本文件（覆盖）。
func (s *ServerSFTPService) WriteFile(ctx context.Context, serverID uint64, filePath, content string) error {
	opCtx, cancel := context.WithTimeout(ctx, writeOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return err
	}
	defer release()
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()

	cleanPath := safePath(filePath)
	f, err := sftpClient.Create(cleanPath)
	if err != nil {
		if os.IsPermission(err) {
			return ErrWithMessage(ErrInvalidParams, "无权限写入: "+cleanPath)
		}
		return ErrWithMessage(ErrSSHNetwork, "创建文件失败")
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write([]byte(content)); err != nil {
		return ErrWithMessage(ErrSSHNetwork, "写入文件失败")
	}
	return nil
}

// ──────────────────────────────────────────────────────────
//  Upload / Download
// ──────────────────────────────────────────────────────────

// Upload 上传文件到远程路径。reader 由调用方提供，size 用于预检。
func (s *ServerSFTPService) Upload(ctx context.Context, serverID uint64, remotePath string, reader io.Reader, size int64) error {
	if size > maxUploadSize {
		return ErrWithMessage(ErrInvalidParams, "文件超过 100MB 上传限制")
	}

	opCtx, cancel := context.WithTimeout(ctx, transferOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return err
	}
	defer release()
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()

	cleanPath := safePath(remotePath)
	dirPath := path.Dir(cleanPath)
	if dirPath == "." || dirPath == "" {
		dirPath = "/"
	}
	if _, err := sftpClient.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			return ErrWithMessage(ErrNotFound, "目标目录不存在: "+dirPath)
		}
		return ErrWithMessage(ErrSSHNetwork, "校验目标目录失败")
	}
	tempPath := path.Join(dirPath, fmt.Sprintf(".%s.uploading.%d", path.Base(cleanPath), time.Now().UnixNano()))
	f, err := sftpClient.Create(tempPath)
	if err != nil {
		if os.IsPermission(err) {
			return ErrWithMessage(ErrInvalidParams, "无权限写入: "+cleanPath)
		}
		return ErrWithMessage(ErrSSHNetwork, "创建远程文件失败")
	}
	defer func() { _ = f.Close() }()
	cleanupTemp := func() {
		_ = f.Close()
		_ = sftpClient.Remove(tempPath)
	}

	if _, err := io.Copy(f, reader); err != nil {
		cleanupTemp()
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return ErrWithMessage(ErrSSHNetwork, "上传已中断，未完成文件已清理")
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ErrWithMessage(ErrSSHTimeout, "上传超时，未完成文件已清理")
		}
		return ErrWithMessage(ErrSSHNetwork, "上传写入失败，未完成文件已清理")
	}
	if err := f.Close(); err != nil {
		_ = sftpClient.Remove(tempPath)
		return ErrWithMessage(ErrSSHNetwork, "上传收尾失败，未完成文件已清理")
	}
	if _, err := sftpClient.Stat(cleanPath); err == nil {
		if err := sftpClient.Remove(cleanPath); err != nil {
			_ = sftpClient.Remove(tempPath)
			return ErrWithMessage(ErrSSHNetwork, "替换已存在文件失败")
		}
	}
	if err := sftpClient.Rename(tempPath, cleanPath); err != nil {
		_ = sftpClient.Remove(tempPath)
		return ErrWithMessage(ErrSSHNetwork, "上传完成但落盘失败")
	}
	return nil
}

// Download 下载远程文件。返回 ReadCloser 供 controller 流式写入。
// 调用方必须关闭返回的三个资源。
func (s *ServerSFTPService) Download(ctx context.Context, serverID uint64, filePath string) (io.ReadCloser, int64, string, *sftp.Client, *ssh.Client, func(), error) {
	opCtx, cancel := context.WithTimeout(ctx, transferOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return nil, 0, "", nil, nil, nil, err
	}
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		release()
		return nil, 0, "", nil, nil, nil, err
	}

	cleanPath := safePath(filePath)
	info, err := sftpClient.Stat(cleanPath)
	if err != nil {
		_ = sftpClient.Close()
		_ = sshClient.Close()
		release()
		if os.IsNotExist(err) {
			return nil, 0, "", nil, nil, nil, ErrWithMessage(ErrNotFound, "文件不存在")
		}
		return nil, 0, "", nil, nil, nil, ErrWithMessage(ErrSSHNetwork, "获取文件信息失败")
	}
	if info.IsDir() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
		release()
		return nil, 0, "", nil, nil, nil, ErrWithMessage(ErrInvalidParams, "目标是目录，无法下载")
	}

	f, err := sftpClient.Open(cleanPath)
	if err != nil {
		_ = sftpClient.Close()
		_ = sshClient.Close()
		release()
		return nil, 0, "", nil, nil, nil, ErrWithMessage(ErrSSHNetwork, "打开远程文件失败")
	}

	return f, info.Size(), path.Base(cleanPath), sftpClient, sshClient, release, nil
}

// ──────────────────────────────────────────────────────────
//  Mkdir / Rename / Remove
// ──────────────────────────────────────────────────────────

// Mkdir 创建远程目录。
func (s *ServerSFTPService) Mkdir(ctx context.Context, serverID uint64, dirPath string) error {
	opCtx, cancel := context.WithTimeout(ctx, writeOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return err
	}
	defer release()
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()

	cleanPath := safePath(dirPath)
	if err := sftpClient.MkdirAll(cleanPath); err != nil {
		if os.IsPermission(err) {
			return ErrWithMessage(ErrInvalidParams, "无权限创建目录")
		}
		return ErrWithMessage(ErrSSHNetwork, "创建目录失败")
	}
	return nil
}

// Rename 重命名文件/目录。
func (s *ServerSFTPService) Rename(ctx context.Context, serverID uint64, oldPath, newPath string) error {
	opCtx, cancel := context.WithTimeout(ctx, writeOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return err
	}
	defer release()
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()

	if err := sftpClient.Rename(safePath(oldPath), safePath(newPath)); err != nil {
		return ErrWithMessage(ErrSSHNetwork, "重命名失败")
	}
	return nil
}

// Remove 删除文件或目录。
func (s *ServerSFTPService) Remove(ctx context.Context, serverID uint64, targetPath string) error {
	opCtx, cancel := context.WithTimeout(ctx, writeOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return err
	}
	defer release()
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()

	cleanPath := safePath(targetPath)
	info, err := sftpClient.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 幂等
		}
		return ErrWithMessage(ErrSSHNetwork, "获取文件信息失败")
	}

	if info.IsDir() {
		// 递归删除目录
		return s.removeDir(sftpClient, cleanPath)
	}
	if err := sftpClient.Remove(cleanPath); err != nil {
		return ErrWithMessage(ErrSSHNetwork, "删除文件失败")
	}
	return nil
}

func (s *ServerSFTPService) removeDir(client *sftp.Client, dirPath string) error {
	entries, err := client.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, e := range entries {
		p := path.Join(dirPath, e.Name())
		if e.IsDir() {
			if err := s.removeDir(client, p); err != nil {
				return err
			}
		} else {
			if err := client.Remove(p); err != nil {
				return err
			}
		}
	}
	return client.RemoveDirectory(dirPath)
}

// ──────────────────────────────────────────────────────────
//  Stat
// ──────────────────────────────────────────────────────────

// Stat 获取远程文件/目录元信息。
func (s *ServerSFTPService) Stat(ctx context.Context, serverID uint64, targetPath string) (FileEntry, error) {
	opCtx, cancel := context.WithTimeout(ctx, listOpTimeout)
	defer cancel()
	release, err := s.acquireSlot(opCtx, serverID)
	if err != nil {
		return FileEntry{}, err
	}
	defer release()
	sftpClient, sshClient, err := s.dialSFTP(opCtx, serverID)
	if err != nil {
		return FileEntry{}, err
	}
	defer func() { _ = sftpClient.Close(); _ = sshClient.Close() }()

	cleanPath := safePath(targetPath)
	info, err := sftpClient.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileEntry{}, ErrWithMessage(ErrNotFound, "路径不存在")
		}
		return FileEntry{}, ErrWithMessage(ErrSSHNetwork, "获取信息失败")
	}
	return FileEntry{
		Name:    info.Name(),
		Path:    cleanPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		Mode:    info.Mode().String(),
		ModTime: info.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

// GetHomeDir 获取用户 home 目录（通过 SSH exec 执行 echo $HOME）。
func (s *ServerSFTPService) GetHomeDir(ctx context.Context, serverID uint64) (string, error) {
	info, err := s.serverSvc.GetServerSSHAuth(ctx, serverID)
	if err != nil {
		return "/", err
	}

	var authMethod ssh.AuthMethod
	switch info.AuthType {
	case "password":
		authMethod = ssh.Password(info.Password)
	case "key":
		signer, err := ssh.ParsePrivateKey([]byte(info.PrivateKey))
		if err != nil {
			return "/", nil
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		return "/", nil
	}

	sshCfg := &ssh.ClientConfig{
		User:            info.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, err := ssh.Dial("tcp", info.Addr, sshCfg)
	if err != nil {
		return "/", nil
	}
	defer func() { _ = client.Close() }()

	sess, err := client.NewSession()
	if err != nil {
		return "/", nil
	}
	defer func() { _ = sess.Close() }()

	out, err := sess.CombinedOutput("echo $HOME")
	if err != nil {
		return "/", nil
	}
	home := strings.TrimSpace(string(out))
	if home == "" {
		return "/", nil
	}
	return home, nil
}
