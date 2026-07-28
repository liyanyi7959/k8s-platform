package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrAccountLocked = errors.New("account is locked due to too many failed attempts")
)

const (
	loginFailKeyPrefix = "login_fail:"
	MaxLoginFailures   = 5
	LoginLockDuration  = 15 * time.Minute
)

type LoginAttemptService struct {
	cache CacheStore
}

type LoginAttemptStatus struct {
	FailedAttempts       int  `json:"failed_attempts"`
	RemainingAttempts    int  `json:"remaining_attempts"`
	MaxAttempts          int  `json:"max_attempts"`
	Locked               bool `json:"locked"`
	LockRemainingSeconds int  `json:"lock_remaining_seconds,omitempty"`
	LockDurationSeconds  int  `json:"lock_duration_seconds,omitempty"`
}

type loginFailData struct {
	Count       int   `json:"count"`
	LockedUntil int64 `json:"locked_until"`
}

func NewLoginAttemptService(cache CacheStore) *LoginAttemptService {
	return &LoginAttemptService{cache: cache}
}

func emptyLoginAttemptStatus() LoginAttemptStatus {
	return LoginAttemptStatus{}
}

func defaultLoginAttemptStatus() LoginAttemptStatus {
	return LoginAttemptStatus{
		RemainingAttempts:   MaxLoginFailures,
		MaxAttempts:         MaxLoginFailures,
		LockDurationSeconds: int(LoginLockDuration.Seconds()),
	}
}

func (las *LoginAttemptService) loadData(ctx context.Context, key string) (loginFailData, bool) {
	if las == nil || !las.cache.Enabled() {
		return loginFailData{}, false
	}
	b, ok, err := las.cache.Get(ctx, key)
	if err != nil || !ok || len(b) == 0 {
		return loginFailData{}, false
	}
	var data loginFailData
	if json.Unmarshal(b, &data) != nil {
		return loginFailData{}, false
	}
	return data, true
}

func (las *LoginAttemptService) buildStatusFromData(ctx context.Context, key string, data loginFailData) LoginAttemptStatus {
	status := defaultLoginAttemptStatus()
	if data.Count < 0 {
		data.Count = 0
	}
	status.FailedAttempts = data.Count

	if data.LockedUntil > 0 {
		now := time.Now().Unix()
		if now >= data.LockedUntil {
			if las != nil && las.cache.Enabled() {
				_ = las.cache.Del(ctx, key)
			}
			return status
		}
		status.Locked = true
		status.RemainingAttempts = 0
		status.LockRemainingSeconds = int(data.LockedUntil - now)
		return status
	}

	remaining := MaxLoginFailures - data.Count
	if remaining < 0 {
		remaining = 0
	}
	status.RemainingAttempts = remaining
	return status
}

func (las *LoginAttemptService) GetStatus(ctx context.Context, username string) LoginAttemptStatus {
	if las == nil || !las.cache.Enabled() {
		return emptyLoginAttemptStatus()
	}
	key := loginFailKeyPrefix + username
	data, ok := las.loadData(ctx, key)
	if !ok {
		return defaultLoginAttemptStatus()
	}
	return las.buildStatusFromData(ctx, key, data)
}

func (las *LoginAttemptService) IsLocked(ctx context.Context, username string) (locked bool, remainSec int) {
	status := las.GetStatus(ctx, username)
	return status.Locked, status.LockRemainingSeconds
}

func (las *LoginAttemptService) RecordFailure(ctx context.Context, username string) (locked bool, remainSec int) {
	status := las.RecordFailureDetail(ctx, username)
	return status.Locked, status.LockRemainingSeconds
}

func (las *LoginAttemptService) RecordFailureDetail(ctx context.Context, username string) LoginAttemptStatus {
	if las == nil || !las.cache.Enabled() {
		return emptyLoginAttemptStatus()
	}

	key := loginFailKeyPrefix + username
	data, _ := las.loadData(ctx, key)
	now := time.Now()

	if data.LockedUntil > 0 {
		if now.Unix() < data.LockedUntil {
			return las.buildStatusFromData(ctx, key, data)
		}
		data = loginFailData{}
	}

	data.Count++
	if data.Count >= MaxLoginFailures {
		data.LockedUntil = now.Add(LoginLockDuration).Unix()
	}

	b, _ := json.Marshal(data)
	ttl := LoginLockDuration
	if data.LockedUntil == 0 {
		ttl = 30 * time.Minute
	}
	_ = las.cache.Set(ctx, key, b, ttl)

	return las.buildStatusFromData(ctx, key, data)
}

func (las *LoginAttemptService) ResetFailures(ctx context.Context, username string) {
	if las == nil || !las.cache.Enabled() {
		return
	}
	_ = las.cache.Del(ctx, loginFailKeyPrefix+username)
}

func FormatLockMessage(remainSec int) string {
	min := remainSec / 60
	sec := remainSec % 60
	if min > 0 {
		return fmt.Sprintf("账号已被锁定，请 %d 分 %02d 秒后重试", min, sec)
	}
	return fmt.Sprintf("账号已被锁定，请 %d 秒后重试", sec)
}
