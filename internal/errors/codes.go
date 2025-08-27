package errors

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorCode 定义业务错误码
type ErrorCode int32

const (
	// 通用错误码
	CodeOK                ErrorCode = 0
	CodeInternalError     ErrorCode = 1000
	CodeInvalidParameter  ErrorCode = 1001
	CodeUnauthorized      ErrorCode = 1002
	CodeForbidden         ErrorCode = 1003
	CodeNotFound          ErrorCode = 1004
	CodeAlreadyExists     ErrorCode = 1005
	CodeTooManyRequests   ErrorCode = 1006

	// 用户相关错误码
	CodeUserNotFound      ErrorCode = 2001
	CodeUserAlreadyExists ErrorCode = 2002
	CodeInvalidUserName   ErrorCode = 2003
	CodeInvalidEmail      ErrorCode = 2004

	// 策略相关错误码
	CodePolicyNotFound      ErrorCode = 3001
	CodePolicyAlreadyExists ErrorCode = 3002
	CodeInvalidPolicy       ErrorCode = 3003
	CodePolicyAttachFailed  ErrorCode = 3004

	// 访问密钥相关错误码
	CodeAccessKeyNotFound     ErrorCode = 4001
	CodeAccessKeyInvalid      ErrorCode = 4002
	CodeAccessKeyExpired      ErrorCode = 4003
	CodeAccessKeyInactive     ErrorCode = 4004
	CodeTooManyAccessKeys     ErrorCode = 4005
	CodeAccessKeyCreateFailed ErrorCode = 4006

	// 权限相关错误码
	CodePermissionDenied ErrorCode = 5001
	CodeInvalidAction    ErrorCode = 5002
	CodeInvalidResource  ErrorCode = 5003
)

// ErrorMessage 错误码对应的消息
var ErrorMessage = map[ErrorCode]string{
	// 通用错误
	CodeOK:                "Success",
	CodeInternalError:     "Internal server error",
	CodeInvalidParameter:  "Invalid parameter",
	CodeUnauthorized:      "Unauthorized",
	CodeForbidden:         "Forbidden",
	CodeNotFound:          "Resource not found",
	CodeAlreadyExists:     "Resource already exists",
	CodeTooManyRequests:   "Too many requests",

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

// ErrorCodeToGRPCCode 将业务错误码映射到gRPC状态码
var ErrorCodeToGRPCCode = map[ErrorCode]codes.Code{
	CodeOK:                codes.OK,
	CodeInternalError:     codes.Internal,
	CodeInvalidParameter:  codes.InvalidArgument,
	CodeUnauthorized:      codes.Unauthenticated,
	CodeForbidden:         codes.PermissionDenied,
	CodeNotFound:          codes.NotFound,
	CodeAlreadyExists:     codes.AlreadyExists,
	CodeTooManyRequests:   codes.ResourceExhausted,

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

// BusinessError 业务错误结构
type BusinessError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// Error 实现error接口
func (e *BusinessError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewBusinessError 创建业务错误
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
		err.Details = details[0]
	}

	return err
}

// ToGRPCError 将业务错误转换为gRPC错误
func (e *BusinessError) ToGRPCError() error {
	grpcCode, exists := ErrorCodeToGRPCCode[e.Code]
	if !exists {
		grpcCode = codes.Internal
	}

	return status.Error(grpcCode, e.Error())
}

// FromError 从标准错误创建业务错误
func FromError(err error) *BusinessError {
	if err == nil {
		return nil
	}

	if bizErr, ok := err.(*BusinessError); ok {
		return bizErr
	}

	return NewBusinessError(CodeInternalError, err.Error())
}