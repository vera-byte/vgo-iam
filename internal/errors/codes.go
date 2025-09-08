// Package errors 提供统一的业务错误处理机制
//
// 本包定义了标准化的业务错误码、错误消息和错误处理函数，支持以下功能：
// - 分类的业务错误码定义（通用、用户、策略、访问密钥、权限等）
// - 国际化错误消息支持
// - gRPC状态码映射
// - 数据库错误转换
// - 标准错误接口实现
//
// 错误码分类：
// - 0-1999: 通用错误码
// - 2000-2999: 用户相关错误码
// - 3000-3999: 策略相关错误码
// - 4000-4999: 访问密钥相关错误码
// - 5000-5999: 权限相关错误码
package errors

import (
	"context"
	"fmt"
	"github.com/gocraft/dbr/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorCode 定义业务错误码类型
// 用于标识不同类型的业务错误，支持国际化和gRPC状态码映射
type ErrorCode int32

const (
	// 通用错误码 (0-1999)
	CodeOK               ErrorCode = 0    // 成功
	CodeInternalError    ErrorCode = 1000 // 内部服务器错误
	CodeInvalidParameter ErrorCode = 1001 // 无效参数
	CodeUnauthorized     ErrorCode = 1002 // 未授权
	CodeForbidden        ErrorCode = 1003 // 禁止访问
	CodeNotFound         ErrorCode = 1004 // 资源未找到
	CodeAlreadyExists    ErrorCode = 1005 // 资源已存在
	CodeTooManyRequests  ErrorCode = 1006 // 请求过多
	CodeNoData           ErrorCode = 1007 // 暂无数据

	// 用户相关错误码 (2000-2999)
	CodeUserNotFound      ErrorCode = 2001 // 用户未找到
	CodeUserAlreadyExists ErrorCode = 2002 // 用户已存在
	CodeInvalidUserName   ErrorCode = 2003 // 无效用户名
	CodeInvalidEmail      ErrorCode = 2004 // 无效邮箱地址

	// 策略相关错误码 (3000-3999)
	CodePolicyNotFound      ErrorCode = 3001 // 策略未找到
	CodePolicyAlreadyExists ErrorCode = 3002 // 策略已存在
	CodeInvalidPolicy       ErrorCode = 3003 // 无效策略文档
	CodePolicyAttachFailed  ErrorCode = 3004 // 策略绑定失败

	// 访问密钥相关错误码 (4000-4999)
	CodeAccessKeyNotFound     ErrorCode = 4001 // 访问密钥未找到
	CodeAccessKeyInvalid      ErrorCode = 4002 // 无效访问密钥
	CodeAccessKeyExpired      ErrorCode = 4003 // 访问密钥已过期
	CodeAccessKeyInactive     ErrorCode = 4004 // 访问密钥未激活
	CodeTooManyAccessKeys     ErrorCode = 4005 // 访问密钥数量过多
	CodeAccessKeyCreateFailed ErrorCode = 4006 // 访问密钥创建失败

	// 权限相关错误码 (5000-5999)
	CodePermissionDenied ErrorCode = 5001 // 权限被拒绝
	CodeInvalidAction    ErrorCode = 5002 // 无效操作
	CodeInvalidResource  ErrorCode = 5003 // 无效资源
)

// ErrorMessageKey 错误码对应的国际化键映射表
// 用于支持多语言错误消息，键值对应locales目录下的翻译文件
var ErrorMessageKey = map[ErrorCode]string{
	// 通用错误
	CodeOK:               "success",
	CodeInternalError:    "error.internal",
	CodeInvalidParameter: "error.invalid_parameter",
	CodeUnauthorized:     "error.unauthorized",
	CodeForbidden:        "error.forbidden",
	CodeNotFound:         "error.not_found",
	CodeAlreadyExists:    "error.already_exists",
	CodeTooManyRequests:  "error.too_many_requests",
	CodeNoData:           "error.no_data",

	// 用户相关错误
	CodeUserNotFound:      "error.user.not_found",
	CodeUserAlreadyExists: "error.user.already_exists",
	CodeInvalidUserName:   "error.user.invalid_name",
	CodeInvalidEmail:      "error.user.invalid_email",

	// 策略相关错误
	CodePolicyNotFound:      "error.policy.not_found",
	CodePolicyAlreadyExists: "error.policy.already_exists",
	CodeInvalidPolicy:       "error.policy.invalid",
	CodePolicyAttachFailed:  "error.policy.attach_failed",

	// 访问密钥相关错误
	CodeAccessKeyNotFound:     "error.access_key.not_found",
	CodeAccessKeyInvalid:      "error.access_key.invalid",
	CodeAccessKeyExpired:      "error.access_key.expired",
	CodeAccessKeyInactive:     "error.access_key.inactive",
	CodeTooManyAccessKeys:     "error.access_key.too_many",
	CodeAccessKeyCreateFailed: "error.access_key.create_failed",

	// 权限相关错误
	CodePermissionDenied: "error.permission.denied",
	CodeInvalidAction:    "error.permission.invalid_action",
	CodeInvalidResource:  "error.permission.invalid_resource",
}

// ErrorMessage 错误码对应的默认英文消息映射表
// 用于向后兼容和翻译失败时的回退消息
var ErrorMessage = map[ErrorCode]string{
	// 通用错误
	CodeOK:               "Success",
	CodeInternalError:    "Internal server error",
	CodeInvalidParameter: "Invalid parameter",
	CodeUnauthorized:     "Unauthorized",
	CodeForbidden:        "Forbidden",
	CodeNotFound:         "Resource not found",
	CodeAlreadyExists:    "Resource already exists",
	CodeTooManyRequests:  "Too many requests",
	CodeNoData:           "暂无数据",

	// 用户相关错误
	CodeUserNotFound:      "User not found",
	CodeUserAlreadyExists: "User already exists",
	CodeInvalidUserName:   "Invalid user name",
	CodeInvalidEmail:      "Invalid email address",

	// 策略相关错误
	CodePolicyNotFound:      "Policy not found",
	CodePolicyAlreadyExists: "Policy already exists",
	CodeInvalidPolicy:       "Invalid policy document",
	CodePolicyAttachFailed:  "Failed to attach policy to user",

	// 访问密钥相关错误
	CodeAccessKeyNotFound:     "Access key not found",
	CodeAccessKeyInvalid:      "Invalid access key",
	CodeAccessKeyExpired:      "Access key expired",
	CodeAccessKeyInactive:     "Access key inactive",
	CodeTooManyAccessKeys:     "Too many access keys for user",
	CodeAccessKeyCreateFailed: "Failed to create access key",

	// 权限相关错误
	CodePermissionDenied: "Permission denied",
	CodeInvalidAction:    "Invalid action",
	CodeInvalidResource:  "Invalid resource",
}

// ErrorCodeToGRPCCode 业务错误码到gRPC状态码的映射表
// 用于将自定义业务错误码转换为标准gRPC状态码
var ErrorCodeToGRPCCode = map[ErrorCode]codes.Code{
	CodeOK:               codes.OK,
	CodeInternalError:    codes.Internal,
	CodeInvalidParameter: codes.InvalidArgument,
	CodeUnauthorized:     codes.Unauthenticated,
	CodeForbidden:        codes.PermissionDenied,
	CodeNotFound:         codes.NotFound,
	CodeAlreadyExists:    codes.AlreadyExists,
	CodeTooManyRequests:  codes.ResourceExhausted,
	CodeNoData:           codes.OK,

	CodeUserNotFound:      codes.NotFound,
	CodeUserAlreadyExists: codes.AlreadyExists,
	CodeInvalidUserName:   codes.InvalidArgument,
	CodeInvalidEmail:      codes.InvalidArgument,

	CodePolicyNotFound:      codes.NotFound,
	CodePolicyAlreadyExists: codes.AlreadyExists,
	CodeInvalidPolicy:       codes.InvalidArgument,
	CodePolicyAttachFailed:  codes.Internal,

	CodeAccessKeyNotFound:     codes.NotFound,
	CodeAccessKeyInvalid:      codes.Unauthenticated,
	CodeAccessKeyExpired:      codes.Unauthenticated,
	CodeAccessKeyInactive:     codes.Unauthenticated,
	CodeTooManyAccessKeys:     codes.ResourceExhausted,
	CodeAccessKeyCreateFailed: codes.Internal,

	CodePermissionDenied: codes.PermissionDenied,
	CodeInvalidAction:    codes.InvalidArgument,
	CodeInvalidResource:  codes.InvalidArgument,
}

// BusinessError 业务错误结构体
// 用于封装业务层的错误信息，包含错误码、消息和可选的详细数据
type BusinessError struct {
	Code    ErrorCode `json:"code"`           // 业务错误码
	Message string    `json:"message"`        // 错误消息
	Data    string    `json:"data,omitempty"` // 可选的错误详细信息
}

// Error 实现标准库error接口
// 返回格式化的错误字符串，用于日志记录和调试
// 返回值: 格式化的错误消息字符串
func (e *BusinessError) Error() string {
	if e.Data != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Translator 翻译器接口
// 用于支持国际化错误消息翻译（临时定义，避免循环依赖）
type Translator interface {
	// Translate 翻译指定键的消息
	// 参数:
	//   - key: 翻译键
	//   - args: 可选的格式化参数
	// 返回值:
	//   - string: 翻译后的消息
	Translate(key string, args ...interface{}) string
}

// NewBusinessError 创建业务错误实例
// 使用默认英文消息创建业务错误，适用于不需要国际化的场景
// 参数:
//   - code: 业务错误码
//   - details: 可选的错误详细信息，只使用第一个参数
//
// 返回值:
//   - *BusinessError: 业务错误实例
func NewBusinessError(code ErrorCode, details ...string) *BusinessError {
	message, exists := ErrorMessage[code]
	if !exists {
		message = "Unknown error"
	}

	err := &BusinessError{
		Code:    code,
		Message: message,
	}

	if len(details) > 0 {
		err.Data = details[0]
	}

	return err
}

// NewBusinessErrorWithTranslator 使用翻译器创建国际化业务错误
// 优先使用翻译器获取本地化消息，翻译失败时回退到默认英文消息
// 参数:
//   - ctx: 请求上下文（预留参数，用于未来扩展）
//   - translator: 翻译器实例，用于获取本地化消息
//   - code: 业务错误码
//   - details: 可选的错误详细信息，只使用第一个参数
//
// 返回值:
//   - *BusinessError: 包含本地化消息的业务错误实例
func NewBusinessErrorWithTranslator(ctx context.Context, translator Translator, code ErrorCode, details ...string) *BusinessError {
	var message string

	// 尝试使用翻译器获取本地化消息
	if translator != nil {
		if key, exists := ErrorMessageKey[code]; exists {
			message = translator.Translate(key)
			// 如果翻译结果和键相同，说明没有找到翻译，使用默认消息
			if message == key {
				if defaultMsg, ok := ErrorMessage[code]; ok {
					message = defaultMsg
				} else {
					message = "Unknown error"
				}
			}
		} else {
			message = "Unknown error"
		}
	} else {
		// 回退到默认消息
		if defaultMsg, exists := ErrorMessage[code]; exists {
			message = defaultMsg
		} else {
			message = "Unknown error"
		}
	}

	err := &BusinessError{
		Code:    code,
		Message: message,
	}

	if len(details) > 0 {
		err.Data = details[0]
	}

	return err
}

// GetLocalizedMessage 获取指定错误码的本地化消息
// 尝试使用翻译器获取本地化消息，失败时返回默认英文消息
// 参数:
//   - translator: 翻译器实例，用于获取本地化消息
//   - code: 业务错误码
//
// 返回值:
//   - string: 本地化错误消息，翻译失败时返回默认英文消息
func GetLocalizedMessage(translator Translator, code ErrorCode) string {
	if translator != nil {
		if key, exists := ErrorMessageKey[code]; exists {
			message := translator.Translate(key)
			// 如果翻译结果和键相同，说明没有找到翻译，使用默认消息
			if message != key {
				return message
			}
		}
	}

	// 回退到默认消息
	if defaultMsg, exists := ErrorMessage[code]; exists {
		return defaultMsg
	}
	return "Unknown error"
}

// ToGRPCError 将业务错误转换为gRPC错误
// 根据业务错误码映射到对应的gRPC状态码
// 返回值:
//   - error: gRPC标准错误
func (e *BusinessError) ToGRPCError() error {
	grpcCode, exists := ErrorCodeToGRPCCode[e.Code]
	if !exists {
		grpcCode = codes.Internal
	}

	return status.Error(grpcCode, e.Error())
}

// HandleDBError 处理数据库错误
// 将数据库特定错误转换为业务错误，特别处理 dbr.ErrNotFound 转换为友好的 "暂无数据" 消息
// 参数:
//   - err: 数据库操作返回的错误
//
// 返回值:
//   - *BusinessError: 转换后的业务错误，如果输入为nil则返回nil
func HandleDBError(err error) *BusinessError {
	if err == nil {
		return nil
	}

	if bizErr, ok := err.(*BusinessError); ok {
		return bizErr
	}

	// 处理 dbr.ErrNotFound 错误，转换为友好的 "暂无数据" 消息
	if err == dbr.ErrNotFound {
		return NewBusinessError(CodeNoData)
	}

	return NewBusinessError(CodeInternalError, err.Error())
}

// FromError 从标准错误创建业务错误
// 如果输入已经是BusinessError则直接返回，否则包装为内部错误
// 参数:
//   - err: 标准错误接口
//
// 返回值:
//   - *BusinessError: 业务错误实例，如果输入为nil则返回nil
func FromError(err error) *BusinessError {
	if err == nil {
		return nil
	}

	if bizErr, ok := err.(*BusinessError); ok {
		return bizErr
	}

	return NewBusinessError(CodeInternalError, err.Error())
}
