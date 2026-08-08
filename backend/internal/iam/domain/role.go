package domain

import (
	"sort"
	"strings"
)

// NamespaceScope 是角色的命名空间作用域值对象，属于 Role 聚合的一部分。
// 不变式：作用域必须指定集群，且仅当角色包含 namespace:* 权限时才允许命名空间作用域。
type NamespaceScope struct {
	ClusterID  uint64
	Namespaces []string
}

// NewNamespaceScope 校验并构造命名空间作用域。无命名空间时返回 nil（无作用域）。
func NewNamespaceScope(clusterID uint64, namespaces []string, permissionCodes []string) (*NamespaceScope, error) {
	normalized := normalizeScopeNamespaces(namespaces)
	if len(normalized) == 0 {
		return nil, nil
	}
	if clusterID == 0 || (permissionCodes != nil && !HasNamespacePermission(permissionCodes)) {
		return nil, ErrInvalidParams
	}
	return &NamespaceScope{ClusterID: clusterID, Namespaces: normalized}, nil
}

func normalizeScopeNamespaces(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
