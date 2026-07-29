package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s-platform-backend/internal/iam/ports"
)

const MaxLoginFailures = 5
const LoginLockDuration = 15 * time.Minute
const loginFailurePrefix = "login_fail:"

type LoginAttemptService struct {
	cache ports.Cache
	now   func() time.Time
}
type LoginAttemptStatus struct {
	FailedAttempts       int  `json:"failed_attempts"`
	RemainingAttempts    int  `json:"remaining_attempts"`
	MaxAttempts          int  `json:"max_attempts"`
	Locked               bool `json:"locked"`
	LockRemainingSeconds int  `json:"lock_remaining_seconds,omitempty"`
	LockDurationSeconds  int  `json:"lock_duration_seconds,omitempty"`
}
type loginFailureData struct {
	Count       int   `json:"count"`
	LockedUntil int64 `json:"locked_until"`
}

func NewLoginAttemptService(cache ports.Cache) *LoginAttemptService {
	return &LoginAttemptService{cache: cache, now: time.Now}
}
func (s *LoginAttemptService) GetStatus(ctx context.Context, username string) LoginAttemptStatus {
	if !s.enabled() {
		return LoginAttemptStatus{}
	}
	key := loginFailurePrefix + username
	data, ok := s.load(ctx, key)
	if !ok {
		return defaultAttemptStatus()
	}
	return s.status(ctx, key, data)
}
func (s *LoginAttemptService) IsLocked(ctx context.Context, username string) (bool, int) {
	status := s.GetStatus(ctx, username)
	return status.Locked, status.LockRemainingSeconds
}
func (s *LoginAttemptService) RecordFailure(ctx context.Context, username string) (bool, int) {
	status := s.RecordFailureDetail(ctx, username)
	return status.Locked, status.LockRemainingSeconds
}
func (s *LoginAttemptService) RecordFailureDetail(ctx context.Context, username string) LoginAttemptStatus {
	if !s.enabled() {
		return LoginAttemptStatus{}
	}
	key := loginFailurePrefix + username
	data, _ := s.load(ctx, key)
	now := s.now()
	if data.LockedUntil > 0 {
		if now.Unix() < data.LockedUntil {
			return s.status(ctx, key, data)
		}
		data = loginFailureData{}
	}
	data.Count++
	if data.Count >= MaxLoginFailures {
		data.LockedUntil = now.Add(LoginLockDuration).Unix()
	}
	encoded, _ := json.Marshal(data)
	ttl := 30 * time.Minute
	if data.LockedUntil > 0 {
		ttl = LoginLockDuration
	}
	_ = s.cache.Set(ctx, key, encoded, ttl)
	return s.status(ctx, key, data)
}
func (s *LoginAttemptService) ResetFailures(ctx context.Context, username string) {
	if s.enabled() {
		_ = s.cache.Del(ctx, loginFailurePrefix+username)
	}
}
func (s *LoginAttemptService) enabled() bool { return s != nil && s.cache != nil && s.cache.Enabled() }
func (s *LoginAttemptService) load(ctx context.Context, key string) (loginFailureData, bool) {
	data, ok, err := s.cache.Get(ctx, key)
	if err != nil || !ok || len(data) == 0 {
		return loginFailureData{}, false
	}
	var result loginFailureData
	if json.Unmarshal(data, &result) != nil {
		return loginFailureData{}, false
	}
	return result, true
}
func (s *LoginAttemptService) status(ctx context.Context, key string, data loginFailureData) LoginAttemptStatus {
	result := defaultAttemptStatus()
	if data.Count < 0 {
		data.Count = 0
	}
	result.FailedAttempts = data.Count
	if data.LockedUntil > 0 {
		now := s.now().Unix()
		if now >= data.LockedUntil {
			_ = s.cache.Del(ctx, key)
			return result
		}
		result.Locked, result.RemainingAttempts, result.LockRemainingSeconds = true, 0, int(data.LockedUntil-now)
		return result
	}
	result.RemainingAttempts = MaxLoginFailures - data.Count
	if result.RemainingAttempts < 0 {
		result.RemainingAttempts = 0
	}
	return result
}
func defaultAttemptStatus() LoginAttemptStatus {
	return LoginAttemptStatus{RemainingAttempts: MaxLoginFailures, MaxAttempts: MaxLoginFailures, LockDurationSeconds: int(LoginLockDuration.Seconds())}
}
func FormatLockMessage(seconds int) string {
	minutes, remainder := seconds/60, seconds%60
	if minutes > 0 {
		return fmt.Sprintf("账号已被锁定，请 %d 分 %02d 秒后重试", minutes, remainder)
	}
	return fmt.Sprintf("账号已被锁定，请 %d 秒后重试", remainder)
}
