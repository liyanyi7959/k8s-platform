package mysql

import (
	"context"
	"sort"
	"strings"

	"gorm.io/gorm"

	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/internal/iam/ports"
)

func (r *AuthRepository) ListUsers(ctx context.Context, filter ports.UserListFilter) ([]ports.UserRecord, int64, error) {
	query := r.db.WithContext(ctx).Model(&userRow{}).Where("deleted_at IS NULL")
	if value := strings.TrimSpace(filter.Keyword); value != "" {
		like := "%" + value + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", like, like, like)
	}
	if value := strings.TrimSpace(filter.Status); value != "" {
		query = query.Where("status = ?", value)
	}
	if filter.RoleID > 0 {
		query = query.Where("EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = users.id AND ur.role_id = ?)", filter.RoleID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []userRow
	if err := query.Order("id ASC").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	roleMap := map[uint64][]ports.UserRoleRecord{}
	if len(ids) > 0 {
		var rows []userRoleJoinRow
		if err := r.db.WithContext(ctx).Raw(`SELECT ur.user_id, r.id AS role_id, r.name, r.code FROM user_roles ur JOIN roles r ON r.id = ur.role_id AND r.deleted_at IS NULL WHERE ur.user_id IN ?`, ids).Scan(&rows).Error; err != nil {
			return nil, 0, err
		}
		for _, row := range rows {
			roleMap[row.UserID] = append(roleMap[row.UserID], ports.UserRoleRecord{ID: row.RoleID, Name: row.Name, Code: row.Code})
		}
	}
	result := make([]ports.UserRecord, 0, len(users))
	for _, user := range users {
		result = append(result, ports.UserRecord{ID: user.ID, Username: user.Username, Nickname: user.Nickname, Email: user.Email, Status: user.Status, Roles: roleMap[user.ID], CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	return result, total, nil
}
func (r *AuthRepository) CreateUser(ctx context.Context, data ports.CreateUserData) (uint64, error) {
	var id uint64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roles, err := resolveUserRoles(tx, data.RoleIDs, data.RoleNames)
		if err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&userRow{}).Where("deleted_at IS NULL AND username = ?", data.Username).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrConflict
		}
		nickname := strings.TrimSpace(data.Nickname)
		if nickname == "" {
			nickname = data.Username
		}
		user := userRow{Username: data.Username, Nickname: nickname, Email: strings.TrimSpace(data.Email), PasswordHash: data.PasswordHash, Status: "active"}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		id = user.ID
		for _, role := range roles {
			if err := tx.Create(&userRoleRow{UserID: id, RoleID: role.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}
func (r *AuthRepository) UpdateUser(ctx context.Context, id uint64, data ports.UpdateUserData) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roles, err := resolveUserRoles(tx, data.RoleIDs, data.RoleNames)
		if err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&userRow{}).Where("deleted_at IS NULL AND id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return domain.ErrNotFound
		}
		updates := map[string]any{}
		if data.Nickname != nil {
			updates["nickname"] = strings.TrimSpace(*data.Nickname)
		}
		if data.Email != nil {
			updates["email"] = strings.TrimSpace(*data.Email)
		}
		if data.Status != nil {
			updates["status"] = *data.Status
		}
		if len(updates) > 0 {
			if err := tx.Model(&userRow{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		if data.RoleIDs != nil || data.RoleNames != nil {
			if err := tx.Where("user_id = ?", id).Delete(&userRoleRow{}).Error; err != nil {
				return err
			}
			for _, role := range roles {
				if err := tx.Create(&userRoleRow{UserID: id, RoleID: role.ID}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
func (r *AuthRepository) DeleteUser(ctx context.Context, id uint64) error {
	return affectedUser(r.db.WithContext(ctx).Model(&userRow{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", r.db.NowFunc()))
}
func (r *AuthRepository) ResetUserPassword(ctx context.Context, id uint64, hash string) error {
	return affectedUser(r.db.WithContext(ctx).Model(&userRow{}).Where("id = ? AND deleted_at IS NULL", id).Update("password_hash", hash))
}
func affectedUser(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func resolveUserRoles(tx *gorm.DB, ids []uint64, names []string) ([]roleRow, error) {
	ids = uniqueIDs(ids)
	if len(ids) > 0 {
		var roles []roleRow
		if err := tx.Where("deleted_at IS NULL AND id IN ?", ids).Order("id ASC").Find(&roles).Error; err != nil {
			return nil, err
		}
		if len(roles) != len(ids) {
			return nil, domain.ErrInvalidParams
		}
		return roles, nil
	}
	names = uniqueNames(names)
	if len(names) > 0 {
		var roles []roleRow
		if err := tx.Where("deleted_at IS NULL AND name IN ?", names).Order("id ASC").Find(&roles).Error; err != nil {
			return nil, err
		}
		if len(roles) != len(names) {
			return nil, domain.ErrInvalidParams
		}
		return roles, nil
	}
	return nil, nil
}
func uniqueIDs(values []uint64) []uint64 {
	seen := map[uint64]struct{}{}
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value > 0 {
			if _, ok := seen[value]; !ok {
				seen[value] = struct{}{}
				result = append(result, value)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
func uniqueNames(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			if _, ok := seen[value]; !ok {
				seen[value] = struct{}{}
				result = append(result, value)
			}
		}
	}
	sort.Strings(result)
	return result
}

type roleRow struct {
	ID   uint64 `gorm:"column:id"`
	Name string `gorm:"column:name"`
	Code string `gorm:"column:code"`
}

func (roleRow) TableName() string { return "roles" }

type userRoleRow struct {
	UserID uint64 `gorm:"column:user_id;primaryKey"`
	RoleID uint64 `gorm:"column:role_id;primaryKey"`
}

func (userRoleRow) TableName() string { return "user_roles" }

type userRoleJoinRow struct {
	UserID uint64 `gorm:"column:user_id"`
	RoleID uint64 `gorm:"column:role_id"`
	Name   string `gorm:"column:name"`
	Code   string `gorm:"column:code"`
}
