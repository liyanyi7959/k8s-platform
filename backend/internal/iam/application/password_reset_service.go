package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/internal/iam/ports"
)

const resetTokenPrefix = "pwd_reset:"
const resetTokenTTL = 10 * time.Minute

type PasswordResetService struct {
	repository ports.AuthRepository
	hasher     ports.PasswordHasher
	cache      ports.Cache
	mail       ports.MailSender
}
type resetTokenData struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Used     bool   `json:"used"`
}

func NewPasswordResetService(repository ports.AuthRepository, hasher ports.PasswordHasher, cache ports.Cache, mail ports.MailSender) *PasswordResetService {
	return &PasswordResetService{repository: repository, hasher: hasher, cache: cache, mail: mail}
}
func (s *PasswordResetService) RequestReset(ctx context.Context, identifier string) (string, string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", "", domain.ErrInvalidParams
	}
	user, err := s.repository.FindUserByIdentifier(ctx, identifier)
	if err != nil {
		return "", "", err
	}
	if user.Status == "disabled" {
		return "", "", domain.ErrUserDisabled
	}
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return "", "", err
	}
	token := base64.URLEncoding.EncodeToString(random)
	encoded, _ := json.Marshal(resetTokenData{UserID: user.ID, Username: user.Username})
	if s.cache != nil && s.cache.Enabled() {
		if err := s.cache.Set(ctx, resetTokenPrefix+token, encoded, resetTokenTTL); err != nil {
			return "", "", err
		}
	}
	if s.mail != nil && s.mail.Enabled() && user.Email != "" {
		subject := "AIOPS 智能运维平台 - 密码重置"
		body := fmt.Sprintf("您好 %s，\n\n您的密码重置验证码为：%s\n\n该验证码 10 分钟内有效。", user.Username, token)
		if err := s.mail.SendMail([]string{user.Email}, subject, body); err == nil {
			return "", user.Username, nil
		}
	}
	return token, user.Username, nil
}
func (s *PasswordResetService) ResetPasswordByToken(ctx context.Context, token, password string) error {
	token = strings.TrimSpace(token)
	if token == "" || strings.TrimSpace(password) == "" {
		return domain.ErrInvalidParams
	}
	if s.cache == nil || !s.cache.Enabled() {
		return errors.New("密码重置服务不可用（需要 Redis 支持）")
	}
	key := resetTokenPrefix + token
	encoded, ok, err := s.cache.Get(ctx, key)
	if err != nil {
		return err
	}
	if !ok || len(encoded) == 0 {
		return domain.ErrResetTokenNotFound
	}
	var data resetTokenData
	if err := json.Unmarshal(encoded, &data); err != nil {
		return err
	}
	if data.Used {
		_ = s.cache.Del(ctx, key)
		return domain.ErrResetTokenUsed
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}
	if err := s.repository.UpdatePassword(ctx, data.UserID, hash); err != nil {
		return err
	}
	_ = s.cache.Del(ctx, key)
	return nil
}
