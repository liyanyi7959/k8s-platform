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
	// db 为 GORM 连接，统一由 main 注入。
	db *gorm.DB

	cache         CacheStore
	rolesPermsTTL time.Duration
}

func NewRbacService(db *gorm.DB, cacheStore CacheStore, rolesPermsTTL time.Duration) *RbacService {
	// RbacService 聚合“用户/角色/权限点”相关的核心业务逻辑。
	if rolesPermsTTL <= 0 {
		rolesPermsTTL = 15 * time.Minute
	}
	return &RbacService{db: db, cache: cacheStore, rolesPermsTTL: rolesPermsTTL}
}

type PermissionItem struct {
	// PermissionItem 为对外返回的权限点视图模型（避免直接暴露 DB 模型）。
	ID   uint64 `json:"id"`
	Code string `json:"code"`
	Desc string `json:"desc"`
}

type RoleItem struct {
	// RoleItem 为对外返回的角色视图模型。
	ID        uint64  `json:"id"`
	Name      string  `json:"name"`
	Desc      *string `json:"desc,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
}

type UserItem struct {
	// UserItem 为对外返回的用户视图模型（含角色名数组）。
	ID        uint64   `json:"id"`
	Username  string   `json:"username"`
	Status    string   `json:"status"`
	RoleIDs   []uint64 `json:"role_ids"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at,omitempty"`
}

type ListPermissionsRequest struct {
	Page     int
	PageSize int
}

func (s *RbacService) ListPermissions(ctx context.Context, req ListPermissionsRequest) (PageResult[PermissionItem], error) {
	// ListPermissions 分页列出权限点。
	page, pageSize := normalizePage(req.Page, req.PageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.Permission{}).Where("deleted_at IS NULL")
	if err := q.Count(&total).Error; err != nil {
		return PageResult[PermissionItem]{}, err
	}
	var rows []model.Permission
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[PermissionItem]{}, err
	}
	list := make([]PermissionItem, 0, len(rows))
	for _, p := range rows {
		desc := ""
		if p.Desc != nil {
			desc = *p.Desc
		}
		list = append(list, PermissionItem{ID: p.ID, Code: p.Code, Desc: desc})
	}
	return PageResult[PermissionItem]{List: list, Total: int(total), Page: page, PageSize: pageSize}, nil
}

type ListRolesRequest struct {
	Page     int
	PageSize int
	Keyword  string
	SortBy   string
	Order    string
}

func (s *RbacService) ListRoles(ctx context.Context, req ListRolesRequest) (PageResult[RoleItem], error) {
	// ListRoles 分页列出角色，可按 keyword 搜索，并支持 created_at 排序。
	page, pageSize := normalizePage(req.Page, req.PageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.Role{}).Where("deleted_at IS NULL")
	kw := strings.TrimSpace(req.Keyword)
	if kw != "" {
		q = q.Where("name LIKE ?", "%"+kw+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return PageResult[RoleItem]{}, err
	}

	orderClause := "id desc"
	if req.SortBy == "created_at" {
		if strings.ToLower(req.Order) == "asc" {
			orderClause = "created_at asc"
		} else {
			orderClause = "created_at desc"
		}
	}
	var rows []model.Role
	if err := q.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return PageResult[RoleItem]{}, err
	}
	list := make([]RoleItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, RoleItem{
			ID:        r.ID,
			Name:      r.Name,
			Desc:      r.Desc,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return PageResult[RoleItem]{List: list, Total: int(total), Page: page, PageSize: pageSize}, nil
}

type CreateRoleRequest struct {
	Name string
	Desc *string
}

func (s *RbacService) CreateRole(ctx context.Context, req CreateRoleRequest) error {
	// CreateRole 创建角色（name 唯一）。
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrInvalidParams
	}
	var exists int64
	if err := s.db.WithContext(ctx).Model(&model.Role{}).Where("deleted_at IS NULL AND name = ?", name).Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return ErrConflict
	}
	r := model.Role{Name: name, Desc: req.Desc}
	if err := s.db.WithContext(ctx).Create(&r).Error; err != nil {
		return err
	}
	return nil
}

