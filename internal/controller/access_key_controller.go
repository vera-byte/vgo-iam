package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// AccessKeyController 访问密钥控制器
type AccessKeyController struct {
	accessKeyService *service.AccessKeyService
}

// NewAccessKeyController 创建访问密钥控制器
// 参数:
//   - accessKeyService: 访问密钥服务
// 返回值:
//   - *AccessKeyController: 访问密钥控制器实例
func NewAccessKeyController(accessKeyService *service.AccessKeyService) *AccessKeyController {
	return &AccessKeyController{
		accessKeyService: accessKeyService,
	}
}

// CreateAccessKeyRequest 创建访问密钥请求
type CreateAccessKeyRequest struct {
	UserName    string `json:"user_name" binding:"required"`
	Description string `json:"description" binding:"omitempty,max=500"`
}

// UpdateAccessKeyRequest 更新访问密钥请求
type UpdateAccessKeyRequest struct {
	Description string `json:"description" binding:"omitempty,max=500"`
	Status      string `json:"status" binding:"omitempty,oneof=Active Inactive"`
}

// ListAccessKeys 获取访问密钥列表
// 参数:
//   - c: Gin上下文
func (akc *AccessKeyController) ListAccessKeys(c *gin.Context) {
	userName := c.Query("user_name")
	if userName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_name parameter is required",
		})
		return
	}

	accessKeys, err := akc.accessKeyService.ListAccessKeys(c.Request.Context(), userName)
	if err != nil {
		if bizErr, ok := err.(*errors.BusinessError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": bizErr.Message,
				"code":  bizErr.Code,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_keys": accessKeys,
	})
}

// CreateAccessKey 创建访问密钥
// 参数:
//   - c: Gin上下文
func (akc *AccessKeyController) CreateAccessKey(c *gin.Context) {
	var req CreateAccessKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	accessKey, err := akc.accessKeyService.CreateAccessKey(c.Request.Context(), req.UserName)
	if err != nil {
		if bizErr, ok := err.(*errors.BusinessError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": bizErr.Message,
				"code":  bizErr.Code,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "access key created successfully",
		"access_key": accessKey,
	})
}

// GetAccessKey 获取访问密钥详情
// 参数:
//   - c: Gin上下文
func (akc *AccessKeyController) GetAccessKey(c *gin.Context) {
	accessKeyID := c.Param("id")
	if accessKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "access key id is required",
		})
		return
	}

	accessKey, err := akc.accessKeyService.GetAccessKey(c.Request.Context(), accessKeyID)
	if err != nil {
		if bizErr, ok := err.(*errors.BusinessError); ok {
			c.JSON(http.StatusNotFound, gin.H{
				"error": bizErr.Message,
				"code":  bizErr.Code,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_key": accessKey,
	})
}

// UpdateAccessKey 更新访问密钥状态
// 参数:
//   - c: Gin上下文
func (akc *AccessKeyController) UpdateAccessKey(c *gin.Context) {
	accessKeyID := c.Param("id")
	if accessKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "access key id is required",
		})
		return
	}

	var req UpdateAccessKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 只更新状态，忽略描述字段
	if req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "status is required",
		})
		return
	}

	accessKey, err := akc.accessKeyService.UpdateStatus(c.Request.Context(), accessKeyID, req.Status)
	if err != nil {
		if bizErr, ok := err.(*errors.BusinessError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": bizErr.Message,
				"code":  bizErr.Code,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "access key updated successfully",
		"access_key": accessKey,
	})
}

// DeleteAccessKey 删除访问密钥
// 参数:
//   - c: Gin上下文
func (akc *AccessKeyController) DeleteAccessKey(c *gin.Context) {
	accessKeyID := c.Param("id")
	if accessKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "access key id is required",
		})
		return
	}

	// 需要用户名参数，这里暂时使用请求头中的用户名
	userName := c.GetHeader("X-Username")
	if userName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user name is required in X-Username header",
		})
		return
	}

	err := akc.accessKeyService.DeleteAccessKey(c.Request.Context(), userName, accessKeyID)
	if err != nil {
		if bizErr, ok := err.(*errors.BusinessError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": bizErr.Message,
				"code":  bizErr.Code,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "access key deleted successfully",
	})
}

// RotateAccessKey 轮换访问密钥
// 参数:
//   - c: Gin上下文
func (akc *AccessKeyController) RotateAccessKey(c *gin.Context) {
	accessKeyID := c.Param("id")
	if accessKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "access key id is required",
		})
		return
	}

	newAccessKey, err := akc.accessKeyService.RotateAccessKey(c.Request.Context(), accessKeyID)
	if err != nil {
		if bizErr, ok := err.(*errors.BusinessError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": bizErr.Message,
				"code":  bizErr.Code,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "access key rotated successfully",
		"new_access_key": newAccessKey,
	})
}

// GetAccessKeyUsage 获取访问密钥使用情况（未实现）
func (akc *AccessKeyController) GetAccessKeyUsage(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "get access key usage not implemented yet",
	})
}

// 以下方法暂时返回未实现错误，等待后续实现

// EnableAccessKey 启用访问密钥（未实现）
func (akc *AccessKeyController) EnableAccessKey(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "enable access key not implemented yet",
	})
}

// DisableAccessKey 禁用访问密钥（未实现）
func (akc *AccessKeyController) DisableAccessKey(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "disable access key not implemented yet",
	})
}

// GetAccessKeyMetadata 获取访问密钥元数据（未实现）
func (akc *AccessKeyController) GetAccessKeyMetadata(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "get access key metadata not implemented yet",
	})
}