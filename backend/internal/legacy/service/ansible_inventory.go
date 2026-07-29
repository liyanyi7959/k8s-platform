package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
)

// nodeAlias 生成 inventory 中的节点别名：角色+序号+IP（IP 的 . 替换为 -）。
// 例如：master01-192-168-19-129、worker02-192-168-19-130。
// 该别名同时作为 k8s 节点名称（bootstrap role 会将 hostname 设为 inventory_hostname）。
func nodeAlias(role string, idx int, ip string) string {
	return provisionapp.InventoryNodeAlias(role, idx, ip)
}

// inventoryHost 表示 inventory 中的单个主机
type inventoryHost struct {
	Alias    string
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
	masterIdx, workerIdx := 0, 0

	for _, node := range nodes {
		server, cred, authType, err := s.getServerCredentialForNode(ctx, node.ServerID)
		if err != nil {
			return "", nil, fmt.Errorf("获取服务器 %d 凭据失败: %w", node.ServerID, err)
		}

		var alias string
		if node.Role == "master" {
			masterIdx++
			alias = nodeAlias("master", masterIdx, server.IP)
		} else {
			workerIdx++
			alias = nodeAlias("worker", workerIdx, server.IP)
		}
		host := inventoryHost{
			Alias:    alias,
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

	inventoryContent, err := marshalInventory(masters, workers, false)
	if err != nil {
		return "", nil, fmt.Errorf("生成 inventory 内容失败: %w", err)
	}

	// 写入临时 inventory 文件
	inventoryFile, err := os.CreateTemp("", "ansible-inventory-*.ini")
	if err != nil {
		return "", nil, fmt.Errorf("创建 inventory 文件失败: %w", err)
	}
	if err := inventoryFile.Chmod(0600); err != nil {
		inventoryFile.Close()
		os.Remove(inventoryFile.Name())
		return "", nil, fmt.Errorf("设置 inventory 文件权限失败: %w", err)
	}
	if _, err := inventoryFile.WriteString(inventoryContent); err != nil {
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

func marshalInventory(masters, workers []inventoryHost, maskSecret bool) (string, error) {
	convert := func(items []inventoryHost) []provisionapp.InventoryHost {
		result := make([]provisionapp.InventoryHost, 0, len(items))
		for _, host := range items {
			result = append(result, provisionapp.InventoryHost{
				Alias: host.Alias, IP: host.IP, SSHPort: host.SSHPort, User: host.User,
				AuthType: host.AuthType, Password: host.Password, KeyFile: host.KeyFile,
			})
		}
		return result
	}
	return provisionapp.MarshalAnsibleInventory(convert(masters), convert(workers), maskSecret)
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
	return resolveAnsibleDir(s.ansibleDir)
}

func resolveAnsibleDir(configured string) string {
	if configured != "" {
		return configured
	}
	candidates := []string{"ansible", filepath.Join("backend", "ansible")}
	if executable, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(executable), "ansible")}, candidates...)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "ansible"
}

// ansiblePlaybookPath 获取 site.yml 的完整路径
func (s *DeployService) ansiblePlaybookPath() string {
	return filepath.Join(s.ansiblePlaybookDir(), "site.yml")
}

// PlanAnsibleConfig 是部署计划对应的 Ansible 执行配置。
type PlanAnsibleConfig struct {
	PlaybookPath string         `json:"playbook_path"`
	Inventory    string         `json:"inventory"`
	ExtraVars    map[string]any `json:"extra_vars"`
}

// GetPlanAnsibleConfig 根据部署计划生成 Ansible 执行配置（inventory + vars）。
// 返回的 inventory 中密码已脱敏。
func (s *DeployService) GetPlanAnsibleConfig(ctx context.Context, planID uint64) (*PlanAnsibleConfig, error) {
	plan, nodes, err := s.getPlanWithNodes(ctx, planID)
	if err != nil {
		return nil, err
	}

	inventory, err := s.buildInventoryContent(ctx, nodes, true)
	if err != nil {
		return nil, err
	}

	extraVars := map[string]any{
		"k8s_version":       plan.K8sVersion,
		"k8s_minor_version": extractMinorVersion(plan.K8sVersion),
		"pod_cidr":          plan.PodCIDR,
		"svc_cidr":          plan.SvcCIDR,
		"cni_type":          plan.CNIType,
		"cluster_name":      plan.ClusterName,
	}

	return &PlanAnsibleConfig{
		PlaybookPath: s.ansiblePlaybookPath(),
		Inventory:    inventory,
		ExtraVars:    extraVars,
	}, nil
}

// buildInventoryContent 生成 inventory 字符串。
// maskSecret 为 true 时密码显示为 ***。
func (s *DeployService) buildInventoryContent(ctx context.Context, nodes []model.DeployPlanNode, maskSecret bool) (string, error) {
	var masters, workers []inventoryHost
	masterIdx, workerIdx := 0, 0
	for _, node := range nodes {
		server, cred, authType, err := s.getServerCredentialForNode(ctx, node.ServerID)
		if err != nil {
			return "", fmt.Errorf("获取服务器 %d 凭据失败: %w", node.ServerID, err)
		}
		var alias string
		if node.Role == "master" {
			masterIdx++
			alias = nodeAlias("master", masterIdx, server.IP)
		} else {
			workerIdx++
			alias = nodeAlias("worker", workerIdx, server.IP)
		}
		host := inventoryHost{
			Alias:    alias,
			IP:       server.IP,
			SSHPort:  server.SSHPort,
			User:     server.User,
			AuthType: authType,
		}
		if authType == "key" {
			host.KeyFile = "~/.ssh/id_rsa"
		} else {
			if maskSecret {
				host.Password = "***"
			} else {
				host.Password = cred
			}
		}
		if node.Role == "master" {
			masters = append(masters, host)
		} else {
			workers = append(workers, host)
		}
	}

	return marshalInventory(masters, workers, maskSecret)
}
