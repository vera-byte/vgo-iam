package service

import (
	"context"
	"fmt"

	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/store"
)

// DeveloperVerificationService 开发者认证服务接口
type DeveloperVerificationService interface {
	SubmitVerification(ctx context.Context, req *SubmitVerificationRequest) (*model.DeveloperVerification, error)
	GetVerification(ctx context.Context, userID int64, developerType model.DeveloperType) (*model.DeveloperVerification, error)
	ListVerifications(ctx context.Context, status model.VerificationStatus, page, pageSize int) ([]*model.DeveloperVerification, int, error)
	ReviewVerification(ctx context.Context, req *ReviewVerificationRequest) error
	CheckVerificationStatus(ctx context.Context, userID int64, developerType model.DeveloperType) (bool, error)
}

// SubmitVerificationRequest 提交认证请求
type SubmitVerificationRequest struct {
	UserID                  int64                   `json:"user_id"`
	DeveloperType           model.DeveloperType     `json:"developer_type"`
	RealName                *string                 `json:"real_name,omitempty"`
	IDCardNumber            *string                 `json:"id_card_number,omitempty"`
	IDCardFrontURL          *string                 `json:"id_card_front_url,omitempty"`
	IDCardBackURL           *string                 `json:"id_card_back_url,omitempty"`
	CompanyName             *string                 `json:"company_name,omitempty"`
	BusinessLicenseNumber   *string                 `json:"business_license_number,omitempty"`
	BusinessLicenseURL      *string                 `json:"business_license_url,omitempty"`
	LegalRepresentative     *string                 `json:"legal_representative,omitempty"`
	CompanyAddress          *string                 `json:"company_address,omitempty"`
}

// ReviewVerificationRequest 审核认证请求
type ReviewVerificationRequest struct {
	VerificationID int64                      `json:"verification_id"`
	ReviewerID     int64                      `json:"reviewer_id"`
	Status         model.VerificationStatus   `json:"status"`
	Comment        string                     `json:"comment"`
}

type developerVerificationService struct {
	verificationStore store.DeveloperVerificationStore
	userStore         store.UserStore
}

// NewDeveloperVerificationService 创建开发者认证服务实例
func NewDeveloperVerificationService(verificationStore store.DeveloperVerificationStore, userStore store.UserStore) DeveloperVerificationService {
	return &developerVerificationService{
		verificationStore: verificationStore,
		userStore:         userStore,
	}
}

// SubmitVerification 提交开发者认证
func (s *developerVerificationService) SubmitVerification(ctx context.Context, req *SubmitVerificationRequest) (*model.DeveloperVerification, error) {
	// 检查用户是否存在
	user, err := s.userStore.GetByID(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 检查是否已有相同类型的认证记录
	existingVerification, err := s.verificationStore.GetByUserIDAndType(ctx, req.UserID, req.DeveloperType)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing verification: %w", err)
	}
	if existingVerification != nil {
		return nil, fmt.Errorf("verification already exists for this developer type")
	}

	// 验证请求参数
	if err := s.validateSubmitRequest(req); err != nil {
		return nil, err
	}

	// 创建认证记录
	verification := model.NewDeveloperVerification(req.UserID, req.DeveloperType)

	// 设置个人开发者信息
	if req.DeveloperType == model.DeveloperTypeIndividual {
		verification.SetIndividualInfo(*req.RealName, *req.IDCardNumber, *req.IDCardFrontURL, *req.IDCardBackURL)
	}

	// 设置企业开发者信息
	if req.DeveloperType == model.DeveloperTypeEnterprise {
		verification.SetEnterpriseInfo(*req.CompanyName, *req.BusinessLicenseNumber, *req.BusinessLicenseURL, *req.LegalRepresentative, *req.CompanyAddress)
	}

	// 保存到数据库
	err = s.verificationStore.Create(ctx, verification)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification: %w", err)
	}

	return verification, nil
}

