package application

import (
	"context"
	"time"

	"k8s-platform-backend/internal/audit/domain"
)

// RunRetentionLoop 定期按保留策略清理过期审计记录，兜底防止日志无限增长。
func RunRetentionLoop(ctx context.Context, service *Service, policy domain.RetentionPolicy, interval time.Duration) {
	if service == nil {
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pruneCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_, _ = service.Prune(pruneCtx, policy)
			cancel()
		}
	}
}
