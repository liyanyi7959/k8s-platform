package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"k8s-platform-backend/internal/model"
	"k8s.io/client-go/tools/clientcmd"
)

// runAnsibleOnMaster uses the plan's only Master as an ephemeral Linux runner.
// Only the runner workspace is cleaned; target state is intentionally preserved for idempotent retry.
func (s *DeployService) runAnsibleOnMaster(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *Task) (string, error) {
	var masterNode *model.DeployPlanNode
	for i := range nodes {
		if nodes[i].Role == "master" {
			masterNode = &nodes[i]
			break
		}
	}
	if masterNode == nil {
		return "", fmt.Errorf("部署计划未配置 Master 节点")
	}

	master, credential, authType, err := s.getServerCredentialForNode(ctx, masterNode.ServerID)
	if err != nil {
		return "", fmt.Errorf("读取 Master Runner 凭据失败: %w", err)
	}
	master.AuthType = authType
	client, err := dialDeploySSH(ctx, master, credential)
	if err != nil {
		return "", fmt.Errorf("连接 Master Runner 失败: %w", err)
	}
	defer client.Close()

	workspace := fmt.Sprintf("/tmp/k8s-platform-deploy-%d-%d", plan.ID, task.ID)
	defer func() {
		_, cleanupErr := runSSHCommand(client, "rm -rf -- "+shellQuote(workspace))
		if cleanupErr != nil {
			task.AppendLog(fmt.Sprintf("[warn] Runner 临时目录清理失败: %v", cleanupErr))
		} else {
			task.AppendLog("[info] Runner 临时目录已清理")
		}
		_ = s.taskStore.Put(task)
	}()

	task.AppendLog(fmt.Sprintf("[info] 使用 Master %s (%s) 作为临时 Ansible Runner", master.Name, master.IP))
	task.AppendLog("[info] 正在检查并安装 Runner 依赖...")
	_ = s.taskStore.Put(task)
	needSSHPass := false
	for _, node := range nodes {
		_, _, nodeAuthType, credErr := s.getServerCredentialForNode(ctx, node.ServerID)
		if credErr != nil {
			return "", credErr
		}
		if nodeAuthType != "key" {
			needSSHPass = true
		}
	}
	bootstrap := runnerBootstrapScript(needSSHPass)
	if output, bootstrapErr := runPrivilegedSSHCommand(client, master, credential, bootstrap); bootstrapErr != nil {
		return "", fmt.Errorf("Master Runner 依赖安装失败: %w", bootstrapErr)
	} else if strings.TrimSpace(output) != "" {
		task.AppendLog("[runner] " + strings.TrimSpace(output))
	}

	archive, err := s.buildRunnerArchive(ctx, plan, nodes, workspace)
	if err != nil {
		return "", err
	}
	if err := uploadRunnerArchive(client, workspace, archive); err != nil {
		return "", fmt.Errorf("上传 Runner 执行包失败: %w", err)
	}
	task.AppendLog("[info] Playbook、inventory 和临时凭据已上传")
	_ = s.taskStore.Put(task)

	writer := newAnsibleLogWriter(task, s.taskStore)
	defer writer.flush()
	command := "cd " + shellQuote(workspace+"/ansible") + " && " +
		"ANSIBLE_HOST_KEY_CHECKING=False ANSIBLE_RETRY_FILES_ENABLED=False ANSIBLE_NOCOLOR=True " +
		"ansible-playbook site.yml -i ../inventory.yml --extra-vars @../extra-vars.json --forks 10"
	if err := runStreamingSSHCommand(ctx, client, command, writer); err != nil {
		return "", fmt.Errorf("Ansible Playbook 执行失败: %w", err)
	}
	kubeconfig, err := runPrivilegedSSHCommand(client, master, credential, "cat /etc/kubernetes/admin.conf")
	if err != nil {
		return "", fmt.Errorf("从 Master 读取 kubeconfig 失败: %w", err)
	}
	return normalizeMasterKubeconfig(kubeconfig, master.IP)
}

func runnerBootstrapScript(needSSHPass bool) string {
	sshpass := ""
	if needSSHPass {
		sshpass = " sshpass"
	}
	return `set -eu
if command -v ansible-playbook >/dev/null 2>&1 && command -v tar >/dev/null 2>&1` + func() string {
		if needSSHPass {
			return ` && command -v sshpass >/dev/null 2>&1`
		}
		return ""
	}() + `; then ansible-playbook --version | head -n 1; exit 0; fi
if command -v apt-get >/dev/null 2>&1; then
  apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y ansible tar gzip` + sshpass + `
else
  pm=""; command -v dnf >/dev/null 2>&1 && pm=dnf; [ -n "$pm" ] || command -v yum >/dev/null 2>&1 && pm=yum
  [ -n "$pm" ] || { echo "unsupported package manager" >&2; exit 1; }
  $pm install -y tar gzip` + sshpass + `
  $pm install -y ansible-core || $pm install -y ansible || { $pm install -y python3-pip; python3 -m pip install ansible-core; }
fi
command -v ansible-playbook >/dev/null 2>&1
command -v tar >/dev/null 2>&1
ansible-playbook --version | head -n 1`
}

