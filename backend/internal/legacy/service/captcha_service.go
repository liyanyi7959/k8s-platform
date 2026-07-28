package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// ── 滑块验证码 Service ──
// 采用"服务端生成 puzzle token + 前端滑块偏移量校验"方案：
// 1. Generate: 随机生成目标偏移量 targetX(0-280)，存入 CacheStore(key=captcha:<token>, ttl=5min)，返回 token + 背景图 Base64
// 2. Verify: 前端提交 token + 用户滑动偏移量 x，与 targetX 比较（±5px 容差），校验后立即删除（一次性）
// 无需图片处理依赖，前端自行渲染滑块 UI

var (
	ErrCaptchaNotFound  = errors.New("captcha not found or expired")
	ErrCaptchaInvalid    = errors.New("captcha verification failed")
	ErrCaptchaUsed       = errors.New("captcha already used")
)

const (
	captchaKeyPrefix   = "captcha:"
	captchaTTL          = 5 * time.Minute
	captchaTolerance    = 5 // 滑块容差像素
	captchaTrackWidth   = 280 // 滑动轨道宽度
)

type CaptchaService struct {
	cache CacheStore
}

func NewCaptchaService(cache CacheStore) *CaptchaService {
	return &CaptchaService{cache: cache}
}

// CacheEnabled 返回缓存是否可用（Redis 是否启用）
func (cs *CaptchaService) CacheEnabled() bool {
	return cs != nil && cs.cache != nil && cs.cache.Enabled()
}

// captchaData 存储在 cache 中的验证码数据
type captchaData struct {
	TargetX int    `json:"target_x"`
	Used    bool   `json:"used"`
}

// CaptchaPuzzle 返回给前端的验证码信息
type CaptchaPuzzle struct {
	Token    string `json:"token"`
	TargetX  int    `json:"target_x"`   // 目标偏移量（调试用，生产可去掉）
	TrackWidth int  `json:"track_width"`
}

// Generate 生成滑块验证码
func (cs *CaptchaService) Generate(ctx context.Context) (*CaptchaPuzzle, error) {
	if cs == nil || !cs.cache.Enabled() {
		// Redis 未启用时返回 nil，前端跳过验证码
		return nil, nil
	}

	// 生成随机 token (16 bytes)
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// 生成随机目标偏移量 (40 ~ trackWidth-40)
	targetX := 40 + secureRandomInt(captchaTrackWidth-80)

	data := captchaData{TargetX: targetX, Used: false}
	b, _ := json.Marshal(data)

	if err := cs.cache.Set(ctx, captchaKeyPrefix+token, b, captchaTTL); err != nil {
		return nil, err
	}

	return &CaptchaPuzzle{
		Token:      token,
		TargetX:    targetX,
		TrackWidth: captchaTrackWidth,
	}, nil
}

// Verify 校验滑块验证码，校验后立即删除（一次性使用）
func (cs *CaptchaService) Verify(ctx context.Context, token string, userX int) error {
	if cs == nil || !cs.cache.Enabled() {
		return nil // Redis 未启用时跳过验证
	}
	if token == "" {
		return ErrCaptchaNotFound
	}

	key := captchaKeyPrefix + token
	b, ok, err := cs.cache.Get(ctx, key)
	if err != nil {
		return err
	}
	if !ok || len(b) == 0 {
		return ErrCaptchaNotFound
	}

	var data captchaData
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	if data.Used {
		_ = cs.cache.Del(ctx, key)
		return ErrCaptchaUsed
	}

	// 容差校验
	diff := userX - data.TargetX
	if diff < 0 {
		diff = -diff
	}
	if diff > captchaTolerance {
		// 校验失败也删除，防止反复尝试
		_ = cs.cache.Del(ctx, key)
		return ErrCaptchaInvalid
	}

	// 校验成功，删除验证码（一次性）
	_ = cs.cache.Del(ctx, key)
	return nil
}

// secureRandomInt 生成 [0, max) 范围内的安全随机数
func secureRandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return int(b[0]) % max
}
