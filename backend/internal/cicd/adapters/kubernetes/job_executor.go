package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	typedbatch "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedcore "k8s.io/client-go/kubernetes/typed/core/v1"

	cicdapp "k8s-platform-backend/internal/cicd/application"
	kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"
)

type kubeClient interface {
	BatchV1() typedbatch.BatchV1Interface
	CoreV1() typedcore.CoreV1Interface
}
type clientProvider interface {
	TypedClient(context.Context, uint64) (kubeClient, error)
}
type serviceProvider struct{ service *kopsclient.K8sService }

func (p serviceProvider) TypedClient(ctx context.Context, clusterID uint64) (kubeClient, error) {
	if p.service == nil {
		return nil, errors.New("kubernetes service is not configured")
	}
	return p.service.TypedClient(ctx, clusterID)
}

type JobExecutor struct {
	clients      clientProvider
	pollInterval time.Duration
}

func NewJobExecutor(service *kopsclient.K8sService) *JobExecutor {
	return &JobExecutor{clients: serviceProvider{service: service}, pollInterval: 2 * time.Second}
}
func NewJobExecutorWithProvider(provider clientProvider, pollInterval time.Duration) *JobExecutor {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	return &JobExecutor{clients: provider, pollInterval: pollInterval}
}

func BuildScript(configYAML string) (string, error) {
	config, err := cicdapp.ParseWorkflow(configYAML)
	if err != nil {
		return "", err
	}
	lines := []string{"set -eu", "echo '__CICD_JOB_STARTED__'"}
	for _, stage := range config.Stages {
		lines = append(lines, "echo '__AIOPS_STAGE_START__|"+stage.Key+"|"+strings.ReplaceAll(stage.Name, "|", "-")+"'")
		for _, step := range stage.Steps {
			command, err := pluginCommand(step)
			if err != nil {
				return "", err
			}
			lines = append(lines, "echo '__AIOPS_STEP_START__|"+stage.Key+"|"+step.Key+"|"+strings.ReplaceAll(step.Name, "|", "-")+"'", command, "echo '__AIOPS_STEP_END__|"+stage.Key+"|"+step.Key+"'")
		}
		lines = append(lines, "echo '__AIOPS_STAGE_END__|"+stage.Key+"'")
	}
	lines = append(lines, "echo '__CICD_JOB_FINISHED__'")
	return strings.Join(lines, "\n"), nil
}
func pluginCommand(step cicdapp.WorkflowStep) (string, error) {
	for key, value := range step.Env {
		if strings.TrimSpace(key) != "" {
			step.Script = "export " + key + "=" + shellQuote(value) + "\n" + step.Script
		}
	}
	switch step.Plugin {
	case "bash", "shell":
		if strings.TrimSpace(step.Script) == "" {
			return "", fmt.Errorf("bash step %q requires script", step.Name)
		}
		return step.Script, nil
	case "git":
		if strings.TrimSpace(step.Repository) == "" {
			return "", fmt.Errorf("git step %q requires repository", step.Name)
		}
		path := step.Path
		if path == "" {
			path = "/workspace/src"
		}
		branch := ""
		if step.Branch != "" {
			branch = " --branch " + shellQuote(step.Branch)
		}
		clone := "git clone --depth 1"
		if step.CredentialsSecret != "" {
			clone = "git -c credential.helper='!f() { echo username=$GIT_USERNAME; echo password=$GIT_PASSWORD; }; f' clone --depth 1"
		}
		return "mkdir -p $(dirname " + shellQuote(path) + ")\n" + clone + branch + " " + shellQuote(step.Repository) + " " + shellQuote(path), nil
	case "kubectl", "helm":
		command := step.Command
		if command == "" {
			command = step.Run
		}
		if command == "" {
			return "", fmt.Errorf("%s step %q requires command", step.Plugin, step.Name)
		}
		return command + " " + strings.Join(step.Args, " "), nil
	case "docker-build":
		image := step.Image
		if image == "" {
			return "", fmt.Errorf("docker-build step %q requires image", step.Name)
		}
		tag := step.Tag
		if tag == "" {
			tag = "latest"
		}
		ctx := step.Context
		if ctx == "" {
			ctx = "."
		}
		return "docker build -t " + shellQuote(image+":"+tag) + " " + shellQuote(ctx) + " && docker push " + shellQuote(image+":"+tag), nil
	case "docker-login":
		registry := step.Image
		if registry == "" {
			return "", fmt.Errorf("docker-login step %q requires image (registry URL)", step.Name)
		}
		if step.CredentialsSecret == "" {
			return "", fmt.Errorf("docker-login step %q requires credentials_secret", step.Name)
		}
		return "echo \"$REGISTRY_PASSWORD\" | docker login -u \"$REGISTRY_USERNAME\" --password-stdin " + shellQuote(registry), nil
	case "env-inject":
		// env-inject: export env vars for subsequent steps in the same Job container
		if len(step.Env) == 0 {
			return "true # env-inject: no env vars defined", nil
		}
		var lines []string
		for key, value := range step.Env {
			if strings.TrimSpace(key) != "" {
				lines = append(lines, "export "+key+"="+shellQuote(value))
			}
		}
		if len(lines) == 0 {
			return "true # env-inject: no valid env vars", nil
		}
		return strings.Join(lines, "\n"), nil
	case "wait":
		// wait: poll a command until it succeeds or timeout
		checkCmd := step.Command
		if checkCmd == "" {
			return "", fmt.Errorf("wait step %q requires command", step.Name)
		}
		timeout := step.Timeout
		if timeout <= 0 {
			timeout = 60
		}
		return fmt.Sprintf("for i in $(seq 1 %d); do if %s; then exit 0; fi; sleep 1; done; echo 'timeout after %ds'; exit 1", timeout, checkCmd, timeout), nil
	case "webhook":
		// webhook: send HTTP request (GET/POST) to a URL
		url := step.URL
		if url == "" {
			return "", fmt.Errorf("webhook step %q requires url", step.Name)
		}
		method := strings.ToUpper(strings.TrimSpace(step.Method))
		if method == "" {
			method = "GET"
		}
		cmd := "curl -sS -X " + method + " " + shellQuote(url)
		if step.Body != "" {
			cmd += " -H 'Content-Type: application/json' -d " + shellQuote(step.Body)
		}
		return cmd, nil
	case "sleep":
		seconds := step.Timeout
		if seconds <= 0 {
			seconds = 1
		}
		return fmt.Sprintf("sleep %d", seconds), nil
	default:
		return "", fmt.Errorf("unsupported pipeline plugin %q", step.Plugin)
	}
}
func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }

