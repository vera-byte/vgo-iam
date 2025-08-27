package config

import (
	"time"

	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

// Config 应用配置
type AppConfig struct {
	GRPC struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"grpc"`
	Database struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"database"`
	Log        LogConfig        `mapstructure:"log"`
	Sentry     SentryConfig     `mapstructure:"sentry"`
	RateLimit  RateLimitConfig  `mapstructure:"ratelimit"`
	Middleware struct {
		Ignore    []string `mapstructure:"ignore"`
		MasterKey string   `mapstructure:"master_key"`
	} `mapstructure:"middleware"`
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

func LodIAMConfig() *AppConfig {
	var cfg *AppConfig
	if err := vgokit.Cfg.Unmarshal(&cfg); err != nil {
		vgokit.Log.Error("读取IAM服务配置失败", zap.Error(err))
		return nil
	}
	return cfg
}
