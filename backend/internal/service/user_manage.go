package service

import (
	"context"
	"errors"
	"sort"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// ──────────────────────────────────────────────────────────
// 用户管理
// ──────────────────────────────────────────────────────────

type UserListParams struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string
}

type UserListItem struct {
	ID        uint64   `json:"id"`
	Username  string   `json:"username"`
	Status    string   `json:"status"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at"`
}

type UserListResult struct {
	Total int64          `json:"total"`
	Items []UserListItem `json:"items"`
}

func (s *RbacService) ListUsers(ctx context.Context, p UserListParams) (*UserListResult, error) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 || p.PageSize > 100 {
		p.PageSize = 20
	}

	q := s.db.WithContext(ctx).Model(&model.User{}).Where("deleted_at IS NULL")
	if v := strings.TrimSpace(p.Keyword); v != "" {
		q = q.Where("username LIKE ?", "%"+v+"%")
	}
	if v := strings.TrimSpace(p.Status); v != "" {
		q = q.Where("status = ?", v)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var users []model.User
	offset := (p.Page - 1) * p.PageSize
	if err := q.Order("id ASC").Offset(offset).Limit(p.PageSize).Find(&users).Error; err != nil {
		return nil, err
	}

	// 批量查询角色
	userIDs := make([]uint64, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}

	type userRoleRow struct {
		UserID   uint64 `gorm:"column:user_id"`
		RoleName string `gorm:"column:name"`
	}
	var urRows []userRoleRow
	if len(userIDs) > 0 {
		if err := s.db.WithContext(ctx).Raw(`
			SELECT ur.user_id, r.name
			FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id AND r.deleted_at IS NULL
			WHERE ur.user_id IN ?
		`, userIDs).Scan(&urRows).Error; err != nil {
			return nil, err
		}
	}
	roleMap := map[uint64][]string{}
	for _, row := range urRows {
		roleMap[row.UserID] = append(roleMap[row.UserID], row.RoleName)
	}

	items := make([]UserListItem, 0, len(users))
	for _, u := range users {
		items = append(items, UserListItem{
			ID:        u.ID,
			Username:  u.Username,
			Status:    u.Status,
			Roles:     roleMap[u.ID],
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &UserListResult{Total: total, Items: items}, nil
}

type CreateUserReq struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	RoleIDs  []uint64 `json:"role_ids"`
	Roles    []string `json:"roles"`
}

func (s *RbacService) CreateUser(ctx context.Context, req CreateUserReq) (uint64, error) {
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return 0, ErrInvalidParams
	}
	if len(password) < 4 {
		return 0, &ServiceError{Kind: ErrInvalidParams, Message: "密码至少4位"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	var id uint64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roles, err := s.resolveRoles(tx, req.RoleIDs, req.Roles)
		if err != nil {
			return err
		}

		// 检查用户名是否重复
		var count int64
		if err2 := tx.Model(&model.User{}).Where("deleted_at IS NULL AND username = ?", username).Count(&count).Error; err2 != nil {
			return err2
		}
		if count > 0 {
			return &ServiceError{Kind: ErrConflict, Message: "用户名已存在"}
		}

		user := model.User{Username: username, PasswordHash: string(hash), Status: "active"}
		if err2 := tx.Create(&user).Error; err2 != nil {
			return err2
		}
		id = user.ID

		// 绑定角色
		if len(roles) > 0 {
			for _, role := range roles {
				if err2 := tx.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err2 != nil {
					return err2
				}
			}
		}
		return nil
	})
	return id, err
}

type UpdateUserReq struct {
	Status  *string  `json:"status"`
	RoleIDs []uint64 `json:"role_ids"`
	Roles   []string `json:"roles"`
}

func (s *RbacService) UpdateUser(ctx context.Context, userID uint64, req UpdateUserReq) error {
	if userID == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roles, err := s.resolveRoles(tx, req.RoleIDs, req.Roles)
		if err != nil {
			return err
		}

		var user model.User
		if err := tx.Where("deleted_at IS NULL AND id = ?", userID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if req.Status != nil {
			st := strings.TrimSpace(*req.Status)
			if st != "active" && st != "disabled" {
				return &ServiceError{Kind: ErrInvalidParams, Message: "status 无效"}
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("status", st).Error; err != nil {
				return err
			}
		}

		if req.RoleIDs != nil || req.Roles != nil {
			// 先删再绑
			if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
				return err
			}
			if len(roles) > 0 {
				for _, role := range roles {
					if err := tx.Create(&model.UserRole{UserID: userID, RoleID: role.ID}).Error; err != nil {
						return err
					}
				}
			}
		}

		s.invalidateUserRolesPerms(ctx, userID)
		return nil
	})
}

