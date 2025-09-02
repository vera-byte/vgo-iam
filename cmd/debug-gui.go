package cmd

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/vera-byte/vgo-iam/internal/api"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/errors"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/service"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	vgokit "github.com/vera-byte/vgo-kit"
	"go.uber.org/zap"
)

// debugGUIFiles 调试界面静态文件（暂时内嵌在代码中）
// var debugGUIFiles embed.FS

func init() {
	RootCmd.AddCommand(DebugGUICmd)
}

// DebugGUICmd Web调试界面命令
var DebugGUICmd = &cobra.Command{
	Use:   "debug-gui",
	Short: "Start web debug GUI for VGO-IAM",
	Long:  "Start a web-based debug interface for testing VGO-IAM functionality",
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化配置
		cfg := config.LodIAMConfig()
		// 初始化服务
		iamServer, session := bootstrap.InitServices(cfg)
		defer session.Close()

		// 创建调试服务器
		debugServer := NewDebugServer(iamServer)
		
		// 启动服务器
		port := 8080
		fmt.Printf("Starting VGO-IAM Debug GUI on http://localhost:%d\n", port)
		vgokit.Log.Info("Debug GUI server starting", zap.Int("port", port))
		
		if err := debugServer.Start(port); err != nil {
			vgokit.Log.Error("Failed to start debug server", zap.Error(err))
		}
	},
}

// DebugServer 调试服务器
type DebugServer struct {
	iamServer *api.IAMServer
	router    *gin.Engine
}

// NewDebugServer 创建调试服务器
func NewDebugServer(iamServer *api.IAMServer) *DebugServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	server := &DebugServer{
		iamServer: iamServer,
		router:    router,
	}

	server.setupRoutes()
	return server
}

// Start 启动服务器
func (s *DebugServer) Start(port int) error {
	return s.router.Run(fmt.Sprintf(":%d", port))
}

// setupRoutes 设置路由
func (s *DebugServer) setupRoutes() {
	// 静态文件
	s.router.GET("/", s.handleIndex)
	s.router.GET("/static/*filepath", s.handleStatic)

	// API路由
	api := s.router.Group("/api")
	{
		// 用户管理
		api.POST("/users", s.handleCreateUser)
		api.GET("/users/:username", s.handleGetUser)
		api.PUT("/users/:username", s.handleUpdateUser)
		api.DELETE("/users/:username", s.handleDeleteUser)
		api.GET("/users", s.handleListUsers)

		// 访问密钥管理
		api.POST("/users/:username/access-keys", s.handleCreateAccessKey)
		api.GET("/users/:username/access-keys", s.handleListAccessKeys)
		api.DELETE("/access-keys/:accessKeyId", s.handleDeleteAccessKey)
		api.POST("/access-keys/verify", s.handleVerifyAccessKey)

		// 策略管理
		api.POST("/policies", s.handleCreatePolicy)
		api.GET("/policies/:name", s.handleGetPolicy)
		api.PUT("/policies/:name", s.handleUpdatePolicy)
		api.DELETE("/policies/:name", s.handleDeletePolicy)
		api.GET("/policies", s.handleListPolicies)

		// 权限检查
		api.POST("/check-permission", s.handleCheckPermission)

		// 应用管理
		api.POST("/applications", s.handleCreateApplication)
		api.GET("/applications/:appId", s.handleGetApplication)
		api.PUT("/applications/:appId", s.handleUpdateApplication)
		api.DELETE("/applications/:appId", s.handleDeleteApplication)
		api.GET("/applications", s.handleListApplications)

		// 开发者认证
		api.POST("/developer-verification", s.handleSubmitDeveloperVerification)
		api.GET("/developer-verification/:username", s.handleGetDeveloperVerification)

		// 系统监控
		api.GET("/metrics", s.handleMetrics)
		api.GET("/health", s.handleHealth)
		api.GET("/logs", s.handleLogs)
		api.GET("/system-info", s.handleSystemInfo)
		
		// 配置管理
		api.GET("/config", s.handleGetConfig)
		api.POST("/config/update", s.handleUpdateConfig)
	}
}

// handleIndex 处理首页
func (s *DebugServer) handleIndex(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, s.getIndexHTML())
}

// handleStatic 处理静态文件
func (s *DebugServer) handleStatic(c *gin.Context) {
	c.String(http.StatusOK, "Static file: %s", c.Param("filepath"))
}

// API处理函数

// handleCreateUser 创建用户
func (s *DebugServer) handleCreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	user, err := s.iamServer.UserService().CreateUser(ctx, req.Username, req.Username, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "username": user.Name})
}

// handleGetUser 获取用户
func (s *DebugServer) handleGetUser(c *gin.Context) {
	username := c.Param("username")
	ctx := context.Background()

	user, err := s.iamServer.UserService().GetUser(ctx, username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// handleUpdateUser 更新用户
func (s *DebugServer) handleUpdateUser(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	if len(username) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度不能超过50个字符"})
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效: " + err.Error()})
		return
	}

	// 验证至少提供一个更新字段
	if req.DisplayName == "" && req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要提供一个更新字段(display_name或email)"})
		return
	}

	// 验证邮箱格式
	if req.Email != "" {
		if len(req.Email) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱长度不能超过100个字符"})
			return
		}
	}

	// 验证显示名称
	if req.DisplayName != "" {
		if len(req.DisplayName) > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "显示名称长度不能超过50个字符"})
			return
		}
	}

	ctx := context.Background()
	
	// 首先获取用户信息
	user, err := s.iamServer.UserService().GetUser(ctx, username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在: " + err.Error()})
		return
	}

	// 更新用户信息
	updatedUser, err := s.iamServer.UserService().UpdateUser(ctx, user.ID, req.DisplayName, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户更新成功",
		"user":    updatedUser,
	})
}

// handleDeleteUser 删除用户
func (s *DebugServer) handleDeleteUser(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	if len(username) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度不能超过50个字符"})
		return
	}

	ctx := context.Background()
	
	// 首先检查用户是否存在
	user, err := s.iamServer.UserService().GetUser(ctx, username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在: " + err.Error()})
		return
	}

	// 删除用户
	err = s.iamServer.UserService().DeleteUser(ctx, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		} else if strings.Contains(err.Error(), "permission") {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限删除此用户"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户删除成功",
		"username": username,
	})
}

// handleListUsers 处理用户列表请求
// 参数: c - Gin上下文
// 功能: 获取系统中所有用户的列表
func (s *DebugServer) handleListUsers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	// 验证页码参数
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "页码必须是大于0的整数"})
		return
	}

	// 验证每页数量参数
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "每页数量必须是1-100之间的整数"})
		return
	}

	ctx := context.Background()
	
	// 获取用户列表
	users, err := s.iamServer.UserService().ListUsers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败: " + err.Error()})
		return
	}

	// 构建响应数据，隐藏敏感信息
	var userList []map[string]interface{}
	for _, user := range users {
		userInfo := map[string]interface{}{
			"id":           user.ID,
			"name":         user.Name,
			"display_name": user.DisplayName,
			"email":        user.Email,
			"created_at":   user.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at":   user.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		userList = append(userList, userInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取用户列表成功",
		"users":   userList,
		"count":   len(userList),
	})
}

