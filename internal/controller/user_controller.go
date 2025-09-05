package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// UserController 用户控制器
type UserController struct {
	userService *service.UserService
}

// NewUserController 创建用户控制器
// 参数:
//   - userService: 用户服务
// 返回值:
//   - *UserController: 用户控制器实例
func NewUserController(userService *service.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	DisplayName string `json:"display_name" binding:"required,min=1,max=100"`
	Email       string `json:"email" binding:"required,email"`
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpdateProfileRequest 更新用户资料请求
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name" binding:"omitempty,max=100"`
	Email       string `json:"email" binding:"omitempty,email"`
}

// Register 用户注册
// 参数:
//   - c: Gin上下文
func (uc *UserController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	user, err := uc.userService.CreateUser(c.Request.Context(), req.Username, req.DisplayName, req.Email)
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
		"message": "user registered successfully",
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Name,
			"display_name": user.DisplayName,
			"email":        user.Email,
		},
	})
}

// Login 用户登录（简化版本）
// 参数:
//   - c: Gin上下文
func (uc *UserController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 获取用户信息
	user, err := uc.userService.GetUser(c.Request.Context(), req.Username)
	if err != nil {
		if bizErr, ok := err.(*errors.BusinessError); ok {
			c.JSON(http.StatusUnauthorized, gin.H{
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

	// TODO: 实现密码验证和JWT令牌生成
	// 这里暂时返回模拟令牌
	token := "mock_token_" + user.Name

	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"token":   token,
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Name,
			"display_name": user.DisplayName,
			"email":        user.Email,
		},
	})
}

// GetProfile 获取用户资料（暂时返回固定用户）
// 参数:
//   - c: Gin上下文
func (uc *UserController) GetProfile(c *gin.Context) {
	// TODO: 从认证中间件获取当前用户
	username := c.GetHeader("X-Username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	user, err := uc.userService.GetUser(c.Request.Context(), username)
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
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Name,
			"display_name": user.DisplayName,
			"email":        user.Email,
			"created_at":   user.CreatedAt,
			"updated_at":   user.UpdatedAt,
		},
	})
}

// UpdateProfile 更新用户资料
// 参数:
//   - c: Gin上下文
func (uc *UserController) UpdateProfile(c *gin.Context) {
	// TODO: 从认证中间件获取当前用户
	username := c.GetHeader("X-Username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	// 获取当前用户
	currentUser, err := uc.userService.GetUser(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not found",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 使用当前值作为默认值
	displayName := req.DisplayName
	if displayName == "" {
		displayName = currentUser.DisplayName
	}
	email := req.Email
	if email == "" {
		email = currentUser.Email
	}

	updatedUser, err := uc.userService.UpdateUser(c.Request.Context(), currentUser.ID, displayName, email)
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
		"message": "profile updated successfully",
		"user": gin.H{
			"id":           updatedUser.ID,
			"username":     updatedUser.Name,
			"display_name": updatedUser.DisplayName,
			"email":        updatedUser.Email,
		},
	})
}

// ListUsers 获取用户列表
// 参数:
//   - c: Gin上下文
func (uc *UserController) ListUsers(c *gin.Context) {
	users, err := uc.userService.ListUsers(c.Request.Context())
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
		"users": users,
	})
}

// CreateUser 创建用户（管理员）
// 参数:
//   - c: Gin上下文
func (uc *UserController) CreateUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	user, err := uc.userService.CreateUser(c.Request.Context(), req.Username, req.DisplayName, req.Email)
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
		"message": "user created successfully",
		"user":    user,
	})
}

// GetUser 获取用户详情
// 参数:
//   - c: Gin上下文
func (uc *UserController) GetUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	user, err := uc.userService.GetUserByID(c.Request.Context(), userID)
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
		"user": user,
	})
}

// UpdateUser 更新用户
// 参数:
//   - c: Gin上下文
func (uc *UserController) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	user, err := uc.userService.UpdateUser(c.Request.Context(), userID, req.DisplayName, req.Email)
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
		"message": "user updated successfully",
		"user":    user,
	})
}

// DeleteUser 删除用户
// 参数:
//   - c: Gin上下文
func (uc *UserController) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	err = uc.userService.DeleteUser(c.Request.Context(), userID)
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
		"message": "user deleted successfully",
	})
}

// GetUserPolicies 获取用户策略
// 参数:
//   - c: Gin上下文
func (uc *UserController) GetUserPolicies(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	policies, err := uc.userService.GetUserPolicies(c.Request.Context(), userID)
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
		"policies": policies,
	})
}

// AttachPolicy 附加策略到用户
// 参数:
//   - c: Gin上下文
func (uc *UserController) AttachPolicy(c *gin.Context) {
	userName := c.Param("id") // 这里假设传入的是用户名
	policyName := c.Param("policy_id") // 这里假设传入的是策略名

	err := uc.userService.AttachPolicy(c.Request.Context(), userName, policyName)
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
		"message": "policy attached successfully",
	})
}

// 以下方法暂时返回未实现错误，等待后续实现

// ChangePassword 修改密码（未实现）
func (uc *UserController) ChangePassword(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "change password not implemented yet",
	})
}

// ForgotPassword 忘记密码（未实现）
func (uc *UserController) ForgotPassword(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "forgot password not implemented yet",
	})
}

// ResetPassword 重置密码（未实现）
func (uc *UserController) ResetPassword(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "reset password not implemented yet",
	})
}

// DeleteAccount 删除账户（未实现）
func (uc *UserController) DeleteAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "delete account not implemented yet",
	})
}

// EnableUser 启用用户（未实现）
func (uc *UserController) EnableUser(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "enable user not implemented yet",
	})
}

// DisableUser 禁用用户（未实现）
func (uc *UserController) DisableUser(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "disable user not implemented yet",
	})
}

// DetachPolicy 从用户分离策略（未实现）
func (uc *UserController) DetachPolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "detach policy not implemented yet",
	})
}