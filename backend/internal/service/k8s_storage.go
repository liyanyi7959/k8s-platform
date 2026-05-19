package service

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CreatePVCInput struct {
	Namespace    string
	Name         string
	StorageClass string
	AccessModes  []string
	Capacity     string
}

func (s *K8sService) CreatePVC(ctx context.Context, clusterID uint64, input CreatePVCInput) error {
	cs, err := s.typedClient(ctx, clusterID)
	if err != nil {
		return err
	}

	namespace := strings.TrimSpace(input.Namespace)
	name := strings.TrimSpace(input.Name)
	capacity := strings.TrimSpace(input.Capacity)
	if namespace == "" || name == "" || capacity == "" {
		return ErrInvalidParams
	}

	accessModes, err := normalizePVCAccessModes(input.AccessModes)
	if err != nil {
		return err
	}

	quantity, err := apiresource.ParseQuantity(capacity)
	if err != nil || quantity.Sign() <= 0 {
		return ErrWithMessage(ErrInvalidParams, "容量格式无效")
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
		},
	}

	if storageClass := strings.TrimSpace(input.StorageClass); storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	_, err = cs.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	return normalizeK8sErr(err)
}

func normalizePVCAccessModes(values []string) ([]corev1.PersistentVolumeAccessMode, error) {
	if len(values) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, nil
	}
	result := make([]corev1.PersistentVolumeAccessMode, 0, len(values))
	seen := make(map[corev1.PersistentVolumeAccessMode]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		mode, ok := mapPVCAccessMode(value)
		if !ok {
			return nil, ErrWithMessage(ErrInvalidParams, "access_modes 无效")
		}
		if _, exists := seen[mode]; exists {
			continue
		}
		seen[mode] = struct{}{}
		result = append(result, mode)
	}
	if len(result) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, nil
	}
	return result, nil
}

func mapPVCAccessMode(value string) (corev1.PersistentVolumeAccessMode, bool) {
	switch strings.TrimSpace(value) {
	case string(corev1.ReadWriteOnce):
		return corev1.ReadWriteOnce, true
	case string(corev1.ReadOnlyMany):
		return corev1.ReadOnlyMany, true
	case string(corev1.ReadWriteMany):
		return corev1.ReadWriteMany, true
	case string(corev1.ReadWriteOncePod):
		return corev1.ReadWriteOncePod, true
	default:
		return "", false
	}
}
