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
