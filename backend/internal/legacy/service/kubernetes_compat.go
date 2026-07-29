package service

import kopsclient "k8s-platform-backend/internal/kops/adapters/kubernetes"

// K8sService is a temporary source-compatible alias. The implementation,
// informer lifecycle, cache policy and error normalization now live in Kops.
type K8sService = kopsclient.K8sService

func NormalizeKubernetesError(err error) error { return kopsclient.NormalizeKubernetesError(err) }
