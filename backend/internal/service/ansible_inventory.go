package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s-platform-backend/internal/model"
)

// inventoryHost 表示 inventory 中的单个主机
type inventoryHost struct {
	IP       string
	SSHPort  int
	User     string
	AuthType string
	Password string // 密码认证
	KeyFile  string // 密钥认证的临时文件路径
}

// generateInventoryFile 从部署计划的节点生成 Ansible inventory 文件
// 返回 inventory 文件路径和清理函数
func (s *DeployService) generateInventoryFile(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode) (string, func(), error) {
	var masters, workers []inventoryHost
	var keyFiles []string

	for _, node := range nodes {
		server, cred, authType, err := s.getServerCredentialForNode(ctx, node.ServerID)
		if err != nil {
			return "", nil, fmt.Errorf("获取服务器 %d 凭据失败: %w", node.ServerID, err)
		}

		host := inventoryHost{
			IP:       server.IP,
			SSHPort:  server.SSHPort,
			User:     server.User,
			AuthType: authType,
		}

		if authType == "key" {
			// 将私钥写入临时文件
			keyFile, err := os.CreateTemp("", "ansible-key-*")
			if err != nil {
				return "", nil, fmt.Errorf("创建密钥临时文件失败: %w", err)
			}
			if _, err := keyFile.WriteString(cred); err != nil {
				keyFile.Close()
				os.Remove(keyFile.Name())
				return "", nil, fmt.Errorf("写入密钥文件失败: %w", err)
			}
			keyFile.Close()
			os.Chmod(keyFile.Name(), 0600)
			host.KeyFile = keyFile.Name()
			keyFiles = append(keyFiles, keyFile.Name())
		} else {
			host.Password = cred
		}

		if node.Role == "master" {
			masters = append(masters, host)
		} else {
			workers = append(workers, host)
		}
	}

	// 生成 inventory 内容
	var sb strings.Builder
	sb.WriteString("[master]\n")
	for _, h := range masters {
		sb.WriteString(formatHostLine(h))
	}
	sb.WriteString("\n[worker]\n")
	for _, h := range workers {
		sb.WriteString(formatHostLine(h))
	}
	// 如果没有 worker 节点，添加一个空行避免空组错误
	if len(workers) == 0 {
		sb.WriteString("localhost ansible_connection=local\n")
	}
	sb.WriteString("\n[all:vars]\n")
	sb.WriteString("ansible_python_interpreter=/usr/bin/python3\n")

	// 写入临时 inventory 文件
	inventoryFile, err := os.CreateTemp("", "ansible-inventory-*.ini")
	if err != nil {
		return "", nil, fmt.Errorf("创建 inventory 文件失败: %w", err)
	}
	if _, err := inventoryFile.WriteString(sb.String()); err != nil {
		inventoryFile.Close()
		os.Remove(inventoryFile.Name())
		return "", nil, fmt.Errorf("写入 inventory 文件失败: %w", err)
	}
	inventoryFile.Close()

	cleanup := func() {
		os.Remove(inventoryFile.Name())
		for _, f := range keyFiles {
			os.Remove(f)
		}
	}

	return inventoryFile.Name(), cleanup, nil
}

// formatHostLine 格式化 inventory 中的主机行
func formatHostLine(h inventoryHost) string {
	var line string
	if h.KeyFile != "" {
		line = fmt.Sprintf("%s ansible_port=%d ansible_user=%s ansible_ssh_private_key_file=%s\n",
			h.IP, h.SSHPort, h.User, h.KeyFile)
	} else {
		line = fmt.Sprintf("%s ansible_port=%d ansible_user=%s ansible_ssh_pass=%s\n",
			h.IP, h.SSHPort, h.User, h.Password)
	}
	return line
}

// getServerCredentialForNode 获取节点服务器的凭据信息
// 返回 server 模型、解密后的凭据字符串、认证类型
func (s *DeployService) getServerCredentialForNode(ctx context.Context, serverID uint64) (model.DeployServer, string, string, error) {
	var server model.DeployServer
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", serverID).First(&server).Error; err != nil {
		return model.DeployServer{}, "", "", fmt.Errorf("服务器不存在: %w", err)
	}

	credentialEnc := server.CredentialEnc
	authType := server.AuthType

	// 如果服务器关联了凭据，从凭据表获取
	if server.CredentialID != nil && *server.CredentialID > 0 {
		var cred model.SSHCredential
		if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *server.CredentialID).First(&cred).Error; err != nil {
			return model.DeployServer{}, "", "", fmt.Errorf("关联的凭据不存在: %w", err)
		}
		credentialEnc = cred.CredentialEnc
		authType = cred.AuthType
	}

	cred, err := decryptText(s.encryptionKey, credentialEnc)
	if err != nil {
		return model.DeployServer{}, "", "", fmt.Errorf("解密凭证失败: %w", err)
	}

	return server, cred, authType, nil
}

// ansiblePlaybookDir 获取 Ansible playbook 的目录路径
func (s *DeployService) ansiblePlaybookDir() string {
	// 优先使用配置的路径，否则使用相对路径
	if s.ansibleDir != "" {
		return s.ansibleDir
	}
	return "ansible"
}

// ansiblePlaybookPath 获取 site.yml 的完整路径
func (s *DeployService) ansiblePlaybookPath() string {
	return filepath.Join(s.ansiblePlaybookDir(), "site.yml")
}
