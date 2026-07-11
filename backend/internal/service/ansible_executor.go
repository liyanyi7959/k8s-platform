package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/options"
	"github.com/apenella/go-ansible/pkg/playbook"
)

// ansibleStepDef 定义 Ansible 部署流水线步骤
type ansibleStepDef struct {
	Key   string
	Title string
	// PlayName 是 site.yml 中 PLAY 的名称，用于匹配输出
	PlayName string
}

// ansibleSteps 定义了 7 个部署步骤与 site.yml 中 PLAY 名称的映射
var ansibleSteps = []ansibleStepDef{
	{Key: "pre_check", Title: "环境预检", PlayName: "环境预检"},
	{Key: "bootstrap", Title: "基础环境初始化", PlayName: "基础环境初始化"},
	{Key: "container_runtime", Title: "容器运行时安装", PlayName: "容器运行时安装"},
	{Key: "kubeadm_init", Title: "Kubernetes Master 初始化", PlayName: "Kubernetes Master 初始化"},
	{Key: "join_workers", Title: "Worker 节点加入集群", PlayName: "Worker 节点加入集群"},
	{Key: "install_cni", Title: "安装 CNI 网络插件", PlayName: "安装 CNI 网络插件"},
	{Key: "register", Title: "节点注册到管理平台", PlayName: "节点注册到管理平台"},
}

// ansibleLogWriter 自定义 io.Writer，逐行捕获 Ansible 输出并写入 Task 日志
type ansibleLogWriter struct {
	task       *Task
	store      *TaskStore
	mu         sync.Mutex
	buf        []byte
	stepIndex  int // 当前执行的步骤索引
	stepDone   bool
}

func newAnsibleLogWriter(task *Task, store *TaskStore) *ansibleLogWriter {
	return &ansibleLogWriter{
		task:  task,
		store: store,
	}
}

// Write 实现 io.Writer 接口，逐行解析 Ansible 输出
func (w *ansibleLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)

	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx == -1 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]

		// 解析步骤进度
		w.parseStepProgress(line)

		// 写入日志
		w.task.AppendLog(line)
	}

	// 批量保存任务状态
	_ = w.store.Put(w.task)

	return len(p), nil
}

// parseStepProgress 从 Ansible 输出行中解析步骤进度
// 匹配 "PLAY [名称]" 格式的行来更新步骤状态
func (w *ansibleLogWriter) parseStepProgress(line string) {
	trimmed := strings.TrimSpace(line)

	// 检测 PLAY 开始：PLAY [环境预检 - 所有节点] 或 PLAY [Kubernetes Master 初始化]
	if strings.HasPrefix(trimmed, "PLAY [") {
		playName := extractPlayName(trimmed)
		for i, step := range ansibleSteps {
			if strings.Contains(playName, step.PlayName) {
				// 标记前序步骤为完成
				for j := 0; j < i && j < len(w.task.Steps); j++ {
					if w.task.Steps[j].Status == StepRunning {
						w.task.Steps[j].Status = StepSuccess
					}
				}
				// 标记当前步骤为执行中
				if i < len(w.task.Steps) {
					w.task.Steps[i].Status = StepRunning
				}
				w.stepIndex = i
				break
			}
		}
	}

	// 检测 PLAY RECAP（全部完成）
	if strings.HasPrefix(trimmed, "PLAY RECAP") {
		for i := range w.task.Steps {
			if w.task.Steps[i].Status == StepRunning {
				w.task.Steps[i].Status = StepSuccess
			}
		}
	}

	// 检测失败：failed=1
	if strings.Contains(trimmed, "failed=1") || strings.Contains(trimmed, "FAILED") {
		if w.stepIndex < len(w.task.Steps) {
			w.task.Steps[w.stepIndex].Status = StepFailed
		}
	}
}

// extractPlayName 从 "PLAY [xxx]" 行中提取方括号内的名称
func extractPlayName(line string) string {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return line[start+1 : end]
}

// ansibleExecuteOptions Ansible 执行参数
type ansibleExecuteOptions struct {
	PlaybookPath string
	Inventory    string
	ExtraVars    map[string]interface{}
	CmdRunDir    string
}

// runAnsiblePlaybook 执行 Ansible Playbook
// 使用 go-ansible 库执行 playbook，输出通过自定义 writer 实时写入 task 日志
func (s *DeployService) runAnsiblePlaybook(ctx context.Context, opts ansibleExecuteOptions, task *Task) error {
	// 创建自定义日志写入器
	writer := newAnsibleLogWriter(task, s.taskStore)

	// 创建执行器
	exec := execute.NewDefaultExecute(
		execute.WithWrite(writer),
		execute.WithWriteError(writer), // stderr 也写入同一 writer
		execute.WithCmdRunDir(opts.CmdRunDir),
	)

	// 连接选项
	connOptions := &options.AnsibleConnectionOptions{
		Connection: "ssh",
		Timeout:     30,
	}

	// 提权选项
	privOptions := &options.AnsiblePrivilegeEscalationOptions{
		Become:       true,
		BecomeMethod: "sudo",
	}

	// Playbook 选项
	pbOptions := &playbook.AnsiblePlaybookOptions{
		Inventory: opts.Inventory,
		ExtraVars: opts.ExtraVars,
		Forks:     "10",
	}

	// 创建 playbook 命令
	cmd := &playbook.AnsiblePlaybookCmd{
		Playbooks:               []string{opts.PlaybookPath},
		Options:                 pbOptions,
		ConnectionOptions:       connOptions,
		PrivilegeEscalationOptions: privOptions,
		Exec:                    exec,
		StdoutCallback:          "default",
	}

	// 记录开始
	task.AppendLog(fmt.Sprintf("[info] 开始执行 Ansible Playbook: %s", opts.PlaybookPath))
	_ = s.taskStore.Put(task)

	// 执行
	err := cmd.Run(ctx)

	// 刷新缓冲区中剩余的日志
	writer.mu.Lock()
	if len(writer.buf) > 0 {
		task.AppendLog(string(writer.buf))
		writer.buf = nil
	}
	writer.mu.Unlock()

	if err != nil {
		task.AppendLog(fmt.Sprintf("[error] Ansible Playbook 执行失败: %v", err))
		_ = s.taskStore.Put(task)
		return err
	}

	task.AppendLog("[info] Ansible Playbook 执行完成")
	_ = s.taskStore.Put(task)
	return nil
}
