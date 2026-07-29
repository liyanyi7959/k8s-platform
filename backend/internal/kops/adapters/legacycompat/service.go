// Package service provides source-compatible names for legacy adapters while
// the implementations live in the Kops Kubernetes adapter. It intentionally
// contains no business behavior and is outside internal/legacy so contexts do
// not depend on the retired service layer.
package service

import kubernetes "k8s-platform-backend/internal/kops/adapters/kubernetes"

type K8sService = kubernetes.K8sService
type ServiceError = kubernetes.ServiceError

var (
	ErrNotFound         = kubernetes.ErrNotFound
	ErrConflict         = kubernetes.ErrConflict
	ErrInvalidParams    = kubernetes.ErrInvalidParams
	ErrCrypto           = kubernetes.ErrCrypto
	ErrK8s              = kubernetes.ErrK8s
	ErrK8sNetwork       = kubernetes.ErrK8sNetwork
	ErrK8sTimeout       = kubernetes.ErrK8sTimeout
	ErrK8sUnauthorized  = kubernetes.ErrK8sUnauthorized
	ErrK8sForbidden     = kubernetes.ErrK8sForbidden
	ErrK8sTLS           = kubernetes.ErrK8sTLS
)

func ErrWithMessage(kind error, message string) error { return kubernetes.ErrWithMessage(kind, message) }
func UserMessage(err error) (string, bool)             { return kubernetes.UserMessage(err) }
func NormalizeKubernetesError(err error) error         { return kubernetes.NormalizeKubernetesError(err) }