func (s *RbacService) resolveRoles(tx *gorm.DB, roleIDs []uint64, roleNames []string) ([]model.Role, error) {
	if tx == nil {
		return nil, errors.New("tx is required")
	}

	ids := normalizeUint64s(roleIDs)
	if len(ids) > 0 {
		var roles []model.Role
		if err := tx.Where("deleted_at IS NULL AND id IN ?", ids).Order("id ASC").Find(&roles).Error; err != nil {
			return nil, err
		}
		if len(roles) != len(ids) {
			return nil, &ServiceError{Kind: ErrInvalidParams, Message: "存在无效角色"}
		}
		return roles, nil
	}

	names := normalizeStrings(roleNames)
	if len(names) > 0 {
		var roles []model.Role
		if err := tx.Where("deleted_at IS NULL AND name IN ?", names).Order("id ASC").Find(&roles).Error; err != nil {
			return nil, err
		}
		if len(roles) != len(names) {
			return nil, &ServiceError{Kind: ErrInvalidParams, Message: "存在无效角色"}
		}
		return roles, nil
	}

	return nil, nil
}

func (s *RbacService) resolvePermissions(tx *gorm.DB, codes []string) ([]model.Permission, error) {
	if tx == nil {
		return nil, errors.New("tx is required")
	}

	normalized := normalizeStrings(codes)
	if len(normalized) == 0 {
		return nil, nil
	}

	var perms []model.Permission
	if err := tx.Where("deleted_at IS NULL AND code IN ?", normalized).Order("code ASC").Find(&perms).Error; err != nil {
		return nil, err
	}
	if len(perms) != len(normalized) {
		return nil, &ServiceError{Kind: ErrInvalidParams, Message: "存在无效权限点"}
	}
	return perms, nil
}

func (s *RbacService) listActiveUserIDsByRole(tx *gorm.DB, roleID uint64) ([]uint64, error) {
	type row struct {
		UserID uint64 `gorm:"column:user_id"`
	}
	var rows []row
	if err := tx.Raw(`
		SELECT DISTINCT ur.user_id
		FROM user_roles ur
		JOIN users u ON u.id = ur.user_id AND u.deleted_at IS NULL
		WHERE ur.role_id = ?
	`, roleID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if row.UserID > 0 {
			ids = append(ids, row.UserID)
		}
	}
	return normalizeUint64s(ids), nil
}

func (s *RbacService) invalidateUserRolesPermsBatch(ctx context.Context, userIDs []uint64) {
	for _, userID := range normalizeUint64s(userIDs) {
		s.invalidateUserRolesPerms(ctx, userID)
	}
}

func normalizeUint64s(values []uint64) []uint64 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint64]struct{}, len(values))
	out := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func (s *RbacService) DeleteUser(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return ErrInvalidParams
	}
	now := s.db.NowFunc()
	result := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ? AND deleted_at IS NULL", userID).Update("deleted_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	s.invalidateUserRolesPerms(ctx, userID)
	return nil
}

