package provisioning

import (
	"strings"
	"testing"

	platformapp "k8s-platform-backend/internal/platform/application"
	provisionapp "k8s-platform-backend/internal/provisioning/application"
	model "k8s-platform-backend/internal/provisioning/domain"
	"k8s.io/client-go/tools/clientcmd"
)

func TestAnsibleRunnerRequiresPersistence(t *testing.T) {
	runner := NewAnsibleRunner(nil, "", nil)
	_, err := runner.Run(t.Context(), model.DeployPlan{}, nil, &platformapp.Task{})
	if err == nil || !strings.Contains(err.Error(), "未初始化") {
		t.Fatalf("Run without persistence error = %v, want runner initialization failure", err)
	}
}

func TestRunnerBootstrapScriptIncludesRequiredDependencies(t *testing.T) {
	withPassword := runnerBootstrapScript(true)
	for _, expected := range []string{"ansible-playbook", "apt-get", "dnf", "yum", "sshpass", "python3 -m pip install ansible-core"} {
		if !strings.Contains(withPassword, expected) {
			t.Fatalf("bootstrap script is missing %q", expected)
		}
	}
	withoutPassword := runnerBootstrapScript(false)
	if strings.Contains(withoutPassword, "install -y tar gzip sshpass") {
		t.Fatal("key-only runner must not require sshpass installation")
	}
}

func TestNormalizeMasterKubeconfig(t *testing.T) {
	raw := `apiVersion: v1
kind: Config
clusters:
- name: cluster
  cluster:
    server: https://127.0.0.1:6443
contexts: []
users: []
`
	normalized, err := normalizeMasterKubeconfig(raw, "192.0.2.10")
	if err != nil {
		t.Fatal(err)
	}
	config, err := clientcmd.Load([]byte(normalized))
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Clusters["cluster"].Server; got != "https://192.0.2.10:6443" {
		t.Fatalf("unexpected API server: %s", got)
	}
}

func TestRunnerShellQuote(t *testing.T) {
	if got := runnerShellQuote("a'b"); got != `'a'"'"'b'` {
		t.Fatalf("unexpected shell quote: %s", got)
	}
}

func TestAnsibleLogWriterStopsStepProgressAfterFailure(t *testing.T) {
	steps := provisionapp.DefaultAnsibleSteps()
	task := &platformapp.Task{Steps: make([]platformapp.TaskStep, len(steps))}
	for index, definition := range steps {
		task.Steps[index] = platformapp.TaskStep{Key: definition.Key, Title: definition.Title, Status: platformapp.StepPending}
	}
	task.Steps[0].Status = platformapp.StepSuccess
	task.Steps[1].Status = platformapp.StepSuccess

	writer := newAnsibleLogWriter(task, nil)
	writer.parseStepProgress("PLAY [容器运行时安装 - 所有节点]")
	writer.parseStepProgress("TASK [container_runtime : 安装 containerd]")
	writer.parseStepProgress(`fatal: [worker-1]: FAILED! => {"msg": "package unavailable"}`)
	writer.parseStepProgress("PLAY [Kubernetes Master 初始化]")
	writer.parseStepProgress("PLAY [Worker 节点加入集群]")

	if got := task.Steps[2].Status; got != platformapp.StepFailed {
		t.Fatalf("failed step status = %q, want %q", got, platformapp.StepFailed)
	}
	if got := writer.currentStepKey(); got != "container_runtime" {
		t.Fatalf("current step key = %q, want container_runtime", got)
	}
	for index := 3; index < len(task.Steps); index++ {
		if got := task.Steps[index].Status; got != platformapp.StepPending {
			t.Fatalf("later step %s status = %q, want %q", task.Steps[index].Key, got, platformapp.StepPending)
		}
	}
}

func TestAnsibleLogWriterKeepsAddonTaskSubstepsInSelectedPlay(t *testing.T) {
	task := &platformapp.Task{
		Type: "install_cluster_addons", Meta: map[string]any{"enabled_steps": []string{"install_addons"}},
		Steps: []platformapp.TaskStep{{Key: "install_addons", Title: "补充安装集群组件", Status: platformapp.StepRunning}},
	}
	writer := newAnsibleLogWriter(task, nil)

	writer.parseStepProgress("PLAY [环境预检 - 所有节点]")
	writer.parseStepProgress("TASK [pre_check : 检查主机连通性]")
	if len(task.Steps[0].SubSteps) != 0 {
		t.Fatalf("unselected play should not create add-on substeps: %#v", task.Steps[0].SubSteps)
	}

	writer.parseStepProgress("PLAY [安装 Kubernetes 扩展组件]")
	writer.parseStepProgress("TASK [install_addons : 安装 metrics-server]")
	writer.parseStepProgress(`fatal: [master-1]: FAILED! => {"msg": "install failed"}`)

	if got := task.Steps[0].Status; got != platformapp.StepFailed {
		t.Fatalf("add-on step status = %q, want %q", got, platformapp.StepFailed)
	}
	if len(task.Steps[0].SubSteps) != 1 || task.Steps[0].SubSteps[0].Status != platformapp.StepFailed {
		t.Fatalf("add-on substeps = %#v, want one failed install task", task.Steps[0].SubSteps)
	}
}

func TestHasAnsibleRecapFailure(t *testing.T) {
	for _, testCase := range []struct {
		line string
		want bool
	}{
		{line: "worker-1 : ok=2 changed=1 unreachable=0 failed=0 skipped=0", want: false},
		{line: "worker-1 : ok=2 changed=1 unreachable=0 failed=2 skipped=0", want: true},
		{line: "worker-1 : ok=0 changed=0 unreachable=1 failed=0 skipped=0", want: true},
	} {
		if got := provisionapp.HasAnsibleRecapFailure(testCase.line); got != testCase.want {
			t.Fatalf("HasAnsibleRecapFailure(%q) = %v, want %v", testCase.line, got, testCase.want)
		}
	}
}
