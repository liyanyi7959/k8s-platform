package mysql

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/internal/iam/ports"
)

func (r *AuthRepository) ListRoles(ctx context.Context) ([]ports.RoleRecord, error) {
	var roles []managedRoleRow
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Order("id ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(roles))
	for _, role := range roles {
		ids = append(ids, role.ID)
	}
	permissionMap := map[uint64][]string{}
	userCountMap := map[uint64]int64{}
	scopeMap := map[uint64]*ports.RoleNamespaceScopeRecord{}
	if len(ids) > 0 {
		var permissions []rolePermissionCodeRow
		if err := r.db.WithContext(ctx).Raw(`SELECT rp.role_id, p.code FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id AND p.deleted_at IS NULL WHERE rp.role_id IN ?`, ids).Scan(&permissions).Error; err != nil {
			return nil, err
		}
		for _, row := range permissions {
			permissionMap[row.RoleID] = append(permissionMap[row.RoleID], row.Code)
		}
		var counts []roleUserCountRow
		if err := r.db.WithContext(ctx).Raw(`SELECT ur.role_id, COUNT(DISTINCT ur.user_id) AS user_count FROM user_roles ur JOIN users u ON u.id = ur.user_id AND u.deleted_at IS NULL WHERE ur.role_id IN ? GROUP BY ur.role_id`, ids).Scan(&counts).Error; err != nil {
			return nil, err
		}
		for _, row := range counts {
			userCountMap[row.RoleID] = row.UserCount
		}
		var scopes []roleScopeJoinRow
		if err := r.db.WithContext(ctx).Raw(`SELECT rns.role_id, rns.cluster_id, c.name AS cluster_name, rns.namespace FROM role_namespace_scopes rns JOIN clusters c ON c.id = rns.cluster_id AND c.deleted_at IS NULL WHERE rns.role_id IN ? ORDER BY rns.role_id ASC, rns.namespace ASC`, ids).Scan(&scopes).Error; err != nil {
			return nil, err
		}
		for _, row := range scopes {
			scope := scopeMap[row.RoleID]
			if scope == nil {
				scope = &ports.RoleNamespaceScopeRecord{ClusterID: row.ClusterID, ClusterName: row.ClusterName, Namespaces: []string{}}
				scopeMap[row.RoleID] = scope
			}
			scope.Namespaces = append(scope.Namespaces, row.Namespace)
		}
	}
	result := make([]ports.RoleRecord, 0, len(roles))
	for _, role := range roles {
		permissions := permissionMap[role.ID]
		sort.Strings(permissions)
		description := ""
		if role.Description != nil {
			description = *role.Description
		}
		result = append(result, ports.RoleRecord{ID: role.ID, Name: role.Name, Code: role.Code, Description: description, Permissions: permissions, NamespaceScope: scopeMap[role.ID], UserCount: userCountMap[role.ID], CreatedAt: role.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	return result, nil
}

func (r *AuthRepository) CreateRole(ctx context.Context, data ports.CreateRoleData) (uint64, error) {
	var id uint64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		permissions, err := resolveManagedPermissions(tx, data.Permissions)
		if err != nil {
			return err
		}
		if err := validateManagedScope(tx, data.Scope, data.Permissions); err != nil {
			return err
		}
		if exists, err := managedRoleExists(tx, "name", data.Name, 0); err != nil {
			return err
		} else if exists {
			return domain.ErrConflict
		}
		if exists, err := managedRoleExists(tx, "code", data.Code, 0); err != nil {
			return err
		} else if exists {
			return domain.ErrConflict
		}
		role := managedRoleRow{Name: data.Name, Code: data.Code, Description: data.Description}
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		id = role.ID
		if err := replaceManagedPermissions(tx, id, permissions); err != nil {
			return err
		}
		return replaceManagedScope(tx, id, data.Scope)
	})
	return id, err
}

func (r *AuthRepository) UpdateRole(ctx context.Context, id uint64, data ports.UpdateRoleData) ([]uint64, error) {
	var affected []uint64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role managedRoleRow
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if (role.Name == "admin" || role.Code == "admin") && ((data.Name != nil && *data.Name != "admin") || (data.Code != nil && *data.Code != "admin")) {
			return domain.ErrConflict
		}
		updates := map[string]any{}
		if data.Name != nil {
			if exists, err := managedRoleExists(tx, "name", *data.Name, id); err != nil {
				return err
			} else if exists {
				return domain.ErrConflict
			}
			updates["name"] = *data.Name
		}
		if data.Code != nil {
			if exists, err := managedRoleExists(tx, "code", *data.Code, id); err != nil {
				return err
			} else if exists {
				return domain.ErrConflict
			}
			updates["code"] = *data.Code
		}
		if data.SetDescription {
			updates["desc"] = data.Description
		}
		if len(updates) > 0 {
			if err := tx.Model(&managedRoleRow{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		permissionCodes := data.Permissions
		if !data.SetPermissions && data.SetScope {
			var err error
			permissionCodes, err = currentManagedPermissionCodes(tx, id)
			if err != nil {
				return err
			}
		}
		if data.SetScope {
			if err := validateManagedScope(tx, data.Scope, permissionCodes); err != nil {
				return err
			}
		}
		if data.SetPermissions {
			permissions, err := resolveManagedPermissions(tx, data.Permissions)
			if err != nil {
				return err
			}
			affected, err = activeManagedUserIDs(tx, id)
			if err != nil {
				return err
			}
			if err := replaceManagedPermissions(tx, id, permissions); err != nil {
				return err
			}
		}
		if data.SetScope {
			if err := replaceManagedScope(tx, id, data.Scope); err != nil {
				return err
			}
		} else if data.SetPermissions && !domain.HasNamespacePermission(data.Permissions) {
			if err := replaceManagedScope(tx, id, nil); err != nil {
				return err
			}
		}
		return nil
	})
	return affected, err
}

func (r *AuthRepository) DeleteRole(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role managedRoleRow
		if err := tx.Where("deleted_at IS NULL AND id = ?", id).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if role.Name == "admin" || role.Code == "admin" {
			return domain.ErrConflict
		}
		users, err := activeManagedUserIDs(tx, id)
		if err != nil {
			return err
		}
		if len(users) > 0 {
			return domain.ErrConflict
		}
		result := tx.Model(&managedRoleRow{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", tx.NowFunc())
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (r *AuthRepository) ListPermissions(ctx context.Context) ([]ports.PermissionRecord, error) {
	var rows []managedPermissionRow
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Order("code ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]ports.PermissionRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, ports.PermissionRecord{ID: row.ID, Code: row.Code, Description: row.Description})
	}
	return result, nil
}

func resolveManagedPermissions(tx *gorm.DB, codes []string) ([]managedPermissionRow, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	var rows []managedPermissionRow
	if err := tx.Where("deleted_at IS NULL AND code IN ?", codes).Order("code ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(codes) {
		return nil, domain.ErrInvalidParams
	}
	return rows, nil
}
func validateManagedScope(tx *gorm.DB, scope *ports.RoleNamespaceScopeData, permissionCodes []string) error {
	if scope == nil || len(scope.Namespaces) == 0 {
		return nil
	}
	if scope.ClusterID == 0 || !domain.HasNamespacePermission(permissionCodes) {
		return domain.ErrInvalidParams
	}
	var count int64
	if err := tx.Table("clusters").Where("deleted_at IS NULL AND id = ?", scope.ClusterID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrInvalidParams
	}
	return nil
}
func replaceManagedPermissions(tx *gorm.DB, roleID uint64, permissions []managedPermissionRow) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&managedRolePermissionRow{}).Error; err != nil {
		return err
	}
	for _, permission := range permissions {
		if err := tx.Create(&managedRolePermissionRow{RoleID: roleID, PermissionID: permission.ID}).Error; err != nil {
			return err
		}
	}
	return nil
}
func replaceManagedScope(tx *gorm.DB, roleID uint64, scope *ports.RoleNamespaceScopeData) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&managedRoleScopeRow{}).Error; err != nil {
		return err
	}
	if scope == nil {
		return nil
	}
	for _, namespace := range scope.Namespaces {
		if err := tx.Create(&managedRoleScopeRow{RoleID: roleID, ClusterID: scope.ClusterID, Namespace: namespace}).Error; err != nil {
			return err
		}
	}
	return nil
}
func managedRoleExists(tx *gorm.DB, column, value string, excludeID uint64) (bool, error) {
	query := tx.Model(&managedRoleRow{}).Where("deleted_at IS NULL AND "+column+" = ?", value)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}
func currentManagedPermissionCodes(tx *gorm.DB, roleID uint64) ([]string, error) {
	var rows []struct {
		Code string `gorm:"column:code"`
	}
	if err := tx.Raw(`SELECT p.code FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id AND p.deleted_at IS NULL WHERE rp.role_id = ?`, roleID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		if value := strings.TrimSpace(row.Code); value != "" {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result, nil
}
func activeManagedUserIDs(tx *gorm.DB, roleID uint64) ([]uint64, error) {
	var rows []struct {
		UserID uint64 `gorm:"column:user_id"`
	}
	if err := tx.Raw(`SELECT ur.user_id FROM user_roles ur JOIN users u ON u.id = ur.user_id AND u.deleted_at IS NULL WHERE ur.role_id = ?`, roleID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]uint64, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.UserID)
	}
	return result, nil
}

type managedRoleRow struct {
	ID          uint64     `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	Code        string     `gorm:"column:code"`
	Description *string    `gorm:"column:desc"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

func (managedRoleRow) TableName() string { return "roles" }

type managedPermissionRow struct {
	ID          uint64  `gorm:"column:id"`
	Code        string  `gorm:"column:code"`
	Description *string `gorm:"column:desc"`
}

func (managedPermissionRow) TableName() string { return "permissions" }

type managedRolePermissionRow struct {
	RoleID       uint64 `gorm:"column:role_id;primaryKey"`
	PermissionID uint64 `gorm:"column:permission_id;primaryKey"`
}

func (managedRolePermissionRow) TableName() string { return "role_permissions" }

type managedRoleScopeRow struct {
	RoleID    uint64 `gorm:"column:role_id;primaryKey"`
	ClusterID uint64 `gorm:"column:cluster_id;primaryKey"`
	Namespace string `gorm:"column:namespace;primaryKey"`
}

func (managedRoleScopeRow) TableName() string { return "role_namespace_scopes" }

type rolePermissionCodeRow struct {
	RoleID uint64 `gorm:"column:role_id"`
	Code   string `gorm:"column:code"`
}
type roleUserCountRow struct {
	RoleID    uint64 `gorm:"column:role_id"`
	UserCount int64  `gorm:"column:user_count"`
}
type roleScopeJoinRow struct {
	RoleID      uint64 `gorm:"column:role_id"`
	ClusterID   uint64 `gorm:"column:cluster_id"`
	ClusterName string `gorm:"column:cluster_name"`
	Namespace   string `gorm:"column:namespace"`
}