func (s *RbacService) ResetPassword(ctx context.Context, userID uint64, newPassword string) error {
	if userID == 0 || strings.TrimSpace(newPassword) == "" {
		return ErrInvalidParams
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ? AND deleted_at IS NULL", userID).Update("password_hash", string(hash))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	s.invalidateUserRolesPerms(ctx, userID)
	return nil
}

// ──────────────────────────────────────────────────────────
// 角色管理
// ──────────────────────────────────────────────────────────

type RoleListItem struct {
	ID             uint64                     `json:"id"`
	Name           string                     `json:"name"`
	Description    string                     `json:"description"`
	Permissions    []string                   `json:"permissions"`
	NamespaceScope *RoleNamespaceScopeSummary `json:"namespace_scope,omitempty"`
	UserCount      int64                      `json:"user_count"`
	Builtin        bool                       `json:"builtin"`
	CreatedAt      string                     `json:"created_at"`
}

type RoleNamespaceScopeSummary struct {
	ClusterID   uint64   `json:"cluster_id"`
	ClusterName string   `json:"cluster_name"`
	Namespaces  []string `json:"namespaces"`
}

type NamespaceScopePayload struct {
	ClusterID  uint64   `json:"cluster_id"`
	Namespaces []string `json:"namespaces"`
}

func (s *RbacService) ListRoles(ctx context.Context) ([]RoleListItem, error) {
	var roles []model.Role
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Order("id ASC").Find(&roles).Error; err != nil {
		return nil, err
	}

	roleIDs := make([]uint64, 0, len(roles))
	for _, r := range roles {
		roleIDs = append(roleIDs, r.ID)
	}

	type rpRow struct {
		RoleID uint64 `gorm:"column:role_id"`
		Code   string `gorm:"column:code"`
	}
	var rpRows []rpRow
	if len(roleIDs) > 0 {
		if err := s.db.WithContext(ctx).Raw(`
			SELECT rp.role_id, p.code
			FROM role_permissions rp
			JOIN permissions p ON p.id = rp.permission_id AND p.deleted_at IS NULL
			WHERE rp.role_id IN ?
		`, roleIDs).Scan(&rpRows).Error; err != nil {
			return nil, err
		}
	}
	permMap := map[uint64][]string{}
	for _, row := range rpRows {
		permMap[row.RoleID] = append(permMap[row.RoleID], row.Code)
	}

	type roleUserRow struct {
		RoleID    uint64 `gorm:"column:role_id"`
		UserCount int64  `gorm:"column:user_count"`
	}
	var userRows []roleUserRow
	if len(roleIDs) > 0 {
		if err := s.db.WithContext(ctx).Raw(`
			SELECT ur.role_id, COUNT(DISTINCT ur.user_id) AS user_count
			FROM user_roles ur
			JOIN users u ON u.id = ur.user_id AND u.deleted_at IS NULL
			WHERE ur.role_id IN ?
			GROUP BY ur.role_id
		`, roleIDs).Scan(&userRows).Error; err != nil {
			return nil, err
		}
	}
	userCountMap := map[uint64]int64{}
	for _, row := range userRows {
		userCountMap[row.RoleID] = row.UserCount
	}

	type roleNamespaceScopeRow struct {
		RoleID      uint64 `gorm:"column:role_id"`
		ClusterID   uint64 `gorm:"column:cluster_id"`
		ClusterName string `gorm:"column:cluster_name"`
		Namespace   string `gorm:"column:namespace"`
	}
	var scopeRows []roleNamespaceScopeRow
	if len(roleIDs) > 0 {
		if err := s.db.WithContext(ctx).Raw(`
			SELECT rns.role_id, rns.cluster_id, c.name AS cluster_name, rns.namespace
			FROM role_namespace_scopes rns
			JOIN clusters c ON c.id = rns.cluster_id AND c.deleted_at IS NULL
			WHERE rns.role_id IN ?
			ORDER BY rns.role_id ASC, rns.namespace ASC
		`, roleIDs).Scan(&scopeRows).Error; err != nil {
			return nil, err
		}
	}
	scopeMap := map[uint64]*RoleNamespaceScopeSummary{}
	for _, row := range scopeRows {
		entry, ok := scopeMap[row.RoleID]
		if !ok {
			entry = &RoleNamespaceScopeSummary{
				ClusterID:   row.ClusterID,
				ClusterName: row.ClusterName,
				Namespaces:  []string{},
			}
			scopeMap[row.RoleID] = entry
		}
		entry.Namespaces = append(entry.Namespaces, row.Namespace)
	}

	items := make([]RoleListItem, 0, len(roles))
	for _, r := range roles {
		desc := ""
		if r.Desc != nil {
			desc = *r.Desc
		}
		permissions := append([]string(nil), permMap[r.ID]...)
		sort.Strings(permissions)
		var namespaceScope *RoleNamespaceScopeSummary
		if currentScope, ok := scopeMap[r.ID]; ok {
			namespaceScope = &RoleNamespaceScopeSummary{
				ClusterID:   currentScope.ClusterID,
				ClusterName: currentScope.ClusterName,
				Namespaces:  append([]string(nil), currentScope.Namespaces...),
			}
		}
		items = append(items, RoleListItem{
			ID:             r.ID,
			Name:           r.Name,
			Description:    desc,
			Permissions:    permissions,
			NamespaceScope: namespaceScope,
			UserCount:      userCountMap[r.ID],
			Builtin:        r.Name == "admin",
			CreatedAt:      r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, nil
}

type CreateRoleReq struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Desc           string                 `json:"desc"`
	Permissions    []string               `json:"permissions"`
	NamespaceScope *NamespaceScopePayload `json:"namespace_scope"`
}

func (r CreateRoleReq) normalizedDescription() *string {
	desc := strings.TrimSpace(r.Description)
	if desc == "" {
		desc = strings.TrimSpace(r.Desc)
	}
	if desc == "" {
		return nil
	}
	return &desc
}

func hasNamespacePermissionCodes(codes []string) bool {
	for _, code := range codes {
		if strings.HasPrefix(strings.TrimSpace(code), "namespace:") {
			return true
		}
	}
	return false
}

func (s *RbacService) normalizeNamespaceScope(tx *gorm.DB, scope *NamespaceScopePayload, permissionCodes []string) (*NamespaceScopePayload, error) {
	if tx == nil {
		return nil, errors.New("tx is required")
	}
	if scope == nil {
		return nil, nil
	}

	namespaces := normalizeStrings(scope.Namespaces)
	if len(namespaces) == 0 {
		return nil, nil
	}
	if scope.ClusterID == 0 {
		return nil, &ServiceError{Kind: ErrInvalidParams, Message: "请选择命名空间所属集群"}
	}
	if !hasNamespacePermissionCodes(permissionCodes) {
		return nil, &ServiceError{Kind: ErrInvalidParams, Message: "请先选择命名空间权限"}
	}

	var count int64
	if err := tx.Model(&model.Cluster{}).Where("deleted_at IS NULL AND id = ?", scope.ClusterID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, &ServiceError{Kind: ErrInvalidParams, Message: "存在无效集群"}
	}

	return &NamespaceScopePayload{ClusterID: scope.ClusterID, Namespaces: namespaces}, nil
}

func (s *RbacService) syncRoleNamespaceScope(tx *gorm.DB, roleID uint64, scope *NamespaceScopePayload) error {
	if tx == nil {
		return errors.New("tx is required")
	}
	if err := tx.Where("role_id = ?", roleID).Delete(&model.RoleNamespaceScope{}).Error; err != nil {
		return err
	}
	if scope == nil || len(scope.Namespaces) == 0 {
		return nil
	}
	rows := make([]model.RoleNamespaceScope, 0, len(scope.Namespaces))
	for _, namespace := range scope.Namespaces {
		rows = append(rows, model.RoleNamespaceScope{RoleID: roleID, ClusterID: scope.ClusterID, Namespace: namespace})
	}
	return tx.Create(&rows).Error
}

func (s *RbacService) currentRolePermissionCodes(tx *gorm.DB, roleID uint64) ([]string, error) {
	if tx == nil {
		return nil, errors.New("tx is required")
	}
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	if err := tx.Raw(`
		SELECT p.code
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id AND p.deleted_at IS NULL
		WHERE rp.role_id = ?
	`, roleID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Code) == "" {
			continue
		}
		codes = append(codes, row.Code)
	}
	sort.Strings(codes)
	return codes, nil
}

func (s *RbacService) CreateRole(ctx context.Context, req CreateRoleReq) (uint64, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, ErrInvalidParams
	}

	var id uint64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		perms, err := s.resolvePermissions(tx, req.Permissions)
		if err != nil {
			return err
		}
		namespaceScope, err := s.normalizeNamespaceScope(tx, req.NamespaceScope, req.Permissions)
		if err != nil {
			return err
		}

		var count int64
		if err2 := tx.Model(&model.Role{}).Where("deleted_at IS NULL AND name = ?", name).Count(&count).Error; err2 != nil {
			return err2
		}
		if count > 0 {
			return &ServiceError{Kind: ErrConflict, Message: "角色名已存在"}
		}

		role := model.Role{Name: name, Desc: req.normalizedDescription()}
		if err2 := tx.Create(&role).Error; err2 != nil {
			return err2
		}
		id = role.ID

		if len(perms) > 0 {
			for _, p := range perms {
				if err2 := tx.Create(&model.RolePermission{RoleID: role.ID, PermissionID: p.ID}).Error; err2 != nil {
					return err2
				}
			}
		}
		if err := s.syncRoleNamespaceScope(tx, role.ID, namespaceScope); err != nil {
			return err
		}
		return nil
	})
	return id, err
}