// handleCreateAccessKey 创建访问密钥
func (s *DebugServer) handleCreateAccessKey(c *gin.Context) {
	username := c.Param("username")
	var req struct {
		Description string `json:"description"`
	}

	c.ShouldBindJSON(&req)

	ctx := context.Background()
	
	// 临时禁用开发者认证检查（仅用于调试）
	originalDeveloperService := s.iamServer.AccessKeyService().GetDeveloperVerificationService()
	s.iamServer.AccessKeyService().SetDeveloperVerificationService(nil)
	
	accessKey, err := s.iamServer.AccessKeyService().CreateAccessKey(ctx, username)
	
	// 恢复原始的开发者认证服务
	s.iamServer.AccessKeyService().SetDeveloperVerificationService(originalDeveloperService)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, accessKey)
}

// handleListAccessKeys 列出访问密钥
func (s *DebugServer) handleListAccessKeys(c *gin.Context) {
	username := c.Param("username")
	ctx := context.Background()

	accessKeys, err := s.iamServer.AccessKeyService().ListAccessKeys(ctx, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, accessKeys)
}

// handleDeleteAccessKey 删除访问密钥
// 参数:
//   - c: Gin上下文
func (s *DebugServer) handleDeleteAccessKey(c *gin.Context) {
	var req struct {
		UserName    string `json:"user_name" binding:"required"`
		AccessKeyID string `json:"access_key_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效: " + err.Error()})
		return
	}

	if req.UserName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	if req.AccessKeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "访问密钥ID不能为空"})
		return
	}

	if len(req.AccessKeyID) < 10 || len(req.AccessKeyID) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "访问密钥ID格式无效"})
		return
	}

	ctx := context.Background()
	
	// 调用访问密钥服务删除访问密钥
	err := s.iamServer.AccessKeyService().DeleteAccessKey(ctx, req.UserName, req.AccessKeyID)
	if err != nil {
		if businessErr, ok := err.(*errors.BusinessError); ok {
			switch businessErr.Code {
			case errors.CodeUserNotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			case errors.CodeAccessKeyNotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "访问密钥不存在"})
			case errors.CodePermissionDenied:
				c.JSON(http.StatusForbidden, gin.H{"error": "没有权限删除此访问密钥"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": businessErr.Message})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除访问密钥失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "访问密钥删除成功"})
}

// handleVerifyAccessKey 验证访问密钥
func (s *DebugServer) handleVerifyAccessKey(c *gin.Context) {
	var req struct {
		AccessKeyID     string `json:"access_key_id" binding:"required"`
		SecretAccessKey string `json:"secret_access_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	accessKey, err := s.iamServer.AccessKeyService().GetAccessKey(ctx, req.AccessKeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	valid := accessKey.SecretAccessKey == req.SecretAccessKey
	c.JSON(http.StatusOK, gin.H{"valid": valid})
}

// handleCheckPermission 检查权限
// 功能: 处理权限检查请求，验证用户对指定资源的操作权限
// 参数: c *gin.Context - Gin上下文对象
// 返回值: 无
func (s *DebugServer) handleCheckPermission(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Action   string `json:"action" binding:"required"`
		Resource string `json:"resource" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效: " + err.Error()})
		return
	}

	// 验证用户名
	if len(req.Username) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度不能超过50个字符"})
		return
	}

	// 验证资源
	if len(req.Resource) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "资源路径长度不能超过200个字符"})
		return
	}

	ctx := context.Background()
	
	// 使用IAMServer的CheckPermission方法进行权限检查
	checkReq := &iamv1.CheckPermissionRequest{
		UserName: req.Username,
		Action:   req.Action,
		Resource: req.Resource,
	}
	
	checkResp, err := s.iamServer.CheckPermission(ctx, checkReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限检查失败: " + err.Error()})
		return
	}

	// 返回权限检查结果
	response := gin.H{
		"success":  true,
		"allowed":  checkResp.Allowed,
		"username": req.Username,
		"action":   req.Action,
		"resource": req.Resource,
		"message":  fmt.Sprintf("权限检查完成，结果: %t", checkResp.Allowed),
	}

	c.JSON(http.StatusOK, response)
}

// 其他API处理函数...
// handleCreatePolicy 创建策略
func (s *DebugServer) handleCreatePolicy(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
		PolicyDocument string `json:"policy_document" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效: " + err.Error()})
		return
	}

	// 验证策略名称
	if len(req.Name) == 0 || len(req.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "策略名称长度必须在1-100个字符之间"})
		return
	}

	// 验证策略文档
	if len(req.PolicyDocument) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "策略文档不能为空"})
		return
	}

	ctx := context.Background()
	
	// 调用策略服务创建策略
	policy, err := s.iamServer.PolicyService().CreatePolicy(ctx, req.Name, req.Description, req.PolicyDocument)
	if err != nil {
		if businessErr, ok := err.(*errors.BusinessError); ok {
			switch businessErr.Code {
			case errors.CodePolicyAlreadyExists:
				c.JSON(http.StatusConflict, gin.H{"error": "策略已存在"})
			case errors.CodeInvalidPolicy:
				c.JSON(http.StatusBadRequest, gin.H{"error": "策略文档格式无效"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": businessErr.Message})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建策略失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": policy, "message": "策略创建成功"})
}

// handleGetPolicy 获取策略
func (s *DebugServer) handleGetPolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "策略名称不能为空"})
		return
	}

	ctx := context.Background()
	
	// 调用策略服务获取策略
	policy, err := s.iamServer.PolicyService().GetPolicy(ctx, policyName)
	if err != nil {
		if businessErr, ok := err.(*errors.BusinessError); ok {
			switch businessErr.Code {
			case errors.CodePolicyNotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": businessErr.Message})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取策略失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": policy})
}

