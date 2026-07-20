package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

type SSHProbeResult struct {
	Status    string  `json:"status"`
	Message   string  `json:"message"`
	OS        string  `json:"os,omitempty"`
	OSVersion string  `json:"os_version,omitempty"`
	Kernel    string  `json:"kernel,omitempty"`
	CPUCores  *uint   `json:"cpu_cores,omitempty"`
	MemoryMB  *uint64 `json:"memory_mb,omitempty"`
	DiskGB    *uint64 `json:"disk_gb,omitempty"`
}

func (s *DeployService) ProbeServerSSH(ctx context.Context, id uint64) (SSHProbeResult, error) {
	if id == 0 {
		return SSHProbeResult{}, ErrInvalidParams
	}
	var row model.DeployServer
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SSHProbeResult{}, ErrNotFound
		}
		return SSHProbeResult{}, err
	}
	// 如果服务器关联了凭据，从凭据表获取凭证
	credentialEnc := row.CredentialEnc
	authType := row.AuthType
	if row.CredentialID != nil && *row.CredentialID > 0 {
		var cred model.SSHCredential
		if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *row.CredentialID).First(&cred).Error; err != nil {
			_ = s.updateServerStatus(ctx, id, "unavailable")
			return SSHProbeResult{}, ErrWithMessage(ErrNotFound, "关联的凭据不存在")
		}
		credentialEnc = cred.CredentialEnc
		authType = cred.AuthType
	}
	credential, err := decryptText(s.encryptionKey, credentialEnc)
	if err != nil {
		_ = s.updateServerStatus(ctx, id, "unavailable")
		return SSHProbeResult{}, ErrWithMessage(ErrCrypto, "服务器凭证解密失败")
	}
	// 临时覆盖 authType 用于 SSH 连接
	row.AuthType = authType
	result, err := probeSSH(ctx, row, credential)
	status := "available"
	if err != nil {
		status = "unavailable"
		result = SSHProbeResult{Status: status, Message: err.Error()}
	}
	updates := map[string]any{"status": status}
	if result.OS != "" {
		updates["os"] = result.OS
	}
	if result.OSVersion != "" {
		updates["os_version"] = result.OSVersion
	}
	if result.Kernel != "" {
		updates["kernel"] = result.Kernel
	}
	if result.CPUCores != nil {
		updates["cpu_cores"] = *result.CPUCores
	}
	if result.MemoryMB != nil {
		updates["memory_mb"] = *result.MemoryMB
	}
	if result.DiskGB != nil {
		updates["disk_gb"] = *result.DiskGB
	}
	if dbErr := s.db.WithContext(ctx).Model(&model.DeployServer{}).Where("id = ?", id).Updates(updates).Error; dbErr != nil {
		return SSHProbeResult{}, dbErr
	}
	if err != nil {
		return result, ErrWithMessage(ErrInvalidParams, result.Message)
	}
	return result, nil
}

func probeSSH(ctx context.Context, row model.DeployServer, credential string) (SSHProbeResult, error) {
	client, err := dialDeploySSH(ctx, row, credential)
	if err != nil {
		return SSHProbeResult{}, err
	}
	defer client.Close()
	// 采集系统信息：OS、内核、CPU 核数、内存(MB)、磁盘(GB)
	output, err := runSSHCommand(client, `uname -s && uname -r && (grep -E '^(NAME|ID|ID_LIKE|VERSION_ID)=' /etc/os-release 2>/dev/null || true) && echo "---HW---" && nproc 2>/dev/null && (free -m 2>/dev/null | awk '/^Mem:/{print $2}' || cat /proc/meminfo 2>/dev/null | awk '/MemTotal/{print $2}') && (df -BG / 2>/dev/null | awk 'NR==2{gsub(/G/,"",$2);print $2}' || echo 0)`)
	if err != nil {
		return SSHProbeResult{}, err
	}
	result := parseSSHProbeOutput(output)
	result.Status = "available"
	result.Message = "SSH 连接成功"
	return result, nil
}

func dialDeploySSH(ctx context.Context, row model.DeployServer, credential string) (*ssh.Client, error) {
	auth, err := buildSSHAuth(row.AuthType, credential)
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User:            row.User,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         8 * time.Second,
	}
	addr := net.JoinHostPort(row.IP, fmt.Sprintf("%d", row.SSHPort))
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败：%w", err)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		// 将技术性 SSH 错误转换为用户友好的提示
		errMsg := err.Error()
		if strings.Contains(errMsg, "unable to authenticate") || strings.Contains(errMsg, "no supported methods remain") {
			return nil, fmt.Errorf("SSH 认证失败：密码错误或用户名不正确")
		}
		if strings.Contains(errMsg, "handshake failed") {
			return nil, fmt.Errorf("SSH 认证失败：服务器拒绝连接，请检查用户名和密码")
		}
		return nil, fmt.Errorf("SSH 认证失败：%w", err)
	}
	return ssh.NewClient(sshConn, chans, reqs), nil
}

func buildSSHAuth(authType, credential string) (ssh.AuthMethod, error) {
	switch strings.TrimSpace(authType) {
	case "password", "":
		return ssh.Password(credential), nil
	case "key":
		signer, err := ssh.ParsePrivateKey([]byte(credential))
		if err != nil {
			return nil, fmt.Errorf("SSH 私钥解析失败：%w", err)
		}
		return ssh.PublicKeys(signer), nil
	default:
		return nil, fmt.Errorf("不支持的 SSH 认证方式：%s", authType)
	}
}

func runSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH 会话失败：%w", err)
	}
	defer session.Close()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(command); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("执行探测命令失败：%s", msg)
	}
	return stdout.String(), nil
}

func parseSSHProbeOutput(output string) SSHProbeResult {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	result := SSHProbeResult{}
	hwStart := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "---HW---" {
			hwStart = i
			break
		}
	}
	// 解析系统信息（---HW--- 之前的部分）
	endIdx := len(lines)
	if hwStart >= 0 {
		endIdx = hwStart
	}
	if endIdx > 0 {
		result.OS = strings.TrimSpace(lines[0])
	}
	if endIdx > 1 {
		result.Kernel = strings.TrimSpace(lines[1])
	}
	for _, line := range lines[2:endIdx] {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), "\"")
		switch key {
		case "NAME":
			if value != "" {
				result.OS = value
			}
		case "VERSION_ID":
			result.OSVersion = value
		}
	}
	// 解析硬件信息（---HW--- 之后：CPU 核数、内存 MB、磁盘 GB）
	if hwStart >= 0 && hwStart+3 < len(lines) {
		if cpu, err := strconv.ParseUint(strings.TrimSpace(lines[hwStart+1]), 10, 64); err == nil && cpu > 0 {
			c := uint(cpu)
			result.CPUCores = &c
		}
		if mem, err := strconv.ParseUint(strings.TrimSpace(lines[hwStart+2]), 10, 64); err == nil && mem > 0 {
			result.MemoryMB = &mem
		}
		if disk, err := strconv.ParseUint(strings.TrimSpace(lines[hwStart+3]), 10, 64); err == nil && disk > 0 {
			result.DiskGB = &disk
		}
	}
	return result
}
