package application

import (
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const MaskedSecretValue = "***"

// MetadataObject is the small Kubernetes metadata surface needed for stable
// list ordering. Runtime adapters remain responsible for fetching objects.
type MetadataObject interface {
	GetName() string
	GetNamespace() string
}

// SortObjectsByMetadata applies the supported Kops list ordering without
// coupling the policy to a specific Kubernetes resource type.
func SortObjectsByMetadata[T MetadataObject](items []T, sortBy, order string) {
	sortBy = strings.TrimSpace(sortBy)
	if sortBy != "metadata.name" && sortBy != "metadata.namespace" {
		return
	}
	descending := strings.EqualFold(strings.TrimSpace(order), "desc")
	sort.SliceStable(items, func(left, right int) bool {
		var leftValue, rightValue string
		if sortBy == "metadata.name" {
			leftValue, rightValue = items[left].GetName(), items[right].GetName()
		} else {
			leftValue, rightValue = items[left].GetNamespace(), items[right].GetNamespace()
		}
		if descending {
			return leftValue > rightValue
		}
		return leftValue < rightValue
	})
}

// MaskSecretForRead copies a Secret while preserving its shape and replacing
// all values. It is safe to use for cached list responses and model evidence.
func MaskSecretForRead(secret *corev1.Secret) *corev1.Secret {
	if secret == nil {
		return nil
	}
	masked := secret.DeepCopy()
	if len(masked.Data) > 0 {
		data := make(map[string][]byte, len(masked.Data))
		for key := range masked.Data {
			data[key] = []byte(MaskedSecretValue)
		}
		masked.Data = data
	}
	if len(masked.StringData) > 0 {
		stringData := make(map[string]string, len(masked.StringData))
		for key := range masked.StringData {
			stringData[key] = MaskedSecretValue
		}
		masked.StringData = stringData
	}
	return masked
}

func MaskSecretsForRead(items []*corev1.Secret) []*corev1.Secret {
	masked := make([]*corev1.Secret, 0, len(items))
	for _, item := range items {
		if item := MaskSecretForRead(item); item != nil {
			masked = append(masked, item)
		}
	}
	return masked
}
