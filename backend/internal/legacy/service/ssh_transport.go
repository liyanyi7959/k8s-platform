package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"

	"golang.org/x/crypto/ssh"
)

// ProbeSSH is the narrow transport port used by Provisioning runtime adapters.
func ProbeSSH(ctx context.Context, server model.DeployServer, credential string) (provisionapp.SSHProbeResult, error) {
	client, err := DialDeploySSH(ctx, server, credential)
	if err != nil {
		return provisionapp.SSHProbeResult{}, err
	}
	defer client.Close()
	output, err := RunSSHCommand(client, `uname -s && uname -r && (grep -E '^(NAME|ID|ID_LIKE|VERSION_ID)=' /etc/os-release 2>/dev/null || true) && echo "---HW---" && nproc 2>/dev/null && (free -m 2>/dev/null | awk '/^Mem:/{print $2}' || cat /proc/meminfo 2>/dev/null | awk '/MemTotal/{print $2}') && (df -BG / 2>/dev/null | awk 'NR==2{gsub(/G/,"",$2);print $2}' || echo 0)`)
	if err != nil {
		return provisionapp.SSHProbeResult{}, err
	}
	result := parseSSHProbeOutput(output)
	result.Status = "available"
	result.Message = "SSH 连接成功"
	return result, nil
}

// DialDeploySSH returns an authenticated SSH client and keeps credential
// cipher handling outside of this retained transport package.
func DialDeploySSH(ctx context.Context, server model.DeployServer, credential string) (*ssh.Client, error) {
	auth, err := buildSSHAuth(server.AuthType, credential)
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: server.User, Auth: []ssh.AuthMethod{auth}, HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 8 * time.Second,
	}
	address := net.JoinHostPort(server.IP, fmt.Sprintf("%d", server.SSHPort))
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	sshConnection, channels, requests, err := ssh.NewClientConn(connection, address, config)
	if err != nil {
		connection.Close()
		message := err.Error()
		switch {
		case strings.Contains(message, "unable to authenticate"), strings.Contains(message, "no supported methods remain"):
			return nil, fmt.Errorf("SSH 认证失败：密码错误或用户名不正确")
		case strings.Contains(message, "handshake failed"):
			return nil, fmt.Errorf("SSH 认证失败：服务器拒绝连接，请检查用户名和密码")
		default:
			return nil, fmt.Errorf("SSH 认证失败: %w", err)
		}
	}
	return ssh.NewClient(sshConnection, channels, requests), nil
}

// RunSSHCommand executes a non-privileged command over an authenticated
// client. Callers own command selection and policy.
func RunSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH 会话失败: %w", err)
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout, session.Stderr = &stdout, &stderr
	if err := session.Run(command); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("执行探测命令失败: %s", message)
	}
	return stdout.String(), nil
}

// RunPrivilegedSSHCommand executes a caller-provided script with the managed
// server credential. The runner adapter owns all provisioning policy.
func RunPrivilegedSSHCommand(client *ssh.Client, server model.DeployServer, credential, script string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout, session.Stderr = &stdout, &stderr
	command := `if [ "$(id -u)" = "0" ]; then sh -c ` + sshShellQuote(script) + `; else sudo -S -p '' sh -c ` + sshShellQuote(script) + `; fi`
	if server.AuthType != "key" {
		session.Stdin = strings.NewReader(credential + "\n")
	}
	if err := session.Run(command); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return stdout.String(), fmt.Errorf("%s", message)
	}
	return stdout.String(), nil
}

func buildSSHAuth(authType, credential string) (ssh.AuthMethod, error) {
	switch strings.TrimSpace(authType) {
	case "password", "":
		return ssh.Password(credential), nil
	case "key":
		signer, err := ssh.ParsePrivateKey([]byte(credential))
		if err != nil {
			return nil, fmt.Errorf("SSH 私钥解析失败: %w", err)
		}
		return ssh.PublicKeys(signer), nil
	default:
		return nil, fmt.Errorf("不支持的 SSH 认证方式: %s", authType)
	}
}

func sshShellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'" }

func parseSSHProbeOutput(output string) provisionapp.SSHProbeResult {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	result := provisionapp.SSHProbeResult{}
	hardwareStart := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == "---HW---" {
			hardwareStart = index
			break
		}
	}
	endIndex := len(lines)
	if hardwareStart >= 0 {
		endIndex = hardwareStart
	}
	if endIndex > 0 {
		result.OS = strings.TrimSpace(lines[0])
	}
	if endIndex > 1 {
		result.Kernel = strings.TrimSpace(lines[1])
	}
	for _, line := range lines[2:endIndex] {
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
	if hardwareStart >= 0 && hardwareStart+3 < len(lines) {
		if cpu, err := strconv.ParseUint(strings.TrimSpace(lines[hardwareStart+1]), 10, 64); err == nil && cpu > 0 {
			value := uint(cpu)
			result.CPUCores = &value
		}
		if memory, err := strconv.ParseUint(strings.TrimSpace(lines[hardwareStart+2]), 10, 64); err == nil && memory > 0 {
			value := uint64(memory)
			result.MemoryMB = &value
		}
		if disk, err := strconv.ParseUint(strings.TrimSpace(lines[hardwareStart+3]), 10, 64); err == nil && disk > 0 {
			value := uint64(disk)
			result.DiskGB = &value
		}
	}
	return result
}
