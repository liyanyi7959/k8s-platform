package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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
	{Key: "install_addons", Title: "安装 Kubernetes 扩展组件", PlayName: "安装 Kubernetes 扩展组件"},
	{Key: "register", Title: "节点注册到管理平台", PlayName: "节点注册到管理平台"},
}

// computeEnabledSteps 根据起始步骤 key 返回需要执行的步骤 key 列表。
// retryFromStep 为空时返回 nil，表示执行全部步骤（不传 retry_enabled_steps）。
func computeEnabledSteps(retryFromStep string) []string {
	if retryFromStep == "" {
		return nil
	}
	started := false
	var steps []string
	for _, step := range ansibleSteps {
		if step.Key == retryFromStep {
			started = true
		}
		if started {
			steps = append(steps, step.Key)
		}
	}
	return steps
}

// ansibleLogWriter 自定义 io.Writer，逐行捕获 Ansible 输出并写入 Task 日志
type ansibleLogWriter struct {
	task      *Task
	store     *TaskStore
	mu        sync.Mutex
	buf       []byte
	stepIndex int // 当前执行的步骤索引
	stepDone  bool
}

func newAnsibleLogWriter(task *Task, store *TaskStore) *ansibleLogWriter {
	return &ansibleLogWriter{
		task:      task,
		store:     store,
		stepIndex: -1,
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

		// 写入日志（带当前 step key）
		w.task.AppendLog(line, w.currentStepKey())
	}

	// 批量保存任务状态
	_ = w.store.Put(w.task)

	return len(p), nil
}

func (w *ansibleLogWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		w.task.AppendLog(string(w.buf), w.currentStepKey())
		w.buf = nil
		_ = w.store.Put(w.task)
	}
}

// currentStepKey 返回当前执行步骤的 key。
func (w *ansibleLogWriter) currentStepKey() string {
	if w.stepIndex >= 0 && w.stepIndex < len(ansibleSteps) {
		return ansibleSteps[w.stepIndex].Key
	}
	return ""
}

// parseStepProgress 从 Ansible 输出行中解析步骤进度
// 匹配 "PLAY [名称]" 更新大步骤，匹配 "TASK [role : task]" 更新子步骤。
func (w *ansibleLogWriter) parseStepProgress(line string) {
	trimmed := strings.TrimSpace(line)
	now := time.Now().UTC()

	// 检测 PLAY 开始：PLAY [环境预检 - 所有节点] 或 PLAY [Kubernetes Master 初始化]
	if strings.HasPrefix(trimmed, "PLAY [") {
		playName := extractPlayName(trimmed)
		for i, step := range ansibleSteps {
			if strings.Contains(playName, step.PlayName) {
				// 标记前序步骤为完成
				for j := 0; j < i && j < len(w.task.Steps); j++ {
					if w.task.Steps[j].Status == StepRunning {
						w.task.Steps[j].Status = StepSuccess
						w.task.Steps[j].FinishedAt = &now
						w.finishRunningSubStep(j, now)
					}
				}
				// 标记当前步骤为执行中（重试场景下已被标记为 success 的步骤不覆盖，避免把跳过的步骤重新置为 running）
				if i < len(w.task.Steps) && w.task.Steps[i].Status != StepSuccess {
					w.task.Steps[i].Status = StepRunning
					if w.task.Steps[i].StartedAt == nil {
						w.task.Steps[i].StartedAt = &now
					}
				}
				percent := i * 100 / len(ansibleSteps)
				w.task.Percent = &percent
				w.stepIndex = i
				break
			}
		}
	}

	// 检测 TASK 开始
	if strings.HasPrefix(trimmed, "TASK [") {
		taskName := extractPlayName(trimmed)
		if taskName != "" && w.stepIndex >= 0 && w.stepIndex < len(w.task.Steps) {
			step := &w.task.Steps[w.stepIndex]
			// 结束当前运行中的子步骤
			w.finishRunningSubStep(w.stepIndex, now)
			// 查找或创建子步骤
			found := false
			for k := range step.SubSteps {
				if step.SubSteps[k].Title == taskName {
					step.SubSteps[k].Status = StepRunning
					step.SubSteps[k].StartedAt = &now
					step.SubSteps[k].FinishedAt = nil
					found = true
					break
				}
			}
			if !found {
				key := fmt.Sprintf("%s-%d", step.Key, len(step.SubSteps))
				step.SubSteps = append(step.SubSteps, TaskSubStep{
					Key:       key,
					Title:     taskName,
					Status:    StepRunning,
					StartedAt: &now,
				})
			}
		}
	}

	// 检测 PLAY RECAP（全部完成）
	if strings.HasPrefix(trimmed, "PLAY RECAP") {
		for i := range w.task.Steps {
			if w.task.Steps[i].Status == StepRunning {
				w.task.Steps[i].Status = StepSuccess
				w.task.Steps[i].FinishedAt = &now
				w.finishRunningSubStep(i, now)
			}
		}
	}

	// 检测失败：failed=1
	if strings.Contains(trimmed, "failed=1") || strings.Contains(trimmed, "FAILED") {
		if w.stepIndex >= 0 && w.stepIndex < len(w.task.Steps) {
			w.task.Steps[w.stepIndex].Status = StepFailed
			w.task.Steps[w.stepIndex].FinishedAt = &now
			w.finishRunningSubStep(w.stepIndex, now)
		}
	}
}

// finishRunningSubStep 将指定大步骤下运行中的子步骤标记为完成（根据最终状态）。
func (w *ansibleLogWriter) finishRunningSubStep(stepIdx int, t time.Time) {
	if stepIdx < 0 || stepIdx >= len(w.task.Steps) {
		return
	}
	step := &w.task.Steps[stepIdx]
	for k := range step.SubSteps {
		if step.SubSteps[k].Status == StepRunning {
			step.SubSteps[k].FinishedAt = &t
			if step.Status == StepFailed {
				step.SubSteps[k].Status = StepFailed
			} else {
				step.SubSteps[k].Status = StepSuccess
			}
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
		execute.WithEnvVar("ANSIBLE_HOST_KEY_CHECKING", "False"),
		execute.WithEnvVar("ANSIBLE_RETRY_FILES_ENABLED", "False"),
		execute.WithEnvVar("ANSIBLE_NOCOLOR", "True"),
	)

	// 连接选项
	connOptions := &options.AnsibleConnectionOptions{
		Connection: "ssh",
		Timeout:    30,
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
		Playbooks:                  []string{opts.PlaybookPath},
		Options:                    pbOptions,
		ConnectionOptions:          connOptions,
		PrivilegeEscalationOptions: privOptions,
		Exec:                       exec,
		StdoutCallback:             "default",
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
