package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// PolicyController 策略控制器
type PolicyController struct {
	policyService *service.PolicyService
}

// NewPolicyController 创建策略控制器
// 参数:
//   - policyService: 策略服务
// 返回值:
//   - *PolicyController: 策略控制器实例
func NewPolicyController(policyService *service.PolicyService) *PolicyController {
	return &PolicyController{
		policyService: policyService,
	}
}

// CreatePolicyRequest 创建策略请求
type CreatePolicyRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
	Document    string `json:"document" binding:"required"`
}

// UpdatePolicyRequest 更新策略请求
type UpdatePolicyRequest struct {
	Description string `json:"description" binding:"omitempty,max=500"`
	Document    string `json:"document" binding:"omitempty"`
}

// ListPolicies 获取策略列表
// 参数:
//   - c: Gin上下文
func (pc *PolicyController) ListPolicies(c *gin.Context) {
	policies, err := pc.policyService.ListPolicies(c.Request.Context())
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

// CreatePolicy 创建策略
// 参数:
//   - c: Gin上下文
func (pc *PolicyController) CreatePolicy(c *gin.Context) {
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	policy, err := pc.policyService.CreatePolicy(c.Request.Context(), req.Name, req.Description, req.Document)
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
		"message": "policy created successfully",
		"policy":  policy,
	})
}

// GetPolicy 获取策略详情
// 参数:
//   - c: Gin上下文
func (pc *PolicyController) GetPolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "policy name is required",
		})
		return
	}

	policy, err := pc.policyService.GetPolicy(c.Request.Context(), policyName)
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
		"policy": policy,
	})
}

// UpdatePolicy 更新策略
// 参数:
//   - c: Gin上下文
func (pc *PolicyController) UpdatePolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "policy name is required",
		})
		return
	}

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	policy, err := pc.policyService.UpdatePolicy(c.Request.Context(), policyName, req.Description, req.Document)
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
		"message": "policy updated successfully",
		"policy":  policy,
	})
}

// DeletePolicy 删除策略
// 参数:
//   - c: Gin上下文
func (pc *PolicyController) DeletePolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "policy name is required",
		})
		return
	}

	err := pc.policyService.DeletePolicy(c.Request.Context(), policyName)
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
		"message": "policy deleted successfully",
	})
}

// ValidatePolicy 验证策略文档（未实现）
func (pc *PolicyController) ValidatePolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "validate policy not implemented yet",
	})
}

// GetPolicyVersions 获取策略版本列表（未实现）
func (pc *PolicyController) GetPolicyVersions(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "get policy versions not implemented yet",
	})
}

// GetPolicyVersion 获取策略特定版本（未实现）
func (pc *PolicyController) GetPolicyVersion(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "get policy version not implemented yet",
	})
}

// SetDefaultPolicyVersion 设置默认策略版本（未实现）
func (pc *PolicyController) SetDefaultPolicyVersion(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "set default policy version not implemented yet",
	})
}

// 以下方法暂时返回未实现错误，等待后续实现

// SimulatePolicy 模拟策略执行（未实现）
func (pc *PolicyController) SimulatePolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "simulate policy not implemented yet",
	})
}

// GetPolicyUsage 获取策略使用情况（未实现）
func (pc *PolicyController) GetPolicyUsage(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "get policy usage not implemented yet",
	})
}