type UpdateRolePermissionsRequest struct {
	PermissionCodes []string
}

func (s *RbacService) UpdateRolePermissions(ctx context.Context, roleID uint64, req UpdateRolePermissionsRequest) error {
	// UpdateRolePermissions 以“全量覆盖”的方式更新角色权限点：
	// 1) 清空 role_permissions
	// 2) 确保 permissions 表中存在对应 code（不存在则创建）
	// 3) 重建 role_permissions 关联
	if roleID == 0 {
		return ErrInvalidParams
	}
	codes := make([]string, 0, len(req.PermissionCodes))
	seen := map[string]bool{}
	for _, raw := range req.PermissionCodes {
		code := strings.TrimSpace(raw)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		codes = append(codes, code)
	}
	sort.Strings(codes)

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 事务保证 role_permissions 不会出现半更新状态。
		var role model.Role
		if err := tx.Where("id = ? AND deleted_at IS NULL", roleID).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		if len(codes) == 0 {
			// 权限点为空表示清空该角色的所有权限。
			return nil
		}

		var perms []model.Permission
		if err := tx.Where("deleted_at IS NULL AND code IN ?", codes).Find(&perms).Error; err != nil {
			return err
		}
		found := map[string]model.Permission{}
		for _, p := range perms {
			found[p.Code] = p
		}

		for _, code := range codes {
			if _, ok := found[code]; ok {
				continue
			}
			// 对前端传入但 DB 不存在的权限点，采用“自动补齐”策略：
			// 便于快速扩展权限点集合（也可在未来改为“必须预置，否则 invalid params”）。
			p := model.Permission{Code: code}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
			found[code] = p
		}

		links := make([]model.RolePermission, 0, len(codes))
		for _, code := range codes {
			links = append(links, model.RolePermission{RoleID: roleID, PermissionID: found[code].ID})
		}
		if err := tx.Create(&links).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	s.invalidateRoleUsersRolesPerms(ctx, roleID)
	return nil
}

func (s *RbacService) GetRolePermissionCodes(ctx context.Context, roleID uint64) ([]string, error) {
	if roleID == 0 {
		return nil, ErrInvalidParams
	}
	var role model.Role
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", roleID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var codes []string
	if err := s.db.WithContext(ctx).
		Model(&model.Permission{}).
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Where("rp.role_id = ? AND permissions.deleted_at IS NULL", roleID).
		Order("permissions.code asc").
		Pluck("permissions.code", &codes).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(codes))
	seen := map[string]bool{}
	for _, c := range codes {
		v := strings.TrimSpace(c)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

type ListUsersRequest struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string
	SortBy   string
	Order    string
}

