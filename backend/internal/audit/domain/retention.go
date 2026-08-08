package domain

import (
	"errors"
	"time"
)

var ErrInvalidRetention = errors.New("invalid audit retention policy")

// RetentionPolicy 定义审计日志保留策略：超过保留天数的记录可被清理。
type RetentionPolicy struct {
	// Days 为保留天数，必须大于 0。
	Days int
}

// CutoffBefore 返回清理截止时间：早于该时间的记录视为过期。
func (p RetentionPolicy) CutoffBefore(now time.Time) (time.Time, error) {
	if p.Days <= 0 {
		return time.Time{}, ErrInvalidRetention
	}
	return now.AddDate(0, 0, -p.Days), nil
}
