package application

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSortObjectsByMetadata(t *testing.T) {
	items := []*corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "zeta", Namespace: "system"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "alpha", Namespace: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "beta", Namespace: "platform"}},
	}
	SortObjectsByMetadata(items, "metadata.name", "asc")
	if items[0].Name != "alpha" || items[2].Name != "zeta" {
		t.Fatalf("name ordering = %q, %q, %q", items[0].Name, items[1].Name, items[2].Name)
	}
	SortObjectsByMetadata(items, "metadata.namespace", "desc")
	if items[0].Namespace != "system" || items[2].Namespace != "default" {
		t.Fatalf("namespace ordering = %q, %q, %q", items[0].Namespace, items[1].Namespace, items[2].Namespace)
	}
	SortObjectsByMetadata(items, "unsupported", "asc")
	if items[0].Namespace != "system" {
		t.Fatal("unsupported sort must preserve order")
	}
}

func TestMaskSecretForReadKeepsOriginalUntouched(t *testing.T) {
	original := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "api-key", Namespace: "platform"},
		Data:       map[string][]byte{"token": []byte("plaintext")},
		StringData: map[string]string{"password": "plain"},
	}
	masked := MaskSecretForRead(original)
	if masked == original || string(masked.Data["token"]) != MaskedSecretValue || masked.StringData["password"] != MaskedSecretValue {
		t.Fatalf("masked secret = %#v", masked)
	}
	if string(original.Data["token"]) != "plaintext" || original.StringData["password"] != "plain" {
		t.Fatal("masking mutated original secret")
	}
	if got := MaskSecretsForRead([]*corev1.Secret{nil, original}); len(got) != 1 || got[0].Name != "api-key" {
		t.Fatalf("MaskSecretsForRead() = %#v", got)
	}
}
