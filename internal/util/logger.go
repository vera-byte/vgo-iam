package util

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/vera-byte/vgo-iam/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.Logger

func InitLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var cores []zapcore.Core

	// 日志级别
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// 输出到文件
	if cfg.Directory != "" && cfg.Filename != "" {
		if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
			return nil, err
		}
		logfile := filepath.Join(cfg.Directory, cfg.Filename)
		fileWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    100, // MB
			MaxBackups: 10,
			MaxAge:     30, // days
			Compress:   true,
		})
		// 文件建议用 JSON encoder
		fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		cores = append(cores, zapcore.NewCore(fileEncoder, fileWriter, level))
	}

	// 输出到终端
	if cfg.ToStdout {
		// 终端建议用 Console encoder
		consoleConfig := zap.NewDevelopmentEncoderConfig()
		consoleConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), level))
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no log output configured")
	}

	Logger = zap.New(zapcore.NewTee(cores...))
	fmt.Println("Logger initialized, pointer:", Logger)
	return Logger, nil
}

// 添加请求ID生成
func GenerateRequestID() string {
	return fmt.Sprintf("req-%s-%d", uuid.New().String()[:8], time.Now().UnixNano()%1000000)
}

// 添加带请求ID的日志函数
func WithRequestID(logger *zap.Logger, reqID string) *zap.Logger {
	if reqID == "" {
		reqID = GenerateRequestID()
	}
	return logger.With(zap.String("request_id", reqID))
}
