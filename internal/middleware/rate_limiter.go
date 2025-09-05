package middleware

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vera-byte/vgo-iam/internal/errors"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	// Allow 检查是否允许请求
	// 参数:
	//   - key: 限流键（通常是IP地址或用户ID）
	// 返回值:
	//   - bool: 是否允许请求
	//   - time.Duration: 重试等待时间
	Allow(key string) (bool, time.Duration)

	// Reset 重置指定键的限流状态
	// 参数:
	//   - key: 限流键
	Reset(key string)

	// GetStats 获取限流统计信息
	// 参数:
	//   - key: 限流键
	// 返回值:
	//   - *RateLimitStats: 统计信息
	GetStats(key string) *RateLimitStats
}

// RateLimitStats 限流统计信息
type RateLimitStats struct {
	Key           string        `json:"key"`            // 限流键
	RequestCount  int64         `json:"request_count"`  // 请求总数
	AllowedCount  int64         `json:"allowed_count"`  // 允许的请求数
	BlockedCount  int64         `json:"blocked_count"`  // 被阻止的请求数
	LastRequest   time.Time     `json:"last_request"`   // 最后请求时间
	ResetTime     time.Time     `json:"reset_time"`     // 重置时间
	RetryAfter    time.Duration `json:"retry_after"`    // 重试等待时间
}

// TokenBucketLimiter 令牌桶限流器
type TokenBucketLimiter struct {
	buckets   map[string]*tokenBucket
	mutex     sync.RWMutex
	capacity  int64         // 桶容量
	refillRate int64        // 令牌补充速率（每秒）
	window    time.Duration // 时间窗口
	logger    *log.Logger
}

// tokenBucket 令牌桶
type tokenBucket struct {
	tokens       int64     // 当前令牌数
	lastRefill   time.Time // 最后补充时间
	requestCount int64     // 请求总数
	allowedCount int64     // 允许的请求数
	blockedCount int64     // 被阻止的请求数
	lastRequest  time.Time // 最后请求时间
}

// NewTokenBucketLimiter 创建令牌桶限流器
// 参数:
//   - capacity: 桶容量
//   - refillRate: 令牌补充速率（每秒）
//   - window: 时间窗口
//   - logger: 日志记录器
// 返回值:
//   - *TokenBucketLimiter: 令牌桶限流器实例
func NewTokenBucketLimiter(capacity, refillRate int64, window time.Duration, logger *log.Logger) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets:    make(map[string]*tokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
		window:     window,
		logger:     logger,
	}
}

// Allow 检查是否允许请求
func (tbl *TokenBucketLimiter) Allow(key string) (bool, time.Duration) {
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	now := time.Now()
	bucket, exists := tbl.buckets[key]
	if !exists {
		bucket = &tokenBucket{
			tokens:      tbl.capacity,
			lastRefill:  now,
			lastRequest: now,
		}
		tbl.buckets[key] = bucket
	}

	// 补充令牌
	tbl.refillTokens(bucket, now)

	// 更新统计信息
	bucket.requestCount++
	bucket.lastRequest = now

	// 检查是否有可用令牌
	if bucket.tokens > 0 {
		bucket.tokens--
		bucket.allowedCount++
		tbl.logger.Printf("Rate limit allow: key=%s, tokens=%d", key, bucket.tokens)
		return true, 0
	}

	// 没有可用令牌，计算重试等待时间
	bucket.blockedCount++
	retryAfter := time.Second / time.Duration(tbl.refillRate)
	tbl.logger.Printf("Rate limit exceeded: key=%s, retry_after=%v", key, retryAfter)
	return false, retryAfter
}

// Reset 重置指定键的限流状态
func (tbl *TokenBucketLimiter) Reset(key string) {
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	delete(tbl.buckets, key)
	tbl.logger.Printf("Rate limit reset: key=%s", key)
}

// GetStats 获取限流统计信息
func (tbl *TokenBucketLimiter) GetStats(key string) *RateLimitStats {
	tbl.mutex.RLock()
	defer tbl.mutex.RUnlock()

	bucket, exists := tbl.buckets[key]
	if !exists {
		return &RateLimitStats{
			Key: key,
		}
	}

	retryAfter := time.Duration(0)
	if bucket.tokens == 0 {
		retryAfter = time.Second / time.Duration(tbl.refillRate)
	}

	return &RateLimitStats{
		Key:          key,
		RequestCount: bucket.requestCount,
		AllowedCount: bucket.allowedCount,
		BlockedCount: bucket.blockedCount,
		LastRequest:  bucket.lastRequest,
		ResetTime:    bucket.lastRefill.Add(tbl.window),
		RetryAfter:   retryAfter,
	}
}