func (s *DeployService) buildRunnerArchive(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, workspace string) ([]byte, error) {
	var masters, workers []inventoryHost
	keys := map[string]string{}
	ignoredDiskHosts := make([]string, 0)
	for _, node := range nodes {
		server, credential, authType, err := s.getServerCredentialForNode(ctx, node.ServerID)
		if err != nil {
			return nil, fmt.Errorf("读取节点 %d 凭据失败: %w", node.ServerID, err)
		}
		host := inventoryHost{Alias: fmt.Sprintf("node-%d", node.ServerID), IP: server.IP, SSHPort: server.SSHPort, User: server.User, AuthType: authType}
		if containsString(plan.PreflightIgnores, fmt.Sprintf("node.%d.disk", node.ServerID)) {
			ignoredDiskHosts = append(ignoredDiskHosts, host.Alias)
		}
		if authType == "key" {
			name := fmt.Sprintf("keys/node-%d", node.ServerID)
			host.KeyFile = workspace + "/" + name
			keys[name] = credential
		} else {
			host.Password = credential
		}
		if node.Role == "master" {
			masters = append(masters, host)
		} else {
			workers = append(workers, host)
		}
	}
	inventory, err := marshalInventory(masters, workers, false)
	if err != nil {
		return nil, err
	}
	extraVars, err := json.Marshal(map[string]any{
		"k8s_version": plan.K8sVersion, "k8s_package_version": strings.TrimPrefix(plan.K8sVersion, "v"),
		"k8s_minor_version": extractMinorVersion(plan.K8sVersion), "pod_cidr": plan.PodCIDR,
		"svc_cidr": plan.SvcCIDR, "cni_type": plan.CNIType, "cluster_name": plan.ClusterName,
		"addons": []string(plan.Addons), "preflight_ignored_disk_hosts": ignoredDiskHosts,
	})
	if err != nil {
		return nil, err
	}

	var result bytes.Buffer
	gz := gzip.NewWriter(&result)
	tw := tar.NewWriter(gz)
	addBytes := func(name string, data []byte, mode int64) error {
		if err := tw.WriteHeader(&tar.Header{Name: filepath.ToSlash(name), Mode: mode, Size: int64(len(data))}); err != nil {
			return err
		}
		_, err := tw.Write(data)
		return err
	}
	if err := addBytes("inventory.yml", []byte(inventory), 0600); err != nil {
		return nil, err
	}
	if err := addBytes("extra-vars.json", extraVars, 0600); err != nil {
		return nil, err
	}
	for name, content := range keys {
		if err := addBytes(name, []byte(content), 0600); err != nil {
			return nil, err
		}
	}
	root := s.ansiblePlaybookDir()
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		name := filepath.Join("ansible", rel)
		if entry.IsDir() {
			if rel == "." {
				return nil
			}
			return tw.WriteHeader(&tar.Header{Name: filepath.ToSlash(name) + "/", Typeflag: tar.TypeDir, Mode: 0755})
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return addBytes(name, data, 0644)
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func uploadRunnerArchive(client *ssh.Client, workspace string, archive []byte) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdin = bytes.NewReader(archive)
	var stderr bytes.Buffer
	session.Stderr = &stderr
	command := "umask 077; mkdir -p " + shellQuote(workspace) + " && chmod 700 " + shellQuote(workspace) + " && tar -xzf - -C " + shellQuote(workspace)
	if err := session.Run(command); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func runStreamingSSHCommand(ctx context.Context, client *ssh.Client, command string, writer io.Writer) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	session.Stdout, session.Stderr = writer, writer
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case err := <-done:
		session.Close()
		return err
	case <-ctx.Done():
		session.Close()
		return ctx.Err()
	}
}

func runPrivilegedSSHCommand(client *ssh.Client, server model.DeployServer, credential, script string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout, session.Stderr = &stdout, &stderr
	quotedScript := shellQuote(script)
	command := `if [ "$(id -u)" = "0" ]; then sh -c ` + quotedScript + `; else sudo -S -p '' sh -c ` + quotedScript + `; fi`
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

func normalizeMasterKubeconfig(raw, masterIP string) (string, error) {
	cfg, err := clientcmd.Load([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("解析 kubeconfig 失败: %w", err)
	}
	for _, cluster := range cfg.Clusters {
		cluster.Server = strings.Replace(cluster.Server, "127.0.0.1", masterIP, 1)
		cluster.Server = strings.Replace(cluster.Server, "localhost", masterIP, 1)
	}
	data, err := clientcmd.Write(*cfg)
	if err != nil {
		return "", fmt.Errorf("序列化 kubeconfig 失败: %w", err)
	}
	return string(data), nil
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'" }
