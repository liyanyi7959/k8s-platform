package service

import (
	"context"
	"errors"
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
		s.db.WithContext(ctx).Raw(`
			SELECT ur.user_id, r.name
			FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id AND r.deleted_at IS NULL
			WHERE ur.user_id IN ?
		`, userIDs).Scan(&urRows)
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
		if len(req.Roles) > 0 {
			var roles []model.Role
			tx.Where("deleted_at IS NULL AND name IN ?", req.Roles).Find(&roles)
			for _, role := range roles {
				tx.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID})
			}
		}
		return nil
	})
	return id, err
}

type UpdateUserReq struct {
	Status *string  `json:"status"`
	Roles  []string `json:"roles"`
}

func (s *RbacService) UpdateUser(ctx context.Context, userID uint64, req UpdateUserReq) error {
	if userID == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
			tx.Model(&model.User{}).Where("id = ?", userID).Update("status", st)
		}

		if req.Roles != nil {
			// 先删再绑
			tx.Where("user_id = ?", userID).Delete(&model.UserRole{})
			if len(req.Roles) > 0 {
				var roles []model.Role
				tx.Where("deleted_at IS NULL AND name IN ?", req.Roles).Find(&roles)
				for _, role := range roles {
					tx.Create(&model.UserRole{UserID: userID, RoleID: role.ID})
				}
			}
		}

		s.invalidateUserRolesPerms(ctx, userID)
		return nil
	})
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
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Desc        string   `json:"desc"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
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
		s.db.WithContext(ctx).Raw(`
			SELECT rp.role_id, p.code
			FROM role_permissions rp
			JOIN permissions p ON p.id = rp.permission_id AND p.deleted_at IS NULL
			WHERE rp.role_id IN ?
		`, roleIDs).Scan(&rpRows)
	}
	permMap := map[uint64][]string{}
	for _, row := range rpRows {
		permMap[row.RoleID] = append(permMap[row.RoleID], row.Code)
	}

	items := make([]RoleListItem, 0, len(roles))
	for _, r := range roles {
		desc := ""
		if r.Desc != nil {
			desc = *r.Desc
		}
		items = append(items, RoleListItem{
			ID:          r.ID,
			Name:        r.Name,
			Desc:        desc,
			Permissions: permMap[r.ID],
			CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, nil
}

type CreateRoleReq struct {
	Name        string   `json:"name"`
	Desc        string   `json:"desc"`
	Permissions []string `json:"permissions"`
}

func (s *RbacService) CreateRole(ctx context.Context, req CreateRoleReq) (uint64, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return 0, ErrInvalidParams
	}

	var id uint64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err2 := tx.Model(&model.Role{}).Where("deleted_at IS NULL AND name = ?", name).Count(&count).Error; err2 != nil {
			return err2
		}
		if count > 0 {
			return &ServiceError{Kind: ErrConflict, Message: "角色名已存在"}
		}

		desc := strings.TrimSpace(req.Desc)
		var descPtr *string
		if desc != "" {
			descPtr = &desc
		}
		role := model.Role{Name: name, Desc: descPtr}
		if err2 := tx.Create(&role).Error; err2 != nil {
			return err2
		}
		id = role.ID

		if len(req.Permissions) > 0 {
			var perms []model.Permission
			tx.Where("deleted_at IS NULL AND code IN ?", req.Permissions).Find(&perms)
			for _, p := range perms {
				tx.Create(&model.RolePermission{RoleID: role.ID, PermissionID: p.ID})
			}
		}
		return nil
	})
	return id, err
}

type UpdateRoleReq struct {
	Desc        *string  `json:"desc"`
	Permissions []string `json:"permissions"`
}

func (s *RbacService) UpdateRole(ctx context.Context, roleID uint64, req UpdateRoleReq) error {
	if roleID == 0 {
		return ErrInvalidParams
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role model.Role
		if err := tx.Where("deleted_at IS NULL AND id = ?", roleID).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if req.Desc != nil {
			desc := strings.TrimSpace(*req.Desc)
			var descPtr *string
			if desc != "" {
				descPtr = &desc
			}
			tx.Model(&model.Role{}).Where("id = ?", roleID).Update("desc", descPtr)
		}

		if req.Permissions != nil {
			tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{})
			if len(req.Permissions) > 0 {
				var perms []model.Permission
				tx.Where("deleted_at IS NULL AND code IN ?", req.Permissions).Find(&perms)
				for _, p := range perms {
					tx.Create(&model.RolePermission{RoleID: roleID, PermissionID: p.ID})
				}
			}
		}
		return nil
	})
}

func (s *RbacService) DeleteRole(ctx context.Context, roleID uint64) error {
	if roleID == 0 {
		return ErrInvalidParams
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

// ListPermissions 列出所有可用权限点。
func (s *RbacService) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var perms []model.Permission
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL").Order("code ASC").Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}