func (e *JobExecutor) Submit(ctx context.Context, request cicdapp.JobRequest) (cicdapp.JobRef, error) {
	if e == nil || e.clients == nil {
		return cicdapp.JobRef{}, errors.New("kubernetes client is not configured")
	}
	if request.ClusterID == 0 || strings.TrimSpace(request.Namespace) == "" {
		return cicdapp.JobRef{}, errors.New("job cluster and namespace are required")
	}
	script, err := BuildScript(request.ConfigYAML)
	if err != nil {
		return cicdapp.JobRef{}, err
	}
	image := strings.TrimSpace(request.RunnerImage)
	if image == "" {
		image = "alpine:3.20"
	}
	name := sanitizeName(request.PipelineName)
	workflow, _ := cicdapp.ParseWorkflow(request.ConfigYAML)
	secretRefs := map[string]bool{}
	var envFrom []corev1.EnvFromSource
	for _, stage := range workflow.Stages {
		for _, step := range stage.Steps {
			if secret := strings.TrimSpace(step.CredentialsSecret); secret != "" && !secretRefs[secret] {
				secretRefs[secret] = true
				envFrom = append(envFrom, corev1.EnvFromSource{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: secret}}})
			}
		}
	}
	labels := map[string]string{"app.kubernetes.io/managed-by": "aiops-cicd", "aiops.cicd/run-id": fmt.Sprint(request.RunID), "aiops.cicd/pipeline-id": fmt.Sprint(request.PipelineID)}
	backoff := int32(0)
	ttl := int32(3600)
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{GenerateName: "cicd-" + name + "-", Namespace: request.Namespace, Labels: labels, Annotations: map[string]string{"aiops.cicd/config-sha": fmt.Sprintf("%x", request.RunID)}}, Spec: batchv1.JobSpec{BackoffLimit: &backoff, TTLSecondsAfterFinished: &ttl, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{RestartPolicy: corev1.RestartPolicyNever, Containers: []corev1.Container{{Name: "runner", Image: image, WorkingDir: "/workspace", Command: []string{"/bin/sh", "-c", script}, Env: []corev1.EnvVar{{Name: "AIOPS_CICD_RUN_ID", Value: fmt.Sprint(request.RunID)}}, EnvFrom: envFrom}}}}}}
	client, err := e.clients.TypedClient(ctx, request.ClusterID)
	if err != nil {
		return cicdapp.JobRef{}, err
	}
	if err := ensureNamespace(ctx, client, request.Namespace); err != nil {
		return cicdapp.JobRef{}, err
	}
	created, err := client.BatchV1().Jobs(request.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return cicdapp.JobRef{}, err
	}
	return cicdapp.JobRef{Name: created.Name, Namespace: request.Namespace, ClusterID: request.ClusterID}, nil
}