// handleUpdatePolicy 更新策略
func (s *DebugServer) handleUpdatePolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "策略名称不能为空"})
		return
	}

	var req struct {
		Description    string `json:"description"`
		PolicyDocument string `json:"policy_document" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效: " + err.Error()})
		return
	}

	// 验证策略文档
	if len(req.PolicyDocument) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "策略文档不能为空"})
		return
	}

	ctx := context.Background()
	
	// 调用策略服务更新策略
	policy, err := s.iamServer.PolicyService().UpdatePolicy(ctx, policyName, req.Description, req.PolicyDocument)
	if err != nil {
		if businessErr, ok := err.(*errors.BusinessError); ok {
			switch businessErr.Code {
			case errors.CodePolicyNotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
			case errors.CodeInvalidPolicy:
				c.JSON(http.StatusBadRequest, gin.H{"error": "策略文档格式无效"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": businessErr.Message})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新策略失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": policy, "message": "策略更新成功"})
}

// handleDeletePolicy 删除策略
func (s *DebugServer) handleDeletePolicy(c *gin.Context) {
	policyName := c.Param("name")
	if policyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "策略名称不能为空"})
		return
	}

	ctx := context.Background()
	
	// 调用策略服务删除策略
	err := s.iamServer.PolicyService().DeletePolicy(ctx, policyName)
	if err != nil {
		if businessErr, ok := err.(*errors.BusinessError); ok {
			switch businessErr.Code {
			case errors.CodePolicyNotFound:
				c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": businessErr.Message})
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除策略失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "策略删除成功"})
}

// handleListPolicies 获取策略列表
func (s *DebugServer) handleListPolicies(c *gin.Context) {
	ctx := context.Background()
	
	// 调用策略服务获取策略列表
	policies, err := s.iamServer.PolicyService().ListPolicies(ctx)
	if err != nil {
		if businessErr, ok := err.(*errors.BusinessError); ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": businessErr.Message})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取策略列表失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": policies, "count": len(policies)})
}

func (s *DebugServer) handleCreateApplication(c *gin.Context) {
	var req struct {
		UserName       string   `json:"user_name" binding:"required"`
		AppName        string   `json:"app_name" binding:"required"`
		AppDescription string   `json:"app_description"`
		AppType        string   `json:"app_type" binding:"required"`
		AppIconURL     string   `json:"app_icon_url"`
		AppWebsite     string   `json:"app_website"`
		CallbackURLs   []string `json:"callback_urls"`
		AllowedOrigins []string `json:"allowed_origins"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// 输入验证
	if len(req.AppName) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application name too long (max 100 characters)"})
		return
	}

	if len(req.AppDescription) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application description too long (max 500 characters)"})
		return
	}

	// 验证应用类型
	validAppTypes := map[string]bool{
		"web":     true,
		"mobile":  true,
		"desktop": true,
		"api":     true,
	}
	if !validAppTypes[req.AppType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application type. Must be one of: web, mobile, desktop, api"})
		return
	}

	ctx := context.Background()
	
	// 获取用户信息
	user, err := s.iamServer.UserService().GetUser(ctx, req.UserName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found: " + err.Error()})
		return
	}

	// 构建创建应用请求
	createReq := &service.CreateApplicationRequest{
		UserID:         user.ID,
		AppName:        req.AppName,
		AppDescription: req.AppDescription,
		AppType:        req.AppType,
		CallbackURLs:   req.CallbackURLs,
		AllowedOrigins: req.AllowedOrigins,
	}

	// 创建应用
	app, err := s.iamServer.ApplicationService().CreateApplication(ctx, createReq)
	if err != nil {
		// 根据错误类型返回不同的HTTP状态码
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else if strings.Contains(err.Error(), "verification") {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": app, "message": "Application created successfully"})
}

func (s *DebugServer) handleGetApplication(c *gin.Context) {
	appIDStr := c.Param("appId")
	if appIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application ID is required"})
		return
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid application ID format. Must be a valid integer"})
		return
	}

	if appID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application ID must be a positive integer"})
		return
	}

	ctx := context.Background()
	app, err := s.iamServer.ApplicationService().GetApplication(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve application: " + err.Error()})
		return
	}

	if app == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": app})
}

func (s *DebugServer) handleUpdateApplication(c *gin.Context) {
	appIDStr := c.Param("appId")
	if appIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用ID不能为空"})
		return
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil || appID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用ID必须是正整数"})
		return
	}

	var req struct {
		UserName       string   `json:"user_name" binding:"required"`
		AppName        string   `json:"app_name" binding:"required"`
		AppDescription string   `json:"app_description"`
		AppType        string   `json:"app_type" binding:"required"`
		AppIconURL     string   `json:"app_icon_url"`
		AppWebsite     string   `json:"app_website"`
		CallbackURLs   []string `json:"callback_urls"`
		AllowedOrigins []string `json:"allowed_origins"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效: " + err.Error()})
		return
	}

	// 验证应用名称
	if len(req.AppName) < 2 || len(req.AppName) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用名称长度必须在2-50个字符之间"})
		return
	}

	// 验证应用描述
	if len(req.AppDescription) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用描述长度不能超过200个字符"})
		return
	}

	// 验证应用类型
	validAppTypes := map[string]bool{
		"web":     true,
		"mobile":  true,
		"desktop": true,
		"api":     true,
	}
	if !validAppTypes[req.AppType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用类型无效，支持的值：web, mobile, desktop, api"})
		return
	}

	// 验证URL格式
	if req.AppWebsite != "" && !strings.HasPrefix(req.AppWebsite, "http://") && !strings.HasPrefix(req.AppWebsite, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用网站URL必须以http://或https://开头"})
		return
	}

	// 验证回调URL
	for _, url := range req.CallbackURLs {
		if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "回调URL必须以http://或https://开头"})
			return
		}
	}

	// 验证允许的来源
	for _, origin := range req.AllowedOrigins {
		if origin != "" && origin != "*" && !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "允许的来源必须是*或以http://或https://开头的URL"})
			return
		}
	}

	ctx := context.Background()
	
	// 获取用户信息
	user, err := s.iamServer.UserService().GetUser(ctx, req.UserName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found: " + err.Error()})
		return
	}

	// 构建更新应用请求
	updateReq := &service.UpdateApplicationRequest{
		ID:             appID,
		UserID:         user.ID,
		AppName:        req.AppName,
		AppDescription: req.AppDescription,
		AppType:        req.AppType,
		CallbackURLs:   req.CallbackURLs,
		AllowedOrigins: req.AllowedOrigins,
	}

	// 更新应用
	err = s.iamServer.ApplicationService().UpdateApplication(ctx, updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取更新后的应用信息
	app, err := s.iamServer.ApplicationService().GetApplication(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, app)
}

func (s *DebugServer) handleDeleteApplication(c *gin.Context) {
	appIDStr := c.Param("appId")
	if appIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用ID不能为空"})
		return
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil || appID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用ID必须是正整数"})
		return
	}

	ctx := context.Background()
	
	// 检查应用是否存在
	app, err := s.iamServer.ApplicationService().GetApplication(ctx, appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取应用信息失败: " + err.Error()})
		return
	}

	if app == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "应用不存在"})
		return
	}

	// 删除应用 (使用固定的userID，实际应用中应该从认证上下文获取)
	userID := int64(1) // TODO: 从认证上下文获取用户ID
	err = s.iamServer.ApplicationService().DeleteApplication(ctx, appID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "应用不存在"})
		} else if strings.Contains(err.Error(), "permission") {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限删除此应用"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除应用失败: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "应用删除成功"})
}

func (s *DebugServer) handleListApplications(c *gin.Context) {
	userName := c.Query("user_name")
	if userName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_name parameter is required"})
		return
	}

	// 验证用户名格式
	if len(userName) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度不能超过50个字符"})
		return
	}

	status := c.DefaultQuery("status", "active")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	// 验证页码参数
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "页码必须是大于0的整数"})
		return
	}

	// 验证每页数量参数
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "每页数量必须是1-100之间的整数"})
		return
	}

	// 验证状态参数
	validStatuses := map[string]bool{
		"active":    true,
		"inactive":  true,
		"suspended": true,
	}
	if !validStatuses[status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "状态参数无效，支持的值：active, inactive, suspended"})
		return
	}

	ctx := context.Background()
	
	// 获取用户信息
	user, err := s.iamServer.UserService().GetUser(ctx, userName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found: " + err.Error()})
		return
	}

	// 转换状态
	var appStatus model.AppStatus
	switch status {
	case "active":
		appStatus = model.AppStatusActive
	case "inactive":
		appStatus = model.AppStatusInactive
	case "suspended":
		appStatus = model.AppStatusSuspended
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	// 获取应用列表
	apps, total, err := s.iamServer.ApplicationService().ListApplications(ctx, user.ID, appStatus, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": apps,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
	})
}

// handleSubmitDeveloperVerification 处理提交开发者认证请求
// 参数: 开发者类型、个人信息或企业信息
// 返回值: 认证记录或错误信息
func (s *DebugServer) handleSubmitDeveloperVerification(c *gin.Context) {
	var req struct {
		DeveloperType         string `json:"developer_type" binding:"required"`
		RealName             string `json:"real_name"`
		IDCardNumber         string `json:"id_card_number"`
		IDCardFrontURL       string `json:"id_card_front_url"`
		IDCardBackURL        string `json:"id_card_back_url"`
		CompanyName          string `json:"company_name"`
		BusinessLicenseNumber string `json:"business_license_number"`
		BusinessLicenseURL   string `json:"business_license_url"`
		LegalRepresentative  string `json:"legal_representative"`
		CompanyAddress       string `json:"company_address"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 验证开发者类型
	if req.DeveloperType != "individual" && req.DeveloperType != "enterprise" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "开发者类型必须是 individual 或 enterprise"})
		return
	}

	ctx := context.Background()

	// 构建gRPC请求
	grpcReq := &iamv1.SubmitDeveloperVerificationRequest{
		DeveloperType: req.DeveloperType,
	}

	// 根据开发者类型设置相应字段
	if req.DeveloperType == "individual" {
		grpcReq.RealName = req.RealName
		grpcReq.IdCardNumber = req.IDCardNumber
		grpcReq.IdCardFrontUrl = req.IDCardFrontURL
		grpcReq.IdCardBackUrl = req.IDCardBackURL
	} else {
		grpcReq.CompanyName = req.CompanyName
		grpcReq.BusinessLicenseNumber = req.BusinessLicenseNumber
		grpcReq.BusinessLicenseUrl = req.BusinessLicenseURL
		grpcReq.LegalRepresentative = req.LegalRepresentative
		grpcReq.CompanyAddress = req.CompanyAddress
	}

	// 调用IAMServer的SubmitDeveloperVerification方法
	resp, err := s.iamServer.SubmitDeveloperVerification(ctx, grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交开发者认证失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "开发者认证提交成功",
		"data":    resp,
	})
}

