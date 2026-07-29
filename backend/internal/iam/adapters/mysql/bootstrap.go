package mysql

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/iam/domain"
)

// EnsureBuiltinRBAC makes the minimum IAM administration data available at
// startup. It is deliberately owned by the IAM persistence adapter because it
// writes the IAM relational schema directly.
func EnsureBuiltinRBAC(db *gorm.DB, adminUsername, adminPassword string) error {
	if db == nil {
		return errors.New("db is required")
	}
	username := strings.TrimSpace(adminUsername)
	if username == "" {
		username = "admin"
	}
	password := adminPassword
	if password == "" {
		password = "admin@123"
	}

	catalog := domain.BuiltinPermissionCatalog()
	codes := make([]string, 0, len(catalog))
	for _, item := range catalog {
		codes = append(codes, item.Code)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var existing []bootstrapPermission
		if err := tx.Where("deleted_at IS NULL AND code IN ?", codes).Find(&existing).Error; err != nil {
			return err
		}
		existingByCode := make(map[string]bootstrapPermission, len(existing))
		for _, permission := range existing {
			existingByCode[permission.Code] = permission
		}
		for _, item := range catalog {
			if current, ok := existingByCode[item.Code]; ok {
				currentDescription := ""
				if current.Description != nil {
					currentDescription = strings.TrimSpace(*current.Description)
				}
				if currentDescription != item.Description {
					description := item.Description
					if err := tx.Model(&bootstrapPermission{}).Where("id = ?", current.ID).Update("desc", &description).Error; err != nil {
						return err
					}
				}
				continue
			}
			description := item.Description
			if err := tx.Create(&bootstrapPermission{Code: item.Code, Description: &description}).Error; err != nil {
				return err
			}
		}

		var adminRole bootstrapRole
		if err := tx.Where("deleted_at IS NULL AND name = ?", "admin").First(&adminRole).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			description := "内置管理员"
			adminRole = bootstrapRole{Name: "admin", Code: "admin", Description: &description}
			if err := tx.Create(&adminRole).Error; err != nil {
				return err
			}
		}

		var permissions []bootstrapPermission
		if err := tx.Where("deleted_at IS NULL AND code IN ?", codes).Find(&permissions).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", adminRole.ID).Delete(&bootstrapRolePermission{}).Error; err != nil {
			return err
		}
		links := make([]bootstrapRolePermission, 0, len(permissions))
		for _, permission := range permissions {
			links = append(links, bootstrapRolePermission{RoleID: adminRole.ID, PermissionID: permission.ID})
		}
		if len(links) > 0 {
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		var adminUser bootstrapUser
		if err := tx.Where("deleted_at IS NULL AND username = ?", username).First(&adminUser).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if hashErr != nil {
				return hashErr
			}
			adminUser = bootstrapUser{Username: username, PasswordHash: string(hash), Status: "active"}
			if err := tx.Create(&adminUser).Error; err != nil {
				return err
			}
		} else if bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(password)) != nil {
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if hashErr != nil {
				return hashErr
			}
			if err := tx.Model(&bootstrapUser{}).Where("id = ?", adminUser.ID).Update("password_hash", string(hash)).Error; err != nil {
				return err
			}
			adminUser.PasswordHash = string(hash)
		}

		var link bootstrapUserRole
		if err := tx.Where("user_id = ? AND role_id = ?", adminUser.ID, adminRole.ID).First(&link).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return tx.Create(&bootstrapUserRole{UserID: adminUser.ID, RoleID: adminRole.ID}).Error
		}
		return nil
	})
}

type bootstrapPermission struct {
	ID          uint64  `gorm:"column:id"`
	Code        string  `gorm:"column:code"`
	Description *string `gorm:"column:desc"`
}

func (bootstrapPermission) TableName() string { return "permissions" }

type bootstrapRole struct {
	ID          uint64  `gorm:"column:id"`
	Name        string  `gorm:"column:name"`
	Code        string  `gorm:"column:code"`
	Description *string `gorm:"column:desc"`
}

func (bootstrapRole) TableName() string { return "roles" }

type bootstrapRolePermission struct {
	RoleID       uint64 `gorm:"column:role_id;primaryKey"`
	PermissionID uint64 `gorm:"column:permission_id;primaryKey"`
}

func (bootstrapRolePermission) TableName() string { return "role_permissions" }

type bootstrapUser struct {
	ID           uint64    `gorm:"column:id"`
	Username     string    `gorm:"column:username"`
	PasswordHash string    `gorm:"column:password_hash"`
	Status       string    `gorm:"column:status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (bootstrapUser) TableName() string { return "users" }

type bootstrapUserRole struct {
	UserID uint64 `gorm:"column:user_id;primaryKey"`
	RoleID uint64 `gorm:"column:role_id;primaryKey"`
}

func (bootstrapUserRole) TableName() string { return "user_roles" }
