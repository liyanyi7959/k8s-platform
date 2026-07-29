package ports

import (
	"context"
	"time"

	"k8s-platform-backend/internal/iam/domain"
)

type AuthRepository interface {
	FindUserByUsername(context.Context, string) (domain.User, error)
	FindUserByID(context.Context, uint64) (domain.User, error)
	FindUserByIdentifier(context.Context, string) (domain.User, error)
	UpdatePassword(context.Context, uint64, string) error
	RolesPermissions(context.Context, uint64) ([]string, []string, error)
	ActiveUserIDsByRoleName(context.Context, string) ([]uint64, error)
}

type PasswordHasher interface {
	Compare(string, string) bool
	Hash(string) (string, error)
}
type Cache interface {
	Enabled() bool
	Get(context.Context, string) ([]byte, bool, error)
	Set(context.Context, string, []byte, time.Duration) error
	Del(context.Context, string) error
}

type MailSender interface {
	Enabled() bool
	SendMail([]string, string, string) error
}

type UserManagementRepository interface {
	ListUsers(context.Context, UserListFilter) ([]UserRecord, int64, error)
	CreateUser(context.Context, CreateUserData) (uint64, error)
	UpdateUser(context.Context, uint64, UpdateUserData) error
	DeleteUser(context.Context, uint64) error
	ResetUserPassword(context.Context, uint64, string) error
}

type RoleManagementRepository interface {
	ListRoles(context.Context) ([]RoleRecord, error)
	CreateRole(context.Context, CreateRoleData) (uint64, error)
	UpdateRole(context.Context, uint64, UpdateRoleData) ([]uint64, error)
	DeleteRole(context.Context, uint64) error
	ListPermissions(context.Context) ([]PermissionRecord, error)
}

type RoleRecord struct {
	ID             uint64
	Name           string
	Code           string
	Description    string
	Permissions    []string
	NamespaceScope *RoleNamespaceScopeRecord
	UserCount      int64
	CreatedAt      string
}

type RoleNamespaceScopeRecord struct {
	ClusterID   uint64
	ClusterName string
	Namespaces  []string
}

type RoleNamespaceScopeData struct {
	ClusterID  uint64
	Namespaces []string
}

type CreateRoleData struct {
	Name, Code  string
	Description *string
	Permissions []string
	Scope       *RoleNamespaceScopeData
}

type UpdateRoleData struct {
	Name, Code     *string
	Description    *string
	SetDescription bool
	Permissions    []string
	SetPermissions bool
	Scope          *RoleNamespaceScopeData
	SetScope       bool
}

type PermissionRecord struct {
	ID          uint64
	Code        string
	Description *string
}

type UserListFilter struct {
	Page, PageSize  int
	Keyword, Status string
	RoleID          uint64
}
type UserRoleRecord struct {
	ID         uint64
	Name, Code string
}
type UserRecord struct {
	ID                                           uint64
	Username, Nickname, Email, Status, CreatedAt string
	Roles                                        []UserRoleRecord
}
type CreateUserData struct {
	Username, Nickname, Email, PasswordHash string
	RoleIDs                                 []uint64
	RoleNames                               []string
}
type UpdateUserData struct {
	Nickname, Email, Status *string
	RoleIDs                 []uint64
	RoleNames               []string
}
