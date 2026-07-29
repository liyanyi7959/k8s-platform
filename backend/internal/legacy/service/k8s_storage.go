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

	accessModes := pvcAccessModes(input.AccessModes)

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

func pvcAccessModes(values []string) []corev1.PersistentVolumeAccessMode {
	if len(values) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	modes := make([]corev1.PersistentVolumeAccessMode, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			modes = append(modes, corev1.PersistentVolumeAccessMode(value))
		}
	}
	if len(modes) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	return modes
}
