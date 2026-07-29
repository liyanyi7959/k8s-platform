package application

import "strings"

// HasAllToolPermissions determines whether every meaningful required
// permission is present in the caller's normalized permission set.
func HasAllToolPermissions(userPermissions, requiredPermissions []string) bool {
	return len(MissingToolPermissions(userPermissions, requiredPermissions)) == 0
}

// MissingToolPermissions preserves required-permission order for stable API
// output while ignoring empty values on either side.
func MissingToolPermissions(userPermissions, requiredPermissions []string) []string {
	if len(requiredPermissions) == 0 {
		return nil
	}
	available := make(map[string]struct{}, len(userPermissions))
	for _, permission := range userPermissions {
		if permission = strings.TrimSpace(permission); permission != "" {
			available[permission] = struct{}{}
		}
	}
	missing := make([]string, 0, len(requiredPermissions))
	for _, permission := range requiredPermissions {
		if permission = strings.TrimSpace(permission); permission == "" {
			continue
		}
		if _, ok := available[permission]; !ok {
			missing = append(missing, permission)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return missing
}
