package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ── 登录限流 Service ──
// 基于 CacheStore (Redis) 实现登录失败计数与账号锁定：
// - 连续失败达 MaxFailures 次后锁定 LockDuration 时间
// - 锁定期内拒绝登录，返回剩余锁定时间
// - 登录成功时自动清零计数

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

func NewLoginAttemptService(cache CacheStore) *LoginAttemptService {
	return &LoginAttemptService{cache: cache}
}

// loginFailData 存储失败次数与锁定截止时间
type loginFailData struct {
	Count    int   `json:"count"`
	LockedUntil int64 `json:"locked_until"` // unix timestamp, 0 表示未锁定
}

// IsLocked 检查账号是否被锁定，返回是否锁定及剩余锁定秒数
func (las *LoginAttemptService) IsLocked(ctx context.Context, username string) (locked bool, remainSec int) {
	if las == nil || !las.cache.Enabled() {
		return false, 0
	}
	key := loginFailKeyPrefix + username
	b, ok, err := las.cache.Get(ctx, key)
	if err != nil || !ok || len(b) == 0 {
		return false, 0
	}
	var data loginFailData
	if json.Unmarshal(b, &data) != nil {
		return false, 0
	}
	if data.LockedUntil == 0 {
		return false, 0
	}
	now := time.Now().Unix()
	if now >= data.LockedUntil {
		// 锁定已过期，清理
		_ = las.cache.Del(ctx, key)
		return false, 0
	}
	return true, int(data.LockedUntil - now)
}

// RecordFailure 记录一次登录失败，达阈值则锁定
func (las *LoginAttemptService) RecordFailure(ctx context.Context, username string) (locked bool, remainSec int) {
	if las == nil || !las.cache.Enabled() {
		return false, 0
	}
	key := loginFailKeyPrefix + username

	var data loginFailData
	if b, ok, err := las.cache.Get(ctx, key); err == nil && ok && len(b) > 0 {
		_ = json.Unmarshal(b, &data)
	}

	// 如果已锁定，直接返回剩余时间
	if data.LockedUntil > 0 {
		now := time.Now().Unix()
		if now >= data.LockedUntil {
			// 锁定过期，重新计数
			data = loginFailData{Count: 1, LockedUntil: 0}
		} else {
			return true, int(data.LockedUntil - now)
		}
	} else {
		data.Count++
	}

	// 达到阈值则锁定
	if data.Count >= MaxLoginFailures {
		data.LockedUntil = time.Now().Add(LoginLockDuration).Unix()
	}

	b, _ := json.Marshal(data)
	ttl := LoginLockDuration
	if data.LockedUntil == 0 {
		ttl = 30 * time.Minute // 未锁定时计数记录 30min 后过期
	}
	_ = las.cache.Set(ctx, key, b, ttl)

	if data.LockedUntil > 0 {
		now := time.Now().Unix()
		return true, int(data.LockedUntil - now)
	}
	return false, 0
}

// ResetFailures 登录成功时清零失败计数
func (las *LoginAttemptService) ResetFailures(ctx context.Context, username string) {
	if las == nil || !las.cache.Enabled() {
		return
	}
	_ = las.cache.Del(ctx, loginFailKeyPrefix + username)
}

// FormatLockMessage 格式化锁定提示文案
func FormatLockMessage(remainSec int) string {
	min := remainSec / 60
	sec := remainSec % 60
	if min > 0 {
		return fmt.Sprintf("账号已被锁定，请 %d 分 %02d 秒后重试", min, sec)
	}
	return fmt.Sprintf("账号已被锁定，请 %d 秒后重试", sec)
}
