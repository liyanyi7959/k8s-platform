package ansible

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Runner struct {
	binPath string
	docker  string
	image   string
}

func resolveBin(envKey string, fallbackName string) string {
	v := strings.TrimSpace(os.Getenv(envKey))
	if v == "" {
		v = fallbackName
	}
	if strings.TrimSpace(v) == "" {
		return ""
	}
	if filepath.IsAbs(v) {
		if st, err := os.Stat(v); err == nil && !st.IsDir() {
			return v
		}
		return ""
	}
	p, err := exec.LookPath(v)
	if err != nil {
		return ""
	}
	return p
}

func shellQuoteSingle(s string) string {
	return "'" + strings.ReplaceAll(s, `'`, `'"'"'`) + "'"
}

func NewRunner() *Runner {
	image := strings.TrimSpace(os.Getenv("K8S_PLATFORM_ANSIBLE_IMAGE"))
	if image == "" {
		image = "cytopia/ansible:latest"
	}

	ansiblePath := resolveBin("K8S_PLATFORM_ANSIBLE_PLAYBOOK_BIN", "ansible-playbook")
	dockerPath := resolveBin("K8S_PLATFORM_DOCKER_BIN", "docker")

	return &Runner{binPath: ansiblePath, docker: dockerPath, image: image}
}

func (r *Runner) RunPlaybook(ctx context.Context, inventory string, playbook string, extraVars map[string]any, extraFiles map[string][]byte, onLine func(string)) error {
	// 1. 创建临时目录
	tmpDir, err := os.MkdirTemp("", "ansible-task-*")
	if err != nil {
		return fmt.Errorf("create temp dir failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if r.binPath == "" {
		if r.docker != "" {
			onLine(fmt.Sprintf("[Runner] ansible-playbook not found, use docker runner (image=%s)", r.image))
		} else {
			onLine("[Runner] missing dependencies on backend host: ansible-playbook and docker not found; install either ansible-playbook or docker, or set K8S_PLATFORM_ANSIBLE_PLAYBOOK_BIN / K8S_PLATFORM_DOCKER_BIN")
		}
	} else {
		onLine(fmt.Sprintf("[Runner] use local ansible-playbook (path=%s)", r.binPath))
	}

	// 2. 写入 inventory 文件
	invPath := filepath.Join(tmpDir, "hosts.ini")
	invText := inventory
	if r.binPath == "" && r.docker != "" {
		invText = strings.ReplaceAll(invText, "ansible_ssh_private_key_file=./key_", "ansible_ssh_private_key_file=/tmp/ansiblekeys/key_")
	}
	if err := os.WriteFile(invPath, []byte(invText), 0644); err != nil {
		return fmt.Errorf("write inventory failed: %w", err)
	}

	// 3. 写入 playbook 文件
	pbPath := filepath.Join(tmpDir, "site.yaml")
	if err := os.WriteFile(pbPath, []byte(playbook), 0644); err != nil {
		return fmt.Errorf("write playbook failed: %w", err)
	}

	// 3.1 写入额外文件（如 SSH Key）
	// 注意：文件权限设置为 0600 (SSH Key 要求严格权限)
	for name, content := range extraFiles {
		fp := filepath.Join(tmpDir, name)
		if err := os.WriteFile(fp, content, 0600); err != nil {
			return fmt.Errorf("write extra file %s failed: %w", name, err)
		}
	}

	// 4. 准备命令
	// export ANSIBLE_HOST_KEY_CHECKING=False 避免首次连接的主机确认提示
	args := []string{"-vv", "-i", invPath, pbPath}
	if len(extraVars) > 0 {
		evPath := filepath.Join(tmpDir, "extra_vars.json")
		b, err := json.Marshal(extraVars)
		if err != nil {
			return fmt.Errorf("marshal extra vars failed: %w", err)
		}
		if err := os.WriteFile(evPath, b, 0644); err != nil {
			return fmt.Errorf("write extra vars failed: %w", err)
		}
		args = append(args, "--extra-vars", "@"+evPath)
	}

	if r.binPath == "" {
		if r.docker != "" {
			dockerArgs := []string{"-vv", "-i", "hosts.ini", "site.yaml"}
			if len(extraVars) > 0 {
				dockerArgs = append(dockerArgs, "--extra-vars", "@extra_vars.json")
			}
			return r.runWithDocker(ctx, tmpDir, dockerArgs, onLine)
		}
		return errors.New("missing dependencies on backend host: ansible-playbook and docker not found")
	}

	cmd := exec.CommandContext(ctx, r.binPath, args...)
	cmd.Env = append(os.Environ(),
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"ANSIBLE_FORCE_COLOR=False",
		"PYTHONUNBUFFERED=1",
	)

	// 5. 管道处理
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout // 合并 stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ansible failed: %w", err)
	}

	// 6. 实时读取日志
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		onLine(scanner.Text())
	}

	// 7. 等待结束
	if err := cmd.Wait(); err != nil {
		onLine(fmt.Sprintf("[Error] %v", err))
		return err
	}
	return nil
}

func (r *Runner) runWithDocker(ctx context.Context, tmpDir string, ansibleArgs []string, onLine func(string)) error {
	image := strings.TrimSpace(r.image)
	if image == "" {
		image = "cytopia/ansible:latest"
	}
	workDir := "/work"

	quotedArgs := make([]string, 0, len(ansibleArgs))
	for _, a := range ansibleArgs {
		quotedArgs = append(quotedArgs, shellQuoteSingle(a))
	}
	shellCmd := strings.Join([]string{
		`set -e`,
		`mkdir -p /tmp/ansiblekeys`,
		`for f in ` + workDir + `/key_*; do if [ -f "$f" ]; then cp "$f" /tmp/ansiblekeys/$(basename "$f"); chmod 600 /tmp/ansiblekeys/$(basename "$f"); fi; done`,
		`ansible-playbook ` + strings.Join(quotedArgs, " "),
	}, "\n")

	dArgs := []string{
		"run", "--rm",
		"-w", workDir,
		"-v", tmpDir + ":" + workDir,
		"-e", "ANSIBLE_HOST_KEY_CHECKING=False",
		"-e", "ANSIBLE_FORCE_COLOR=False",
		"-e", "PYTHONUNBUFFERED=1",
		image,
		"sh", "-lc", shellCmd,
	}
	cmd := exec.CommandContext(ctx, r.docker, dArgs...)
	cmd.Env = os.Environ()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start docker ansible failed: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		onLine(scanner.Text())
	}
	if err := cmd.Wait(); err != nil {
		onLine(fmt.Sprintf("[Error] %v", err))
		return err
	}
	return nil
}

func (r *Runner) RunStep(ctx context.Context, stepKey string, onLine func(string)) error {
	// 兼容旧接口，暂时保留
	return r.runMock(ctx, onLine)
}

func (r *Runner) runMock(ctx context.Context, onLine func(string)) error {
	onLine("[Mock] ansible-playbook running...")
	onLine("[Mock] Warning: ansible-playbook not found in PATH")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}
	onLine("[Mock] Task failed: runner dependencies missing")
	return errors.New("ansible-playbook not found and docker not found")
}
