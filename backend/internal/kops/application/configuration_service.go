package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type ConfigurationResource string

const (
	ConfigurationConfigMap ConfigurationResource = "configmap"
	ConfigurationSecret    ConfigurationResource = "secret"
)

type ConfigurationListQuery struct {
	ClusterID uint64
	Namespace string
	SortBy    string
	Order     string
}

type ConfigurationReference struct {
	ClusterID uint64
	Namespace string
	Name      string
}

type ConfigMapEditInput struct {
	ClusterID uint64             `json:"-"`
	Namespace string             `json:"namespace"`
	Name      string             `json:"name"`
	Labels    map[string]*string `json:"labels"`
	Data      map[string]*string `json:"data"`
}

type SecretEditInput struct {
	ClusterID uint64             `json:"-"`
	Namespace string             `json:"namespace"`
	Name      string             `json:"name"`
	Type      *string            `json:"type"`
	Labels    map[string]*string `json:"labels"`
	Data      map[string]*string `json:"data"`
}

type RelatedOwner struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	UID  string `json:"uid,omitempty"`
}

type RelatedController struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type RelatedPod struct {
	Namespace string         `json:"namespace"`
	Name      string         `json:"name"`
	Phase     string         `json:"phase"`
	Node      string         `json:"node"`
	Ready     string         `json:"ready"`
	Restarts  int            `json:"restarts"`
	Owners    []RelatedOwner `json:"owners"`
}

type RelatedPodsResult struct {
	Pods        []RelatedPod        `json:"pods"`
	Controllers []RelatedController `json:"controllers"`
}

type ConfigurationRuntime interface {
	List(context.Context, ConfigurationResource, ConfigurationListQuery) (any, error)
	YAML(context.Context, ConfigurationResource, ConfigurationReference) (any, error)
	Delete(context.Context, ConfigurationResource, ConfigurationReference) error
	EditConfigMap(context.Context, ConfigMapEditInput) error
	EditSecret(context.Context, SecretEditInput) error
	SecretObject(context.Context, ConfigurationReference) (map[string]any, error)
	Pods(context.Context, uint64, string) ([]any, error)
}

type ConfigurationService struct{ runtime ConfigurationRuntime }

func NewConfigurationService(runtime ConfigurationRuntime) *ConfigurationService {
	return &ConfigurationService{runtime: runtime}
}

