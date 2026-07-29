package mysql

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/iam/domain"
)

type AuthRepository struct{ db *gorm.DB }

func NewAuthRepository(db *gorm.DB) *AuthRepository { return &AuthRepository{db: db} }
func (r *AuthRepository) FindUserByUsername(ctx context.Context, username string) (domain.User, error) {
	var row userRow
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND username = ?", username).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}
	return toUser(row), nil
}
func (r *AuthRepository) FindUserByID(ctx context.Context, id uint64) (domain.User, error) {
	var row userRow
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	return toUser(row), nil
}
func (r *AuthRepository) FindUserByIdentifier(ctx context.Context, identifier string) (domain.User, error) {
	var row userRow
	if err := r.db.WithContext(ctx).Where("deleted_at IS NULL AND (username = ? OR email = ?)", identifier, identifier).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	return toUser(row), nil
}
func (r *AuthRepository) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	return r.db.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Update("password_hash", hash).Error
}
func (r *AuthRepository) RolesPermissions(ctx context.Context, id uint64) ([]string, []string, error) {
	var roleRows []textRow
	if err := r.db.WithContext(ctx).Raw(`SELECT r.name AS value FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id = ? AND r.deleted_at IS NULL`, id).Scan(&roleRows).Error; err != nil {
		return nil, nil, err
	}
	var permissionRows []textRow
	if err := r.db.WithContext(ctx).Raw(`SELECT DISTINCT p.code AS value FROM user_roles ur JOIN role_permissions rp ON ur.role_id = rp.role_id JOIN permissions p ON rp.permission_id = p.id WHERE ur.user_id = ? AND p.deleted_at IS NULL`, id).Scan(&permissionRows).Error; err != nil {
		return nil, nil, err
	}
	roles, permissions := values(roleRows), values(permissionRows)
	return roles, permissions, nil
}
func (r *AuthRepository) ActiveUserIDsByRoleName(ctx context.Context, name string) ([]uint64, error) {
	var role struct{ ID uint64 }
	if err := r.db.WithContext(ctx).Table("roles").Select("id").Where("deleted_at IS NULL AND name = ?", name).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	var rows []struct {
		UserID uint64 `gorm:"column:user_id"`
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT ur.user_id FROM user_roles ur JOIN users u ON ur.user_id = u.id WHERE ur.role_id = ? AND u.deleted_at IS NULL`, role.ID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.UserID)
	}
	return ids, nil
}

type BcryptHasher struct{}

func (BcryptHasher) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
func (BcryptHasher) Hash(password string) (string, error) {
	value, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(value), err
}

type userRow struct {
	ID           uint64    `gorm:"column:id"`
	Username     string    `gorm:"column:username"`
	Email        string    `gorm:"column:email"`
	Nickname     string    `gorm:"column:nickname"`
	Status       string    `gorm:"column:status"`
	PasswordHash string    `gorm:"column:password_hash"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}
type textRow struct {
	Value string `gorm:"column:value"`
}

func (userRow) TableName() string { return "users" }
func toUser(row userRow) domain.User {
	return domain.User{ID: row.ID, Username: row.Username, Email: row.Email, Status: row.Status, PasswordHash: row.PasswordHash}
}
func values(rows []textRow) []string {
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		if value := strings.TrimSpace(row.Value); value != "" {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
