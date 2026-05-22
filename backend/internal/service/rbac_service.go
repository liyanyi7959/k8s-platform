package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

var (
	ErrAuthUserNotFound         = errors.New("auth user not found")
	ErrAuthUserDisabled         = errors.New("auth user disabled")
	ErrAuthPasswordIncorrect    = errors.New("auth password incorrect")
	ErrAuthOldPasswordIncorrect = errors.New("auth old password incorrect")
)

type RbacService struct {
	db            *gorm.DB
	cache         CacheStore
	rolesPermsTTL time.Duration
}

func NewRbacService(db *gorm.DB, cacheStore CacheStore, rolesPermsTTL time.Duration) *RbacService {
	if rolesPermsTTL <= 0 {
		rolesPermsTTL = 15 * time.Minute
	}
	return &RbacService{db: db, cache: cacheStore, rolesPermsTTL: rolesPermsTTL}
}

type AuthUser struct {
	ID       int64
	Username string
	Status   string
	Roles    []string
	Perms    []string
}

func (s *RbacService) Authenticate(ctx context.Context, username, password string) (*AuthUser, error) {
	user := strings.TrimSpace(username)
	pass := password
	if user == "" || strings.TrimSpace(pass) == "" {
		return nil, ErrInvalidParams
	}
	var row model.User
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND username = ?", user).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAuthUserNotFound
		}
		return nil, err
	}
	if row.Status == "disabled" {
		return nil, ErrAuthUserDisabled
	}
	if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(pass)) != nil {
		return nil, ErrAuthPasswordIncorrect
	}
	roles, perms, err := s.GetUserRolesPerms(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &AuthUser{ID: int64(row.ID), Username: row.Username, Status: row.Status, Roles: roles, Perms: perms}, nil
}

func (s *RbacService) ChangePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	if userID == 0 || strings.TrimSpace(oldPassword) == "" || strings.TrimSpace(newPassword) == "" {
		return ErrInvalidParams
	}
	var row model.User
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", userID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(oldPassword)) != nil {
		return ErrAuthOldPasswordIncorrect
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("password_hash", string(hash)).Error; err != nil {
		return err
	}
	s.invalidateUserRolesPerms(ctx, userID)
	return nil
}

func (s *RbacService) GetUserRolesPerms(ctx context.Context, userID uint64) ([]string, []string, error) {
	if userID == 0 {
		return nil, nil, ErrInvalidParams
	}

	if s.cache != nil && s.cache.Enabled() && s.rolesPermsTTL > 0 {
		key := s.userRolesPermsCacheKey(userID)
		if b, ok, err := s.cache.Get(ctx, key); err == nil && ok && len(b) > 0 {
			var cached struct {
				Roles []string `json:"roles"`
				Perms []string `json:"perms"`
			}
			if json.Unmarshal(b, &cached) == nil {
				return cached.Roles, cached.Perms, nil
			}
		}
	}

	type roleRow struct {
		Name string `gorm:"column:name"`
	}
	var roleNames []roleRow
	if err := s.db.WithContext(ctx).
		Raw(`SELECT r.name FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id = ? AND r.deleted_at IS NULL`, userID).
		Scan(&roleNames).Error; err != nil {
		return nil, nil, err
	}
	roles := make([]string, 0, len(roleNames))
	for _, r := range roleNames {
		if strings.TrimSpace(r.Name) != "" {
			roles = append(roles, r.Name)
		}
	}
	sort.Strings(roles)

	type permRow struct {
		Code string `gorm:"column:code"`
	}
	var permCodes []permRow
	if err := s.db.WithContext(ctx).
		Raw(`SELECT DISTINCT p.code FROM user_roles ur JOIN role_permissions rp ON ur.role_id = rp.role_id JOIN permissions p ON rp.permission_id = p.id WHERE ur.user_id = ? AND p.deleted_at IS NULL`, userID).
		Scan(&permCodes).Error; err != nil {
		return nil, nil, err
	}
	perms := make([]string, 0, len(permCodes))
	for _, p := range permCodes {
		if strings.TrimSpace(p.Code) != "" {
			perms = append(perms, p.Code)
		}
	}
	sort.Strings(perms)

	if s.cache != nil && s.cache.Enabled() && s.rolesPermsTTL > 0 {
		key := s.userRolesPermsCacheKey(userID)
		if b, err := json.Marshal(struct {
			Roles []string `json:"roles"`
			Perms []string `json:"perms"`
		}{Roles: roles, Perms: perms}); err == nil && len(b) > 0 {
			_ = s.cache.Set(ctx, key, b, s.rolesPermsTTL)
		}
	}

	return roles, perms, nil
}

func (s *RbacService) userRolesPermsCacheKey(userID uint64) string {
	return fmt.Sprintf("rbac:v1:user:%d:roles_perms", userID)
}

func (s *RbacService) InvalidateRoleUsersPerms(ctx context.Context, roleName string) error {
	if s == nil || s.db == nil {
		return nil
	}
	name := strings.TrimSpace(roleName)
	if name == "" {
		return nil
	}

	var role model.Role
	if err := s.db.WithContext(ctx).Where("deleted_at IS NULL AND name = ?", name).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	userIDs, err := s.listActiveUserIDsByRole(s.db.WithContext(ctx), role.ID)
	if err != nil {
		return err
	}
	s.invalidateUserRolesPermsBatch(ctx, userIDs)
	return nil
}

func (s *RbacService) invalidateUserRolesPerms(ctx context.Context, userID uint64) {
	if userID == 0 || s.cache == nil || !s.cache.Enabled() {
		return
	}
	_ = s.cache.Del(ctx, s.userRolesPermsCacheKey(userID))
}
