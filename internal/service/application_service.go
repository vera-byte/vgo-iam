package service

import (
	"context"
	"fmt"

	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/store"
)

// ApplicationService 应用管理服务接口
type ApplicationService interface {
	CreateApplication(ctx context.Context, req *CreateApplicationRequest) (*model.Application, error)
	GetApplication(ctx context.Context, id int64) (*model.Application, error)
	ListApplications(ctx context.Context, userID int64, status model.AppStatus, page, pageSize int) ([]*model.Application, int, error)
	UpdateApplication(ctx context.Context, req *UpdateApplicationRequest) error
	DeleteApplication(ctx context.Context, id int64, userID int64) error
	CheckApplicationOwnership(ctx context.Context, appID, userID int64) (bool, error)
}

// CreateApplicationRequest 创建应用请求
type CreateApplicationRequest struct {
	UserID         int64    `json:"user_id"`
	AppName        string   `json:"app_name"`
	AppDescription string   `json:"app_description"`
	AppType        string   `json:"app_type"`
	CallbackURLs   []string `json:"callback_urls,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
}

// UpdateApplicationRequest 更新应用请求
type UpdateApplicationRequest struct {
	ID             int64    `json:"id"`
	UserID         int64    `json:"user_id"`
	AppName        string   `json:"app_name"`
	AppDescription string   `json:"app_description"`
	AppType        string   `json:"app_type"`
	CallbackURLs   []string `json:"callback_urls,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
}

type applicationService struct {
	appStore                     store.ApplicationStore
	userStore                    store.UserStore
	developerVerificationService DeveloperVerificationService
}

// NewApplicationService 创建应用管理服务实例
func NewApplicationService(appStore store.ApplicationStore, userStore store.UserStore, developerVerificationService DeveloperVerificationService) ApplicationService {
	return &applicationService{
		appStore:                     appStore,
		userStore:                    userStore,
		developerVerificationService: developerVerificationService,
	}
}

// CreateApplication 创建应用
func (s *applicationService) CreateApplication(ctx context.Context, req *CreateApplicationRequest) (*model.Application, error) {
	// 检查用户是否存在
	user, err := s.userStore.GetByID(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 检查用户是否已通过开发者认证
	// 个人开发者需要通过个人认证，企业开发者需要通过企业认证
	individualVerified, err := s.developerVerificationService.CheckVerificationStatus(ctx, req.UserID, model.DeveloperTypeIndividual)
	if err != nil {
		return nil, fmt.Errorf("failed to check individual verification: %w", err)
	}

	enterpriseVerified, err := s.developerVerificationService.CheckVerificationStatus(ctx, req.UserID, model.DeveloperTypeEnterprise)
	if err != nil {
		return nil, fmt.Errorf("failed to check enterprise verification: %w", err)
	}

	if !individualVerified && !enterpriseVerified {
		return nil, fmt.Errorf("user must pass developer verification before creating applications")
	}

	// 验证请求参数
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// 检查应用名称是否已存在
	existingApp, err := s.appStore.GetByUserIDAndName(ctx, req.UserID, req.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing application: %w", err)
	}
	if existingApp != nil {
		return nil, fmt.Errorf("application name already exists")
	}

	// 创建应用
	appType := model.AppType(req.AppType)
	app := model.NewApplication(req.UserID, req.AppName, req.AppDescription, appType)
	app.CallbackURLs = model.StringArray(req.CallbackURLs)
	app.AllowedOrigins = model.StringArray(req.AllowedOrigins)

	// 保存到数据库
	err = s.appStore.Create(ctx, app)
	if err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	return app, nil
}

// GetApplication 获取应用信息
func (s *applicationService) GetApplication(ctx context.Context, id int64) (*model.Application, error) {
	app, err := s.appStore.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	return app, nil
}

// ListApplications 获取应用列表
func (s *applicationService) ListApplications(ctx context.Context, userID int64, status model.AppStatus, page, pageSize int) ([]*model.Application, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	apps, total, err := s.appStore.List(ctx, userID, status, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list applications: %w", err)
	}

	return apps, total, nil
}

// UpdateApplication 更新应用信息
func (s *applicationService) UpdateApplication(ctx context.Context, req *UpdateApplicationRequest) error {
	// 检查应用是否存在
	app, err := s.appStore.GetByID(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}
	if app == nil {
		return fmt.Errorf("application not found")
	}

	// 检查用户是否有权限修改该应用
	if app.UserID != req.UserID {
		return fmt.Errorf("permission denied: user does not own this application")
	}

	// 验证请求参数
	if err := s.validateUpdateRequest(req); err != nil {
		return err
	}

	// 如果修改了应用名称，检查新名称是否已存在
	if app.AppName != req.AppName {
		existingApp, err := s.appStore.GetByUserIDAndName(ctx, req.UserID, req.AppName)
		if err != nil {
			return fmt.Errorf("failed to check existing application: %w", err)
		}
		if existingApp != nil {
			return fmt.Errorf("application name already exists")
		}
	}

	// 更新应用信息
	appType := model.AppType(req.AppType)
	app.UpdateInfo(req.AppName, req.AppDescription, "", "", appType, req.CallbackURLs, req.AllowedOrigins)

	// 保存到数据库
	err = s.appStore.Update(ctx, app)
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}

	return nil
}

