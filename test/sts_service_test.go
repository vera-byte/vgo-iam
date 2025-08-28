package test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/service"
)

// 定义测试用的错误变量
var (
	ErrUserNotFound       = sql.ErrNoRows
	ErrPolicyNotFound     = sql.ErrNoRows
	ErrCredentialNotFound = sql.ErrNoRows
)

// MockTemporaryCredentialStore 模拟临时凭证存储
type MockTemporaryCredentialStore struct {
	mock.Mock
}

func (m *MockTemporaryCredentialStore) Create(tc *model.TemporaryCredential, masterKey string) error {
	args := m.Called(tc, masterKey)
	return args.Error(0)
}

func (m *MockTemporaryCredentialStore) GetByID(id int64) (*model.TemporaryCredential, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TemporaryCredential), args.Error(1)
}

func (m *MockTemporaryCredentialStore) GetByAccessKeyID(accessKeyID string, masterKey string) (*model.TemporaryCredential, error) {
	args := m.Called(accessKeyID, masterKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TemporaryCredential), args.Error(1)
}

func (m *MockTemporaryCredentialStore) GetBySessionToken(sessionToken string, masterKey string) (*model.TemporaryCredential, error) {
	args := m.Called(sessionToken, masterKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TemporaryCredential), args.Error(1)
}

func (m *MockTemporaryCredentialStore) ListByUser(userID int64) ([]*model.TemporaryCredential, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TemporaryCredential), args.Error(1)
}

func (m *MockTemporaryCredentialStore) ListActive() ([]*model.TemporaryCredential, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TemporaryCredential), args.Error(1)
}

func (m *MockTemporaryCredentialStore) ListExpired() ([]*model.TemporaryCredential, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TemporaryCredential), args.Error(1)
}

func (m *MockTemporaryCredentialStore) UpdateStatus(id int64, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockTemporaryCredentialStore) Revoke(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockTemporaryCredentialStore) RevokeBySessionToken(sessionToken string) error {
	args := m.Called(sessionToken)
	return args.Error(0)
}

func (m *MockTemporaryCredentialStore) Refresh(id int64, durationSeconds int32) (*model.TemporaryCredential, error) {
	args := m.Called(id, durationSeconds)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TemporaryCredential), args.Error(1)
}

func (m *MockTemporaryCredentialStore) CleanupExpired() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTemporaryCredentialStore) Delete(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockUserStore 模拟用户存储
type MockUserStore struct {
	mock.Mock
}

func (m *MockUserStore) Create(user *model.User) (int64, error) {
	args := m.Called(user)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserStore) GetByID(id int64) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserStore) GetByName(name string) (*model.User, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserStore) Update(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserStore) Delete(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserStore) List() ([]*model.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}

func (m *MockUserStore) GetByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserStore) AttachPolicy(userID int64, policyID int) error {
	args := m.Called(userID, policyID)
	return args.Error(0)
}

func (m *MockUserStore) DetachPolicy(userID int64, policyID int) error {
	args := m.Called(userID, policyID)
	return args.Error(0)
}

func (m *MockUserStore) ListPolicies(userID int64) ([]*model.Policy, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Policy), args.Error(1)
}

// MockPolicyStore 模拟策略存储
type MockPolicyStore struct {
	mock.Mock
}

func (m *MockPolicyStore) Create(policy *model.Policy) error {
	args := m.Called(policy)
	return args.Error(0)
}

func (m *MockPolicyStore) GetByID(id int) (*model.Policy, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Policy), args.Error(1)
}

func (m *MockPolicyStore) GetByName(name string) (*model.Policy, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Policy), args.Error(1)
}

func (m *MockPolicyStore) Update(policy *model.Policy) error {
	args := m.Called(policy)
	return args.Error(0)
}

func (m *MockPolicyStore) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockPolicyStore) List() ([]*model.Policy, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Policy), args.Error(1)
}

// STSServiceTestSuite STS服务测试套件
type STSServiceTestSuite struct {
	suite.Suite
	stsService          *service.STSService
	mockTempCredStore   *MockTemporaryCredentialStore
	mockUserStore       *MockUserStore
	mockPolicyStore     *MockPolicyStore
	masterKey          string
}

// SetupTest 设置测试环境
func (suite *STSServiceTestSuite) SetupTest() {
	suite.mockTempCredStore = &MockTemporaryCredentialStore{}
	suite.mockUserStore = &MockUserStore{}
	suite.mockPolicyStore = &MockPolicyStore{}
	suite.masterKey = "test-master-key-32-bytes-long!!"

	// 创建STS服务实例
	suite.stsService = service.NewSTSService(
		suite.mockTempCredStore,
		suite.mockUserStore,
		suite.mockPolicyStore,
		suite.masterKey,
	)
}

