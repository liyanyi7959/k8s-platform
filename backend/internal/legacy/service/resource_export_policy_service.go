package service

import (
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

type ResourceExportPolicyService struct{}

func NewResourceExportPolicyService() *ResourceExportPolicyService {
	return &ResourceExportPolicyService{}
}

func (s *ResourceExportPolicyService) IsSensitiveKind(kind string) bool {
	return strings.EqualFold(strings.TrimSpace(kind), "secret")
}

func (s *ResourceExportPolicyService) CanExposeToAI(kind string) bool {
	return !s.IsSensitiveKind(kind)
}

func (s *ResourceExportPolicyService) ExportYAML(kind string, yamlText string) (string, bool, error) {
	raw := strings.TrimSpace(yamlText)
	if raw == "" {
		return raw, false, nil
	}
	if !s.CanExposeToAI(kind) {
		return "", false, ErrWithMessage(ErrK8sForbidden, "AI direct YAML export is disabled for sensitive resource kind")
	}
	return raw, false, nil
}

func (s *ResourceExportPolicyService) ExportMaskedYAML(kind string, yamlText string) (string, bool, error) {
	raw := strings.TrimSpace(yamlText)
	if raw == "" {
		return raw, false, nil
	}
	masked, changed := s.MaskYAML(kind, raw)
	return masked, changed, nil
}

func (s *ResourceExportPolicyService) MaskYAML(kind string, yamlText string) (string, bool) {
	raw := strings.TrimSpace(yamlText)
	if raw == "" {
		return raw, false
	}
	if !s.IsSensitiveKind(kind) {
		return raw, false
	}
	var obj any
	if err := yaml.Unmarshal([]byte(raw), &obj); err != nil {
		return raw, true
	}
	masked := maskSensitiveValue(kind, obj)
	b, err := yaml.Marshal(masked)
	if err != nil {
		return raw, true
	}
	return strings.TrimSpace(string(b)), true
}

func (s *ResourceExportPolicyService) SanitizeObject(kind string, obj map[string]any) (map[string]any, bool) {
	if obj == nil {
		return nil, false
	}
	if !s.IsSensitiveKind(kind) {
		return cloneAnyMap(obj), false
	}
	masked, _ := maskSensitiveValue(kind, obj).(map[string]any)
	if masked == nil {
		return cloneAnyMap(obj), true
	}
	return masked, true
}

func maskSensitiveValue(kind string, v any) any {
	if strings.EqualFold(strings.TrimSpace(kind), "secret") {
		return maskSecretValue(v)
	}
	return cloneAnyValue(v)
}

func maskSecretValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			lowerKey := strings.ToLower(strings.TrimSpace(key))
			switch lowerKey {
			case "data", "stringdata", "binarydata":
				if child, ok := value.(map[string]any); ok {
					maskedChild := make(map[string]any, len(child))
					for childKey := range child {
						maskedChild[childKey] = "***"
					}
					out[key] = maskedChild
				} else {
					out[key] = fmt.Sprintf("*** (%T)", value)
				}
			default:
				out[key] = maskSecretValue(value)
			}
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, maskSecretValue(item))
		}
		return out
	default:
		return v
	}
}

func cloneAnyValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneAnyValue(item))
		}
		return out
	default:
		return v
	}
}

func cloneAnyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = cloneAnyValue(value)
	}
	return out
}