func (s *RbacService) ListUsers(ctx context.Context, req ListUsersRequest) (PageResult[UserItem], error) {
	// ListUsers 分页列出用户，可按 username 模糊搜索、按 status 过滤，并支持 created_at 排序。
	page, pageSize := normalizePage(req.Page, req.PageSize)
	var total int64
	q := s.db.WithContext(ctx).Model(&model.User{}).Where("deleted_at IS NULL")
	kw := strings.TrimSpace(req.Keyword)
	if kw != "" {
		q = q.Where("username LIKE ?", "%"+kw+"%")
	}
	if st := strings.TrimSpace(req.Status); st != "" {
		q = q.Where("status = ?", st)
	}
	if err := q.Count(&total).Error; err != nil {
		return PageResult[UserItem]{}, err
	}

	orderClause := "id desc"
	if req.SortBy == "created_at" {
		if strings.ToLower(req.Order) == "asc" {
			orderClause = "created_at asc"
		} else {
			orderClause = "created_at desc"
		}
	}
	var users []model.User
	if err := q.Order(orderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return PageResult[UserItem]{}, err
	}

	ids := make([]uint64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	roleMap, err := s.listUserRoleNames(ctx, ids)
	if err != nil {
		return PageResult[UserItem]{}, err
	}
	roleIDMap, err := s.listUserRoleIDs(ctx, ids)
	if err != nil {
		return PageResult[UserItem]{}, err
	}

	list := make([]UserItem, 0, len(users))
	for _, u := range users {
		list = append(list, UserItem{
			ID:        u.ID,
			Username:  u.Username,
			Status:    u.Status,
			RoleIDs:   roleIDMap[u.ID],
			Roles:     roleMap[u.ID],
			CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return PageResult[UserItem]{List: list, Total: int(total), Page: page, PageSize: pageSize}, nil
}

type CreateUserRequest struct {
	Username string
	Password string
	RoleIDs  []uint64
	Status   string
}

func (s *RbacService) CreateUser(ctx context.Context, req CreateUserRequest) error {
	// CreateUser 创建用户：
	// - bcrypt hash 存储密码
	// - 通过 user_roles 绑定角色（role_ids 为全量集合）
	username := strings.TrimSpace(req.Username)
	if username == "" || strings.TrimSpace(req.Password) == "" {
		return ErrInvalidParams
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "disabled" {
		return ErrInvalidParams
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	roleIDs := uniqueUint64(req.RoleIDs)
	var createdUserID uint64
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 事务保证用户创建与角色绑定一致性。
		var exists int64
		if err := tx.Model(&model.User{}).Where("deleted_at IS NULL AND username = ?", username).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return ErrConflict
		}

		u := model.User{Username: username, PasswordHash: string(hash), Status: status}
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		createdUserID = u.ID

		if len(roleIDs) == 0 {
			// 不绑定角色也允许创建用户（后续可再 patch 绑定）。
			return nil
		}
		var roles []model.Role
		if err := tx.Where("deleted_at IS NULL AND id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
		if len(roles) != len(roleIDs) {
			// 有角色不存在时直接判定参数非法，避免写入脏关联。
			return ErrInvalidParams
		}
		links := make([]model.UserRole, 0, len(roleIDs))
		for _, rid := range roleIDs {
			links = append(links, model.UserRole{UserID: u.ID, RoleID: rid})
		}
		return tx.Create(&links).Error
	}); err != nil {
		return err
	}
	s.invalidateUserRolesPerms(ctx, createdUserID)
	return nil
}

type PatchUserRequest struct {
	Status  *string
	RoleIDs *[]uint64
}

func (s *RbacService) PatchUser(ctx context.Context, userID uint64, req PatchUserRequest) error {
	// PatchUser 支持部分更新：
	// - status：仅允许 active/disabled
	// - role_ids：全量覆盖更新用户角色（先删后建）
	if userID == 0 {
		return ErrInvalidParams
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if req.Status != nil {
			status := strings.TrimSpace(*req.Status)
			if status != "active" && status != "disabled" {
				return ErrInvalidParams
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("status", status).Error; err != nil {
				return err
			}
		}

		if req.RoleIDs != nil {
			// 注意：role_ids 采用全量覆盖策略，调用方需传入最终期望的角色集合。
			roleIDs := uniqueUint64(*req.RoleIDs)
			if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
				return err
			}
			if len(roleIDs) == 0 {
				return nil
			}
			var roles []model.Role
			if err := tx.Where("deleted_at IS NULL AND id IN ?", roleIDs).Find(&roles).Error; err != nil {
				return err
			}
			if len(roles) != len(roleIDs) {
				return ErrInvalidParams
			}
			links := make([]model.UserRole, 0, len(roleIDs))
			for _, rid := range roleIDs {
				links = append(links, model.UserRole{UserID: userID, RoleID: rid})
			}
			return tx.Create(&links).Error
		}
		return nil
	}); err != nil {
		return err
	}
	if req.Status != nil || req.RoleIDs != nil {
		s.invalidateUserRolesPerms(ctx, userID)
	}
	return nil
}

func (s *RbacService) ResetUserPassword(ctx context.Context, userID uint64, newPassword string) error {
	// ResetUserPassword 用于管理员重置指定用户密码。
	// 与 ChangePassword 不同，这里不校验旧密码。
	if userID == 0 || strings.TrimSpace(newPassword) == "" {
		return ErrInvalidParams
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ? AND deleted_at IS NULL", userID).Update("password_hash", string(hash))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

type AuthUser struct {
	// AuthUser 为登录成功后返回给控制器的简化结构。
	ID       int64
	Username string
	Status   string
	Roles    []string
	Perms    []string
}

func (s *RbacService) Authenticate(ctx context.Context, username, password string) (*AuthUser, error) {
	// Authenticate 校验用户名密码并返回该用户的角色与权限点集合。
	// 安全策略：对外尽量不区分“用户不存在/密码错误/用户禁用”，统一返回 not found / unauthorized。
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
	// bcrypt.CompareHashAndPassword 会在密码不匹配时返回错误。
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
	// ChangePassword 用于当前登录用户修改密码（需要校验旧密码）。
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
	return s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("password_hash", string(hash)).Error
}

func (s *RbacService) GetUserRolesPerms(ctx context.Context, userID uint64) ([]string, []string, error) {
	// GetUserRolesPerms 查询用户拥有的角色名与权限点 code。
	// 说明：这里使用 Raw SQL 进行 join 查询，避免写复杂的 GORM Preload 关联。
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

func (s *RbacService) invalidateUserRolesPerms(ctx context.Context, userID uint64) {
	if userID == 0 || s.cache == nil || !s.cache.Enabled() {
		return
	}
	_ = s.cache.Del(ctx, s.userRolesPermsCacheKey(userID))
}

func (s *RbacService) invalidateRoleUsersRolesPerms(ctx context.Context, roleID uint64) {
	if roleID == 0 || s.cache == nil || !s.cache.Enabled() {
		return
	}
	userIDs, err := s.listUserIDsByRole(ctx, roleID)
	if err != nil {
		return
	}
	for _, uid := range userIDs {
		s.invalidateUserRolesPerms(ctx, uid)
	}
}

func (s *RbacService) listUserIDsByRole(ctx context.Context, roleID uint64) ([]uint64, error) {
	type row struct {
		UserID uint64 `gorm:"column:user_id"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`SELECT ur.user_id FROM user_roles ur WHERE ur.role_id = ?`, roleID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]uint64, 0, len(rows))
	for _, r := range rows {
		if r.UserID > 0 {
			out = append(out, r.UserID)
		}
	}
	return uniqueUint64(out), nil
}

func (s *RbacService) listUserRoleIDs(ctx context.Context, userIDs []uint64) (map[uint64][]uint64, error) {
	out := map[uint64][]uint64{}
	if len(userIDs) == 0 {
		return out, nil
	}
	type row struct {
		UserID uint64 `gorm:"column:user_id"`
		RoleID uint64 `gorm:"column:role_id"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Raw(`SELECT ur.user_id, ur.role_id FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id IN ? AND r.deleted_at IS NULL`, userIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], r.RoleID)
	}
	for id := range out {
		out[id] = uniqueUint64(out[id])
	}
	return out, nil
}

func (s *RbacService) listUserRoleNames(ctx context.Context, userIDs []uint64) (map[uint64][]string, error) {
	// listUserRoleNames 批量查询用户的角色名，返回 map[userID][]roleName。
	out := map[uint64][]string{}
	if len(userIDs) == 0 {
		return out, nil
	}
	type row struct {
		UserID uint64 `gorm:"column:user_id"`
		Name   string `gorm:"column:name"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Raw(`SELECT ur.user_id, r.name FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id IN ? AND r.deleted_at IS NULL`, userIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], r.Name)
	}
	for id := range out {
		sort.Strings(out[id])
	}
	return out, nil
}

// normalizePage 已迁移至 helpers.go（多个 service 共用）。

func uniqueUint64(in []uint64) []uint64 {
	// uniqueUint64 对 uint64 列表去重、过滤 0，并排序。
	seen := map[uint64]bool{}
	out := make([]uint64, 0, len(in))
	for _, v := range in {
		if v == 0 || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
