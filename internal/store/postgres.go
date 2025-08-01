package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gocraft/dbr/v2"
	_ "github.com/lib/pq"
	"github.com/vera-byte/vgo-iam/internal/util"
	"go.uber.org/zap"
)

// PostgresStore PostgreSQL存储
type PostgresStore struct {
	DB      *sql.DB
	Session *dbr.Session
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	// 标准化DSN格式
	if !strings.Contains(dsn, "://") {
		dsn = "postgres://" + strings.TrimPrefix(dsn, "postgresql://")
	}

	// 创建标准连接
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db open failed: %w", err)
	}

	// 带超时的连接测试
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if pingErr := db.PingContext(ctx); pingErr != nil {
		db.Close()
		return nil, fmt.Errorf("db ping failed: %w", err)
	}

	// 优化连接池配置
	db.SetMaxOpenConns(25)                  // 建议值: (2 * CPU核心数) + 备用连接
	db.SetMaxIdleConns(5)                   // 小于MaxOpenConns
	db.SetConnMaxLifetime(30 * time.Minute) // 避免云服务断开
	db.SetConnMaxIdleTime(5 * time.Minute)  // 主动回收闲置连接
	monitorConnection(db, 10*time.Second)
	// 创建dbr连接（关键修正）
	conn, err := dbr.Open("postgres", dsn, nil)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("dbr open failed: %w", err)
	}

	return &PostgresStore{
		DB:      db,
		Session: conn.NewSession(nil),
	}, nil
}

// 连接健康监控
func monitorConnection(db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := db.PingContext(ctx)
		cancel()

		if err != nil {
			util.Logger.Warn("Database connection unhealthy", zap.Error(err))
		}
	}
}

// 关闭方法 (带重试机制)
func (s *PostgresStore) Close() error {
	const maxRetries = 3
	var errs []error

	closeFunc := func(name string, closer func() error) {
		for i := 0; i < maxRetries; i++ {
			if err := closer(); err == nil {
				return
			}
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
		}
		errs = append(errs, fmt.Errorf("failed to close %s after %d retries", name, maxRetries))
	}

	closeFunc("DB", s.DB.Close)
	closeFunc("dbr session", s.Session.Close)

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// 添加重试工具函数
func withRetry(maxRetries int, fn func() error) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		// 指数退避重试
		time.Sleep(time.Duration(1<<i) * 100 * time.Millisecond)
	}
	return fmt.Errorf("after %d retries: %w", maxRetries, err)
}

// 修改查询执行方法
func (s *PostgresStore) ExecQuery(query string, args ...interface{}) (sql.Result, error) {
	return s.Session.Exec(query, args...)
}

// 添加带重试的查询方法
func (s *PostgresStore) ExecQueryWithRetry(query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	err := withRetry(3, func() error {
		var execErr error
		result, execErr = s.ExecQuery(query, args...)
		return execErr
	})
	return result, err
}
