package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/vera-byte/vgo-iam/internal/errors"
	vgokit "github.com/vera-byte/vgo-kit"
	"github.com/vera-byte/vgo-kit/i18n"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StandardResponse 标准API响应格式
// 统一的API响应结构，包含错误码、消息和数据
type StandardResponse struct {
	Code    int         `json:"code"`           // 业务错误码，使用项目定义的错误码
	Message string      `json:"message"`        // 响应消息
	Data    interface{} `json:"data,omitempty"` // 响应数据
}

// SuccessResponse 创建成功响应
// data: 响应数据
// 返回: 标准响应结构
func SuccessResponse(data interface{}) *StandardResponse {
	return &StandardResponse{
		Code:    int(errors.CodeOK),
		Message: "success",
		Data:    data,
	}
}

// SuccessResponseWithMessage 创建带自定义消息的成功响应
// data: 响应数据
// message: 自定义消息
// 返回: 标准响应结构
func SuccessResponseWithMessage(data interface{}, message string) *StandardResponse {
	return &StandardResponse{
		Code:    int(errors.CodeOK),
		Message: message,
		Data:    data,
	}
}

// SuccessResponseI18n 创建国际化成功响应
// ctx: 上下文，用于获取语言信息
// data: 响应数据
// translator: 翻译器
// 返回: 标准响应结构
func SuccessResponseI18n(ctx context.Context, data interface{}, translator i18n.Translator) *StandardResponse {
	// 设置翻译器的语言（从上下文获取）
	translator.SetLanguage(i18n.GetLanguageFromContext(ctx))
	message := translator.Translate("message.success")
	return &StandardResponse{
		Code:    int(errors.CodeOK),
		Message: message,
		Data:    data,
	}
}

// ErrorResponse 创建错误响应
// code: 错误码
// message: 错误消息
// 返回: 标准响应结构
func ErrorResponse(code errors.ErrorCode, message string) *StandardResponse {
	return &StandardResponse{
		Code:    int(code),
		Message: message,
		Data:    nil,
	}
}

// BusinessErrorResponse 从业务错误创建响应
// err: 业务错误
// 返回: 标准响应结构
func BusinessErrorResponse(err *errors.BusinessError) *StandardResponse {
	return &StandardResponse{
		Code:    int(err.Code),
		Message: err.Message,
		Data:    nil,
	}
}

