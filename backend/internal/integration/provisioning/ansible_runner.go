package provisioning

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

	"gorm.io/gorm"
	"k8s.io/client-go/tools/clientcmd"

	platformapp "k8s-platform-backend/internal/platform/application"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
	secretcrypto "k8s-platform-backend/internal/transport/secretcrypto"
	sshtransport "k8s-platform-backend/internal/transport/ssh"

	"golang.org/x/crypto/ssh"
)

// AnsibleRunner owns the remote Ansible execution archive, runner bootstrap,
// credential materialization, and streamed task-log projection for the
// Provisioning deployment executor.
type AnsibleRunner struct {
	db            *gorm.DB
	encryptionKey string
	config        *provisionapp.DeployConfigService
	taskStore     *platformapp.TaskStore
}

func NewAnsibleRunner(db *gorm.DB, encryptionKey string, taskStore *platformapp.TaskStore) *AnsibleRunner {
	return &AnsibleRunner{
		db:            db,
		encryptionKey: encryptionKey,
		config:        provisionapp.NewDeployConfigService(db),
		taskStore:     taskStore,
	}
}

// Run uses the plan's Master as a short-lived Linux Ansible runner. Only its
// workspace is removed afterwards; the target state remains for idempotent
// retry and diagnostics.
func (r *AnsibleRunner) Run(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, task *platformapp.Task) (string, error) {
	if r == nil || r.db == nil || task == nil {
		return "", fmt.Errorf("Ansible Runner 未初始化")
	}
	logErr := func(msg string, err error) error {
		task.AppendLog(fmt.Sprintf("[error] %s: %v", msg, err), activeTaskStepKey(task))
		r.putTask(task)
		return fmt.Errorf("%s: %w", msg, err)
	}

	var masterNode *model.DeployPlanNode
	for index := range nodes {
		if nodes[index].Role == "master" {
			masterNode = &nodes[index]
			break
		}
	}
	if masterNode == nil {
		return "", logErr("部署计划未配置 Master 节点", fmt.Errorf("master node not found"))
	}

	master, credential, authType, err := r.serverCredential(ctx, masterNode.ServerID)
	if err != nil {
		return "", logErr("读取 Master Runner 凭据失败", err)
	}
	master.AuthType = authType
	client, err := sshtransport.Dial(ctx, deploymentSSHConfig(master), credential)
	if err != nil {
		return "", logErr("连接 Master Runner 失败", err)
	}
	defer client.Close()

	workspace := fmt.Sprintf("/tmp/k8s-platform-deploy-%d-%d", plan.ID, task.ID)
	defer func() {
		_, cleanupErr := sshtransport.RunCommand(client, "rm -rf -- "+runnerShellQuote(workspace))
		if cleanupErr != nil {
			task.AppendLog(fmt.Sprintf("[warn] Runner 临时目录清理失败: %v", cleanupErr), activeTaskStepKey(task))
		} else {
			task.AppendLog("[info] Runner 临时目录已清理", activeTaskStepKey(task))
		}
		r.putTask(task)
	}()

	task.AppendLog(fmt.Sprintf("[info] 使用 Master %s (%s) 作为临时 Ansible Runner", master.Name, master.IP), activeTaskStepKey(task))
	task.AppendLog("[info] 正在检查并安装 Runner 依赖...", activeTaskStepKey(task))
	r.putTask(task)

	needSSHPass := false
	for _, node := range nodes {
		_, _, nodeAuthType, credentialErr := r.serverCredential(ctx, node.ServerID)
		if credentialErr != nil {
			return "", logErr(fmt.Sprintf("读取节点 %d 凭据失败", node.ServerID), credentialErr)
		}
		if nodeAuthType != "key" {
			needSSHPass = true
		}
	}
	if output, bootstrapErr := sshtransport.RunPrivilegedCommand(client, master.AuthType, credential, runnerBootstrapScript(needSSHPass)); bootstrapErr != nil {
		if strings.TrimSpace(output) != "" {
			task.AppendLog("[runner] "+strings.TrimSpace(output), activeTaskStepKey(task))
		}
		return "", logErr("Master Runner 依赖安装失败", bootstrapErr)
	} else if strings.TrimSpace(output) != "" {
		task.AppendLog("[runner] "+strings.TrimSpace(output), activeTaskStepKey(task))
		r.putTask(task)
	}

	archive, err := r.buildArchive(ctx, plan, nodes, workspace, task)
	if err != nil {
		return "", logErr("构建 Runner 执行包失败", err)
	}
	if err := uploadRunnerArchive(client, workspace, archive); err != nil {
		return "", logErr("上传 Runner 执行包失败", err)
	}
	task.AppendLog("[info] Playbook、inventory 和临时凭据已上传", activeTaskStepKey(task))
	r.putTask(task)

	writer := newAnsibleLogWriter(task, r.taskStore)
	defer writer.flush()
	command := "cd " + runnerShellQuote(workspace+"/ansible") + " && " +
		"ANSIBLE_HOST_KEY_CHECKING=False ANSIBLE_RETRY_FILES_ENABLED=False ANSIBLE_NOCOLOR=True " +
		"ansible-playbook site.yml -i ../inventory.yml --extra-vars @../extra-vars.json --forks 10"
	if err := runStreamingSSHCommand(ctx, client, command, writer); err != nil {
		return "", logErr("Ansible Playbook 执行失败", err)
	}
	kubeconfig, err := sshtransport.RunPrivilegedCommand(client, master.AuthType, credential, "cat /etc/kubernetes/admin.conf")
	if err != nil {
		return "", logErr("从 Master 读取 kubeconfig 失败", err)
	}
	return normalizeMasterKubeconfig(kubeconfig, master.IP)
}

