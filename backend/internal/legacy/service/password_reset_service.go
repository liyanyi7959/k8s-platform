package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"k8s-platform-backend/internal/legacy/model"
)

// ── 找回密码 Service ──
// 流程：
// 1. RequestReset: 用户提交用户名或邮箱 → 生成 reset token 存 Redis(10min) → 返回 token
//    （无邮件服务时直接返回 token，前端展示重置链接；有邮件服务时可改为发送邮件）
// 2. ResetPasswordByToken: 用户提交 token + 新密码 → 校验 token → 更新密码 → 删除 token

var (
	ErrResetTokenNotFound = errors.New("reset token not found or expired")
	ErrResetTokenUsed     = errors.New("reset token already used")
)

const (
	resetTokenKeyPrefix = "pwd_reset:"
	resetTokenTTL       = 10 * time.Minute
)

type PasswordResetService struct {
	db    *gorm.DB
	cache CacheStore
	mail  *MailService
}

func NewPasswordResetService(db *gorm.DB, cache CacheStore, mail *MailService) *PasswordResetService {
	return &PasswordResetService{db: db, cache: cache, mail: mail}
}

// resetTokenData 存储在 cache 中的重置 token 数据
type resetTokenData struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Used     bool   `json:"used"`
}

// RequestReset 根据用户名或邮箱生成密码重置 token
func (prs *PasswordResetService) RequestReset(ctx context.Context, identifier string) (token string, username string, err error) {
	if prs == nil || prs.db == nil {
		return "", "", ErrInvalidParams
	}
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", "", ErrInvalidParams
	}

	// 按用户名或邮箱查找用户
	var row model.User
	query := prs.db.WithContext(ctx).Where("deleted_at IS NULL AND (username = ? OR email = ?)", identifier, identifier)
	if err := query.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", ErrNotFound
		}
		return "", "", err
	}

	if row.Status == "disabled" {
		return "", "", ErrAuthUserDisabled
	}

	// 生成安全随机 token
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", err
	}
	token = base64.URLEncoding.EncodeToString(tokenBytes)

	data := resetTokenData{
		UserID:   row.ID,
		Username: row.Username,
		Used:     false,
	}
	b, _ := json.Marshal(data)

	if prs.cache.Enabled() {
		if err := prs.cache.Set(ctx, resetTokenKeyPrefix+token, b, resetTokenTTL); err != nil {
			return "", "", err
		}
	}

	// 如果用户有邮箱且邮件服务可用，发送密码重置邮件
	if prs.mail != nil && prs.mail.Enabled() && row.Email != "" {
		subject := "AIOPS 智能运维平台 - 密码重置"
		body := fmt.Sprintf("您好 %s，\n\n您正在重置 AIOPS 平台密码。\n\n重置验证码：%s\n\n该验证码 10 分钟内有效，请尽快使用。\n\n如非本人操作，请忽略此邮件。\n\nAIOPS 运维平台", row.Username, token)
		if err := prs.mail.SendMail([]string{row.Email}, subject, body); err != nil {
			// 邮件发送失败不阻塞流程，仍返回 token（降级为直接展示）
			return token, row.Username, nil
		}
		// 邮件发送成功，不返回 token（前端提示"已发送邮件"）
		return "", row.Username, nil
	}

	return token, row.Username, nil
}

// ResetPasswordByToken 通过重置 token 设置新密码
func (prs *PasswordResetService) ResetPasswordByToken(ctx context.Context, token string, newPassword string) error {
	if prs == nil || prs.db == nil {
		return ErrInvalidParams
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.TrimSpace(newPassword) == "" {
		return ErrInvalidParams
	}

	// 如果 Redis 未启用，不支持找回密码
	if !prs.cache.Enabled() {
		return errors.New("密码重置服务不可用（需 Redis 支持）")
	}

	key := resetTokenKeyPrefix + token
	b, ok, err := prs.cache.Get(ctx, key)
	if err != nil {
		return err
	}
	if !ok || len(b) == 0 {
		return ErrResetTokenNotFound
	}

	var data resetTokenData
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	if data.Used {
		_ = prs.cache.Del(ctx, key)
		return ErrResetTokenUsed
	}

	// 生成新密码 hash
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 更新密码
	if err := prs.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", data.UserID).Update("password_hash", string(hash)).Error; err != nil {
		return err
	}

	// 删除 token（一次性）
	_ = prs.cache.Del(ctx, key)
	return nil
}
