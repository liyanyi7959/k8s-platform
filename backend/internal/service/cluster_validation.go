package service

import "strings"

const (
	maxKubeconfigContentSize = 1024 * 1024
	maxEncodedKubeconfigSize = maxKubeconfigContentSize*4/3 + 4096
)

func validateClusterName(name string) error {
	if len(name) > 120 || !clusterNamePattern.MatchString(name) {
		return ErrWithMessage(ErrInvalidParams, "集群名称需以字母或数字开头，最长 120 字符，仅可包含字母、数字、点、横线和下划线")
	}
	return nil
}

func validateKubeconfigSize(kubeconfig string) error {
	if len(strings.TrimSpace(kubeconfig)) > maxKubeconfigContentSize {
		return ErrWithMessage(ErrInvalidParams, "kubeconfig 内容不能超过 1MB")
	}
	return nil
}
