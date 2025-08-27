package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/vera-byte/vgo-iam/internal/model"
)

// DeveloperVerificationStore 开发者认证存储接口
type DeveloperVerificationStore interface {
	Create(ctx context.Context, verification *model.DeveloperVerification) error
	GetByUserIDAndType(ctx context.Context, userID int64, developerType model.DeveloperType) (*model.DeveloperVerification, error)
	GetByID(ctx context.Context, id int64) (*model.DeveloperVerification, error)
	List(ctx context.Context, status model.VerificationStatus, page, pageSize int) ([]*model.DeveloperVerification, int, error)
	Update(ctx context.Context, verification *model.DeveloperVerification) error
	UpdateStatus(ctx context.Context, id int64, status model.VerificationStatus, reviewerID int64, comment string) error
}

type developerVerificationStore struct {
	db *sql.DB
}

// NewDeveloperVerificationStore 创建开发者认证存储实例
func NewDeveloperVerificationStore(db *sql.DB) DeveloperVerificationStore {
	return &developerVerificationStore{db: db}
}

// Create 创建开发者认证记录
func (s *developerVerificationStore) Create(ctx context.Context, verification *model.DeveloperVerification) error {
	query := `
		INSERT INTO developer_verifications (
			user_id, developer_type, status, real_name, id_card_number, 
			id_card_front_url, id_card_back_url, company_name, business_license_number,
			business_license_url, legal_representative, company_address, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
		verification.UserID,
		verification.DeveloperType,
		verification.Status,
		verification.RealName,
		verification.IDCardNumber,
		verification.IDCardFrontURL,
		verification.IDCardBackURL,
		verification.CompanyName,
		verification.BusinessLicenseNumber,
		verification.BusinessLicenseURL,
		verification.LegalRepresentative,
		verification.CompanyAddress,
		verification.CreatedAt,
		verification.UpdatedAt,
	).Scan(&verification.ID)

	return err
}

// GetByUserIDAndType 根据用户ID和开发者类型获取认证记录
func (s *developerVerificationStore) GetByUserIDAndType(ctx context.Context, userID int64, developerType model.DeveloperType) (*model.DeveloperVerification, error) {
	query := `
		SELECT 
			dv.id, dv.user_id, dv.developer_type, dv.status,
			dv.real_name, dv.id_card_number, dv.id_card_front_url, dv.id_card_back_url,
			dv.company_name, dv.business_license_number, dv.business_license_url,
			dv.legal_representative, dv.company_address,
			dv.reviewer_id, dv.review_comment, dv.reviewed_at,
			dv.created_at, dv.updated_at,
			u.name as user_name,
			r.name as reviewer_name
		FROM developer_verifications dv
		JOIN users u ON dv.user_id = u.id
		LEFT JOIN users r ON dv.reviewer_id = r.id
		WHERE dv.user_id = $1 AND dv.developer_type = $2
	`

	verification := &model.DeveloperVerification{}
	var reviewerName sql.NullString
	err := s.db.QueryRowContext(ctx, query, userID, developerType).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.DeveloperType,
		&verification.Status,
		&verification.RealName,
		&verification.IDCardNumber,
		&verification.IDCardFrontURL,
		&verification.IDCardBackURL,
		&verification.CompanyName,
		&verification.BusinessLicenseNumber,
		&verification.BusinessLicenseURL,
		&verification.LegalRepresentative,
		&verification.CompanyAddress,
		&verification.ReviewerID,
		&verification.ReviewComment,
		&verification.ReviewedAt,
		&verification.CreatedAt,
		&verification.UpdatedAt,
		&verification.UserName,
		&reviewerName,
	)

	if reviewerName.Valid {
		verification.ReviewerName = reviewerName.String
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return verification, err
}

// GetByID 根据ID获取认证记录
func (s *developerVerificationStore) GetByID(ctx context.Context, id int64) (*model.DeveloperVerification, error) {
	query := `
		SELECT 
			dv.id, dv.user_id, dv.developer_type, dv.status,
			dv.real_name, dv.id_card_number, dv.id_card_front_url, dv.id_card_back_url,
			dv.company_name, dv.business_license_number, dv.business_license_url,
			dv.legal_representative, dv.company_address,
			dv.reviewer_id, dv.review_comment, dv.reviewed_at,
			dv.created_at, dv.updated_at,
			u.name as user_name,
			r.name as reviewer_name
		FROM developer_verifications dv
		JOIN users u ON dv.user_id = u.id
		LEFT JOIN users r ON dv.reviewer_id = r.id
		WHERE dv.id = $1
	`

	verification := &model.DeveloperVerification{}
	var reviewerName sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.DeveloperType,
		&verification.Status,
		&verification.RealName,
		&verification.IDCardNumber,
		&verification.IDCardFrontURL,
		&verification.IDCardBackURL,
		&verification.CompanyName,
		&verification.BusinessLicenseNumber,
		&verification.BusinessLicenseURL,
		&verification.LegalRepresentative,
		&verification.CompanyAddress,
		&verification.ReviewerID,
		&verification.ReviewComment,
		&verification.ReviewedAt,
		&verification.CreatedAt,
		&verification.UpdatedAt,
		&verification.UserName,
		&reviewerName,
	)

	if reviewerName.Valid {
		verification.ReviewerName = reviewerName.String
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return verification, err
}

// List 获取认证记录列表
func (s *developerVerificationStore) List(ctx context.Context, status model.VerificationStatus, page, pageSize int) ([]*model.DeveloperVerification, int, error) {
	// 构建查询条件
	var conditions []string
	var args []interface{}
	argIndex := 1

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("dv.status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 获取总数
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM developer_verifications dv
		JOIN users u ON dv.user_id = u.id
		%s
	`, whereClause)

	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	query := fmt.Sprintf(`
		SELECT 
			dv.id, dv.user_id, dv.developer_type, dv.status,
			dv.real_name, dv.id_card_number, dv.id_card_front_url, dv.id_card_back_url,
			dv.company_name, dv.business_license_number, dv.business_license_url,
			dv.legal_representative, dv.company_address,
			dv.reviewer_id, dv.review_comment, dv.reviewed_at,
			dv.created_at, dv.updated_at,
			u.name as user_name,
			r.name as reviewer_name
		FROM developer_verifications dv
		JOIN users u ON dv.user_id = u.id
		LEFT JOIN users r ON dv.reviewer_id = r.id
		%s
		ORDER BY dv.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var verifications []*model.DeveloperVerification
	for rows.Next() {
		verification := &model.DeveloperVerification{}
		var reviewerName sql.NullString
		err := rows.Scan(
			&verification.ID,
			&verification.UserID,
			&verification.DeveloperType,
			&verification.Status,
			&verification.RealName,
			&verification.IDCardNumber,
			&verification.IDCardFrontURL,
			&verification.IDCardBackURL,
			&verification.CompanyName,
			&verification.BusinessLicenseNumber,
			&verification.BusinessLicenseURL,
			&verification.LegalRepresentative,
			&verification.CompanyAddress,
			&verification.ReviewerID,
			&verification.ReviewComment,
			&verification.ReviewedAt,
			&verification.CreatedAt,
			&verification.UpdatedAt,
			&verification.UserName,
			&reviewerName,
		)
		if err != nil {
			return nil, 0, err
		}
		if reviewerName.Valid {
			verification.ReviewerName = reviewerName.String
		}
		verifications = append(verifications, verification)
	}

	return verifications, total, rows.Err()
}

// Update 更新认证记录
func (s *developerVerificationStore) Update(ctx context.Context, verification *model.DeveloperVerification) error {
	query := `
		UPDATE developer_verifications SET
			status = $2,
			real_name = $3,
			id_card_number = $4,
			id_card_front_url = $5,
			id_card_back_url = $6,
			company_name = $7,
			business_license_number = $8,
			business_license_url = $9,
			legal_representative = $10,
			company_address = $11,
			reviewer_id = $12,
			review_comment = $13,
			reviewed_at = $14,
			updated_at = $15
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		verification.ID,
		verification.Status,
		verification.RealName,
		verification.IDCardNumber,
		verification.IDCardFrontURL,
		verification.IDCardBackURL,
		verification.CompanyName,
		verification.BusinessLicenseNumber,
		verification.BusinessLicenseURL,
		verification.LegalRepresentative,
		verification.CompanyAddress,
		verification.ReviewerID,
		verification.ReviewComment,
		verification.ReviewedAt,
		verification.UpdatedAt,
	)

	return err
}

// UpdateStatus 更新认证状态
func (s *developerVerificationStore) UpdateStatus(ctx context.Context, id int64, status model.VerificationStatus, reviewerID int64, comment string) error {
	query := `
		UPDATE developer_verifications SET
			status = $2,
			reviewer_id = $3,
			review_comment = $4,
			reviewed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, id, status, reviewerID, comment)
	return err
}