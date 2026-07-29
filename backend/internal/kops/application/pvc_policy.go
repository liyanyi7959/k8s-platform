package application

import (
	"strings"

	apiresource "k8s.io/apimachinery/pkg/api/resource"
)

var allowedPVCAccessModes = map[string]struct{}{
	"ReadWriteOnce":    {},
	"ReadOnlyMany":     {},
	"ReadWriteMany":    {},
	"ReadWriteOncePod": {},
}

// NormalizePVCAccessModes applies the platform's explicit PVC access-mode
// allowlist before the Kubernetes adapter constructs a typed API object.
func NormalizePVCAccessModes(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{"ReadWriteOnce"}, nil
	}
	seen := make(map[string]struct{}, len(values))
	modes := make([]string, 0, len(values))
	for _, raw := range values {
		mode := strings.TrimSpace(raw)
		if mode == "" {
			continue
		}
		if _, ok := allowedPVCAccessModes[mode]; !ok {
			return nil, ErrInvalidParams
		}
		if _, duplicate := seen[mode]; duplicate {
			continue
		}
		seen[mode] = struct{}{}
		modes = append(modes, mode)
	}
	if len(modes) == 0 {
		return []string{"ReadWriteOnce"}, nil
	}
	return modes, nil
}

func ValidatePVCCapacity(value string) error {
	quantity, err := apiresource.ParseQuantity(strings.TrimSpace(value))
	if err != nil || quantity.Sign() <= 0 {
		return ErrInvalidParams
	}
	return nil
}