// WriteJSONResponse 写入JSON响应
// w: HTTP响应写入器
// statusCode: HTTP状态码
// response: 响应数据
func WriteJSONResponse(w http.ResponseWriter, statusCode int, response *StandardResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		vgokit.Log.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// WriteErrorResponse 写入错误响应
// w: HTTP响应写入器
// err: 错误信息
func WriteErrorResponse(w http.ResponseWriter, err error) {
	// 检查是否为业务错误
	if businessErr, ok := err.(*errors.BusinessError); ok {
		// 业务错误，使用自定义错误码
		response := BusinessErrorResponse(businessErr)
		httpStatus := getHTTPStatusFromBusinessError(businessErr.Code)
		WriteJSONResponse(w, httpStatus, response)
		return
	}

	// gRPC错误处理
	s := status.Convert(err)
	response := &StandardResponse{
		Code:    int(s.Code()),
		Message: s.Message(),
		Data:    nil,
	}

	// 设置HTTP状态码
	httpStatus := getHTTPStatusFromGRPCCode(s.Code())
	WriteJSONResponse(w, httpStatus, response)
}

// getHTTPStatusFromBusinessError 从业务错误码获取HTTP状态码
// code: 业务错误码
// 返回: HTTP状态码
func getHTTPStatusFromBusinessError(code errors.ErrorCode) int {
	switch code {
	case errors.CodeOK:
		return http.StatusOK
	case errors.CodeInvalidParameter:
		return http.StatusBadRequest
	case errors.CodeUserNotFound, errors.CodePolicyNotFound, errors.CodeAccessKeyNotFound, errors.CodeNotFound:
		return http.StatusNotFound
	case errors.CodeUserAlreadyExists, errors.CodePolicyAlreadyExists, errors.CodeAlreadyExists:
		return http.StatusConflict
	case errors.CodePermissionDenied:
		return http.StatusForbidden
	case errors.CodeUnauthorized:
		return http.StatusUnauthorized
	case errors.CodeNoData:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// ResponseWrapper 响应包装器，用于将原始响应转换为标准格式
type ResponseWrapper struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	ctx        context.Context
	translator i18n.Translator
}

// NewResponseWrapper 创建响应包装器
// w: 原始响应写入器
// 返回: 响应包装器
func NewResponseWrapper(w http.ResponseWriter) *ResponseWrapper {
	return &ResponseWrapper{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
}

// NewResponseWrapperWithI18n 创建带国际化支持的响应包装器
// w: 原始响应写入器
// ctx: 上下文
// translator: 翻译器
// 返回: 响应包装器
func NewResponseWrapperWithI18n(w http.ResponseWriter, ctx context.Context, translator i18n.Translator) *ResponseWrapper {
	return &ResponseWrapper{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
		ctx:            ctx,
		translator:     translator,
	}
}

// Write 写入响应数据
// data: 响应数据
// 返回: 写入字节数和错误
func (rw *ResponseWrapper) Write(data []byte) (int, error) {
	return rw.body.Write(data)
}

// WriteHeader 写入响应状态码
// statusCode: HTTP状态码
func (rw *ResponseWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
}

// Flush 刷新响应，将包装后的标准格式响应写入原始响应写入器
func (rw *ResponseWrapper) Flush() {
	// 如果是成功响应，包装为标准格式
	if rw.statusCode >= 200 && rw.statusCode < 300 {
		// 尝试解析原始响应数据
		var originalData interface{}
		if rw.body.Len() > 0 {
			if err := json.Unmarshal(rw.body.Bytes(), &originalData); err != nil {
				// 如果解析失败，使用原始字符串
				originalData = rw.body.String()
			}
		}

		// 创建标准成功响应
		var standardResp *StandardResponse
		if rw.translator != nil && rw.ctx != nil {
			// 使用国际化响应
			standardResp = SuccessResponseI18n(rw.ctx, originalData, rw.translator)
		} else {
			// 使用默认响应
			standardResp = SuccessResponse(originalData)
		}

		// 先编码JSON数据以获取准确的长度
		responseData, err := json.Marshal(standardResp)
		if err != nil {
			vgokit.Log.Error("Failed to marshal standard response", zap.Error(err))
			return
		}

		// 清除可能存在的Content-Length头部，让Go自动设置
		rw.ResponseWriter.Header().Del("Content-Length")
		// 设置响应头
		rw.ResponseWriter.Header().Set("Content-Type", "application/json")
		rw.ResponseWriter.WriteHeader(rw.statusCode)

		// 写入标准响应
		if _, err := rw.ResponseWriter.Write(responseData); err != nil {
			vgokit.Log.Error("Failed to write standard response", zap.Error(err))
		}
	} else {
		// 错误响应直接写入（已经在错误处理器中处理过）
		rw.ResponseWriter.WriteHeader(rw.statusCode)
		io.Copy(rw.ResponseWriter, rw.body)
	}
}

// StandardResponseMiddleware 标准响应中间件
// 将所有成功响应包装为标准格式
// handler: 下一个处理器
// 返回: HTTP处理器
func StandardResponseMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 创建响应包装器
		wrapper := NewResponseWrapper(w)

		// 调用下一个处理器
		handler.ServeHTTP(wrapper, r)

		// 刷新响应
		wrapper.Flush()
	})
}

// StandardResponseMiddlewareWithI18n 带国际化支持的标准响应中间件
// 将所有成功响应包装为标准格式，并支持国际化
// translator: 翻译器实例
// handler: 下一个处理器
// 返回: HTTP处理器
func StandardResponseMiddlewareWithI18n(translator i18n.Translator) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 创建带国际化支持的响应包装器
			wrapper := NewResponseWrapperWithI18n(w, r.Context(), translator)

			// 调用下一个处理器
			handler.ServeHTTP(wrapper, r)

			// 刷新响应
			wrapper.Flush()
		})
	}
}

// getHTTPStatusFromGRPCCode 从gRPC错误码获取HTTP状态码
// code: gRPC错误码
// 返回: HTTP状态码
func getHTTPStatusFromGRPCCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.AlreadyExists:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
