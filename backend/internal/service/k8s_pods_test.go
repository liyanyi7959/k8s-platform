package service

import (
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestBuildPodExecOptions_DisablesStderrWhenTTY(t *testing.T) {
	container := " app "
	options := buildPodExecOptions(&container, nil, true, testReader{}, testWriter{}, testWriter{})

	if options.Container != "app" {
		t.Fatalf("expected trimmed container, got %q", options.Container)
	}
	if !options.TTY {
		t.Fatal("expected tty enabled")
	}
	if options.Stderr {
		t.Fatal("expected stderr disabled when tty is enabled")
	}
	if len(options.Command) != 1 || options.Command[0] != "/bin/sh" {
		t.Fatalf("expected default command /bin/sh, got %#v", options.Command)
	}
	if !options.Stdin || !options.Stdout {
		t.Fatal("expected stdin/stdout enabled")
	}
}

func TestBuildPodExecOptions_EnablesStderrWithoutTTY(t *testing.T) {
	options := buildPodExecOptions(nil, []string{"bash"}, false, nil, testWriter{}, testWriter{})

	if options.TTY {
		t.Fatal("expected tty disabled")
	}
	if !options.Stderr {
		t.Fatal("expected stderr enabled when tty is disabled")
	}
	if len(options.Command) != 1 || options.Command[0] != "bash" {
		t.Fatalf("expected command bash, got %#v", options.Command)
	}
	if options.Stdin {
		t.Fatal("expected stdin disabled")
	}
	if !options.Stdout {
		t.Fatal("expected stdout enabled")
	}
}

func TestBuildPodLogOptions_UsesTailLinesWhenPositive(t *testing.T) {
	tailLines := int64(120)
	options := buildPodLogOptions(" app ", true, tailLines, false)

	if options.Container != "app" {
		t.Fatalf("expected trimmed container, got %q", options.Container)
	}
	if !options.Follow {
		t.Fatal("expected follow enabled")
	}
	if options.TailLines == nil || *options.TailLines != tailLines {
		t.Fatalf("expected tail lines %d, got %#v", tailLines, options.TailLines)
	}
}

func TestBuildPodLogOptions_OmitsTailLinesWhenZero(t *testing.T) {
	options := buildPodLogOptions("", false, 0, false)

	if options.Follow {
		t.Fatal("expected follow disabled")
	}
	if options.TailLines != nil {
		t.Fatalf("expected nil tail lines for full log request, got %#v", *options.TailLines)
	}
}

func TestBuildPodLogOptions_UsesPreviousInstanceWhenRequested(t *testing.T) {
	options := buildPodLogOptions("worker", false, 200, true)

	if !options.Previous {
		t.Fatal("expected previous log flag enabled")
	}
	if options.Container != "worker" {
		t.Fatalf("expected worker container, got %q", options.Container)
	}
}

func TestNormalizePodExecError_KubeletRefused(t *testing.T) {
	pod := &corev1.Pod{}
	pod.Spec.NodeName = "node4"
	err := normalizePodExecError(errors.New("error dialing backend: dial tcp 10.11.27.35:10250: connect: connection refused"), pod, []string{"sh"})

	if !errors.Is(err, ErrK8sNetwork) {
		t.Fatalf("expected ErrK8sNetwork, got %v", err)
	}
	msg, ok := UserMessage(err)
	if !ok {
		t.Fatal("expected user message")
	}
	if msg == "" || !containsAll(msg, []string{"node4", "10250", "kubelet"}) {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestNormalizePodExecError_CommandMissing(t *testing.T) {
	err := normalizePodExecError(errors.New("OCI runtime exec failed: exec failed: unable to start container process: exec: \"sh\": executable file not found in $PATH"), nil, []string{"sh"})

	if !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("expected ErrInvalidParams, got %v", err)
	}
	msg, ok := UserMessage(err)
	if !ok || msg == "" {
		t.Fatal("expected user message")
	}
	if !containsAll(msg, []string{"sh", "/bin/sh", "bash"}) {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestPodExecCommandCandidates_FallbackShells(t *testing.T) {
	candidates := podExecCommandCandidates([]string{"sh"})
	got := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if len(candidate) != 1 {
			t.Fatalf("unexpected candidate: %#v", candidate)
		}
		got = append(got, candidate[0])
	}

	if !containsAll(strings.Join(got, ","), []string{"sh", "/bin/sh", "bash", "/bin/bash", "ash", "/bin/ash"}) {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}

func TestPodExecCommandCandidates_PreservesExplicitCommand(t *testing.T) {
	candidates := podExecCommandCandidates([]string{"ls", "-al"})
	if len(candidates) != 1 {
		t.Fatalf("expected single explicit candidate, got %#v", candidates)
	}
	if len(candidates[0]) != 2 || candidates[0][0] != "ls" || candidates[0][1] != "-al" {
		t.Fatalf("unexpected explicit candidate: %#v", candidates)
	}
}

func containsAll(text string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(text, sub) {
			return false
		}
	}
	return true
}

type testReader struct{}

func (testReader) Read(p []byte) (int, error) {
	return 0, nil
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
