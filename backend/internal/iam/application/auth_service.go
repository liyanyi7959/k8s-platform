package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/internal/iam/ports"
)

type AuthService struct {
	repository ports.AuthRepository
	hasher     ports.PasswordHasher
	cache      ports.Cache
	ttl        time.Duration
}

func NewAuthService(repository ports.AuthRepository, hasher ports.PasswordHasher, cache ports.Cache, ttl time.Duration) *AuthService {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &AuthService{repository: repository, hasher: hasher, cache: cache, ttl: ttl}
}

func (s *AuthService) Authenticate(ctx context.Context, username, password string) (domain.Principal, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(password) == "" {
		return domain.Principal{}, domain.ErrInvalidParams
	}
	user, err := s.repository.FindUserByUsername(ctx, username)
	if err != nil {
		return domain.Principal{}, err
	}
	if user.Status == "disabled" {
		return domain.Principal{}, domain.ErrUserDisabled
	}
	if !s.hasher.Compare(user.PasswordHash, password) {
		return domain.Principal{}, domain.ErrPasswordIncorrect
	}
	roles, permissions, err := s.RolesPermissions(ctx, user.ID)
	if err != nil {
		return domain.Principal{}, err
	}
	return domain.Principal{ID: int64(user.ID), Username: user.Username, Status: user.Status, Roles: roles, Permissions: permissions}, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uint64, oldPassword, newPassword string) error {
	if userID == 0 || strings.TrimSpace(oldPassword) == "" || strings.TrimSpace(newPassword) == "" {
		return domain.ErrInvalidParams
	}
	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !s.hasher.Compare(user.PasswordHash, oldPassword) {
		return domain.ErrOldPasswordIncorrect
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	if err := s.repository.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	s.InvalidateUser(ctx, userID)
	return nil
}

func (s *AuthService) RolesPermissions(ctx context.Context, userID uint64) ([]string, []string, error) {
	if userID == 0 {
		return nil, nil, domain.ErrInvalidParams
	}
	key := cacheKey(userID)
	if s.cache != nil && s.cache.Enabled() && s.ttl > 0 {
		if data, ok, err := s.cache.Get(ctx, key); err == nil && ok {
			var cached struct {
				Roles       []string `json:"roles"`
				Permissions []string `json:"perms"`
			}
			if json.Unmarshal(data, &cached) == nil {
				return cached.Roles, cached.Permissions, nil
			}
		}
	}
	roles, permissions, err := s.repository.RolesPermissions(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if s.cache != nil && s.cache.Enabled() && s.ttl > 0 {
		if data, err := json.Marshal(struct {
			Roles       []string `json:"roles"`
			Permissions []string `json:"perms"`
		}{roles, permissions}); err == nil {
			_ = s.cache.Set(ctx, key, data, s.ttl)
		}
	}
	return roles, permissions, nil
}

func (s *AuthService) InvalidateRole(ctx context.Context, roleName string) error {
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		return nil
	}
	ids, err := s.repository.ActiveUserIDsByRoleName(ctx, roleName)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil
		}
		return err
	}
	for _, id := range ids {
		s.InvalidateUser(ctx, id)
	}
	return nil
}
func (s *AuthService) InvalidateUser(ctx context.Context, userID uint64) {
	if userID > 0 && s.cache != nil && s.cache.Enabled() {
		_ = s.cache.Del(ctx, cacheKey(userID))
	}
}
func cacheKey(userID uint64) string { return fmt.Sprintf("rbac:v1:user:%d:roles_perms", userID) }
