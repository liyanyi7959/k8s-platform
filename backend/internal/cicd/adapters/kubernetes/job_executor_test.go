package kubernetes

import (
	"context"
	"strings"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	cicdapp "k8s-platform-backend/internal/cicd/application"
)

type fakeProvider struct{ client kubeClient }

func (p fakeProvider) TypedClient(context.Context, uint64) (kubeClient, error) {
	return p.client, nil
}

func TestBuildScriptParsesStagesAndSteps(t *testing.T) {
	script, err := BuildScript("stages:\n  - name: build\n    steps:\n      - name: compile\n        run: echo hello\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "echo hello") || !strings.Contains(script, "__AIOPS_STAGE_START__") || !strings.Contains(script, "__CICD_JOB_FINISHED__") {
		t.Fatalf("script=%q", script)
	}
}

func TestBuildScriptCompilesPluginsAndMarkers(t *testing.T) {
	script, err := BuildScript("stages:\n  - key: source\n    steps:\n      - key: checkout\n        plugin: git\n        repository: https://example.com/app.git\n        branch: main\n      - key: test\n        plugin: bash\n        script: go test ./...\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"git clone", "go test ./...", "__AIOPS_STEP_START__|source|checkout", "__AIOPS_STEP_END__|source|test"} {
		if !strings.Contains(script, part) {
			t.Fatalf("script missing %q: %s", part, script)
		}
	}
}

func TestDockerLoginPlugin(t *testing.T) {
	script, err := BuildScript("stages:\n  - key: login\n    steps:\n      - key: registry\n        plugin: docker-login\n        image: harbor.example.com\n        credentials_secret: regcred\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"docker login", "harbor.example.com"} {
		if !strings.Contains(script, part) {
			t.Fatalf("script missing %q: %s", part, script)
		}
	}
}

func TestEnvInjectPlugin(t *testing.T) {
	script, err := BuildScript("stages:\n  - key: env\n    steps:\n      - key: inject\n        plugin: env-inject\n        env:\n          FOO: bar\n          BAZ: qux\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"export FOO='bar'", "export BAZ='qux'"} {
		if !strings.Contains(script, part) {
			t.Fatalf("script missing %q: %s", part, script)
		}
	}
}

func TestWaitPlugin(t *testing.T) {
	script, err := BuildScript("stages:\n  - key: wait\n    steps:\n      - key: rollout\n        plugin: wait\n        command: kubectl rollout status deployment/app\n        timeout: 30\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"kubectl rollout status deployment/app", "seq 1 30"} {
		if !strings.Contains(script, part) {
			t.Fatalf("script missing %q: %s", part, script)
		}
	}
}

func TestWebhookPlugin(t *testing.T) {
	script, err := BuildScript("stages:\n  - key: notify\n    steps:\n      - key: slack\n        plugin: webhook\n        url: https://hooks.slack.com/services/xxx\n        method: POST\n        body: '{\"text\":\"deploy done\"}'\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"curl", "POST", "hooks.slack.com"} {
		if !strings.Contains(script, part) {
			t.Fatalf("script missing %q: %s", part, script)
		}
	}
}

func TestSleepPlugin(t *testing.T) {
	script, err := BuildScript("stages:\n  - key: delay\n    steps:\n      - key: pause\n        plugin: sleep\n        timeout: 10\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "sleep 10") {
		t.Fatalf("script missing %q: %s", "sleep 10", script)
	}
}

func TestBuildScriptRejectsInvalidYAML(t *testing.T) {
	if _, err := BuildScript("stages: ["); err == nil {
		t.Fatal("expected invalid yaml error")
	}
}

func TestSubmitCreatesLabeledJob(t *testing.T) {
	client := fake.NewSimpleClientset()
	executor := NewJobExecutorWithProvider(fakeProvider{client: client}, 1)
	ref, err := executor.Submit(context.Background(), cicdapp.JobRequest{RunID: 12, PipelineID: 3, PipelineName: "frontend-ci", ClusterID: 7, Namespace: "cicd", RunnerImage: "alpine:3.20", ConfigYAML: "stages: []"})
	if err != nil {
		t.Fatal(err)
	}
	job, err := client.BatchV1().Jobs(ref.Namespace).Get(context.Background(), ref.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if job.Labels["aiops.cicd/run-id"] != "12" || job.Spec.Template.Spec.Containers[0].Image != "alpine:3.20" {
		t.Fatalf("job=%#v", job)
	}
}

func TestCancelDeletesJob(t *testing.T) {
	client := fake.NewSimpleClientset(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "cicd-job", Namespace: "cicd"}})
	executor := NewJobExecutorWithProvider(fakeProvider{client: client}, 1)
	if err := executor.Cancel(context.Background(), cicdapp.JobRef{Name: "cicd-job", Namespace: "cicd", ClusterID: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.BatchV1().Jobs("cicd").Get(context.Background(), "cicd-job", metav1.GetOptions{}); err == nil {
		t.Fatal("job still exists")
	}
}

func TestWaitReturnsWhenJobCompletes(t *testing.T) {
	client := fake.NewSimpleClientset(&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "cicd-job", Namespace: "cicd"}, Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: "True"}}}})
	executor := NewJobExecutorWithProvider(fakeProvider{client: client}, 1)
	seen := ""
	err := executor.Wait(context.Background(), cicdapp.JobRef{Name: "cicd-job", Namespace: "cicd", ClusterID: 1}, func(progress cicdapp.JobProgress) { seen = progress.Status })
	if err != nil || seen != "success" {
		t.Fatalf("err=%v status=%q", err, seen)
	}
}
