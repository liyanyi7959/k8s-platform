package service

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestNormalizePVCAccessModesDefaults(t *testing.T) {
	modes, err := normalizePVCAccessModes(nil)
	if err != nil {
		t.Fatalf("normalizePVCAccessModes() error = %v", err)
	}
	if len(modes) != 1 || modes[0] != corev1.ReadWriteOnce {
		t.Fatalf("unexpected modes: %#v", modes)
	}
}

func TestNormalizePVCAccessModesRejectsInvalid(t *testing.T) {
	if _, err := normalizePVCAccessModes([]string{"invalid"}); err == nil {
		t.Fatal("expected error for invalid access mode")
	}
}
