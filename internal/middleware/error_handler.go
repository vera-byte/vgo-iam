package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/vera-byte/vgo-iam/internal/errors"
)

// ErrorHandler 统一错误处理中间件
type ErrorHandler struct {
	logger *log.Logger
}

// NewErrorHandler 创建错误处理中间件实例
// 参数:
//   - logger: 日志记录器
// 返回值:
//   - *ErrorHandler: 错误处理中间件实例
func NewErrorHandler(logger *log.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// Handle Gin框架的错误处理中间件
// 返回值:
//   - gin.HandlerFunc: Gin中间件函数
func (h *ErrorHandler) Handle() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			h.handlePanic(c, fmt.Errorf("panic recovered: %s", err))
		} else if err, ok := recovered.(error); ok {
			h.handlePanic(c, err)
		} else {
			h.handlePanic(c, fmt.Errorf("unknown panic: %v", recovered))
		}
	})
}

// HandleError 处理业务错误
// 参数:
//   - c: Gin上下文
//   - err: 错误信息
func (h *ErrorHandler) HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// 记录错误日志
	h.logError(c, err)

	// 根据错误类型返回相应的HTTP状态码和错误信息
	switch e := err.(type) {
	case *errors.BusinessError:
		h.handleBusinessError(c, e)
	default:
		h.handleInternalError(c, err)
	}
}

// handlePanic 处理panic错误
func (h *ErrorHandler) handlePanic(c *gin.Context, err error) {
	h.logger.Printf("Panic recovered: %v, path: %s, method: %s, stack: %s",
		err.Error(),
		c.Request.URL.Path,
		c.Request.Method,
		string(debug.Stack()),
	)

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Internal server error",
		"code":  errors.CodeInternalError,
	})
}

// handleBusinessError 处理业务错误
func (h *ErrorHandler) handleBusinessError(c *gin.Context, err *errors.BusinessError) {
	statusCode := h.getHTTPStatusCode(err.Code)
	c.JSON(statusCode, gin.H{
		"error": err.Message,
		"code":  err.Code,
	})
}



// handleInternalError 处理内部错误
func (h *ErrorHandler) handleInternalError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Internal server error",
		"code":  "INTERNAL_ERROR",
	})
}

// logError 记录错误日志
func (h *ErrorHandler) logError(c *gin.Context, err error) {
	requestID := c.GetHeader("X-Request-ID")

	h.logger.Printf("Request error: %v, request_id: %s, path: %s, method: %s, user_agent: %s, remote_addr: %s",
		err.Error(),
		requestID,
		c.Request.URL.Path,
		c.Request.Method,
		c.Request.UserAgent(),
		c.ClientIP(),
	)
}

// getHTTPStatusCode 根据业务错误码获取HTTP状态码
func (h *ErrorHandler) getHTTPStatusCode(code errors.ErrorCode) int {
	switch code {
	case errors.CodeUserNotFound, errors.CodePolicyNotFound, errors.CodeNotFound:
		return http.StatusNotFound
	case errors.CodeUserAlreadyExists, errors.CodePolicyAlreadyExists, errors.CodeAlreadyExists:
		return http.StatusConflict
	case errors.CodeInvalidUserName, errors.CodeInvalidEmail, errors.CodeInvalidParameter:
		return http.StatusBadRequest
	case errors.CodeUnauthorized:
		return http.StatusUnauthorized
	case errors.CodeForbidden, errors.CodePermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}