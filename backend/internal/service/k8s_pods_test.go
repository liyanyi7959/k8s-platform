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
	if len(options.Command) != 1 || options.Command[0] != "sh" {
		t.Fatalf("expected default command sh, got %#v", options.Command)
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
