package store

import (
	"time"

	"github.com/vera-byte/vgo-iam/internal/model"

	"github.com/gocraft/dbr/v2"
)

// UserStore 用户存储接口
type UserStore interface {
	Create(user *model.User) error
	GetByID(id int) (*model.User, error)
	GetByName(name string) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	List() ([]*model.User, error)
	Update(user *model.User) error
	Delete(id int) error
	AttachPolicy(userID, policyID int) error
	DetachPolicy(userID, policyID int) error
	ListPolicies(userID int) ([]*model.Policy, error)
}

// userStore 用户存储实现
type userStore struct {
	session *dbr.Session
}

// NewUserStore 创建用户存储实例
func NewUserStore(session *dbr.Session) UserStore {
	return &userStore{session: session}
}

func (s *userStore) Create(user *model.User) error {

	_, err := s.session.InsertInto("users").
		Columns(
			"name",
			"display_name",
			"email",
		).
		Values(
			user.Name,
			user.DisplayName,
			user.Email,
		).Exec()

	return err
}

func (s *userStore) GetByID(id int) (*model.User, error) {
	var user model.User
	err := s.session.Select("*").
		From("users").
		Where("id = ?", id).
		LoadOne(&user)

	return &user, err
}

func (s *userStore) GetByName(name string) (*model.User, error) {
	var user model.User
	err := s.session.Select("*").
		From("users").
		Where("name = ?", name).
		LoadOne(&user)

	return &user, err
}

func (s *userStore) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := s.session.Select("*").
		From("users").
		Where("email = ?", email).
		LoadOne(&user)

	return &user, err
}

func (s *userStore) List() ([]*model.User, error) {
	var users []*model.User
	_, err := s.session.Select("*").
		From("users").
		Load(&users)
	return users, err
}

func (s *userStore) Update(user *model.User) error {
	_, err := s.session.Update("users").
		Set("display_name", user.DisplayName).
		Set("email", user.Email).
		Set("password_hash", user.Password).
		Set("updated_at", time.Now()).
		Where("id = ?", user.ID).
		Exec()
	return err
}

func (s *userStore) Delete(id int) error {
	_, err := s.session.DeleteFrom("users").
		Where("id = ?", id).
		Exec()
	return err
}

func (s *userStore) AttachPolicy(userID, policyID int) error {
	_, err := s.session.InsertInto("user_policies").
		Columns("user_id", "policy_id").
		Values(userID, policyID).
		Exec()
	return err
}

func (s *userStore) DetachPolicy(userID, policyID int) error {
	_, err := s.session.DeleteFrom("user_policies").
		Where("user_id = ? AND policy_id = ?", userID, policyID).
		Exec()
	return err
}

func (s *userStore) ListPolicies(userID int) ([]*model.Policy, error) {
	var policies []*model.Policy
	_, err := s.session.Select("p.*").
		From("policies p").
		Join("user_policies up", "p.id = up.policy_id").
		Where("up.user_id = ?", userID).
		Load(&policies)
	return policies, err
}