func (s *ConfigurationService) List(ctx context.Context, resource ConfigurationResource, query ConfigurationListQuery) (any, error) {
	if !validConfigurationResource(resource) || query.ClusterID == 0 {
		return nil, ErrInvalidParams
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	query.Namespace = strings.TrimSpace(query.Namespace)
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.Order = strings.TrimSpace(query.Order)
	return s.runtime.List(ctx, resource, query)
}

func (s *ConfigurationService) YAML(ctx context.Context, resource ConfigurationResource, ref ConfigurationReference) (any, error) {
	if err := validateConfigurationReference(resource, ref); err != nil {
		return nil, err
	}
	if s == nil || s.runtime == nil {
		return nil, ErrConflict
	}
	return s.runtime.YAML(ctx, resource, normalizeConfigurationReference(ref))
}

func (s *ConfigurationService) Delete(ctx context.Context, resource ConfigurationResource, ref ConfigurationReference) error {
	if err := validateConfigurationReference(resource, ref); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.Delete(ctx, resource, normalizeConfigurationReference(ref))
}

func (s *ConfigurationService) EditConfigMap(ctx context.Context, input ConfigMapEditInput) error {
	if err := normalizeConfigMapEdit(&input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.EditConfigMap(ctx, input)
}

func (s *ConfigurationService) EditSecret(ctx context.Context, input SecretEditInput) error {
	if err := normalizeSecretEdit(&input); err != nil {
		return err
	}
	if s == nil || s.runtime == nil {
		return ErrConflict
	}
	return s.runtime.EditSecret(ctx, input)
}

func (s *ConfigurationService) RevealSecret(ctx context.Context, ref ConfigurationReference) (string, error) {
	if err := validateConfigurationReference(ConfigurationSecret, ref); err != nil {
		return "", err
	}
	if s == nil || s.runtime == nil {
		return "", ErrConflict
	}
	object, err := s.runtime.SecretObject(ctx, normalizeConfigurationReference(ref))
	if err != nil {
		return "", err
	}
	return secretRevealText(object)
}

func (s *ConfigurationService) Related(ctx context.Context, resource ConfigurationResource, ref ConfigurationReference) (RelatedPodsResult, error) {
	if err := validateConfigurationReference(resource, ref); err != nil {
		return RelatedPodsResult{}, err
	}
	if s == nil || s.runtime == nil {
		return RelatedPodsResult{}, ErrConflict
	}
	ref = normalizeConfigurationReference(ref)
	pods, err := s.runtime.Pods(ctx, ref.ClusterID, ref.Namespace)
	if err != nil {
		return RelatedPodsResult{}, err
	}
	return relatedPods(resource, pods, ref.Name), nil
}

func validConfigurationResource(resource ConfigurationResource) bool {
	return resource == ConfigurationConfigMap || resource == ConfigurationSecret
}

func validateConfigurationReference(resource ConfigurationResource, ref ConfigurationReference) error {
	if !validConfigurationResource(resource) || ref.ClusterID == 0 || strings.TrimSpace(ref.Namespace) == "" || strings.TrimSpace(ref.Name) == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizeConfigurationReference(ref ConfigurationReference) ConfigurationReference {
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	ref.Name = strings.TrimSpace(ref.Name)
	return ref
}

func normalizeConfigMapEdit(input *ConfigMapEditInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" {
		return ErrInvalidParams
	}
	return nil
}

func normalizeSecretEdit(input *SecretEditInput) error {
	if input == nil {
		return ErrInvalidParams
	}
	input.Namespace = strings.TrimSpace(input.Namespace)
	input.Name = strings.TrimSpace(input.Name)
	if input.Type != nil {
		value := strings.TrimSpace(*input.Type)
		input.Type = &value
	}
	if input.ClusterID == 0 || input.Namespace == "" || input.Name == "" {
		return ErrInvalidParams
	}
	return nil
}

func secretRevealText(object map[string]any) (string, error) {
	metadata := configurationMap(object["metadata"])
	data := configurationMap(object["data"])
	stringData := configurationMap(object["stringData"])
	revealedData := make(map[string]string, len(data))
	for key, raw := range data {
		revealedData[key] = decodeSecretText(raw)
	}
	revealedStringData := make(map[string]string, len(stringData))
	for key, raw := range stringData {
		revealedStringData[key] = configurationString(raw)
	}
	payload := map[string]any{
		"apiVersion": configurationString(object["apiVersion"]),
		"kind":       configurationString(object["kind"]),
		"metadata": map[string]any{
			"namespace": configurationString(metadata["namespace"]),
			"name":      configurationString(metadata["name"]),
		},
		"type": object["type"],
		"data": revealedData,
	}
	if len(revealedStringData) > 0 {
		payload["stringData"] = revealedStringData
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal secret reveal: %w", err)
	}
	return string(encoded), nil
}

func decodeSecretText(raw any) string {
	text, ok := raw.(string)
	if !ok {
		return configurationString(raw)
	}
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil || !utf8.Valid(decoded) {
		return text
	}
	return string(decoded)
}

func relatedPods(resource ConfigurationResource, items []any, name string) RelatedPodsResult {
	pods := make([]RelatedPod, 0, len(items))
	controllerSet := make(map[string]RelatedController)
	for _, item := range items {
		pod := configurationMap(item)
		if pod == nil || !podUsesConfigurationResource(resource, pod, name) {
			continue
		}
		metadata := configurationMap(pod["metadata"])
		podName, namespace := configurationString(metadata["name"]), configurationString(metadata["namespace"])
		if podName == "" || namespace == "" {
			continue
		}
		owners := configurationOwners(pod)
		for _, owner := range owners {
			controllerSet[owner.Kind+"/"+owner.Name] = RelatedController{Kind: owner.Kind, Name: owner.Name}
		}
		ready, restarts := configurationReadyAndRestarts(pod)
		spec, status := configurationMap(pod["spec"]), configurationMap(pod["status"])
		pods = append(pods, RelatedPod{Namespace: namespace, Name: podName, Phase: configurationString(status["phase"]), Node: configurationString(spec["nodeName"]), Ready: ready, Restarts: restarts, Owners: owners})
	}
	controllers := make([]RelatedController, 0, len(controllerSet))
	for _, controller := range controllerSet {
		controllers = append(controllers, controller)
	}
	return RelatedPodsResult{Pods: pods, Controllers: controllers}
}

func podUsesConfigurationResource(resource ConfigurationResource, pod map[string]any, name string) bool {
	spec := configurationMap(pod["spec"])
	for _, volume := range configurationSlice(spec["volumes"]) {
		value := configurationMap(volume)
		if resource == ConfigurationConfigMap && configurationString(configurationMap(value["configMap"])["name"]) == name {
			return true
		}
		if resource == ConfigurationSecret && (configurationString(configurationMap(value["secret"])["secretName"]) == name || configurationString(configurationMap(configurationMap(value["csi"])["nodePublishSecretRef"])["name"]) == name) {
			return true
		}
		for _, source := range configurationSlice(configurationMap(value["projected"])["sources"]) {
			sourceMap := configurationMap(source)
			if resource == ConfigurationConfigMap && configurationString(configurationMap(sourceMap["configMap"])["name"]) == name {
				return true
			}
			if resource == ConfigurationSecret && configurationString(configurationMap(sourceMap["secret"])["name"]) == name {
				return true
			}
		}
	}
	if resource == ConfigurationSecret {
		for _, pullSecret := range configurationSlice(spec["imagePullSecrets"]) {
			if configurationString(configurationMap(pullSecret)["name"]) == name {
				return true
			}
		}
	}
	for _, key := range []string{"containers", "initContainers"} {
		for _, container := range configurationSlice(spec[key]) {
			value := configurationMap(container)
			for _, source := range configurationSlice(value["envFrom"]) {
				entry := configurationMap(source)
				if resource == ConfigurationConfigMap && configurationString(configurationMap(entry["configMapRef"])["name"]) == name {
					return true
				}
				if resource == ConfigurationSecret && configurationString(configurationMap(entry["secretRef"])["name"]) == name {
					return true
				}
			}
			for _, env := range configurationSlice(value["env"]) {
				from := configurationMap(configurationMap(env)["valueFrom"])
				if resource == ConfigurationConfigMap && configurationString(configurationMap(from["configMapKeyRef"])["name"]) == name {
					return true
				}
				if resource == ConfigurationSecret && configurationString(configurationMap(from["secretKeyRef"])["name"]) == name {
					return true
				}
			}
		}
	}
	return false
}

func configurationOwners(pod map[string]any) []RelatedOwner {
	result := make([]RelatedOwner, 0)
	for _, item := range configurationSlice(configurationMap(pod["metadata"])["ownerReferences"]) {
		owner := configurationMap(item)
		kind, name := configurationString(owner["kind"]), configurationString(owner["name"])
		if kind != "" && name != "" {
			result = append(result, RelatedOwner{Kind: kind, Name: name, UID: configurationString(owner["uid"])})
		}
	}
	return result
}

func configurationReadyAndRestarts(pod map[string]any) (string, int) {
	statuses := configurationSlice(configurationMap(pod["status"])["containerStatuses"])
	if len(statuses) == 0 {
		return "-", 0
	}
	ready, restarts := 0, 0
	for _, item := range statuses {
		status := configurationMap(item)
		if value, ok := status["ready"].(bool); ok && value {
			ready++
		}
		switch value := status["restartCount"].(type) {
		case float64:
			restarts += int(value)
		case int:
			restarts += value
		}
	}
	return fmt.Sprintf("%d/%d", ready, len(statuses)), restarts
}

func configurationMap(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}
func configurationSlice(value any) []any {
	result, _ := value.([]any)
	return result
}
func configurationString(value any) string {
	result, _ := value.(string)
	return result
}
