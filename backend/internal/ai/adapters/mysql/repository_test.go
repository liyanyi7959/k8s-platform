package mysql

import (
	"testing"

	"k8s-platform-backend/internal/ai/ports"
)

func TestProviderUpdatesCanClearEncryptedAPIKey(t *testing.T) {
	var cleared *string
	updates := providerUpdates(ports.ProviderPatch{APIKeyEnc: &cleared})
	value, ok := updates["api_key_enc"]
	if !ok || value != nil {
		t.Fatalf("api_key_enc clear update = %#v, want nil", value)
	}
}

func TestProviderUpdatesKeepsEncryptedAPIKeyValue(t *testing.T) {
	encoded := "ciphertext"
	pointer := &encoded
	updates := providerUpdates(ports.ProviderPatch{APIKeyEnc: &pointer})
	if got := updates["api_key_enc"]; got != encoded {
		t.Fatalf("api_key_enc update = %#v, want %q", got, encoded)
	}
}
