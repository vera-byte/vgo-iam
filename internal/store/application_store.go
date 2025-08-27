package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/vera-byte/vgo-iam/internal/model"
)

// ApplicationStore 应用存储接口
type ApplicationStore interface {
	Create(ctx context.Context, app *model.Application) error
	GetByID(ctx context.Context, id int64) (*model.Application, error)
	GetByUserIDAndName(ctx context.Context, userID int64, name string) (*model.Application, error)
	List(ctx context.Context, userID int64, status model.AppStatus, page, pageSize int) ([]*model.Application, int, error)
	Update(ctx context.Context, app *model.Application) error
	UpdateStatus(ctx context.Context, id int64, status model.AppStatus) error
	Delete(ctx context.Context, id int64) error
}

type applicationStore struct {
	db *sql.DB
}

// NewApplicationStore 创建应用存储实例
func NewApplicationStore(db *sql.DB) ApplicationStore {
	return &applicationStore{db: db}
}

// Create 创建应用
func (s *applicationStore) Create(ctx context.Context, app *model.Application) error {
	query := `
		INSERT INTO applications (
			user_id, app_name, app_description, app_type, status,
			callback_urls, allowed_origins, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err := s.db.QueryRowContext(ctx, query,
		app.UserID,
		app.AppName,
		app.AppDescription,
		app.AppType,
		app.Status,
		app.CallbackURLs,
		app.AllowedOrigins,
		app.CreatedAt,
		app.UpdatedAt,
	).Scan(&app.ID)

	return err
}

// GetByID 根据ID获取应用
func (s *applicationStore) GetByID(ctx context.Context, id int64) (*model.Application, error) {
	query := `
		SELECT 
			a.id, a.user_id, a.app_name, a.app_description, a.app_type, a.status,
			a.callback_urls, a.allowed_origins, a.created_at, a.updated_at,
			u.name as user_name
		FROM applications a
		JOIN users u ON a.user_id = u.id
		WHERE a.id = $1
	`

	app := &model.Application{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&app.ID,
		&app.UserID,
		&app.AppName,
		&app.AppDescription,
		&app.AppType,
		&app.Status,
		&app.CallbackURLs,
		&app.AllowedOrigins,
		&app.CreatedAt,
		&app.UpdatedAt,
		&app.UserName,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return app, err
}

// GetByUserIDAndName 根据用户ID和应用名称获取应用
func (s *applicationStore) GetByUserIDAndName(ctx context.Context, userID int64, name string) (*model.Application, error) {
	query := `
		SELECT 
			a.id, a.user_id, a.app_name, a.app_description, a.app_type, a.status,
			a.callback_urls, a.allowed_origins, a.created_at, a.updated_at,
			u.name as user_name
		FROM applications a
		JOIN users u ON a.user_id = u.id
		WHERE a.user_id = $1 AND a.app_name = $2
	`

	app := &model.Application{}
	err := s.db.QueryRowContext(ctx, query, userID, name).Scan(
		&app.ID,
		&app.UserID,
		&app.AppName,
		&app.AppDescription,
		&app.AppType,
		&app.Status,
		&app.CallbackURLs,
		&app.AllowedOrigins,
		&app.CreatedAt,
		&app.UpdatedAt,
		&app.UserName,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return app, err
}

// List 获取应用列表
func (s *applicationStore) List(ctx context.Context, userID int64, status model.AppStatus, page, pageSize int) ([]*model.Application, int, error) {
	// 构建查询条件
	var conditions []string
	var args []interface{}
	argIndex := 1

	if userID > 0 {
		conditions = append(conditions, fmt.Sprintf("a.user_id = $%d", argIndex))
		args = append(args, userID)
		argIndex++
	}

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argIndex))
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
		FROM applications a
		JOIN users u ON a.user_id = u.id
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
			a.id, a.user_id, a.app_name, a.app_description, a.app_type, a.status,
			a.callback_urls, a.allowed_origins, a.created_at, a.updated_at,
			u.name as user_name
		FROM applications a
		JOIN users u ON a.user_id = u.id
		%s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var applications []*model.Application
	for rows.Next() {
		app := &model.Application{}
		err := rows.Scan(
			&app.ID,
			&app.UserID,
			&app.AppName,
			&app.AppDescription,
			&app.AppType,
			&app.Status,
			&app.CallbackURLs,
			&app.AllowedOrigins,
			&app.CreatedAt,
			&app.UpdatedAt,
			&app.UserName,
		)
		if err != nil {
			return nil, 0, err
		}
		applications = append(applications, app)
	}

	return applications, total, rows.Err()
}

// Update 更新应用
func (s *applicationStore) Update(ctx context.Context, app *model.Application) error {
	query := `
		UPDATE applications SET
			app_name = $2,
			app_description = $3,
			app_type = $4,
			status = $5,
			callback_urls = $6,
			allowed_origins = $7,
			updated_at = $8
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		app.ID,
		app.AppName,
		app.AppDescription,
		app.AppType,
		app.Status,
		app.CallbackURLs,
		app.AllowedOrigins,
		app.UpdatedAt,
	)

	return err
}

// UpdateStatus 更新应用状态
func (s *applicationStore) UpdateStatus(ctx context.Context, id int64, status model.AppStatus) error {
	query := `
		UPDATE applications SET
			status = $2,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, id, status)
	return err
}

// Delete 删除应用
func (s *applicationStore) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM applications WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}