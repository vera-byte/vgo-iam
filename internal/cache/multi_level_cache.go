package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
)

// MultiLevelCache 多级缓存实现
// 包含内存缓存（L1）和Redis缓存（L2）
type MultiLevelCache struct {
	memoryCache *cache.Cache // L1缓存：内存缓存
	redisClient *redis.Client // L2缓存：Redis缓存
	mu          sync.RWMutex
	defaultTTL  time.Duration
}

// CacheConfig 缓存配置
type CacheConfig struct {
	RedisAddr     string        // Redis地址
	RedisPassword string        // Redis密码
	RedisDB       int           // Redis数据库
	MemoryTTL     time.Duration // 内存缓存TTL
	MemoryCleanup time.Duration // 内存缓存清理间隔
	DefaultTTL    time.Duration // 默认TTL
}

// NewMultiLevelCache 创建多级缓存实例
// 参数:
//   - config: 缓存配置
// 返回值:
//   - *MultiLevelCache: 多级缓存实例
//   - error: 创建过程中的错误
func NewMultiLevelCache(config *CacheConfig) (*MultiLevelCache, error) {
	// 创建内存缓存
	memoryCache := cache.New(config.MemoryTTL, config.MemoryCleanup)

	// 创建Redis客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &MultiLevelCache{
		memoryCache: memoryCache,
		redisClient: redisClient,
		defaultTTL:  config.DefaultTTL,
	}, nil
}

// Get 获取缓存值
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//   - dest: 目标对象指针
// 返回值:
//   - bool: 是否找到缓存
//   - error: 获取过程中的错误
func (c *MultiLevelCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	// 先从L1缓存（内存）获取
	if value, found := c.memoryCache.Get(key); found {
		if err := c.unmarshalValue(value, dest); err != nil {
			return false, fmt.Errorf("failed to unmarshal L1 cache value: %w", err)
		}
		return true, nil
	}

	// L1缓存未命中，从L2缓存（Redis）获取
	value, err := c.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // 缓存未命中
		}
		return false, fmt.Errorf("failed to get from Redis: %w", err)
	}

	// 反序列化Redis中的值
	if err := json.Unmarshal([]byte(value), dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal Redis value: %w", err)
	}

	// 将值回填到L1缓存
	c.memoryCache.Set(key, value, cache.DefaultExpiration)

	return true, nil
}

// Set 设置缓存值
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//   - value: 缓存值
//   - ttl: 过期时间（0表示使用默认TTL）
// 返回值:
//   - error: 设置过程中的错误
func (c *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	// 序列化值
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// 设置到L1缓存（内存）
	c.memoryCache.Set(key, string(data), ttl)

	// 设置到L2缓存（Redis）
	if err := c.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set to Redis: %w", err)
	}

	return nil
}

// Delete 删除缓存
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
// 返回值:
//   - error: 删除过程中的错误
func (c *MultiLevelCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 从L1缓存删除
	c.memoryCache.Delete(key)

	// 从L2缓存删除
	if err := c.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	return nil
}

// Clear 清空所有缓存
// 参数:
//   - ctx: 上下文
// 返回值:
//   - error: 清空过程中的错误
func (c *MultiLevelCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 清空L1缓存
	c.memoryCache.Flush()

	// 清空L2缓存（注意：这会清空整个Redis数据库）
	if err := c.redisClient.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush Redis: %w", err)
	}

	return nil
}

// GetStats 获取缓存统计信息
// 返回值:
//   - map[string]interface{}: 统计信息
func (c *MultiLevelCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"memory_items": c.memoryCache.ItemCount(),
		"redis_connected": c.redisClient.Ping(context.Background()).Err() == nil,
	}
}

// Close 关闭缓存连接
// 返回值:
//   - error: 关闭过程中的错误
func (c *MultiLevelCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 清空内存缓存
	c.memoryCache.Flush()

	// 关闭Redis连接
	if err := c.redisClient.Close(); err != nil {
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	return nil
}

// unmarshalValue 反序列化缓存值
func (c *MultiLevelCache) unmarshalValue(value interface{}, dest interface{}) error {
	switch v := value.(type) {
	case string:
		return json.Unmarshal([]byte(v), dest)
	case []byte:
		return json.Unmarshal(v, dest)
	default:
		return fmt.Errorf("unsupported cache value type: %T", value)
	}
}

// GetMemoryCache 获取内存缓存实例（用于测试）
func (c *MultiLevelCache) GetMemoryCache() *cache.Cache {
	return c.memoryCache
}

// GetRedisClient 获取Redis客户端实例（用于测试）
func (c *MultiLevelCache) GetRedisClient() *redis.Client {
	return c.redisClient
}