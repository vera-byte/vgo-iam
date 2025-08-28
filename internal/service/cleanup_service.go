package service

import (
	"context"
	"time"

	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

// CleanupService 清理服务，负责定期清理过期的临时凭证
type CleanupService struct {
	stsService *STSService
	ticker     *time.Ticker
	done       chan bool
	interval   time.Duration
}

// NewCleanupService 创建清理服务实例
// stsService: STS服务实例
// interval: 清理间隔时间
// 返回: 清理服务实例
func NewCleanupService(stsService *STSService, interval time.Duration) *CleanupService {
	if interval <= 0 {
		interval = 1 * time.Hour // 默认每小时清理一次
	}

	return &CleanupService{
		stsService: stsService,
		interval:   interval,
		done:       make(chan bool),
	}
}

// Start 启动清理服务
// ctx: 上下文
func (c *CleanupService) Start(ctx context.Context) {
	c.ticker = time.NewTicker(c.interval)

	vgokit.Log.Info("启动临时凭证清理服务",
		zap.Duration("interval", c.interval))

	go func() {
		for {
			select {
			case <-c.ticker.C:
				c.performCleanup(ctx)
			case <-c.done:
				vgokit.Log.Info("停止临时凭证清理服务")
				return
			case <-ctx.Done():
				vgokit.Log.Info("上下文取消，停止临时凭证清理服务")
				return
			}
		}
	}()
}

// Stop 停止清理服务
func (c *CleanupService) Stop() {
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.done)
}

// performCleanup 执行清理操作
// ctx: 上下文
func (c *CleanupService) performCleanup(ctx context.Context) {
	start := time.Now()
	vgokit.Log.Debug("开始清理过期的临时凭证")

	// 清理过期的临时凭证
	cleanedCount, err := c.stsService.CleanupExpiredCredentials()
	if err != nil {
		vgokit.Log.Error("清理过期临时凭证失败",
			zap.Error(err))
		return
	}

	duration := time.Since(start)
	if cleanedCount > 0 {
		vgokit.Log.Info("清理过期临时凭证完成",
			zap.Int64("cleaned_count", cleanedCount),
			zap.Duration("duration", duration))
	} else {
		vgokit.Log.Debug("没有过期的临时凭证需要清理",
			zap.Duration("duration", duration))
	}
}

// ForceCleanup 强制执行一次清理
// ctx: 上下文
// 返回: 清理的凭证数量和错误信息
func (c *CleanupService) ForceCleanup(ctx context.Context) (int64, error) {
	vgokit.Log.Info("强制执行临时凭证清理")
	c.performCleanup(ctx)
	return c.stsService.CleanupExpiredCredentials()
}

// GetStats 获取清理服务统计信息
// 返回: 统计信息
func (c *CleanupService) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"interval_seconds": c.interval.Seconds(),
		"is_running":       c.ticker != nil,
		"service_type":     "temporary_credential_cleanup",
	}
}