// GetVerification 获取开发者认证信息
func (s *developerVerificationService) GetVerification(ctx context.Context, userID int64, developerType model.DeveloperType) (*model.DeveloperVerification, error) {
	verification, err := s.verificationStore.GetByUserIDAndType(ctx, userID, developerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	return verification, nil
}

// ListVerifications 获取认证列表
func (s *developerVerificationService) ListVerifications(ctx context.Context, status model.VerificationStatus, page, pageSize int) ([]*model.DeveloperVerification, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	verifications, total, err := s.verificationStore.List(ctx, status, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list verifications: %w", err)
	}

	return verifications, total, nil
}

// ReviewVerification 审核开发者认证
func (s *developerVerificationService) ReviewVerification(ctx context.Context, req *ReviewVerificationRequest) error {
	// 检查认证记录是否存在
	verification, err := s.verificationStore.GetByID(ctx, req.VerificationID)
	if err != nil {
		return fmt.Errorf("failed to get verification: %w", err)
	}
	if verification == nil {
		return fmt.Errorf("verification not found")
	}

	// 检查当前状态是否可以审核
	if !verification.IsPending() {
		return fmt.Errorf("verification is not in pending status")
	}

	// 检查审核员是否存在
	reviewer, err := s.userStore.GetByID(req.ReviewerID)
	if err != nil {
		return fmt.Errorf("failed to get reviewer: %w", err)
	}
	if reviewer == nil {
		return fmt.Errorf("reviewer not found")
	}

	// 验证审核状态
	if req.Status != model.VerificationStatusApproved && req.Status != model.VerificationStatusRejected {
		return fmt.Errorf("invalid review status")
	}

	// 更新认证状态
	err = s.verificationStore.UpdateStatus(ctx, req.VerificationID, req.Status, req.ReviewerID, req.Comment)
	if err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}

	return nil
}

// CheckVerificationStatus 检查开发者认证状态
func (s *developerVerificationService) CheckVerificationStatus(ctx context.Context, userID int64, developerType model.DeveloperType) (bool, error) {
	verification, err := s.verificationStore.GetByUserIDAndType(ctx, userID, developerType)
	if err != nil {
		return false, fmt.Errorf("failed to get verification: %w", err)
	}

	if verification == nil {
		return false, nil
	}

	return verification.IsApproved(), nil
}

// validateSubmitRequest 验证提交认证请求
func (s *developerVerificationService) validateSubmitRequest(req *SubmitVerificationRequest) error {
	if req.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	if req.DeveloperType != model.DeveloperTypeIndividual && req.DeveloperType != model.DeveloperTypeEnterprise {
		return fmt.Errorf("invalid developer type")
	}

	// 验证个人开发者信息
	if req.DeveloperType == model.DeveloperTypeIndividual {
		if req.RealName == nil || *req.RealName == "" {
			return fmt.Errorf("real name is required for individual developer")
		}
		if req.IDCardNumber == nil || *req.IDCardNumber == "" {
			return fmt.Errorf("ID card number is required for individual developer")
		}
		if req.IDCardFrontURL == nil || *req.IDCardFrontURL == "" {
			return fmt.Errorf("ID card front URL is required for individual developer")
		}
		if req.IDCardBackURL == nil || *req.IDCardBackURL == "" {
			return fmt.Errorf("ID card back URL is required for individual developer")
		}
	}

	// 验证企业开发者信息
	if req.DeveloperType == model.DeveloperTypeEnterprise {
		if req.CompanyName == nil || *req.CompanyName == "" {
			return fmt.Errorf("company name is required for enterprise developer")
		}
		if req.BusinessLicenseNumber == nil || *req.BusinessLicenseNumber == "" {
			return fmt.Errorf("business license number is required for enterprise developer")
		}
		if req.BusinessLicenseURL == nil || *req.BusinessLicenseURL == "" {
			return fmt.Errorf("business license URL is required for enterprise developer")
		}
		if req.LegalRepresentative == nil || *req.LegalRepresentative == "" {
			return fmt.Errorf("legal representative is required for enterprise developer")
		}
		if req.CompanyAddress == nil || *req.CompanyAddress == "" {
			return fmt.Errorf("company address is required for enterprise developer")
		}
	}

	return nil
}