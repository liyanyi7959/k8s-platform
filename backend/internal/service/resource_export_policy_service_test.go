package service

import "testing"

func TestResourceExportPolicyService_SanitizeObjectMasksSecretData(t *testing.T) {
	svc := NewResourceExportPolicyService()
	input := map[string]any{
		"kind": "Secret",
		"metadata": map[string]any{
			"name": "demo",
		},
		"data": map[string]any{
			"username": "YWRtaW4=",
			"password": "c2VjcmV0",
		},
		"stringData": map[string]any{
			"token": "plain-secret",
		},
	}

	output, masked := svc.SanitizeObject("Secret", input)
	if !masked {
		t.Fatal("expected secret object to be masked")
	}

	data, _ := output["data"].(map[string]any)
	if data["username"] != "***" || data["password"] != "***" {
		t.Fatalf("expected masked secret data, got %#v", data)
	}

	stringData, _ := output["stringData"].(map[string]any)
	if stringData["token"] != "***" {
		t.Fatalf("expected masked stringData, got %#v", stringData)
	}
}

func TestResourceExportPolicyService_ExportYAMLBlocksSensitiveKind(t *testing.T) {
	svc := NewResourceExportPolicyService()

	if svc.CanExposeToAI("Secret") {
		t.Fatal("expected secret direct AI exposure to be disabled")
	}

	_, _, err := svc.ExportYAML("Secret", "apiVersion: v1\nkind: Secret\nmetadata:\n  name: demo")
	if err == nil {
		t.Fatal("expected direct secret yaml export to fail")
	}
}

func TestResourceExportPolicyService_ExportMaskedYAMLAllowsSensitiveKind(t *testing.T) {
	svc := NewResourceExportPolicyService()

	yamlText, masked, err := svc.ExportMaskedYAML("Secret", "apiVersion: v1\nkind: Secret\ndata:\n  password: c2VjcmV0\n")
	if err != nil {
		t.Fatalf("expected masked export to succeed, got %v", err)
	}
	if !masked {
		t.Fatal("expected masked export to mark result as masked")
	}
	if yamlText == "" {
		t.Fatal("expected masked yaml text")
	}
}
