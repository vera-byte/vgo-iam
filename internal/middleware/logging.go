package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vera-byte/vgo-iam/internal/util"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// LoggingConfig 日志拦截器配置
type LoggingConfig struct {
	// 是否启用请求日志
	LogRequests bool
	// 是否启用响应日志
	LogResponses bool
	// 是否启用错误日志
	LogErrors bool
	// 是否启用性能日志
	LogPerformance bool
	// 排除的方法列表
	ExcludeMethods []string
	// 是否启用敏感信息脱敏
	SanitizeEnabled bool
}

// DefaultLoggingConfig 返回默认的日志配置
// 返回值:
//   - *LoggingConfig: 默认日志配置
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		LogRequests:     true,
		LogResponses:    false, // 默认不记录响应以减少日志量
		LogErrors:       true,
		LogPerformance:  true,
		ExcludeMethods:  []string{"/grpc.health.v1.Health/Check"},
		SanitizeEnabled: true,
	}
}

// LoggingUnaryInterceptor 创建一元日志拦截器
// 返回值:
//   - grpc.UnaryServerInterceptor: gRPC一元服务器拦截器
func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return LoggingInterceptor(nil)
}

// LoggingStreamInterceptor 创建流日志拦截器
// 返回值:
//   - grpc.StreamServerInterceptor: gRPC流服务器拦截器
func LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		clientIP := getClientIP(ss.Context())
		userAgent := getUserAgent(ss.Context())
		requestID := generateRequestID()

		// 基础日志字段
		logFields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.String("type", "stream"),
		}

		vgokit.Log.Info("gRPC流请求开始", logFields...)

		// 执行请求
		err := handler(srv, ss)

		// 计算执行时间
		duration := time.Since(start)
		logFields = append(logFields, zap.Duration("duration", duration))

		if err != nil {
			st := status.Convert(err)
			errorFields := append(logFields,
				zap.String("error_code", st.Code().String()),
				zap.String("error_message", sanitizeErrorMessage(st.Message(), true)),
			)
			vgokit.Log.Error("gRPC流请求执行失败", errorFields...)
		} else {
			vgokit.Log.Info("gRPC流请求执行成功", logFields...)
		}

		return err
	}
}

// LoggingInterceptor 创建日志拦截器
// 参数:
//   - config: 日志配置，如果为nil则使用默认配置
// 返回值:
//   - grpc.UnaryServerInterceptor: gRPC一元服务器拦截器
func LoggingInterceptor(config *LoggingConfig) grpc.UnaryServerInterceptor {
	if config == nil {
		config = DefaultLoggingConfig()
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 检查是否需要排除此方法
		for _, excludeMethod := range config.ExcludeMethods {
			if info.FullMethod == excludeMethod {
				return handler(ctx, req)
			}
		}

		start := time.Now()
		clientIP := getClientIP(ctx)
		userAgent := getUserAgent(ctx)
		requestID := generateRequestID()

		// 基础日志字段
		logFields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		}

		// 记录请求日志
		if config.LogRequests {
			reqData := sanitizeRequestData(req, config.SanitizeEnabled)
			logFields = append(logFields, zap.Any("request", reqData))
			vgokit.Log.Info("gRPC请求开始", logFields...)
		}

		// 执行请求
		resp, err := handler(ctx, req)

		// 计算执行时间
		duration := time.Since(start)
		logFields = append(logFields, zap.Duration("duration", duration))

		// 记录性能日志
		if config.LogPerformance {
			if duration > 1*time.Second {
				vgokit.Log.Warn("gRPC请求执行时间过长", logFields...)
			} else if duration > 500*time.Millisecond {
				vgokit.Log.Info("gRPC请求执行时间较长", logFields...)
			}
		}

		// 记录错误日志
		if err != nil && config.LogErrors {
			st := status.Convert(err)
			errorFields := append(logFields,
				zap.String("error_code", st.Code().String()),
				zap.String("error_message", sanitizeErrorMessage(st.Message(), config.SanitizeEnabled)),
			)
			vgokit.Log.Error("gRPC请求执行失败", errorFields...)
		} else {
			// 记录成功的响应日志
			if config.LogResponses {
				respData := sanitizeResponseData(resp, config.SanitizeEnabled)
				logFields = append(logFields, zap.Any("response", respData))
			}
			vgokit.Log.Info("gRPC请求执行成功", logFields...)
		}

		return resp, err
	}
}

// getClientIP 获取客户端IP地址
// 参数:
//   - ctx: 上下文
// 返回值:
//   - string: 客户端IP地址
func getClientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}
	return p.Addr.String()
}

// getUserAgent 获取用户代理
// 参数:
//   - ctx: 上下文
// 返回值:
//   - string: 用户代理字符串
func getUserAgent(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "unknown"
	}

	userAgents := md.Get("user-agent")
	if len(userAgents) > 0 {
		return userAgents[0]
	}
	return "unknown"
}

// generateRequestID 生成请求ID
// 返回值:
//   - string: 唯一的请求ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// sanitizeRequestData 脱敏请求数据
// 参数:
//   - req: 请求数据
//   - enabled: 是否启用脱敏
// 返回值:
//   - interface{}: 脱敏后的请求数据
func sanitizeRequestData(req interface{}, enabled bool) interface{} {
	if !enabled {
		return req
	}

	// 将请求转换为JSON字符串进行脱敏
	jsonData, err := json.Marshal(req)
	if err != nil {
		return req
	}

	// 使用脱敏工具处理
	sanitized := util.SanitizeString(string(jsonData))

	// 尝试解析回结构化数据
	var result interface{}
	if err := json.Unmarshal([]byte(sanitized), &result); err != nil {
		// 如果解析失败，返回脱敏后的字符串
		return sanitized
	}

	return result
}

// sanitizeResponseData 脱敏响应数据
// 参数:
//   - resp: 响应数据
//   - enabled: 是否启用脱敏
// 返回值:
//   - interface{}: 脱敏后的响应数据
func sanitizeResponseData(resp interface{}, enabled bool) interface{} {
	if !enabled || resp == nil {
		return resp
	}

	// 将响应转换为JSON字符串进行脱敏
	jsonData, err := json.Marshal(resp)
	if err != nil {
		return resp
	}

	// 使用脱敏工具处理
	sanitized := util.SanitizeString(string(jsonData))

	// 尝试解析回结构化数据
	var result interface{}
	if err := json.Unmarshal([]byte(sanitized), &result); err != nil {
		// 如果解析失败，返回脱敏后的字符串
		return sanitized
	}

	return result
}

// sanitizeErrorMessage 脱敏错误消息
// 参数:
//   - message: 错误消息
//   - enabled: 是否启用脱敏
// 返回值:
//   - string: 脱敏后的错误消息
func sanitizeErrorMessage(message string, enabled bool) string {
	if !enabled {
		return message
	}
	return util.SanitizeString(message)
}