// handleGetDeveloperVerification 处理获取开发者认证信息请求
// 参数: 用户名、开发者类型
// 返回值: 认证信息或错误信息
func (s *DebugServer) handleGetDeveloperVerification(c *gin.Context) {
	username := c.Param("username")
	developerType := c.Query("developer_type")

	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	if developerType == "" {
		developerType = "individual" // 默认为个人开发者
	}

	// 验证开发者类型
	if developerType != "individual" && developerType != "enterprise" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "开发者类型必须是 individual 或 enterprise"})
		return
	}

	ctx := context.Background()

	// 构建gRPC请求
	grpcReq := &iamv1.GetDeveloperVerificationRequest{
		UserName:      username,
		DeveloperType: developerType,
	}

	// 调用IAMServer的GetDeveloperVerification方法
	resp, err := s.iamServer.GetDeveloperVerification(ctx, grpcReq)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到开发者认证信息"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取开发者认证信息失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取开发者认证信息成功",
		"data":    resp,
	})
}

// getIndexHTML 返回调试界面HTML
func (s *DebugServer) getIndexHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>VGO-IAM Debug GUI</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .header {
            background: #fff;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        
        .header h1 {
            color: #2c3e50;
            margin-bottom: 10px;
        }
        
        .tabs {
            display: flex;
            background: #fff;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        
        .tab {
            flex: 1;
            padding: 15px;
            text-align: center;
            cursor: pointer;
            border-bottom: 3px solid transparent;
            transition: all 0.3s;
        }
        
        .tab:hover {
            background: #f8f9fa;
        }
        
        .tab.active {
            border-bottom-color: #3498db;
            color: #3498db;
        }
        
        .content {
            background: #fff;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        
        .tab-content {
            display: none;
        }
        
        .tab-content.active {
            display: block;
        }
        
        .form-group {
            margin-bottom: 15px;
        }
        
        .form-group label {
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
        }
        
        .form-group input,
        .form-group textarea,
        .form-group select {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
        }
        
        .form-group textarea {
            height: 100px;
            resize: vertical;
        }
        
        .btn {
            background: #3498db;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: background 0.3s;
        }
        
        .btn:hover {
            background: #2980b9;
        }
        
        .btn-danger {
            background: #e74c3c;
        }
        
        .btn-danger:hover {
            background: #c0392b;
        }
        
        .result {
            margin-top: 20px;
            padding: 15px;
            border-radius: 4px;
            white-space: pre-wrap;
            font-family: monospace;
            max-height: 400px;
            overflow-y: auto;
        }
        
        .result.success {
            background: #d4edda;
            border: 1px solid #c3e6cb;
            color: #155724;
        }
        
        .result.error {
            background: #f8d7da;
            border: 1px solid #f5c6cb;
            color: #721c24;
        }
        
        .result.loading {
            background: #d1ecf1;
            border: 1px solid #bee5eb;
            color: #0c5460;
        }
        
        .loading-spinner {
            display: inline-block;
            width: 16px;
            height: 16px;
            border: 2px solid #f3f3f3;
            border-top: 2px solid #3498db;
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin-right: 8px;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .notification {
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 15px 20px;
            border-radius: 4px;
            color: white;
            font-weight: 500;
            z-index: 1000;
            transform: translateX(400px);
            transition: transform 0.3s ease;
        }
        
        .notification.show {
            transform: translateX(0);
        }
        
        .notification.success {
            background: #28a745;
        }
        
        .notification.error {
            background: #dc3545;
        }
        
        .notification.info {
            background: #17a2b8;
        }
        
        .form-group.error input,
        .form-group.error select,
        .form-group.error textarea {
            border-color: #dc3545;
            box-shadow: 0 0 0 0.2rem rgba(220, 53, 69, 0.25);
        }
        
        .form-group .error-message {
            color: #dc3545;
            font-size: 12px;
            margin-top: 5px;
            display: none;
        }
        
        .form-group.error .error-message {
            display: block;
        }
        
        .btn:disabled {
            background: #6c757d;
            cursor: not-allowed;
        }
        
        .btn:disabled:hover {
            background: #6c757d;
        }
        
        .grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
        }
        
        @media (max-width: 768px) {
            .grid {
                grid-template-columns: 1fr;
            }
            
            .tabs {
                flex-direction: column;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 VGO-IAM Debug GUI</h1>
            <p>Identity and Access Management 调试工具</p>
        </div>
        
        <div class="tabs">
            <div class="tab active" onclick="showTab('users')">👤 用户管理</div>
            <div class="tab" onclick="showTab('keys')">🔑 访问密钥</div>
            <div class="tab" onclick="showTab('permissions')">🛡️ 权限检查</div>
            <div class="tab" onclick="showTab('apps')">📱 应用管理</div>
            <div class="tab" onclick="showTab('monitoring')">📊 系统监控</div>
            <div class="tab" onclick="showTab('config')">⚙️ 配置管理</div>
        </div>
        
        <div class="content">
            <!-- 用户管理 -->
            <div id="users" class="tab-content active">
                <h2>用户管理</h2>
                <div class="grid">
                    <div>
                        <h3>创建用户</h3>
                        <form id="createUserForm">
                            <div class="form-group">
                                <label>用户名</label>
                                <input type="text" name="username" required>
                            </div>
                            <div class="form-group">
                                <label>密码</label>
                                <input type="password" name="password" required>
                            </div>
                            <div class="form-group">
                                <label>邮箱</label>
                                <input type="email" name="email">
                            </div>
                            <div class="form-group">
                                <label>手机</label>
                                <input type="tel" name="phone">
                            </div>
                            <button type="submit" class="btn">创建用户</button>
                        </form>
                    </div>
                    
                    <div>
                        <h3>查询用户</h3>
                        <form id="getUserForm">
                            <div class="form-group">
                                <label>用户名</label>
                                <input type="text" name="username" required>
                            </div>
                            <button type="submit" class="btn">查询用户</button>
                        </form>
                        
                        <h3 style="margin-top: 20px;">删除用户</h3>
                        <form id="deleteUserForm">
                            <div class="form-group">
                                <label>用户名</label>
                                <input type="text" name="username" required>
                            </div>
                            <button type="submit" class="btn btn-danger">删除用户</button>
                        </form>
                    </div>
                </div>
                <div id="usersResult" class="result" style="display: none;"></div>
            </div>
            
            <!-- 访问密钥 -->
            <div id="keys" class="tab-content">
                <h2>访问密钥管理</h2>
                <div class="grid">
                    <div>
                        <h3>创建访问密钥</h3>
                        <form id="createKeyForm">
                            <div class="form-group">
                                <label>用户名</label>
                                <input type="text" name="username" required>
                            </div>
                            <div class="form-group">
                                <label>描述</label>
                                <input type="text" name="description">
                            </div>
                            <button type="submit" class="btn">创建密钥</button>
                        </form>
                        
                        <h3 style="margin-top: 20px;">查询用户密钥</h3>
                        <form id="listKeysForm">
                            <div class="form-group">
                                <label>用户名</label>
                                <input type="text" name="username" required>
                            </div>
                            <button type="submit" class="btn">查询密钥</button>
                        </form>
                    </div>
                    
                    <div>
                        <h3>验证访问密钥</h3>
                        <form id="verifyKeyForm">
                            <div class="form-group">
                                <label>Access Key ID</label>
                                <input type="text" name="access_key_id" required>
                            </div>
                            <div class="form-group">
                                <label>Secret Access Key</label>
                                <input type="text" name="secret_access_key" required>
                            </div>
                            <button type="submit" class="btn">验证密钥</button>
                        </form>
                    </div>
                </div>
                <div id="keysResult" class="result" style="display: none;"></div>
            </div>
            
            <!-- 权限检查 -->
            <div id="permissions" class="tab-content">
                <h2>权限检查</h2>
                <form id="checkPermissionForm">
                    <div class="form-group">
                        <label>用户名</label>
                        <input type="text" name="username" required>
                    </div>
                    <div class="form-group">
                        <label>操作</label>
                        <select name="action" required>
                            <option value="">选择操作</option>
                            <option value="read">读取</option>
                            <option value="write">写入</option>
                            <option value="delete">删除</option>
                            <option value="admin">管理</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>资源</label>
                        <input type="text" name="resource" placeholder="例如: /api/users" required>
                    </div>
                    <button type="submit" class="btn">检查权限</button>
                </form>
                <div id="permissionsResult" class="result" style="display: none;"></div>
            </div>
            
            <!-- 应用管理 -->
            <div id="apps" class="tab-content">
                <h2>应用管理</h2>
                <div class="grid">
                    <div>
                        <h3>创建应用</h3>
                        <form id="createAppForm">
                            <div class="form-group">
                                <label>应用名称</label>
                                <input type="text" name="app_name" required>
                            </div>
                            <div class="form-group">
                                <label>应用描述</label>
                                <textarea name="app_description" rows="3"></textarea>
                            </div>
                            <div class="form-group">
                                <label>应用类型</label>
                                <select name="app_type" required>
                                    <option value="">选择应用类型</option>
                                    <option value="web">Web应用</option>
                                    <option value="mobile">移动应用</option>
                                    <option value="desktop">桌面应用</option>
                                    <option value="api">API应用</option>
                                </select>
                            </div>
                            <div class="form-group">
                                <label>应用图标URL</label>
                                <input type="url" name="app_icon_url">
                            </div>
                            <div class="form-group">
                                <label>应用网站</label>
                                <input type="url" name="app_website">
                            </div>
                            <div class="form-group">
                                <label>回调URL (多个用逗号分隔)</label>
                                <input type="text" name="callback_urls" placeholder="https://example.com/callback">
                            </div>
                            <div class="form-group">
                                <label>允许的来源 (多个用逗号分隔)</label>
                                <input type="text" name="allowed_origins" placeholder="https://example.com">
                            </div>
                            <button type="submit" class="btn">创建应用</button>
                        </form>
                        
                        <h3 style="margin-top: 20px;">查询应用</h3>
                        <form id="getAppForm">
                            <div class="form-group">
                                <label>应用ID</label>
                                <input type="number" name="app_id" required>
                            </div>
                            <button type="submit" class="btn">查询应用</button>
                        </form>
                    </div>
                    
                    <div>
                        <h3>应用列表</h3>
                        <form id="listAppsForm">
                            <div class="form-group">
                                <label>用户名</label>
                                <input type="text" name="user_name" required>
                            </div>
                            <div class="form-group">
                                <label>页码</label>
                                <input type="number" name="page" value="1" min="1">
                            </div>
                            <div class="form-group">
                                <label>每页数量</label>
                                <input type="number" name="page_size" value="10" min="1" max="100">
                            </div>
                            <button type="submit" class="btn">获取应用列表</button>
                        </form>
                        
                        <h3 style="margin-top: 20px;">更新应用</h3>
                        <form id="updateAppForm">
                            <div class="form-group">
                                <label>应用ID</label>
                                <input type="number" name="app_id" required>
                            </div>
                            <div class="form-group">
                                <label>应用名称</label>
                                <input type="text" name="app_name" required>
                            </div>
                            <div class="form-group">
                                <label>应用描述</label>
                                <textarea name="app_description" rows="2"></textarea>
                            </div>
                            <div class="form-group">
                                <label>应用类型</label>
                                <select name="app_type" required>
                                    <option value="web">Web应用</option>
                                    <option value="mobile">移动应用</option>
                                    <option value="desktop">桌面应用</option>
                                    <option value="api">API应用</option>
                                </select>
                            </div>
                            <button type="submit" class="btn">更新应用</button>
                        </form>
                        
                        <h3 style="margin-top: 20px;">删除应用</h3>
                        <form id="deleteAppForm">
                            <div class="form-group">
                                <label>应用ID</label>
                                <input type="number" name="app_id" required>
                            </div>
                            <button type="submit" class="btn btn-danger">删除应用</button>
                        </form>
                    </div>
                </div>
                <div id="appsResult" class="result" style="display: none;"></div>
            </div>
            
            <!-- 系统监控 -->
            <div id="monitoring" class="tab-content">
                <h2>系统监控</h2>
                <div class="grid">
                    <div>
                        <h3>系统信息</h3>
                        <button id="refreshSystemInfo" class="btn">刷新系统信息</button>
                        <div id="systemInfoResult" class="result" style="display: none;"></div>
                        
                        <h3 style="margin-top: 20px;">健康检查</h3>
                        <button id="checkHealth" class="btn">检查健康状态</button>
                        <div id="healthResult" class="result" style="display: none;"></div>
                    </div>
                    
                    <div>
                        <h3>性能指标</h3>
                        <button id="getMetrics" class="btn">获取Prometheus指标</button>
                        <div id="metricsResult" class="result" style="display: none;"></div>
                        
                        <h3 style="margin-top: 20px;">系统日志</h3>
                        <form id="getLogsForm">
                            <div class="form-group">
                                <label>日志级别</label>
                                <select name="level">
                                    <option value="info">Info</option>
                                    <option value="warn">Warning</option>
                                    <option value="error">Error</option>
                                    <option value="debug">Debug</option>
                                </select>
                            </div>
                            <div class="form-group">
                                <label>限制条数</label>
                                <input type="number" name="limit" value="100" min="1" max="1000">
                            </div>
                            <button type="submit" class="btn">获取日志</button>
                        </form>
                        <div id="logsResult" class="result" style="display: none;"></div>
                    </div>
                </div>
            </div>
            
            <!-- 配置管理 -->
            <div id="config" class="tab-content">
                <h2>配置管理</h2>
                <div class="grid">
                    <!-- 查看当前配置 -->
                    <div class="card">
                        <h3>当前配置</h3>
                        <button type="button" class="btn" onclick="loadCurrentConfig()">加载配置</button>
                        <div id="currentConfigResult" class="result" style="display: none;"></div>
                    </div>
                    
                    <!-- 更新配置 -->
                    <div class="card">
                        <h3>更新配置</h3>
                        <form id="updateConfigForm">
                            <div class="form-group">
                                <label>配置项</label>
                                <select name="configKey" required>
                                    <option value="">请选择配置项</option>
                                    <option value="log.level">日志级别</option>
                                    <option value="log.to_stdout">控制台输出</option>
                                    <option value="sentry.enabled">Sentry启用</option>
                                    <option value="ratelimit.enabled">限流启用</option>
                                    <option value="ratelimit.limit">限流数量</option>
                                    <option value="sts.default_duration">STS默认有效期</option>
                                    <option value="sts.auto_cleanup">STS自动清理</option>
                                </select>
                            </div>
                            <div class="form-group">
                                <label>配置值</label>
                                <input type="text" name="configValue" placeholder="输入新的配置值" required>
                            </div>
                            <button type="submit" class="btn">更新配置</button>
                        </form>
                        <div id="updateConfigResult" class="result" style="display: none;"></div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // 切换标签页
        function showTab(tabName) {
            // 隐藏所有标签内容
            document.querySelectorAll('.tab-content').forEach(content => {
                content.classList.remove('active');
            });
            
            // 移除所有标签的激活状态
            document.querySelectorAll('.tab').forEach(tab => {
                tab.classList.remove('active');
            });
            
            // 显示选中的标签内容
            document.getElementById(tabName).classList.add('active');
            
            // 激活选中的标签
            document.querySelector('.tab[onclick*="' + tabName + '"]').classList.add('active');
        };
        
        // 通知系统
        function showNotification(message, type) {
            type = type || 'info';
            const notification = document.createElement('div');
            notification.className = 'notification ' + type;
            notification.textContent = message;
            document.body.appendChild(notification);
            
            // 显示通知
            setTimeout(() => notification.classList.add('show'), 100);
            
            // 3秒后自动隐藏
            setTimeout(() => {
                notification.classList.remove('show');
                setTimeout(() => document.body.removeChild(notification), 300);
            }, 3000);
        }
        
        // 表单验证
        function validateForm(form) {
            let isValid = true;
            const requiredFields = form.querySelectorAll('[required]');
            
            requiredFields.forEach(field => {
                const formGroup = field.closest('.form-group');
                const errorMessage = formGroup.querySelector('.error-message');
                
                if (!field.value.trim()) {
                    formGroup.classList.add('error');
                    if (!errorMessage) {
                        const error = document.createElement('div');
                        error.className = 'error-message';
                        error.textContent = '此字段为必填项';
                        formGroup.appendChild(error);
                    }
                    isValid = false;
                } else {
                    formGroup.classList.remove('error');
                }
            });
            
            return isValid;
        }
        
        // 清除表单错误
        function clearFormErrors(form) {
            form.querySelectorAll('.form-group.error').forEach(group => {
                group.classList.remove('error');
            });
        }
        
        // 设置按钮加载状态
        function setButtonLoading(button, loading) {
            if (loading) {
                button.disabled = true;
                button.dataset.originalText = button.textContent;
                button.innerHTML = '<span class="loading-spinner"></span>处理中...';
            } else {
                button.disabled = false;
                button.textContent = button.dataset.originalText || button.textContent;
            }
        }
        
        // 显示加载状态
        function showLoading(elementId) {
            const element = document.getElementById(elementId);
            element.style.display = 'block';
            element.className = 'result loading';
            element.innerHTML = '<span class="loading-spinner"></span>正在处理请求...';
        }
        
        // 通用API请求函数
        async function apiRequest(url, options = {}) {
            try {
                const response = await fetch(url, {
                    headers: {
                        'Content-Type': 'application/json',
                        ...options.headers
                    },
                    ...options
                });
                
                const data = await response.json();
                return { success: response.ok, data, status: response.status };
            } catch (error) {
                console.error('API请求错误:', error);
                return { success: false, error: '网络请求失败: ' + error.message };
            }
        }
        
        // 显示结果
        function showResult(elementId, result) {
            const element = document.getElementById(elementId);
            element.style.display = 'block';
            
            if (result.success) {
                element.className = 'result success';
                const message = result.data.message || '操作成功';
                const data = result.data.data || result.data;
                element.textContent = message + '\n\n' + JSON.stringify(data, null, 2);
                showNotification(message, 'success');
            } else {
                element.className = 'result error';
                const errorMsg = result.data?.error || result.error || '操作失败';
                element.textContent = '错误: ' + errorMsg;
                if (result.data && typeof result.data === 'object') {
                    element.textContent += '\n\n' + JSON.stringify(result.data, null, 2);
                }
                showNotification(errorMsg, 'error');
            }
        }
        
        // 用户管理事件
        document.getElementById('createUserForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            // 清除之前的错误
            clearFormErrors(form);
            
            // 验证表单
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            // 设置加载状态
            setButtonLoading(submitButton, true);
            showLoading('usersResult');
            
            const formData = new FormData(form);
            const data = Object.fromEntries(formData);
            
            const result = await apiRequest('/api/users', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            
            // 恢复按钮状态
            setButtonLoading(submitButton, false);
            showResult('usersResult', result);
            
            // 如果成功，清空表单
            if (result.success) {
                form.reset();
            }
        });
        
        document.getElementById('getUserForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('usersResult');
            
            const formData = new FormData(form);
            const username = formData.get('username');
            
            const result = await apiRequest('/api/users/' + username);
            
            setButtonLoading(submitButton, false);
            showResult('usersResult', result);
        });
        
        document.getElementById('deleteUserForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            const formData = new FormData(form);
            const username = formData.get('username');
            
            if (!confirm('确定要删除用户 ' + username + ' 吗？')) {
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('usersResult');
            
            const result = await apiRequest('/api/users/' + username, {
                method: 'DELETE'
            });
            
            setButtonLoading(submitButton, false);
            showResult('usersResult', result);
            
            if (result.success) {
                form.reset();
            }
        });
        
        // 访问密钥事件
        document.getElementById('createKeyForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('keysResult');
            
            const formData = new FormData(form);
            const username = formData.get('username');
            const description = formData.get('description');
            
            const result = await apiRequest('/api/users/' + username + '/access-keys', {
                method: 'POST',
                body: JSON.stringify({ description })
            });
            
            setButtonLoading(submitButton, false);
            showResult('keysResult', result);
            
            if (result.success) {
                form.reset();
            }
        });
        
        document.getElementById('listKeysForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('keysResult');
            
            const formData = new FormData(form);
            const username = formData.get('username');
            
            const result = await apiRequest('/api/users/' + username + '/access-keys');
            
            setButtonLoading(submitButton, false);
            showResult('keysResult', result);
        });
        
        document.getElementById('verifyKeyForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('keysResult');
            
            const formData = new FormData(form);
            const data = Object.fromEntries(formData);
            
            const result = await apiRequest('/api/access-keys/verify', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            
            setButtonLoading(submitButton, false);
            showResult('keysResult', result);
        });
        
        // 权限检查事件
        document.getElementById('checkPermissionForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('permissionsResult');
            
            const formData = new FormData(form);
            const data = Object.fromEntries(formData);
            
            const result = await apiRequest('/api/check-permission', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            
            setButtonLoading(submitButton, false);
            showResult('permissionsResult', result);
        });
        
        // 应用管理事件
        document.getElementById('createAppForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('appsResult');
            
            const formData = new FormData(form);
            const data = Object.fromEntries(formData);
            
            // 处理数组字段
            if (data.callback_urls) {
                data.callback_urls = data.callback_urls.split(',').map(url => url.trim()).filter(url => url);
            }
            if (data.allowed_origins) {
                data.allowed_origins = data.allowed_origins.split(',').map(origin => origin.trim()).filter(origin => origin);
            }
            
            const result = await apiRequest('/api/applications', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            
            setButtonLoading(submitButton, false);
            showResult('appsResult', result);
            
            if (result.success) {
                form.reset();
            }
        });
        
        document.getElementById('getAppForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('appsResult');
            
            const formData = new FormData(form);
            const appId = formData.get('app_id');
            
            const result = await apiRequest('/api/applications/' + appId);
            
            setButtonLoading(submitButton, false);
            showResult('appsResult', result);
        });
        
        document.getElementById('listAppsForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('appsResult');
            
            const formData = new FormData(form);
            const userName = formData.get('user_name');
            const page = formData.get('page') || 1;
            const pageSize = formData.get('page_size') || 10;
            
            const params = new URLSearchParams({
                user_name: userName,
                page: page,
                page_size: pageSize
            });
            
            const result = await apiRequest('/api/applications?' + params.toString());
            
            setButtonLoading(submitButton, false);
            showResult('appsResult', result);
        });
        
        document.getElementById('updateAppForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('appsResult');
            
            const formData = new FormData(form);
            const data = Object.fromEntries(formData);
            const appId = data.app_id;
            delete data.app_id;
            
            const result = await apiRequest('/api/applications/' + appId, {
                method: 'PUT',
                body: JSON.stringify(data)
            });
            
            setButtonLoading(submitButton, false);
            showResult('appsResult', result);
            
            if (result.success) {
                form.reset();
            }
        });
        
        document.getElementById('deleteAppForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            const formData = new FormData(form);
            const appId = formData.get('app_id');
            
            if (!confirm('确定要删除应用 ID ' + appId + ' 吗？')) {
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('appsResult');
            
            const result = await apiRequest('/api/applications/' + appId, {
                method: 'DELETE'
            });
            
            setButtonLoading(submitButton, false);
            showResult('appsResult', result);
            
            if (result.success) {
                form.reset();
            }
        });
        
        // 监控功能事件
        document.getElementById('refreshSystemInfo').addEventListener('click', async () => {
            const button = document.getElementById('refreshSystemInfo');
            setButtonLoading(button, true);
            showLoading('systemInfoResult');
            
            const result = await apiRequest('/api/system-info');
            
            setButtonLoading(button, false);
            showResult('systemInfoResult', result);
        });
        
        document.getElementById('checkHealth').addEventListener('click', async () => {
            const button = document.getElementById('checkHealth');
            setButtonLoading(button, true);
            showLoading('healthResult');
            
            const result = await apiRequest('/api/health');
            
            setButtonLoading(button, false);
            showResult('healthResult', result);
        });
        
        document.getElementById('getMetrics').addEventListener('click', async () => {
            const button = document.getElementById('getMetrics');
            setButtonLoading(button, true);
            showLoading('metricsResult');
            
            try {
                const response = await fetch('/api/metrics');
                const metricsText = await response.text();
                
                setButtonLoading(button, false);
                
                const resultDiv = document.getElementById('metricsResult');
                resultDiv.style.display = 'block';
                resultDiv.innerHTML = '<h4>Prometheus 指标:</h4><pre>' + metricsText + '</pre>';
            } catch (error) {
                setButtonLoading(button, false);
                showResult('metricsResult', { success: false, error: error.message });
            }
        });
        
        document.getElementById('getLogsForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            setButtonLoading(submitButton, true);
            showLoading('logsResult');
            
            const formData = new FormData(form);
            const level = formData.get('level');
            const limit = formData.get('limit');
            
            const params = new URLSearchParams({
                level: level,
                limit: limit
            });
            
            const result = await apiRequest('/api/logs?' + params.toString());
            
            setButtonLoading(submitButton, false);
            showResult('logsResult', result);
        });
        
        // 配置管理相关事件处理
        document.getElementById('updateConfigForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const submitButton = form.querySelector('button[type="submit"]');
            
            clearFormErrors(form);
            if (!validateForm(form)) {
                showNotification('请填写所有必填字段', 'error');
                return;
            }
            
            setButtonLoading(submitButton, true);
            showLoading('updateConfigResult');
            
            const formData = new FormData(form);
            const configKey = formData.get('configKey');
            const configValue = formData.get('configValue');
            
            const result = await apiRequest('/api/config/update', {
                method: 'POST',
                body: JSON.stringify({
                    key: configKey,
                    value: configValue
                })
            });
            
            setButtonLoading(submitButton, false);
            showResult('updateConfigResult', result);
        });
        
        // 加载当前配置
        async function loadCurrentConfig() {
            showLoading('currentConfigResult');
            const result = await apiRequest('/api/config');
            showResult('currentConfigResult', result);
        }
    </script>