// DeleteApplication 删除应用
func (s *applicationService) DeleteApplication(ctx context.Context, id int64, userID int64) error {
	// 检查应用是否存在
	app, err := s.appStore.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}
	if app == nil {
		return fmt.Errorf("application not found")
	}

	// 检查用户是否有权限删除该应用
	if app.UserID != userID {
		return fmt.Errorf("permission denied: user does not own this application")
	}

	// 删除应用
	err = s.appStore.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	return nil
}

// CheckApplicationOwnership 检查应用所有权
func (s *applicationService) CheckApplicationOwnership(ctx context.Context, appID, userID int64) (bool, error) {
	app, err := s.appStore.GetByID(ctx, appID)
	if err != nil {
		return false, fmt.Errorf("failed to get application: %w", err)
	}
	if app == nil {
		return false, nil
	}

	return app.UserID == userID, nil
}

// validateCreateRequest 验证创建应用请求
func (s *applicationService) validateCreateRequest(req *CreateApplicationRequest) error {
	if req.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if req.AppName == "" {
		return fmt.Errorf("application name is required")
	}

	if len(req.AppName) > 100 {
		return fmt.Errorf("application name too long (max 100 characters)")
	}

	if len(req.AppDescription) > 500 {
		return fmt.Errorf("application description too long (max 500 characters)")
	}

	// 验证应用类型
	appType := model.AppType(req.AppType)
	if appType != model.AppTypeWeb && appType != model.AppTypeMobile && 
	   appType != model.AppTypeDesktop && appType != model.AppTypeAPI {
		return fmt.Errorf("invalid application type")
	}

	return nil
}

// validateUpdateRequest 验证更新应用请求
func (s *applicationService) validateUpdateRequest(req *UpdateApplicationRequest) error {
	if req.ID <= 0 {
		return fmt.Errorf("invalid application ID")
	}

	if req.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if req.AppName == "" {
		return fmt.Errorf("application name is required")
	}

	if len(req.AppName) > 100 {
		return fmt.Errorf("application name too long (max 100 characters)")
	}

	if len(req.AppDescription) > 500 {
		return fmt.Errorf("application description too long (max 500 characters)")
	}

	// 验证应用类型
	appType := model.AppType(req.AppType)
	if appType != model.AppTypeWeb && appType != model.AppTypeMobile && 
	   appType != model.AppTypeDesktop && appType != model.AppTypeAPI {
		return fmt.Errorf("invalid application type")
	}

	return nil
}