func ensureNamespace(ctx context.Context, client kubeClient, namespace string) error {
	_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil || !apierrors.IsNotFound(err) {
		return err
	}
	_, err = client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{"app.kubernetes.io/managed-by": "aiops-cicd"}}}, metav1.CreateOptions{})
	return err
}

func (e *JobExecutor) Wait(ctx context.Context, ref cicdapp.JobRef, progress func(cicdapp.JobProgress)) error {
	if e == nil || e.clients == nil {
		return errors.New("kubernetes client is not configured")
	}
	client, err := e.clients.TypedClient(ctx, ref.ClusterID)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(e.pollInterval)
	defer ticker.Stop()
	var lastLog string
	for {
		job, getErr := client.BatchV1().Jobs(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				return errors.New("kubernetes job was deleted")
			}
			return getErr
		}
		if logText := e.readPodLog(ctx, client, ref.Namespace, ref.Name); logText != "" && logText != lastLog {
			lastLog = logText
			current := parseProgress(logText)
			current.Log = logText
			progress(current)
		}
		if jobFailed(job) {
			progress(cicdapp.JobProgress{Status: "failed", Log: lastLog, Finished: true})
			return errors.New("kubernetes job failed")
		}
		if jobComplete(job) {
			progress(cicdapp.JobProgress{Status: "success", Log: lastLog, Finished: true, Succeeded: true})
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
func parseProgress(logText string) cicdapp.JobProgress {
	lines := strings.Split(logText, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		parts := strings.Split(lines[i], "|")
		if len(parts) < 2 {
			continue
		}
		switch parts[0] {
		case "__AIOPS_STEP_START__":
			if len(parts) >= 4 {
				return cicdapp.JobProgress{Status: "running", StageKey: parts[1], StepKey: parts[2]}
			}
		case "__AIOPS_STEP_END__":
			if len(parts) >= 3 {
				return cicdapp.JobProgress{Status: "success", StageKey: parts[1], StepKey: parts[2]}
			}
		case "__AIOPS_STAGE_START__":
			if len(parts) >= 2 {
				return cicdapp.JobProgress{Status: "running", StageKey: parts[1]}
			}
		}
	}
	return cicdapp.JobProgress{Status: "running"}
}

func (e *JobExecutor) Cancel(ctx context.Context, ref cicdapp.JobRef) error {
	if e == nil || e.clients == nil {
		return errors.New("kubernetes client is not configured")
	}
	if ref.Name == "" || ref.Namespace == "" || ref.ClusterID == 0 {
		return nil
	}
	client, err := e.clients.TypedClient(ctx, ref.ClusterID)
	if err != nil {
		return err
	}
	return client.BatchV1().Jobs(ref.Namespace).Delete(ctx, ref.Name, metav1.DeleteOptions{PropagationPolicy: ptr(metav1.DeletePropagationBackground)})
}
func (e *JobExecutor) readPodLog(ctx context.Context, client kubeClient, namespace, jobName string) string {
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.Set{"job-name": jobName}.AsSelector().String()})
	if err != nil || len(pods.Items) == 0 {
		return ""
	}
	stream, err := client.CoreV1().Pods(namespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{Container: "runner"}).Stream(ctx)
	if err != nil {
		return ""
	}
	defer stream.Close()
	data, err := io.ReadAll(stream)
	if err != nil {
		return ""
	}
	return string(data)
}
func jobComplete(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
func jobFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
func sanitizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "pipeline"
	}
	if len(result) > fortyNine {
		return result[:fortyNine]
	}
	return result
}

const fortyNine = 49

func ptr[T any](value T) *T { return &value }