</body>
</html>`
}

// 全局变量记录启动时间
var startTime = time.Now()

// 监控相关处理函数

// handleMetrics 处理指标查询
// 功能: 返回Prometheus格式的指标数据
// 参数: c - Gin上下文
// 返回值: 无
func (s *DebugServer) handleMetrics(c *gin.Context) {
	// 这里应该集成实际的metrics收集器
	// 暂时返回模拟数据
	metricsData := `# HELP vgo_iam_requests_total Total number of requests
# TYPE vgo_iam_requests_total counter
vgo_iam_requests_total{method="GET",status="200"} 1234
vgo_iam_requests_total{method="POST",status="200"} 567
vgo_iam_requests_total{method="POST",status="400"} 12

# HELP vgo_iam_request_duration_seconds Request duration in seconds
# TYPE vgo_iam_request_duration_seconds histogram
vgo_iam_request_duration_seconds_bucket{le="0.1"} 100
vgo_iam_request_duration_seconds_bucket{le="0.5"} 200
vgo_iam_request_duration_seconds_bucket{le="1.0"} 250
vgo_iam_request_duration_seconds_bucket{le="+Inf"} 300
vgo_iam_request_duration_seconds_sum 45.6
vgo_iam_request_duration_seconds_count 300
`
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, metricsData)
}

// handleHealth 处理健康检查
// 功能: 返回服务健康状态
// 参数: c - Gin上下文
// 返回值: 无
func (s *DebugServer) handleHealth(c *gin.Context) {
	// 检查数据库连接
	ctx := context.Background()
	_, err := s.iamServer.UserService().GetUser(ctx, "health-check")
	
	health := gin.H{
		"status": "healthy",
		"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
		"version": "1.0.0",
		"database": "connected",
	}
	
	if err != nil && !strings.Contains(err.Error(), "not found") {
		health["status"] = "unhealthy"
		health["database"] = "disconnected"
		health["error"] = err.Error()
		c.JSON(http.StatusServiceUnavailable, health)
		return
	}
	
	c.JSON(http.StatusOK, health)
}

// handleLogs 处理日志查询
// 功能: 返回系统日志信息
// 参数: c - Gin上下文
// 返回值: 无
func (s *DebugServer) handleLogs(c *gin.Context) {
	level := c.DefaultQuery("level", "info")
	limit := c.DefaultQuery("limit", "100")
	
	// 模拟日志数据
	logs := []gin.H{
		{
			"timestamp": "2024-01-20T10:30:00Z",
			"level": "info",
			"message": "User login successful",
			"username": "admin",
			"ip": "192.168.1.100",
		},
		{
			"timestamp": "2024-01-20T10:29:45Z",
			"level": "warn",
			"message": "Failed login attempt",
			"username": "unknown",
			"ip": "192.168.1.200",
		},
		{
			"timestamp": "2024-01-20T10:29:30Z",
			"level": "error",
			"message": "Database connection timeout",
			"error": "connection timeout after 30s",
		},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
		"level": level,
		"limit": limit,
		"total": len(logs),
	})
}

// handleSystemInfo 处理系统信息查询
// 功能: 返回系统运行时信息
// 参数: c - Gin上下文
// 返回值: 无
func (s *DebugServer) handleSystemInfo(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	systemInfo := gin.H{
		"go_version": runtime.Version(),
		"go_os": runtime.GOOS,
		"go_arch": runtime.GOARCH,
		"cpu_count": runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
		"memory": gin.H{
			"alloc_mb": bToMb(m.Alloc),
			"total_alloc_mb": bToMb(m.TotalAlloc),
			"sys_mb": bToMb(m.Sys),
			"gc_runs": m.NumGC,
		},
		"uptime_seconds": time.Since(startTime).Seconds(),
	}
	
	c.JSON(http.StatusOK, systemInfo)
}

// bToMb 字节转换为MB
// 功能: 将字节数转换为MB
// 参数: b - 字节数
// 返回值: MB数
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// handleGetConfig 处理获取配置请求
// 功能: 获取当前应用配置信息
// 参数: c - Gin上下文
// 返回值: 无
func (s *DebugServer) handleGetConfig(c *gin.Context) {
	// 获取当前配置
	cfg := config.LodIAMConfig()
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法加载配置",
		})
		return
	}
	
	// 返回配置信息（隐藏敏感信息）
	configData := map[string]interface{}{
		"grpc": map[string]interface{}{
			"port": cfg.GRPC.Port,
		},
		"database": map[string]interface{}{
			"dsn": "[HIDDEN]", // 隐藏数据库连接字符串
		},
		"log": map[string]interface{}{
			"level":     cfg.Log.Level,
			"format":    cfg.Log.Format,
			"directory": cfg.Log.Directory,
			"filename":  cfg.Log.Filename,
			"to_stdout": cfg.Log.ToStdout,
		},
		"sentry": map[string]interface{}{
			"enabled":     cfg.Sentry.Enabled,
			"dsn":         "[HIDDEN]", // 隐藏Sentry DSN
			"environment": cfg.Sentry.Environment,
		},
		"ratelimit": map[string]interface{}{
			"enabled":    cfg.RateLimit.Enabled,
			"type":       cfg.RateLimit.Type,
			"limit":      cfg.RateLimit.Limit,
			"window":     cfg.RateLimit.Window.String(),
			"prefix":     cfg.RateLimit.Prefix,
			"redis_addr": cfg.RateLimit.RedisAddr,
			"redis_db":   cfg.RateLimit.RedisDB,
		},
		"middleware": map[string]interface{}{
			"ignore":     cfg.Middleware.Ignore,
			"master_key": "[HIDDEN]", // 隐藏主密钥
		},
		"sts": map[string]interface{}{
			"default_duration":          cfg.STS.DefaultDuration.String(),
			"max_duration":              cfg.STS.MaxDuration.String(),
			"min_duration":              cfg.STS.MinDuration.String(),
			"cleanup_interval":          cfg.STS.CleanupInterval.String(),
			"auto_cleanup":              cfg.STS.AutoCleanup,
			"max_credentials_per_user":  cfg.STS.MaxCredentialsPerUser,
		},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    configData,
	})
}

// handleUpdateConfig 处理更新配置请求
// 功能: 更新指定的配置项（注意：这是演示功能，实际生产环境中配置更新需要更复杂的处理）
// 参数: c - Gin上下文
// 返回值: 无
func (s *DebugServer) handleUpdateConfig(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}
	
	// 注意：这里只是演示功能，实际生产环境中需要：
	// 1. 验证配置项的有效性
	// 2. 持久化配置更改
	// 3. 重新加载配置或重启相关服务
	// 4. 记录配置更改日志
	
	// 模拟配置更新结果
	result := map[string]interface{}{
		"key":       req.Key,
		"old_value": "[PREVIOUS_VALUE]",
		"new_value": req.Value,
		"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		"note":      "配置更新成功（演示模式，实际未持久化）",
	}
	
	vgokit.Log.Info("配置更新请求", 
		zap.String("key", req.Key),
		zap.String("value", req.Value),
	)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置更新成功",
		"data":    result,
	})
}