func (r *AnsibleRunner) putTask(task *platformapp.Task) {
	if r != nil && r.taskStore != nil && task != nil {
		_ = r.taskStore.Put(task)
	}
}

func (r *AnsibleRunner) serverCredential(ctx context.Context, serverID uint64) (model.DeployServer, string, string, error) {
	var server model.DeployServer
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", serverID).First(&server).Error; err != nil {
		return model.DeployServer{}, "", "", fmt.Errorf("服务器不存在: %w", err)
	}
	ciphertext, authType := server.CredentialEnc, server.AuthType
	if server.CredentialID != nil && *server.CredentialID > 0 {
		var credential model.SSHCredential
		if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", *server.CredentialID).First(&credential).Error; err != nil {
			return model.DeployServer{}, "", "", fmt.Errorf("关联的凭据不存在: %w", err)
		}
		ciphertext, authType = credential.CredentialEnc, credential.AuthType
	}
	secret, err := secretcrypto.Decrypt(r.encryptionKey, ciphertext)
	if err != nil {
		return model.DeployServer{}, "", "", fmt.Errorf("解密凭证失败: %w", err)
	}
	return server, secret, authType, nil
}

func (r *AnsibleRunner) buildArchive(ctx context.Context, plan model.DeployPlan, nodes []model.DeployPlanNode, workspace string, task *platformapp.Task) ([]byte, error) {
	masters, workers := make([]provisionapp.InventoryHost, 0), make([]provisionapp.InventoryHost, 0)
	keys := map[string]string{}
	ignoredDiskHosts := make([]string, 0)
	masterIndex, workerIndex := 0, 0
	for _, node := range nodes {
		server, credential, authType, err := r.serverCredential(ctx, node.ServerID)
		if err != nil {
			return nil, fmt.Errorf("读取节点 %d 凭据失败: %w", node.ServerID, err)
		}
		role := "worker"
		index := workerIndex + 1
		if strings.TrimSpace(node.Role) == "master" {
			role = "master"
			masterIndex++
			index = masterIndex
		} else {
			workerIndex++
		}
		host := provisionapp.InventoryHost{
			Alias: provisionapp.InventoryNodeAlias(role, index, server.IP), IP: server.IP,
			SSHPort: server.SSHPort, User: server.User, AuthType: authType,
		}
		if provisionapp.IsPreflightIgnore(plan.PreflightIgnores, fmt.Sprintf("node.%d.disk", node.ServerID)) {
			ignoredDiskHosts = append(ignoredDiskHosts, host.Alias)
		}
		if authType == "key" {
			name := fmt.Sprintf("keys/node-%d", node.ServerID)
			host.KeyFile = workspace + "/" + name
			keys[name] = credential
		} else {
			host.Password = credential
		}
		if role == "master" {
			masters = append(masters, host)
		} else {
			workers = append(workers, host)
		}
	}
	inventory, err := provisionapp.MarshalAnsibleInventory(masters, workers, false)
	if err != nil {
		return nil, err
	}

	retryFromStep, _ := task.Meta["retry_from_step"].(string)
	enabledSteps := provisionapp.ComputeEnabledAnsibleSteps(retryFromStep)
	if requestedSteps := taskMetaStringSlice(task.Meta, "enabled_steps"); len(requestedSteps) > 0 {
		enabledSteps = requestedSteps
	}
	if retryFromStep != "" {
		task.AppendLog(fmt.Sprintf("[info] 从步骤 %s 开始重试，将跳过已成功的步骤", retryFromStep), activeTaskStepKey(task))
		r.putTask(task)
	}
	extraVarsMap := map[string]any{
		"k8s_version": plan.K8sVersion, "k8s_package_version": strings.TrimPrefix(plan.K8sVersion, "v"),
		"k8s_minor_version": provisionapp.KubernetesMinorVersion(plan.K8sVersion), "pod_cidr": plan.PodCIDR,
		"svc_cidr": plan.SvcCIDR, "cni_type": plan.CNIType, "cluster_name": plan.ClusterName,
		"addons": []string(plan.Addons), "helm_install": plan.HelmInstall, "preflight_ignored_disk_hosts": ignoredDiskHosts,
	}
	if r.config != nil {
		if repos, repositoryErr := r.config.ListRepositories(ctx, ""); repositoryErr == nil {
			var yumMirrors, aptMirrors []map[string]any
			for _, repo := range repos {
				if !repo.Enabled {
					continue
				}
				switch repo.RepoType {
				case "container_mirror":
					if repo.IsDefault {
						imageRepository := strings.TrimPrefix(repo.URL, "https://")
						imageRepository = strings.TrimPrefix(imageRepository, "http://")
						extraVarsMap["k8s_image_repository"] = strings.TrimSuffix(imageRepository, "/")
					}
				case "yum":
					targetOS := ""
					if repo.MirrorOf != nil {
						targetOS = strings.ToLower(strings.TrimSpace(*repo.MirrorOf))
					}
					yumMirrors = append(yumMirrors, map[string]any{"name": repo.Name, "url": strings.TrimSuffix(repo.URL, "/"), "priority": repo.Priority, "target_os": targetOS})
				case "apt":
					aptMirrors = append(aptMirrors, map[string]any{"name": repo.Name, "url": strings.TrimSuffix(repo.URL, "/"), "priority": repo.Priority})
				}
			}
			if len(yumMirrors) > 0 {
				extraVarsMap["yum_mirrors"] = yumMirrors
			}
			if len(aptMirrors) > 0 {
				extraVarsMap["apt_mirrors"] = aptMirrors
			}
		}
	}
	if len(enabledSteps) > 0 {
		extraVarsMap["retry_enabled_steps"] = enabledSteps
	}
	extraVars, err := json.Marshal(extraVarsMap)
	if err != nil {
		return nil, err
	}

	var result bytes.Buffer
	gzipWriter := gzip.NewWriter(&result)
	tarWriter := tar.NewWriter(gzipWriter)
	addBytes := func(name string, data []byte, mode int64) error {
		if err := tarWriter.WriteHeader(&tar.Header{Name: filepath.ToSlash(name), Mode: mode, Size: int64(len(data))}); err != nil {
			return err
		}
		_, err := tarWriter.Write(data)
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
	root := provisioningPlaybookDir()
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.Join("ansible", relativePath)
		if entry.IsDir() {
			if relativePath == "." {
				return nil
			}
			return tarWriter.WriteHeader(&tar.Header{Name: filepath.ToSlash(name) + "/", Typeflag: tar.TypeDir, Mode: 0755})
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return addBytes(name, data, 0644)
	}); err != nil {
		return nil, err
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func provisioningPlaybookDir() string {
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

func taskMetaStringSlice(meta map[string]any, key string) []string {
	if meta == nil {
		return nil
	}
	values := make([]string, 0)
	switch raw := meta[key].(type) {
	case []string:
		values = raw
	case []any:
		for _, value := range raw {
			if text, ok := value.(string); ok {
				values = append(values, text)
			}
		}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
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

func uploadRunnerArchive(client *ssh.Client, workspace string, archive []byte) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdin = bytes.NewReader(archive)
	var stderr bytes.Buffer
	session.Stderr = &stderr
	command := "umask 077; mkdir -p " + runnerShellQuote(workspace) + " && chmod 700 " + runnerShellQuote(workspace) + " && tar -xzf - -C " + runnerShellQuote(workspace)
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
	defer session.Close()
	session.Stdout, session.Stderr = writer, writer
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func normalizeMasterKubeconfig(raw, masterIP string) (string, error) {
	config, err := clientcmd.Load([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("解析 kubeconfig 失败: %w", err)
	}
	for _, cluster := range config.Clusters {
		cluster.Server = strings.Replace(cluster.Server, "127.0.0.1", masterIP, 1)
		cluster.Server = strings.Replace(cluster.Server, "localhost", masterIP, 1)
	}
	data, err := clientcmd.Write(*config)
	if err != nil {
		return "", fmt.Errorf("序列化 kubeconfig 失败: %w", err)
	}
	return string(data), nil
}

func runnerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func activeTaskStepKey(task *platformapp.Task) string {
	if task == nil {
		return ""
	}
	for _, step := range task.Steps {
		if step.Status == platformapp.StepRunning {
			return step.Key
		}
	}
	for _, step := range task.Steps {
		if step.Status != platformapp.StepSuccess {
			return step.Key
		}
	}
	return ""
}
