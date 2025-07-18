package store

import (
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/vera-byte/vgo-iam/internal/model"
)

// PolicyStore 策略存储接口
type PolicyStore interface {
	Create(policy *model.Policy) error
	GetByID(id int) (*model.Policy, error)
	GetByName(name string) (*model.Policy, error)
	List() ([]*model.Policy, error)
	Update(policy *model.Policy) error
	Delete(id int) error
}

// policyStore 策略存储实现
type policyStore struct {
	session *dbr.Session
}

// NewPolicyStore 创建策略存储实例
func NewPolicyStore(session *dbr.Session) PolicyStore {
	return &policyStore{session: session}
}

func (s *policyStore) Create(policy *model.Policy) error {
	_, err := s.session.InsertInto("policies").
		Columns(
			"name",
			"description",
			"policy_document",
		).
		Values(
			policy.Name,
			policy.Description,
			policy.PolicyDocument,
		).Exec()

	return err
}

func (s *policyStore) GetByID(id int) (*model.Policy, error) {
	var policy model.Policy
	err := s.session.Select("*").
		From("policies").
		Where("id = ?", id).
		LoadOne(&policy)

	return &policy, err
}

func (s *policyStore) GetByName(name string) (*model.Policy, error) {
	var policy model.Policy
	err := s.session.Select("*").
		From("policies").
		Where("name = ?", name).
		LoadOne(&policy)

	return &policy, err
}

func (s *policyStore) List() ([]*model.Policy, error) {
	var policies []*model.Policy
	_, err := s.session.Select("*").
		From("policies").
		Load(&policies)
	return policies, err
}

func (s *policyStore) Update(policy *model.Policy) error {
	_, err := s.session.Update("policies").
		Set("description", policy.Description).
		Set("policy_document", policy.PolicyDocument).
		Set("updated_at", time.Now()).
		Where("id = ?", policy.ID).
		Exec()
	return err
}

func (s *policyStore) Delete(id int) error {
	_, err := s.session.DeleteFrom("policies").
		Where("id = ?", id).
		Exec()
	return err
}
