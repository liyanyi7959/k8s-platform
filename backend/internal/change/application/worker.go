package application

import (
	"context"
	"time"
)

// RunExecutionRecoveryLoop 进程内轮询回收超时执行，兜底避免僵尸执行永久
// 占用状态。interval 为轮询周期，timeout 为执行超时阈值；单次回收带独立
// 超时，避免阻塞应用退出。
func RunExecutionRecoveryLoop(ctx context.Context, service *Service, interval, timeout time.Duration) {
	if service == nil {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			recoveryCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = service.RecoverTimedOutExecutions(recoveryCtx, timeout)
			cancel()
		}
	}
}