// TearDownTest 清理测试环境
func (suite *STSServiceTestSuite) TearDownTest() {
	suite.mockTempCredStore.AssertExpectations(suite.T())
	suite.mockUserStore.AssertExpectations(suite.T())
	suite.mockPolicyStore.AssertExpectations(suite.T())
}

// TestGetSessionToken_Success 测试获取会话令牌成功
func (suite *STSServiceTestSuite) TestGetSessionToken_Success() {
	// 测试GetSessionToken成功场景
	req := &iamv1.GetSessionTokenRequest{
		DurationSeconds: 3600,
	}

	// 模拟用户存在
	suite.mockUserStore.On("GetByName", "testuser").Return(&model.User{
		ID:   1,
		Name: "testuser",
	}, nil)

	// 模拟创建临时凭证
	suite.mockTempCredStore.On("Create", mock.AnythingOfType("*model.TemporaryCredential"), mock.AnythingOfType("string")).Return(nil)

	resp, err := suite.stsService.GetSessionToken(context.Background(), req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotNil(suite.T(), resp.Credentials)
	assert.NotEmpty(suite.T(), resp.Credentials.AccessKeyId)
	assert.NotEmpty(suite.T(), resp.Credentials.SecretAccessKey)
	assert.NotEmpty(suite.T(), resp.Credentials.SessionToken)
}

// TestGetSessionToken_UserNotFound 测试获取会话令牌用户不存在
func (suite *STSServiceTestSuite) TestGetSessionToken_UserNotFound() {
	// 测试GetSessionToken用户不存在场景
	req := &iamv1.GetSessionTokenRequest{
		DurationSeconds: 3600,
	}

	// 模拟用户不存在
	suite.mockUserStore.On("GetByName", "nonexistent").Return(nil, ErrUserNotFound)

	resp, err := suite.stsService.GetSessionToken(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Equal(suite.T(), ErrUserNotFound, err)
}

// TestAssumeRole_Success 测试假设角色成功
func (suite *STSServiceTestSuite) TestAssumeRole_Success() {
	// 测试AssumeRole成功场景
	req := &iamv1.AssumeRoleRequest{
		RoleArn:         "arn:aws:iam::123456789012:role/TestRole",
		RoleSessionName: "TestSession",
		DurationSeconds: 3600,
	}

	// 模拟策略存在
	suite.mockPolicyStore.On("GetByName", "TestRole").Return(&model.Policy{
		ID:             1,
		Name:           "TestRole",
		PolicyDocument: "{\"Version\":\"2012-10-17\",\"Statement\":[]}",
	}, nil)

	// 模拟创建临时凭证
	suite.mockTempCredStore.On("Create", mock.AnythingOfType("*model.TemporaryCredential"), mock.AnythingOfType("string")).Return(nil)

	resp, err := suite.stsService.AssumeRole(context.Background(), req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotNil(suite.T(), resp.Credentials)
	assert.NotEmpty(suite.T(), resp.Credentials.AccessKeyId)
	assert.NotEmpty(suite.T(), resp.Credentials.SecretAccessKey)
	assert.NotEmpty(suite.T(), resp.Credentials.SessionToken)
	assert.NotNil(suite.T(), resp.AssumedRoleUser)
	assert.Equal(suite.T(), req.RoleArn, resp.AssumedRoleUser.Arn)
}

// TestAssumeRole_PolicyNotFound 测试假设角色策略不存在
func (suite *STSServiceTestSuite) TestAssumeRole_PolicyNotFound() {
	// 测试AssumeRole策略不存在场景
	req := &iamv1.AssumeRoleRequest{
		RoleArn:         "arn:aws:iam::123456789012:role/NonExistentRole",
		RoleSessionName: "TestSession",
		DurationSeconds: 3600,
	}

	// 模拟策略不存在
	suite.mockPolicyStore.On("GetByName", "NonExistentRole").Return(nil, ErrPolicyNotFound)

	resp, err := suite.stsService.AssumeRole(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Equal(suite.T(), ErrPolicyNotFound, err)
}

// TestRefreshToken_Success 测试刷新令牌成功
func (suite *STSServiceTestSuite) TestRefreshToken_Success() {
	// 测试RefreshToken成功场景
	req := &iamv1.RefreshTokenRequest{
		SessionToken:    "test-session-token",
		DurationSeconds: 7200,
	}

	// 模拟临时凭证存在
	existingCred := &model.TemporaryCredential{
		ID:              1,
		AccessKeyID:     "AKIATEST123",
		SecretAccessKey: "secret123",
		SessionToken:    "test-session-token",
		ExpiresAt:       time.Now().Add(time.Hour),
	}
	suite.mockTempCredStore.On("GetBySessionToken", "test-session-token", mock.AnythingOfType("string")).Return(existingCred, nil)

	// 模拟刷新凭证
	suite.mockTempCredStore.On("Refresh", mock.AnythingOfType("int64"), mock.AnythingOfType("int32")).Return(existingCred, nil)

	resp, err := suite.stsService.RefreshToken(context.Background(), req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotNil(suite.T(), resp.Credentials)
	assert.NotEmpty(suite.T(), resp.Credentials.AccessKeyId)
	assert.NotEmpty(suite.T(), resp.Credentials.SecretAccessKey)
	assert.NotEmpty(suite.T(), resp.Credentials.SessionToken)
}

// TestRefreshToken_CredentialNotFound 测试刷新令牌凭证不存在
func (suite *STSServiceTestSuite) TestRefreshToken_CredentialNotFound() {
	// 测试RefreshToken凭证不存在场景
	req := &iamv1.RefreshTokenRequest{
		SessionToken:    "nonexistent-token",
		DurationSeconds: 7200,
	}

	// 模拟临时凭证不存在
	suite.mockTempCredStore.On("GetBySessionToken", "nonexistent-token", mock.AnythingOfType("string")).Return(nil, ErrCredentialNotFound)

	resp, err := suite.stsService.RefreshToken(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
	assert.Equal(suite.T(), ErrCredentialNotFound, err)
}

// TestValidateTemporaryCredential_Success 测试验证临时凭证成功
func (suite *STSServiceTestSuite) TestValidateTemporaryCredential_Success() {
	// 测试验证临时凭证成功场景
	accessKeyID := "AKIATEST123"
	sessionToken := "test-session-token"

	// 模拟临时凭证存在且未过期
	cred := &model.TemporaryCredential{
		ID:              1,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: "secret123",
		SessionToken:    sessionToken,
		ExpiresAt:       time.Now().Add(time.Hour),
	}
	suite.mockTempCredStore.On("GetByAccessKeyID", accessKeyID, mock.AnythingOfType("string")).Return(cred, nil)

	result, err := suite.stsService.ValidateTemporaryCredential(accessKeyID, sessionToken)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
}

// TestValidateTemporaryCredential_Expired 测试验证临时凭证已过期
func (suite *STSServiceTestSuite) TestValidateTemporaryCredential_Expired() {
	// 测试验证临时凭证已过期场景
	accessKeyID := "AKIATEST123"
	sessionToken := "test-session-token"

	// 模拟临时凭证存在但已过期
	cred := &model.TemporaryCredential{
		ID:              1,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: "secret123",
		SessionToken:    sessionToken,
		ExpiresAt:       time.Now().Add(-time.Hour), // 已过期
	}
	suite.mockTempCredStore.On("GetByAccessKeyID", accessKeyID, mock.AnythingOfType("string")).Return(cred, nil)

	result, err := suite.stsService.ValidateTemporaryCredential(accessKeyID, sessionToken)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
}

// TestCleanupExpiredCredentials_Success 测试清理过期凭证成功
func (suite *STSServiceTestSuite) TestCleanupExpiredCredentials_Success() {
	// 模拟清理过期凭证成功
	suite.mockTempCredStore.On("CleanupExpired").Return(int64(5), nil)

	count, err := suite.stsService.CleanupExpiredCredentials()

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(5), count)
}

// TestGetStore 测试获取存储接口
func (suite *STSServiceTestSuite) TestGetStore() {
	store := suite.stsService.GetStore()
	assert.NotNil(suite.T(), store)
	assert.Equal(suite.T(), suite.mockTempCredStore, store)
}

// TestSTSServiceTestSuite 运行测试套件
func TestSTSServiceTestSuite(t *testing.T) {
	suite.Run(t, new(STSServiceTestSuite))
}

// TestNewSTSService 测试创建STS服务
func TestNewSTSService(t *testing.T) {
	mockTempCredStore := &MockTemporaryCredentialStore{}
	mockUserStore := &MockUserStore{}
	mockPolicyStore := &MockPolicyStore{}
	masterKey := "test-master-key-32-bytes-long!!"

	stsService := service.NewSTSService(mockTempCredStore, mockUserStore, mockPolicyStore, masterKey)

	assert.NotNil(t, stsService)
	assert.Equal(t, mockTempCredStore, stsService.GetStore())
}