package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// ApplicationController 应用控制器
type ApplicationController struct {
	applicationService service.ApplicationService
}

// NewApplicationController 创建应用控制器
// 参数:
//   - applicationService: 应用服务
// 返回值:
//   - *ApplicationController: 应用控制器实例
func NewApplicationController(applicationService service.ApplicationService) *ApplicationController {
	return &ApplicationController{
		applicationService: applicationService,
	}
}

// CreateApplicationRequest 创建应用请求
type CreateApplicationRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Type        string `json:"type" binding:"required,oneof=web mobile desktop api"`
	RedirectURI string `json:"redirect_uri" binding:"omitempty,url"`
}

// UpdateApplicationRequest 更新应用请求
type UpdateApplicationRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Type        string `json:"type" binding:"omitempty,oneof=web mobile desktop api"`
	RedirectURI string `json:"redirect_uri" binding:"omitempty,url"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// ListApplications 获取应用列表
// 参数:
//   - c: Gin上下文
func (ac *ApplicationController) ListApplications(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// TODO: 将status字符串转换为model.AppStatus类型
	applications, total, err := ac.applicationService.ListApplications(c.Request.Context(), userID, model.AppStatusActive, page, pageSize)
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
		"applications": applications,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
	})
}

// CreateApplication 创建应用
// 参数:
//   - c: Gin上下文
func (ac *ApplicationController) CreateApplication(c *gin.Context) {
	// 获取当前用户ID
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 构建服务层请求
	serviceReq := &service.CreateApplicationRequest{
		UserID:         userID,
		AppName:        req.Name,
		AppDescription: req.Description,
		AppType:        req.Type,
		CallbackURLs:   []string{req.RedirectURI},
	}

	application, err := ac.applicationService.CreateApplication(c.Request.Context(), serviceReq)
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
		"message":     "application created successfully",
		"application": application,
	})
}

// GetApplication 获取应用详情
// 参数:
//   - c: Gin上下文
func (ac *ApplicationController) GetApplication(c *gin.Context) {
	appID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid application id",
		})
		return
	}

	application, err := ac.applicationService.GetApplication(c.Request.Context(), appID)
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
		"application": application,
	})
}

// UpdateApplication 更新应用
// 参数:
//   - c: Gin上下文
func (ac *ApplicationController) UpdateApplication(c *gin.Context) {
	appID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid application id",
		})
		return
	}

	// 获取当前用户ID
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 构建服务层请求
	serviceReq := &service.UpdateApplicationRequest{
		ID:             appID,
		UserID:         userID,
		AppName:        req.Name,
		AppDescription: req.Description,
		AppType:        req.Type,
		CallbackURLs:   []string{req.RedirectURI},
	}

	err = ac.applicationService.UpdateApplication(c.Request.Context(), serviceReq)
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
		"message": "application updated successfully",
	})
}

// DeleteApplication 删除应用
// 参数:
//   - c: Gin上下文
func (ac *ApplicationController) DeleteApplication(c *gin.Context) {
	appID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid application id",
		})
		return
	}

	// 获取当前用户ID
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	err = ac.applicationService.DeleteApplication(c.Request.Context(), appID, userID)
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
		"message": "application deleted successfully",
	})
}

// GetApplicationCredentials 获取应用凭证（未实现）
func (ac *ApplicationController) GetApplicationCredentials(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "get application credentials not implemented yet",
	})
}

// RegenerateApplicationSecret 重新生成应用密钥（未实现）
func (ac *ApplicationController) RegenerateApplicationSecret(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "regenerate application secret not implemented yet",
	})
}

// 以下方法暂时返回未实现错误，等待后续实现

// GetApplicationUsage 获取应用使用情况（未实现）
func (ac *ApplicationController) GetApplicationUsage(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "get application usage not implemented yet",
	})
}

// GetApplicationLogs 获取应用日志（未实现）
func (ac *ApplicationController) GetApplicationLogs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "get application logs not implemented yet",
	})
}

// UpdateApplicationStatus 更新应用状态（未实现）
func (ac *ApplicationController) UpdateApplicationStatus(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "update application status not implemented yet",
	})
}