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

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/model"
)

// BuiltinPermissions 返回系统预置权限点列表。
// 权限点是"后端授权"的最小单位：路由中间件会校验 token 中是否包含对应 code。
func BuiltinPermissions() map[string]string {
	return map[string]string{
		"cluster:read":         "集群查看",
		"cluster:create":       "集群创建",
		"k8s:read":             "K8s 资源查看",
		"k8s:write":            "K8s 资源管理",
		"k8s:exec":             "K8s Pod 终端",
		"k8s:secret_reveal":    "K8s Secret 明文查看",
		"k8s:rbac_read":        "K8s RBAC 查看",
		"k8s:rbac_write":       "K8s RBAC 管理",
		"k8s:permission_audit": "K8s 最小权限分析",
		"user:read":            "用户/审计查看",
		"user:write":           "用户/角色管理",
	}
}

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
		password = "admin"
	}

	requiredPerms := BuiltinPermissions()
	allCodes := make([]string, 0, len(requiredPerms))
	for k := range requiredPerms {
		allCodes = append(allCodes, k)
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		// 事务：保证初始化过程的原子性（中途失败不会留下半成品数据）。
		// 权限点：只补齐缺失的条目
		var existing []model.Permission
		if err := tx.Where("deleted_at IS NULL AND code IN ?", allCodes).Find(&existing).Error; err != nil {
			return err
		}
		existingMap := map[string]model.Permission{}
		for _, p := range existing {
			existingMap[p.Code] = p
		}
		for code, desc := range requiredPerms {
			if _, ok := existingMap[code]; ok {
				continue
			}
			d := desc
			if err := tx.Create(&model.Permission{Code: code, Desc: &d}).Error; err != nil {
				return err
			}
		}

		// 角色：admin（不存在则创建）。
		var role model.Role
		if err := tx.Where("deleted_at IS NULL AND name = ?", "admin").First(&role).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			desc := "内置管理员"
			role = model.Role{Name: "admin", Desc: &desc}
			if err := tx.Create(&role).Error; err != nil {
				return err
			}
		}

		var perms []model.Permission
		if err := tx.Where("deleted_at IS NULL AND code IN ?", allCodes).Find(&perms).Error; err != nil {
			return err
		}
		// 绑定权限：采用全量覆盖，避免遗漏与重复。
		if err := tx.Where("role_id = ?", role.ID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		links := make([]model.RolePermission, 0, len(perms))
		for _, p := range perms {
			links = append(links, model.RolePermission{RoleID: role.ID, PermissionID: p.ID})
		}
		if len(links) > 0 {
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}

		// 管理员用户：不存在则创建（密码使用 bcrypt hash）。
		var user model.User
		if err := tx.Where("deleted_at IS NULL AND username = ?", username).First(&user).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			user = model.User{Username: username, PasswordHash: string(hash), Status: "active"}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		}

		// 绑定 admin 角色（不存在则创建关联）。
		var ur model.UserRole
		if err := tx.Where("user_id = ? AND role_id = ?", user.ID, role.ID).First(&ur).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
