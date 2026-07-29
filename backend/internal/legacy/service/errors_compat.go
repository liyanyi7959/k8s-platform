// Package service retains only compatibility names while callers are moved to
// their bounded-context adapters. Error policy is owned by the Kops
// Kubernetes transport.
package service

import kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"

var (
	ErrNotFound         = kopsclient.ErrNotFound
	ErrConflict         = kopsclient.ErrConflict
	ErrInvalidParams    = kopsclient.ErrInvalidParams
	ErrCrypto           = kopsclient.ErrCrypto
	ErrK8s              = kopsclient.ErrK8s
	ErrK8sNetwork       = kopsclient.ErrK8sNetwork
	ErrK8sTimeout       = kopsclient.ErrK8sTimeout
	ErrK8sUnauthorized  = kopsclient.ErrK8sUnauthorized
	ErrK8sForbidden     = kopsclient.ErrK8sForbidden
	ErrK8sTLS           = kopsclient.ErrK8sTLS
)

type ServiceError = kopsclient.ServiceError

func ErrWithMessage(kind error, message string) error { return kopsclient.ErrWithMessage(kind, message) }
func UserMessage(err error) (string, bool)             { return kopsclient.UserMessage(err) }
