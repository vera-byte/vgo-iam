package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	vgokit "github.com/vera-byte/vgo-kit"
)

// RotationScheduler 访问密钥轮换调度器
type RotationScheduler struct {
	service    *AccessKeyService
	interval   time.Duration
	running    bool
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewRotationScheduler 创建新的轮换调度器
func NewRotationScheduler(service *AccessKeyService, interval time.Duration) *RotationScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &RotationScheduler{
		service:  service,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动调度器
func (rs *RotationScheduler) Start() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.running {
		return
	}

	rs.running = true
	rs.wg.Add(1)

	go rs.run()
	vgokit.Log.Info("访问密钥轮换调度器已启动")
}

// Stop 停止调度器
func (rs *RotationScheduler) Stop() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if !rs.running {
		return
	}

	rs.running = false
	rs.cancel()
	rs.wg.Wait()
	vgokit.Log.Info("访问密钥轮换调度器已停止")
}

// IsRunning 检查调度器是否运行中
func (rs *RotationScheduler) IsRunning() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.running
}

// run 调度器主循环
func (rs *RotationScheduler) run() {
	defer rs.wg.Done()

	ticker := time.NewTicker(rs.interval)
	defer ticker.Stop()

	for {
		select {
		case <-rs.ctx.Done():
			return
		case <-ticker.C:
			rs.processRotations()
		}
	}
}

// processRotations 处理轮换任务
func (rs *RotationScheduler) processRotations() {
	defer func() {
		if r := recover(); r != nil {
			vgokit.Log.Error(fmt.Sprintf("轮换调度器发生panic: %v", r))
		}
	}()

	vgokit.Log.Debug("开始处理访问密钥轮换任务")

	// 处理预定的轮换任务
	if err := rs.service.ProcessScheduledRotations(rs.ctx); err != nil {
		vgokit.Log.Error(fmt.Sprintf("处理预定轮换任务失败: %v", err))
	}

	// 检查并轮换过期密钥
	if err := rs.service.CheckAndRotateExpiredKeys(rs.ctx); err != nil {
		vgokit.Log.Error(fmt.Sprintf("检查并轮换过期密钥失败: %v", err))
	}

	vgokit.Log.Debug("访问密钥轮换任务处理完成")
}

// SetInterval 设置调度间隔
func (rs *RotationScheduler) SetInterval(interval time.Duration) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.interval = interval
	vgokit.Log.Info(fmt.Sprintf("轮换调度器间隔已更新: %v", interval))
}

// GetInterval 获取调度间隔
func (rs *RotationScheduler) GetInterval() time.Duration {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.interval
}