// bootstrap.go 提供系统启动时的初始化逻辑（幂等）。
//
// 设计说明：
// - 将初始化数据的逻辑从 main.go 移入 service 层，使其可测试、可复用
// - 所有函数均为幂等操作：重复执行不会产生副作用
// - 包含：内置 RBAC 数据（权限点/角色/管理员用户）的创建
package service

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/iam/domain"
)

// EnsureBuiltinRBAC 初始化最小可用 RBAC 数据：
//   - 预置权限点（permissions）
//   - 预置 admin 角色并绑定上述权限点
//   - 预置管理员用户并绑定 admin 角色
//
// 目的：
//   - 开发/测试环境可以"开箱即用"登录后台
//   - 即便数据库为空，也能创建初始管理员避免"无管理员可登录"的死锁
//
// 该函数是幂等的，可安全重复调用。
func EnsureBuiltinRBAC(gdb *gorm.DB, adminUsername, adminPassword string) error {
	if gdb == nil {
		return errors.New("db is required")
	}
	username := strings.TrimSpace(adminUsername)
	password := adminPassword
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin@123"
	}

	permissionCatalog := domain.BuiltinPermissionCatalog()
	allCodes := make([]string, 0, len(permissionCatalog))
	for _, item := range permissionCatalog {
		allCodes = append(allCodes, item.Code)
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 事务：保证初始化过程的原子性（中途失败不会留下半成品数据）。
		// 权限点：只补齐缺失的条目
		var existing []bootstrapPermission
		if err := tx.Where("deleted_at IS NULL AND code IN ?", allCodes).Find(&existing).Error; err != nil {
			return err
		}
		existingMap := map[string]bootstrapPermission{}
		for _, p := range existing {
			existingMap[p.Code] = p
		}
		for _, item := range permissionCatalog {
			d := item.Description
			if existing, ok := existingMap[item.Code]; ok {
				currentDesc := ""
				if existing.Desc != nil {
					currentDesc = strings.TrimSpace(*existing.Desc)
				}
				if currentDesc != d {
					if err := tx.Model(&bootstrapPermission{}).Where("id = ?", existing.ID).Update("desc", &d).Error; err != nil {
						return err
					}
				}
				continue
			}
			if err := tx.Create(&bootstrapPermission{Code: item.Code, Desc: &d}).Error; err != nil {
				return err
			}
		}

		// 角色：admin（不存在则创建）。
		var role bootstrapRole
		if err := tx.Where("deleted_at IS NULL AND name = ?", "admin").First(&role).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			desc := "内置管理员"
			role = bootstrapRole{Name: "admin", Code: "admin", Desc: &desc}
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
		}

		var perms []bootstrapPermission
		if err := tx.Where("deleted_at IS NULL AND code IN ?", allCodes).Find(&perms).Error; err != nil {
			return err
		}
		// 绑定权限：采用全量覆盖，避免遗漏与重复。
		if err := tx.Where("role_id = ?", role.ID).Delete(&bootstrapRolePermission{}).Error; err != nil {
			return err
		}
		links := make([]bootstrapRolePermission, 0, len(perms))
		for _, p := range perms {
			links = append(links, bootstrapRolePermission{RoleID: role.ID, PermissionID: p.ID})
		}
		if len(links) > 0 {
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		// 管理员用户：不存在则创建（密码使用 bcrypt hash）。
		var user bootstrapUser
		if err := tx.Where("deleted_at IS NULL AND username = ?", username).First(&user).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			user = bootstrapUser{Username: username, PasswordHash: string(hash), Status: "active"}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		} else if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			hash, genErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if genErr != nil {
				return genErr
			}
			if err := tx.Model(&bootstrapUser{}).Where("id = ?", user.ID).Update("password_hash", string(hash)).Error; err != nil {
				return err
			}
			user.PasswordHash = string(hash)
		}

		// 绑定 admin 角色（不存在则创建关联）。
		var ur bootstrapUserRole
		if err := tx.Where("user_id = ? AND role_id = ?", user.ID, role.ID).First(&ur).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Create(&bootstrapUserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type bootstrapPermission struct {
	ID   uint64  `gorm:"column:id"`
	Code string  `gorm:"column:code"`
	Desc *string `gorm:"column:desc"`
}

func (bootstrapPermission) TableName() string { return "permissions" }

type bootstrapRole struct {
	ID   uint64  `gorm:"column:id"`
	Name string  `gorm:"column:name"`
	Code string  `gorm:"column:code"`
	Desc *string `gorm:"column:desc"`
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
