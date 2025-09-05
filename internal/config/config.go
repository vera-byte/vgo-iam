package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`   // 服务器配置
	Database DatabaseConfig `yaml:"database"` // 数据库配置
	Redis    RedisConfig    `yaml:"redis"`    // Redis配置
	Cache    CacheConfig    `yaml:"cache"`    // 缓存配置
	STS      STSConfig      `yaml:"sts"`      // STS配置
	Security SecurityConfig `yaml:"security"` // 安全配置
	Logging  LoggingConfig  `yaml:"logging"`  // 日志配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string        `yaml:"host"`          // 服务器主机
	Port         int           `yaml:"port"`          // 服务器端口
	Environment  string        `yaml:"environment"`   // 运行环境 (development, test, production)
	StaticDir    string        `yaml:"static_dir"`    // 静态文件目录
	UploadDir    string        `yaml:"upload_dir"`    // 上传文件目录
	ReadTimeout  time.Duration `yaml:"read_timeout"`  // 读取超时
	WriteTimeout time.Duration `yaml:"write_timeout"` // 写入超时
	IdleTimeout  time.Duration `yaml:"idle_timeout"`  // 空闲超时
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DSN             string        `yaml:"dsn"`               // 数据源名称
	MaxOpenConns    int           `yaml:"max_open_conns"`    // 最大打开连接数
	MaxIdleConns    int           `yaml:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"` // 连接最大生命周期
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"` // 连接最大空闲时间
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`     // Redis地址
	Password string `yaml:"password"` // Redis密码
	DB       int    `yaml:"db"`       // Redis数据库
}

// CacheConfig 缓存配置
type CacheConfig struct {
	MemoryTTL     time.Duration `yaml:"memory_ttl"`     // 内存缓存TTL
	MemoryCleanup time.Duration `yaml:"memory_cleanup"` // 内存缓存清理间隔
	DefaultTTL    time.Duration `yaml:"default_ttl"`    // 默认TTL
}

// 注意：STSConfig已在app.go中定义，这里不重复定义

// SecurityConfig 安全配置
type SecurityConfig struct {
	MasterKey    string `yaml:"master_key"`    // 主密钥
	JWTSecret    string `yaml:"jwt_secret"`    // JWT密钥
	EncryptionKey string `yaml:"encryption_key"` // 加密密钥
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level"`  // 日志级别
	Format string `yaml:"format"` // 日志格式
	Output string `yaml:"output"` // 日志输出
}

// LoadConfig 加载配置文件
// 参数:
//   - configPath: 配置文件路径
// 返回值:
//   - *Config: 配置实例
//   - error: 加载过程中的错误
func LoadConfig(configPath string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析YAML配置
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	config.setDefaults()

	// 验证配置
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// LoadConfigFromEnv 从环境变量加载配置
// 返回值:
//   - *Config: 配置实例
//   - error: 加载过程中的错误
func LoadConfigFromEnv() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Host:         getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvIntOrDefault("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDurationOrDefault("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDurationOrDefault("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDurationOrDefault("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			DSN:             getEnvOrDefault("DATABASE_DSN", ""),
			MaxOpenConns:    getEnvIntOrDefault("DATABASE_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvIntOrDefault("DATABASE_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDurationOrDefault("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvDurationOrDefault("DATABASE_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Addr:     getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       getEnvIntOrDefault("REDIS_DB", 0),
		},
		Cache: CacheConfig{
			MemoryTTL:     getEnvDurationOrDefault("CACHE_MEMORY_TTL", 5*time.Minute),
			MemoryCleanup: getEnvDurationOrDefault("CACHE_MEMORY_CLEANUP", 10*time.Minute),
			DefaultTTL:    getEnvDurationOrDefault("CACHE_DEFAULT_TTL", 1*time.Hour),
		},
		STS: STSConfig{
			DefaultDuration: getEnvDurationOrDefault("STS_DEFAULT_DURATION", 1*time.Hour),
			MaxDuration:     getEnvDurationOrDefault("STS_MAX_DURATION", 12*time.Hour),
			MinDuration:     getEnvDurationOrDefault("STS_MIN_DURATION", 15*time.Minute),
		},
		Security: SecurityConfig{
			MasterKey:     getEnvOrDefault("SECURITY_MASTER_KEY", ""),
			JWTSecret:     getEnvOrDefault("SECURITY_JWT_SECRET", ""),
			EncryptionKey: getEnvOrDefault("SECURITY_ENCRYPTION_KEY", ""),
		},
		Logging: LoggingConfig{
			Level:  getEnvOrDefault("LOG_LEVEL", "info"),
			Format: getEnvOrDefault("LOG_FORMAT", "json"),
			Output: getEnvOrDefault("LOG_OUTPUT", "stdout"),
		},
	}

	// 设置默认值
	config.setDefaults()

	// 验证配置
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// setDefaults 设置默认值
func (c *Config) setDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 60 * time.Second
	}

	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}
	if c.Database.ConnMaxLifetime == 0 {
		c.Database.ConnMaxLifetime = 5 * time.Minute
	}
	if c.Database.ConnMaxIdleTime == 0 {
		c.Database.ConnMaxIdleTime = 5 * time.Minute
	}

	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}

	if c.Cache.MemoryTTL == 0 {
		c.Cache.MemoryTTL = 5 * time.Minute
	}
	if c.Cache.MemoryCleanup == 0 {
		c.Cache.MemoryCleanup = 10 * time.Minute
	}
	if c.Cache.DefaultTTL == 0 {
		c.Cache.DefaultTTL = 1 * time.Hour
	}

	if c.STS.DefaultDuration == 0 {
		c.STS.DefaultDuration = 1 * time.Hour
	}
	if c.STS.MaxDuration == 0 {
		c.STS.MaxDuration = 12 * time.Hour
	}
	if c.STS.MinDuration == 0 {
		c.STS.MinDuration = 15 * time.Minute
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}
}

// validate 验证配置
func (c *Config) validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	if c.Security.MasterKey == "" {
		return fmt.Errorf("security master key is required")
	}

	if c.Security.JWTSecret == "" {
		return fmt.Errorf("security JWT secret is required")
	}

	if c.Security.EncryptionKey == "" {
		return fmt.Errorf("security encryption key is required")
	}

	return nil
}

// GetDatabaseDSN 获取数据库DSN
func (c *Config) GetDatabaseDSN() string {
	return c.Database.DSN
}

// GetServerAddr 获取服务器地址
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetRedisAddr 获取Redis地址
func (c *Config) GetRedisAddr() string {
	return c.Redis.Addr
}

// 辅助函数
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := time.ParseDuration(value); err == nil {
			return int(intValue)
		}
	}
	return defaultValue
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}