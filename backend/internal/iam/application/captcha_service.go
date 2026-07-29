package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"k8s-platform-backend/internal/iam/domain"
	"k8s-platform-backend/internal/iam/ports"
)

const captchaPrefix = "captcha:"
const captchaTTL = 5 * time.Minute
const captchaTolerance = 5
const captchaTrackWidth = 280

type CaptchaService struct{ cache ports.Cache }
type CaptchaPuzzle struct {
	Token      string `json:"token"`
	TargetX    int    `json:"target_x"`
	TrackWidth int    `json:"track_width"`
}
type captchaData struct {
	TargetX int  `json:"target_x"`
	Used    bool `json:"used"`
}

func NewCaptchaService(cache ports.Cache) *CaptchaService { return &CaptchaService{cache: cache} }
func (s *CaptchaService) CacheEnabled() bool              { return s != nil && s.cache != nil && s.cache.Enabled() }
func (s *CaptchaService) Generate(ctx context.Context) (*CaptchaPuzzle, error) {
	if !s.CacheEnabled() {
		return nil, nil
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	target := 40 + secureRandomInt(captchaTrackWidth-80)
	encoded, _ := json.Marshal(captchaData{TargetX: target})
	if err := s.cache.Set(ctx, captchaPrefix+token, encoded, captchaTTL); err != nil {
		return nil, err
	}
	return &CaptchaPuzzle{Token: token, TargetX: target, TrackWidth: captchaTrackWidth}, nil
}
func (s *CaptchaService) Verify(ctx context.Context, token string, userX int) error {
	if !s.CacheEnabled() {
		return nil
	}
	if token == "" {
		return domain.ErrCaptchaNotFound
	}
	key := captchaPrefix + token
	encoded, ok, err := s.cache.Get(ctx, key)
	if err != nil {
		return err
	}
	if !ok || len(encoded) == 0 {
		return domain.ErrCaptchaNotFound
	}
	var data captchaData
	if err := json.Unmarshal(encoded, &data); err != nil {
		return err
	}
	if data.Used {
		_ = s.cache.Del(ctx, key)
		return domain.ErrCaptchaUsed
	}
	difference := userX - data.TargetX
	if difference < 0 {
		difference = -difference
	}
	_ = s.cache.Del(ctx, key)
	if difference > captchaTolerance {
		return domain.ErrCaptchaInvalid
	}
	return nil
}
func secureRandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	value := make([]byte, 1)
	_, _ = rand.Read(value)
	return int(value[0]) % max
}
