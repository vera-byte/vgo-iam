package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/vera-byte/vgo-iam/internal/api"
	"github.com/vera-byte/vgo-iam/internal/bootstrap"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/service"
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

	c.JSON(http.StatusNotImplemented, gin.H{"message": "用户更新功能暂未实现"})
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

	c.JSON(http.StatusNotImplemented, gin.H{"message": "用户删除功能暂未实现"})
}

// handleListUsers 列出用户
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

	c.JSON(http.StatusNotImplemented, gin.H{"message": "用户列表功能暂未实现"})
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
func (s *DebugServer) handleDeleteAccessKey(c *gin.Context) {
	accessKeyId := c.Param("accessKeyId")
	if accessKeyId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "访问密钥ID不能为空"})
		return
	}

	if len(accessKeyId) < 10 || len(accessKeyId) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "访问密钥ID格式无效"})
		return
	}

	c.JSON(http.StatusNotImplemented, gin.H{"message": "访问密钥删除功能暂未实现"})
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

	// 验证操作
	validActions := map[string]bool{
		"read":   true,
		"write":  true,
		"delete": true,
		"admin":  true,
	}
	if !validActions[req.Action] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "操作类型无效，支持的值：read, write, delete, admin"})
		return
	}

	// 验证资源
	if len(req.Resource) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "资源路径长度不能超过200个字符"})
		return
	}

	c.JSON(http.StatusNotImplemented, gin.H{"message": "权限检查功能暂未实现"})
}

// 其他API处理函数...
func (s *DebugServer) handleCreatePolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (s *DebugServer) handleGetPolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (s *DebugServer) handleUpdatePolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (s *DebugServer) handleDeletePolicy(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (s *DebugServer) handleListPolicies(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
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

func (s *DebugServer) handleSubmitDeveloperVerification(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
}

func (s *DebugServer) handleGetDeveloperVerification(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not implemented yet"})
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
                return { success: false, error: error.message };
            }
        }
        
        // 显示结果
        function showResult(elementId, result) {
            const element = document.getElementById(elementId);
            element.style.display = 'block';
            
            if (result.success) {
                element.className = 'result success';
                element.textContent = JSON.stringify(result.data, null, 2);
            } else {
                element.className = 'result error';
                element.textContent = result.error || JSON.stringify(result.data, null, 2);
            }
        }
        
        // 用户管理事件
        document.getElementById('createUserForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = Object.fromEntries(formData);
            
            const result = await apiRequest('/api/users', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            
            showResult('usersResult', result);
        });
        
        document.getElementById('getUserForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const username = formData.get('username');
            
            const result = await apiRequest('/api/users/' + username);
            showResult('usersResult', result);
        });
        
        document.getElementById('deleteUserForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const username = formData.get('username');
            
            if (!confirm('确定要删除用户 ' + username + ' 吗？')) {
                return;
            }
            
            const result = await apiRequest('/api/users/' + username, {
                method: 'DELETE'
            });
            
            showResult('usersResult', result);
        });
        
        // 访问密钥事件
        document.getElementById('createKeyForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const username = formData.get('username');
            const description = formData.get('description');
            
            const result = await apiRequest('/api/users/' + username + '/access-keys', {
                method: 'POST',
                body: JSON.stringify({ description })
            });
            
            showResult('keysResult', result);
        });
        
        document.getElementById('listKeysForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const username = formData.get('username');
            
            const result = await apiRequest('/api/users/' + username + '/access-keys');
            showResult('keysResult', result);
        });
        
        document.getElementById('verifyKeyForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = Object.fromEntries(formData);
            
            const result = await apiRequest('/api/access-keys/verify', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            
            showResult('keysResult', result);
        });
        
        // 权限检查事件
        document.getElementById('checkPermissionForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = Object.fromEntries(formData);
            
            const result = await apiRequest('/api/check-permission', {
                method: 'POST',
                body: JSON.stringify(data)
            });
            
            showResult('permissionsResult', result);
        });
        
        // 应用管理事件
        document.getElementById('createAppForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
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
            
            showResult('appsResult', result);
        });
        
        document.getElementById('getAppForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const appId = formData.get('app_id');
            
            const result = await apiRequest('/api/applications/' + appId);
            showResult('appsResult', result);
        });
        
        document.getElementById('listAppsForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const userName = formData.get('user_name');
            const page = formData.get('page') || 1;
            const pageSize = formData.get('page_size') || 10;
            
            const params = new URLSearchParams({
                user_name: userName,
                page: page,
                page_size: pageSize
            });
            
            const result = await apiRequest('/api/applications?' + params.toString());
            showResult('appsResult', result);
        });
        
        document.getElementById('updateAppForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = Object.fromEntries(formData);
            const appId = data.app_id;
            delete data.app_id;
            
            const result = await apiRequest('/api/applications/' + appId, {
                method: 'PUT',
                body: JSON.stringify(data)
            });
            
            showResult('appsResult', result);
        });
        
        document.getElementById('deleteAppForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const appId = formData.get('app_id');
            
            if (!confirm('确定要删除应用 ID ' + appId + ' 吗？')) {
                return;
            }
            
            const result = await apiRequest('/api/applications/' + appId, {
                method: 'DELETE'
            });
            
            showResult('appsResult', result);
        });
    </script>
</body>
</html>`
}