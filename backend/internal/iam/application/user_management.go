package application

import (
	"context"
	"strings"

	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/internal/iam/ports"
)

type UserManagement struct {
	repository ports.UserManagementRepository
	hasher     ports.PasswordHasher
	auth       *AuthService
}

func NewUserManagement(repository ports.UserManagementRepository, hasher ports.PasswordHasher, auth *AuthService) *UserManagement {
	return &UserManagement{repository: repository, hasher: hasher, auth: auth}
}

type UserListParams struct {
	Page, PageSize  int
	Keyword, Status string
	RoleID          uint64
}
type UserRoleItem struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}
type UserListItem struct {
	ID        uint64         `json:"id"`
	Username  string         `json:"username"`
	Nickname  string         `json:"nickname"`
	Email     string         `json:"email"`
	Status    string         `json:"status"`
	Enabled   bool           `json:"enabled"`
	Roles     []UserRoleItem `json:"roles"`
	CreatedAt string         `json:"created_at"`
}
type UserListResult struct {
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Items    []UserListItem `json:"items"`
}
type CreateUserRequest struct {
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	RoleIDs  []uint64 `json:"role_ids"`
	Roles    []string `json:"roles"`
}
type UpdateUserRequest struct {
	Nickname *string  `json:"nickname"`
	Email    *string  `json:"email"`
	Status   *string  `json:"status"`
	RoleIDs  []uint64 `json:"role_ids"`
	Roles    []string `json:"roles"`
}

func (s *UserManagement) List(ctx context.Context, params UserListParams) (*UserListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	rows, total, err := s.repository.ListUsers(ctx, ports.UserListFilter{Page: params.Page, PageSize: params.PageSize, Keyword: params.Keyword, Status: params.Status, RoleID: params.RoleID})
	if err != nil {
		return nil, err
	}
	items := make([]UserListItem, 0, len(rows))
	for _, row := range rows {
		roles := make([]UserRoleItem, 0, len(row.Roles))
		for _, role := range row.Roles {
			roles = append(roles, UserRoleItem{ID: role.ID, Name: role.Name, Code: role.Code})
		}
		items = append(items, UserListItem{ID: row.ID, Username: row.Username, Nickname: row.Nickname, Email: row.Email, Status: row.Status, Enabled: row.Status == "active", Roles: roles, CreatedAt: row.CreatedAt})
	}
	return &UserListResult{Total: total, Page: params.Page, PageSize: params.PageSize, Items: items}, nil
}
func (s *UserManagement) Create(ctx context.Context, request CreateUserRequest) (uint64, error) {
	request.Username, request.Password = strings.TrimSpace(request.Username), strings.TrimSpace(request.Password)
	if request.Username == "" || request.Password == "" {
		return 0, domain.ErrInvalidParams
	}
	if len(request.Password) < 8 {
		return 0, domain.ErrInvalidParams
	}
	hash, err := s.hasher.Hash(request.Password)
	if err != nil {
		return 0, err
	}
	return s.repository.CreateUser(ctx, ports.CreateUserData{Username: request.Username, Nickname: request.Nickname, Email: request.Email, PasswordHash: hash, RoleIDs: request.RoleIDs, RoleNames: request.Roles})
}
func (s *UserManagement) Update(ctx context.Context, id uint64, request UpdateUserRequest) error {
	if id == 0 {
		return domain.ErrInvalidParams
	}
	if request.Status != nil {
		value := strings.TrimSpace(*request.Status)
		if value != "active" && value != "disabled" {
			return domain.ErrInvalidParams
		}
		request.Status = &value
	}
	if err := s.repository.UpdateUser(ctx, id, ports.UpdateUserData{Nickname: request.Nickname, Email: request.Email, Status: request.Status, RoleIDs: request.RoleIDs, RoleNames: request.Roles}); err != nil {
		return err
	}
	s.auth.InvalidateUser(ctx, id)
	return nil
}
func (s *UserManagement) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return domain.ErrInvalidParams
	}
	if err := s.repository.DeleteUser(ctx, id); err != nil {
		return err
	}
	s.auth.InvalidateUser(ctx, id)
	return nil
}
func (s *UserManagement) ResetPassword(ctx context.Context, id uint64, password string) error {
	if id == 0 || strings.TrimSpace(password) == "" {
		return domain.ErrInvalidParams
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}
	if err := s.repository.ResetUserPassword(ctx, id, hash); err != nil {
		return err
	}
	s.auth.InvalidateUser(ctx, id)
	return nil
}
