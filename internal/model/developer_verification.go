package model

import (
	"time"
)

// DeveloperType 开发者类型
type DeveloperType string

const (
	DeveloperTypeIndividual DeveloperType = "individual" // 个人开发者
	DeveloperTypeEnterprise DeveloperType = "enterprise" // 企业开发者
)

// VerificationStatus 认证状态
type VerificationStatus string

const (
	VerificationStatusPending  VerificationStatus = "pending"  // 待审核
	VerificationStatusApproved VerificationStatus = "approved" // 已通过
	VerificationStatusRejected VerificationStatus = "rejected" // 已拒绝
)

// DeveloperVerification 开发者认证信息
type DeveloperVerification struct {
	ID             int64              `json:"id" db:"id"`
	UserID         int64              `json:"user_id" db:"user_id"`
	DeveloperType  DeveloperType      `json:"developer_type" db:"developer_type"`
	Status         VerificationStatus `json:"status" db:"status"`

	// 个人开发者信息
	RealName        *string `json:"real_name,omitempty" db:"real_name"`
	IDCardNumber    *string `json:"id_card_number,omitempty" db:"id_card_number"`
	IDCardFrontURL  *string `json:"id_card_front_url,omitempty" db:"id_card_front_url"`
	IDCardBackURL   *string `json:"id_card_back_url,omitempty" db:"id_card_back_url"`

	// 企业开发者信息
	CompanyName            *string `json:"company_name,omitempty" db:"company_name"`
	BusinessLicenseNumber  *string `json:"business_license_number,omitempty" db:"business_license_number"`
	BusinessLicenseURL     *string `json:"business_license_url,omitempty" db:"business_license_url"`
	LegalRepresentative    *string `json:"legal_representative,omitempty" db:"legal_representative"`
	CompanyAddress         *string `json:"company_address,omitempty" db:"company_address"`

	// 审核信息
	ReviewerID    *int64     `json:"reviewer_id,omitempty" db:"reviewer_id"`
	ReviewComment *string    `json:"review_comment,omitempty" db:"review_comment"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// 关联信息
	UserName     string `json:"user_name,omitempty" db:"user_name"`
	ReviewerName string `json:"reviewer_name,omitempty" db:"reviewer_name"`
}

// NewDeveloperVerification 创建新的开发者认证记录
func NewDeveloperVerification(userID int64, developerType DeveloperType) *DeveloperVerification {
	now := time.Now()
	return &DeveloperVerification{
		UserID:        userID,
		DeveloperType: developerType,
		Status:        VerificationStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// IsIndividual 是否为个人开发者
func (dv *DeveloperVerification) IsIndividual() bool {
	return dv.DeveloperType == DeveloperTypeIndividual
}

// IsEnterprise 是否为企业开发者
func (dv *DeveloperVerification) IsEnterprise() bool {
	return dv.DeveloperType == DeveloperTypeEnterprise
}

// IsApproved 是否已通过审核
func (dv *DeveloperVerification) IsApproved() bool {
	return dv.Status == VerificationStatusApproved
}

// IsPending 是否待审核
func (dv *DeveloperVerification) IsPending() bool {
	return dv.Status == VerificationStatusPending
}

// IsRejected 是否已拒绝
func (dv *DeveloperVerification) IsRejected() bool {
	return dv.Status == VerificationStatusRejected
}

// SetIndividualInfo 设置个人开发者信息
func (dv *DeveloperVerification) SetIndividualInfo(realName, idCardNumber, idCardFrontURL, idCardBackURL string) {
	dv.RealName = &realName
	dv.IDCardNumber = &idCardNumber
	dv.IDCardFrontURL = &idCardFrontURL
	dv.IDCardBackURL = &idCardBackURL
}

// SetEnterpriseInfo 设置企业开发者信息
func (dv *DeveloperVerification) SetEnterpriseInfo(companyName, businessLicenseNumber, businessLicenseURL, legalRepresentative, companyAddress string) {
	dv.CompanyName = &companyName
	dv.BusinessLicenseNumber = &businessLicenseNumber
	dv.BusinessLicenseURL = &businessLicenseURL
	dv.LegalRepresentative = &legalRepresentative
	dv.CompanyAddress = &companyAddress
}

// SetReviewInfo 设置审核信息
func (dv *DeveloperVerification) SetReviewInfo(reviewerID int64, status VerificationStatus, comment string) {
	now := time.Now()
	dv.ReviewerID = &reviewerID
	dv.Status = status
	dv.ReviewComment = &comment
	dv.ReviewedAt = &now
	dv.UpdatedAt = now
}