type UpdateRoleReq struct {
	Description    *string                `json:"description"`
	Desc           *string                `json:"desc"`
	Permissions    []string               `json:"permissions"`
	NamespaceScope *NamespaceScopePayload `json:"namespace_scope"`
}

func (r UpdateRoleReq) normalizedDescription() (*string, bool) {
	if r.Description == nil && r.Desc == nil {
		return nil, false
	}
	desc := ""
	if r.Description != nil {
		desc = strings.TrimSpace(*r.Description)
	} else if r.Desc != nil {
		desc = strings.TrimSpace(*r.Desc)
	}
	if desc == "" {
		return nil, true
	}
	return &desc, true
}

func (s *RbacService) UpdateRole(ctx context.Context, roleID uint64, req UpdateRoleReq) error {
	if roleID == 0 {
		return ErrInvalidParams
	}
	var affectedUserIDs []uint64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role model.Role
		if err := tx.Where("deleted_at IS NULL AND id = ?", roleID).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if descPtr, ok := req.normalizedDescription(); ok {
			if err := tx.Model(&model.Role{}).Where("id = ?", roleID).Update("desc", descPtr).Error; err != nil {
				return err
			}
		}

		var permissionCodes []string
		if req.Permissions != nil {
			permissionCodes = append([]string(nil), req.Permissions...)
		} else if req.NamespaceScope != nil {
			currentCodes, err := s.currentRolePermissionCodes(tx, roleID)
			if err != nil {
				return err
			}
			permissionCodes = currentCodes
		}

		var namespaceScope *NamespaceScopePayload
		if req.NamespaceScope != nil {
			resolvedNamespaceScope, err := s.normalizeNamespaceScope(tx, req.NamespaceScope, permissionCodes)
			if err != nil {
				return err
			}
			namespaceScope = resolvedNamespaceScope
		}

		if req.Permissions != nil {
			perms, err := s.resolvePermissions(tx, req.Permissions)
			if err != nil {
				return err
			}
			affectedUserIDs, err = s.listActiveUserIDsByRole(tx, roleID)
			if err != nil {
				return err
			}
			if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
				return err
			}
			if len(perms) > 0 {
				for _, p := range perms {
					if err := tx.Create(&model.RolePermission{RoleID: roleID, PermissionID: p.ID}).Error; err != nil {
						return err
					}
				}
			}
		}

		if req.NamespaceScope != nil {
			if err := s.syncRoleNamespaceScope(tx, roleID, namespaceScope); err != nil {
				return err
			}
		} else if req.Permissions != nil && !hasNamespacePermissionCodes(req.Permissions) {
			if err := s.syncRoleNamespaceScope(tx, roleID, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.invalidateUserRolesPermsBatch(ctx, affectedUserIDs)
	return nil
}

func (s *RbacService) DeleteRole(ctx context.Context, roleID uint64) error {
	if roleID == 0 {
		return ErrInvalidParams
	}
	var role model.Role
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", roleID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if role.Name == "admin" {
		return &ServiceError{Kind: ErrConflict, Message: "内置管理员角色不可删除"}
	}
	var assignedCount int64
	if err := s.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT ur.user_id)
		FROM user_roles ur
		JOIN users u ON u.id = ur.user_id AND u.deleted_at IS NULL
		WHERE ur.role_id = ?
	`, roleID).Scan(&assignedCount).Error; err != nil {
		return err
	}
	if assignedCount > 0 {
		return &ServiceError{Kind: ErrConflict, Message: "角色已分配给用户，无法删除"}
	}
	now := s.db.NowFunc()
	result := s.db.WithContext(ctx).Model(&model.Role{}).Where("id = ? AND deleted_at IS NULL", roleID).Update("deleted_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

type PermissionListItem struct {
	ID            uint64 `json:"id"`
	Code          string `json:"code"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	CategoryLabel string `json:"category_label"`
	Builtin       bool   `json:"builtin"`
}

// ListPermissions 列出所有可用权限点。
func (s *RbacService) ListPermissions(ctx context.Context) ([]PermissionListItem, error) {
	var perms []model.Permission
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Order("code ASC").Find(&perms).Error; err != nil {
		return nil, err
	}
	items := make([]PermissionListItem, 0, len(perms))
	for _, perm := range perms {
		meta := DescribePermission(perm.Code, perm.Desc)
		items = append(items, PermissionListItem{
			ID:            perm.ID,
			Code:          perm.Code,
			Description:   meta.Description,
			Category:      meta.Category,
			CategoryLabel: meta.CategoryLabel,
			Builtin:       meta.Builtin,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		leftOrder := permissionCategoryOrder(items[i].Category)
		rightOrder := permissionCategoryOrder(items[j].Category)
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		if items[i].CategoryLabel == items[j].CategoryLabel {
			return items[i].Code < items[j].Code
		}
		return items[i].CategoryLabel < items[j].CategoryLabel
	})
	return items, nil
}
