package service

import (
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
	masked := maskSecretYAMLValue(obj)
	b, err := yaml.Marshal(masked)
	if err != nil {
		return raw, true
	}
	return strings.TrimSpace(string(b)), true
}

func maskSecretYAMLValue(v any) any {
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
					out[key] = "***"
				}
			default:
				out[key] = maskSecretYAMLValue(value)
			}
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, maskSecretYAMLValue(item))
		}
		return out
	default:
		return v
	}
}

