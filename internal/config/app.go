package config

import (
	"time"

	vgokit "github.com/vera-byte/vgo-kit"
	"github.com/vera-byte/vgo-kit/cache"
	vgogrpc "github.com/vera-byte/vgo-kit/grpc"
	"go.uber.org/zap"
)

// Config 应用配置
type AppConfig struct {
	GRPC struct {
		Server vgogrpc.ServerConfig `mapstructure:"server"`
		Client struct {
			Connections map[string]vgogrpc.ClientConfig `mapstructure:"connections"`
		} `mapstructure:"client"`
	} `mapstructure:"grpc"`
	HTTP struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"http"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Log        LogConfig       `mapstructure:"log"`
	Sentry     SentryConfig    `mapstructure:"sentry"`
	RateLimit  RateLimitConfig `mapstructure:"ratelimit"`
	Middleware struct {
		Ignore         []string `mapstructure:"ignore"`
		MasterKey      string   `mapstructure:"master_key"`
		AllowedHeaders []string `mapstructure:"allowed_headers"`
	} `mapstructure:"middleware"`
	Cache *cache.CacheConfig `mapstructure:"cache"`
	STS   STSConfig          `mapstructure:"sts"`
}
type LogConfig struct {
	Level     string `mapstructure:"level"`     // 日志级别: debug/info/warn/error
	Format    string `mapstructure:"format"`    // 日志格式: json/console
	Directory string `mapstructure:"directory"` // 日志文件目录
	Filename  string `mapstructure:"filename"`  // 日志文件名
	ToStdout  bool   `mapstructure:"to_stdout"` // 是否输出到终端
}

type SentryConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	DSN         string `mapstructure:"dsn"`
	Environment string `mapstructure:"environment"`
}

type RateLimitConfig struct {
	Enabled   bool          `mapstructure:"enabled"`
	Type      string        `mapstructure:"type"`
	Limit     int           `mapstructure:"limit"`
	Window    time.Duration `mapstructure:"window"`
	Prefix    string        `mapstructure:"prefix"`
	RedisAddr string        `mapstructure:"redis_addr"`
	RedisDB   int           `mapstructure:"redis_db"`
	RedisPass string        `mapstructure:"redis_pass"`
}

// STSConfig STS临时凭证配置
type STSConfig struct {
	// 默认凭证有效期
	DefaultDuration time.Duration `mapstructure:"default_duration"`
	// 最大凭证有效期
	MaxDuration time.Duration `mapstructure:"max_duration"`
	// 最小凭证有效期
	MinDuration time.Duration `mapstructure:"min_duration"`
	// 清理过期凭证的间隔时间
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
	// 是否启用自动清理
	AutoCleanup bool `mapstructure:"auto_cleanup"`
	// 每个用户最大凭证数量
	MaxCredentialsPerUser int `mapstructure:"max_credentials_per_user"`
}

func LodIAMConfig() *AppConfig {
	var cfg *AppConfig
	if err := vgokit.Cfg.Unmarshal(&cfg); err != nil {
		vgokit.Log.Error("读取IAM服务配置失败", zap.Error(err))
		return nil
	}
	return cfg
}
