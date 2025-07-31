package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vera-byte/vgo-iam/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.Logger

func InitLogger(cfg config.LogConfig) error {
	var cores []zapcore.Core

	// 日志级别
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// 输出到文件
	if cfg.Directory != "" && cfg.Filename != "" {
		if err := os.MkdirAll(cfg.Directory, 0755); err != nil {
			return err
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
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	}

	if len(cores) == 0 {
		return fmt.Errorf("no log output configured")
	}

	Logger = zap.New(zapcore.NewTee(cores...))
	fmt.Println("Logger initialized, pointer:", Logger)
	return nil
}
