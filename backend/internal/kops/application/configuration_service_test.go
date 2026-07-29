package application

import (
	"context"
	"strings"
	"testing"
)

type configurationRuntimeSpy struct{ configMap ConfigMapEditInput }

func (*configurationRuntimeSpy) List(context.Context, ConfigurationResource, ConfigurationListQuery) (any, error) {
	return nil, nil
}
func (*configurationRuntimeSpy) YAML(context.Context, ConfigurationResource, ConfigurationReference) (any, error) {
	return nil, nil
}
func (*configurationRuntimeSpy) Delete(context.Context, ConfigurationResource, ConfigurationReference) error {
	return nil
}
func (spy *configurationRuntimeSpy) EditConfigMap(_ context.Context, value ConfigMapEditInput) error {
	spy.configMap = value
	return nil
}
func (*configurationRuntimeSpy) EditSecret(context.Context, SecretEditInput) error { return nil }
func (*configurationRuntimeSpy) SecretObject(context.Context, ConfigurationReference) (map[string]any, error) {
	return nil, nil
}
func (*configurationRuntimeSpy) Pods(context.Context, uint64, string) ([]any, error) { return nil, nil }

func TestConfigurationServiceNormalizesConfigMapAndRedactsSecretData(t *testing.T) {
	spy := &configurationRuntimeSpy{}
	service := NewConfigurationService(spy)
	if err := service.EditConfigMap(context.Background(), ConfigMapEditInput{ClusterID: 2, Namespace: " ops ", Name: " api "}); err != nil {
		t.Fatalf("EditConfigMap() error=%v", err)
	}
	if spy.configMap.Namespace != "ops" || spy.configMap.Name != "api" {
		t.Fatalf("input=%#v", spy.configMap)
	}
	text, err := secretRevealText(map[string]any{"apiVersion": "v1", "kind": "Secret", "metadata": map[string]any{"namespace": "ops", "name": "api"}, "data": map[string]any{"token": "aGVsbG8="}})
	if err != nil || !strings.Contains(text, "hello") {
		t.Fatalf("text=%q err=%v", text, err)
	}
}
