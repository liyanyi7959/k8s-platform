package service

import "testing"

func TestAnsibleLogWriterStopsStepProgressAfterFailure(t *testing.T) {
	task := &Task{Steps: make([]TaskStep, len(ansibleSteps))}
	for i, definition := range ansibleSteps {
		task.Steps[i] = TaskStep{Key: definition.Key, Title: definition.Title, Status: StepPending}
	}
	task.Steps[0].Status = StepSuccess
	task.Steps[1].Status = StepSuccess

	writer := newAnsibleLogWriter(task, nil)
	writer.parseStepProgress("PLAY [容器运行时安装 - 所有节点]")
	writer.parseStepProgress("TASK [container_runtime : 安装 containerd]")
	writer.parseStepProgress("fatal: [worker-1]: FAILED! => {\"msg\": \"package unavailable\"}")
	writer.parseStepProgress("PLAY [Kubernetes Master 初始化]")
	writer.parseStepProgress("PLAY [Worker 节点加入集群]")

	if got := task.Steps[2].Status; got != StepFailed {
		t.Fatalf("failed step status = %q, want %q", got, StepFailed)
	}
	if got := writer.currentStepKey(); got != "container_runtime" {
		t.Fatalf("current step key = %q, want container_runtime", got)
	}
	for i := 3; i < len(task.Steps); i++ {
		if got := task.Steps[i].Status; got != StepPending {
			t.Fatalf("later step %s status = %q, want %q", task.Steps[i].Key, got, StepPending)
		}
	}
}

func TestHasAnsibleRecapFailure(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{line: "worker-1 : ok=2 changed=1 unreachable=0 failed=0 skipped=0", want: false},
		{line: "worker-1 : ok=2 changed=1 unreachable=0 failed=2 skipped=0", want: true},
		{line: "worker-1 : ok=0 changed=0 unreachable=1 failed=0 skipped=0", want: true},
	}
	for _, tt := range tests {
		if got := hasAnsibleRecapFailure(tt.line); got != tt.want {
			t.Fatalf("hasAnsibleRecapFailure(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestAnsibleLogWriterKeepsAddonTaskSubstepsInSelectedPlay(t *testing.T) {
	task := &Task{
		Type:  "install_cluster_addons",
		Meta:  map[string]any{"enabled_steps": []string{"install_addons"}},
		Steps: []TaskStep{{Key: "install_addons", Title: "补充安装集群组件", Status: StepRunning}},
	}
	writer := newAnsibleLogWriter(task, nil)

	writer.parseStepProgress("PLAY [环境预检 - 所有节点]")
	writer.parseStepProgress("TASK [pre_check : 检查主机连通性]")
	if len(task.Steps[0].SubSteps) != 0 {
		t.Fatalf("unselected play should not create add-on substeps: %#v", task.Steps[0].SubSteps)
	}

	writer.parseStepProgress("PLAY [安装 Kubernetes 扩展组件]")
	writer.parseStepProgress("TASK [install_addons : 安装 metrics-server]")
	writer.parseStepProgress("fatal: [master-1]: FAILED! => {\"msg\": \"install failed\"}")

	if got := task.Steps[0].Status; got != StepFailed {
		t.Fatalf("add-on step status = %q, want %q", got, StepFailed)
	}
	if len(task.Steps[0].SubSteps) != 1 || task.Steps[0].SubSteps[0].Status != StepFailed {
		t.Fatalf("add-on substeps = %#v, want one failed install task", task.Steps[0].SubSteps)
	}
}