// refillTokens 补充令牌
func (tbl *TokenBucketLimiter) refillTokens(bucket *tokenBucket, now time.Time) {
	elapsed := now.Sub(bucket.lastRefill)
	if elapsed <= 0 {
		return
	}

	// 计算应该补充的令牌数
	tokensToAdd := int64(elapsed.Seconds()) * tbl.refillRate
	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.tokens+tokensToAdd, tbl.capacity)
		bucket.lastRefill = now
	}
}

// min 返回两个int64中的较小值
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	limiter      RateLimiter
	errorHandler *ErrorHandler
	logger       *log.Logger
	keyFunc      func(*gin.Context) string // 生成限流键的函数
}

// NewRateLimitMiddleware 创建限流中间件
// 参数:
//   - limiter: 限流器
//   - errorHandler: 错误处理器
//   - logger: 日志记录器
//   - keyFunc: 生成限流键的函数
// 返回值:
//   - *RateLimitMiddleware: 限流中间件实例
func NewRateLimitMiddleware(
	limiter RateLimiter,
	errorHandler *ErrorHandler,
	logger *log.Logger,
	keyFunc func(*gin.Context) string,
) *RateLimitMiddleware {
	if keyFunc == nil {
		keyFunc = DefaultKeyFunc
	}

	return &RateLimitMiddleware{
		limiter:      limiter,
		errorHandler: errorHandler,
		logger:       logger,
		keyFunc:      keyFunc,
	}
}

// Handler 限流中间件处理函数
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (rlm *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成限流键
		key := rlm.keyFunc(c)

		// 检查是否允许请求
		allowed, retryAfter := rlm.limiter.Allow(key)
		if !allowed {
			// 设置响应头
			c.Header("X-RateLimit-Limit", "100") // 这里应该从配置中获取
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(retryAfter).Unix(), 10))
			c.Header("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))

			// 返回限流错误
			errorMsg := fmt.Sprintf("rate limit exceeded, retry after %v", retryAfter)
			rlm.errorHandler.HandleError(c, errors.NewBusinessError(errors.CodeTooManyRequests, errorMsg))
			c.Abort()
			return
		}

		// 获取统计信息并设置响应头
		stats := rlm.limiter.GetStats(key)
		if stats != nil {
			c.Header("X-RateLimit-Limit", "100") // 这里应该从配置中获取
			remaining := 100 - (stats.RequestCount % 100) // 简化计算
			c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(stats.ResetTime.Unix(), 10))
		}

		rlm.logger.Printf("Rate limit check passed: key=%s", key)
		c.Next()
	}
}

// DefaultKeyFunc 默认的键生成函数（使用客户端IP）
// 参数:
//   - c: Gin上下文
// 返回值:
//   - string: 限流键
func DefaultKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// UserKeyFunc 基于用户ID的键生成函数
// 参数:
//   - c: Gin上下文
// 返回值:
//   - string: 限流键
func UserKeyFunc(c *gin.Context) string {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		return c.ClientIP() // 回退到IP
	}
	return fmt.Sprintf("user:%d", userID)
}

// APIKeyFunc 基于API路径的键生成函数
// 参数:
//   - c: Gin上下文
// 返回值:
//   - string: 限流键
func APIKeyFunc(c *gin.Context) string {
	return fmt.Sprintf("%s:%s", c.ClientIP(), c.Request.URL.Path)
}

// CompositeKeyFunc 组合键生成函数（IP + 用户ID + API路径）
// 参数:
//   - c: Gin上下文
// 返回值:
//   - string: 限流键
func CompositeKeyFunc(c *gin.Context) string {
	userID, exists := GetCurrentUserID(c)
	if exists {
		return fmt.Sprintf("%s:user:%d:%s", c.ClientIP(), userID, c.Request.URL.Path)
	}
	return fmt.Sprintf("%s:anonymous:%s", c.ClientIP(), c.Request.URL.Path)
}

// CleanupExpiredBuckets 清理过期的令牌桶（应该定期调用）
// 参数:
//   - maxAge: 最大存活时间
func (tbl *TokenBucketLimiter) CleanupExpiredBuckets(maxAge time.Duration) {
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, bucket := range tbl.buckets {
		if now.Sub(bucket.lastRequest) > maxAge {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(tbl.buckets, key)
	}

	if len(expiredKeys) > 0 {
		tbl.logger.Printf("Cleaned up %d expired rate limit buckets", len(expiredKeys))
	}
}

// StartCleanupWorker 启动清理工作协程
// 参数:
//   - interval: 清理间隔
//   - maxAge: 最大存活时间
func (tbl *TokenBucketLimiter) StartCleanupWorker(interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			tbl.CleanupExpiredBuckets(maxAge)
		}
	}()

	tbl.logger.Printf("Rate limiter cleanup worker started: interval=%v, maxAge=%v", interval, maxAge)
}