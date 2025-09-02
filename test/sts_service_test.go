package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vera-byte/vgo-iam/internal/config"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/store"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	"github.com/vera-byte/vgo-kit/db"
	vgoconfig "github.com/vera-byte/vgo-kit/config"
)

// STSServiceIntegrationTestSuite STS服务集成测试套件
type STSServiceIntegrationTestSuite struct {
	suite.Suite
	dbStore    *db.PostgresStore
	userStore  store.UserStore
	policyStore store.PolicyStore
	tempCredStore store.TemporaryCredentialStore
	stsService *service.STSService
	cfg        *config.AppConfig
	testUser   *model.User
	testPolicy *model.Policy
}

// SetupSuite 测试套件初始化
func (suite *STSServiceIntegrationTestSuite) SetupSuite() {
	// 使用vgo-kit加载配置文件
	v, err := vgoconfig.LoadConfig("../config/config.yaml")
	if err != nil {
		suite.T().Fatalf("无法加载配置文件: %v", err)
	}

	// 反序列化配置到结构体
	suite.cfg = &config.AppConfig{}
	if err := v.Unmarshal(suite.cfg); err != nil {
		suite.T().Fatalf("无法解析配置文件: %v", err)
	}

	// 使用vgo-kit初始化数据库连接
	suite.dbStore, err = db.NewPostgresStore(suite.cfg.Database.DSN)
	if err != nil {
		suite.T().Fatalf("无法连接到测试数据库: %v", err)
	}

	// 初始化存储层
	suite.userStore = store.NewUserStore(suite.dbStore.Session)
	suite.policyStore = store.NewPolicyStore(suite.dbStore.Session)
	suite.tempCredStore = store.NewTemporaryCredentialStore(suite.dbStore.Session)

	// 创建STS服务实例
	suite.stsService = service.NewSTSService(
		suite.tempCredStore,
		suite.userStore,
		suite.policyStore,
		suite.cfg.Middleware.MasterKey,
	)
}

// SetupTest 每个测试前的准备工作
func (suite *STSServiceIntegrationTestSuite) SetupTest() {
	// 清理测试数据
	suite.cleanupTestData()

	// 创建测试用户（ID必须为1，因为STS服务硬编码使用userID=1）
	testUser := &model.User{
		Name:        "testuser",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}
	userID, err := suite.userStore.Create(testUser)
	if err != nil {
		suite.T().Fatalf("创建测试用户失败: %v", err)
	}
	testUser.ID = userID
	suite.testUser = testUser

	// 如果创建的用户ID不是1，需要确保ID为1的用户存在
	if userID != 1 {
		// 检查ID为1的用户是否存在，如果不存在则创建
		existingUser, err := suite.userStore.GetByID(1)
		if err != nil {
			// 用户不存在，创建一个ID为1的用户
			testUser1 := &model.User{
				Name:        "testuser1",
				DisplayName: "Test User 1",
				Email:       "test1@example.com",
			}
			// 直接插入ID为1的用户
			suite.dbStore.Session.InsertInto("users").
				Columns("id", "name", "display_name", "email", "created_at", "updated_at").
				Values(1, testUser1.Name, testUser1.DisplayName, testUser1.Email, time.Now(), time.Now()).
				Exec()
		} else {
			// 用户已存在，使用现有用户
			suite.testUser = existingUser
		}
	}

	// 创建测试策略
	testPolicy := &model.Policy{
		Name:           "TestRole",
		PolicyDocument: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`,
		Description:    "Test policy for STS",
	}
	err = suite.policyStore.Create(testPolicy)
	if err != nil {
		suite.T().Fatalf("创建测试策略失败: %v", err)
	}
	suite.testPolicy = testPolicy
}

// TearDownTest 每个测试后的清理工作
func (suite *STSServiceIntegrationTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

// TearDownSuite 测试套件清理
func (suite *STSServiceIntegrationTestSuite) TearDownSuite() {
	if suite.dbStore != nil {
		suite.dbStore.Close()
	}
}

// cleanupTestData 清理测试数据
func (suite *STSServiceIntegrationTestSuite) cleanupTestData() {
	// 清理临时凭证
	suite.dbStore.Session.DeleteFrom("temporary_credentials").Exec()
	// 清理用户
	suite.dbStore.Session.DeleteFrom("users").Where("name = ?", "testuser").Exec()
	// 清理策略
	suite.dbStore.Session.DeleteFrom("policies").Where("name = ?", "TestRole").Exec()
}

// TestGetSessionToken_Success 测试获取会话令牌成功
func (suite *STSServiceIntegrationTestSuite) TestGetSessionToken_Success() {
	req := &iamv1.GetSessionTokenRequest{
		DurationSeconds: 3600,
	}

	resp, err := suite.stsService.GetSessionToken(context.Background(), req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), resp)
	assert.NotNil(suite.T(), resp.Credentials)
	assert.NotEmpty(suite.T(), resp.Credentials.AccessKeyId)
	assert.NotEmpty(suite.T(), resp.Credentials.SecretAccessKey)
	assert.NotEmpty(suite.T(), resp.Credentials.SessionToken)
	assert.NotNil(suite.T(), resp.Credentials.Expiration)
}

// TestGetSessionToken_UserNotFound 测试用户不存在（删除ID为1的用户）
func (suite *STSServiceIntegrationTestSuite) TestGetSessionToken_UserNotFound() {
	// 删除ID为1的用户以模拟用户不存在的情况
	suite.dbStore.Session.DeleteFrom("users").Where("id = ?", 1).Exec()

	req := &iamv1.GetSessionTokenRequest{
		DurationSeconds: 3600,
	}

	resp, err := suite.stsService.GetSessionToken(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)

	// 恢复ID为1的用户
	testUser1 := &model.User{
		Name:        "testuser1",
		DisplayName: "Test User 1",
		Email:       "test1@example.com",
	}
	suite.dbStore.Session.InsertInto("users").
		Columns("id", "name", "display_name", "email", "created_at", "updated_at").
		Values(1, testUser1.Name, testUser1.DisplayName, testUser1.Email, time.Now(), time.Now()).
		Exec()
}

// TestAssumeRole_Success 测试假设角色成功
func (suite *STSServiceIntegrationTestSuite) TestAssumeRole_Success() {
	req := &iamv1.AssumeRoleRequest{
		RoleArn:         "arn:aws:iam::123456789012:role/TestRole",
		RoleSessionName: "TestSession",
		DurationSeconds: 3600,
	}

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
func (suite *STSServiceIntegrationTestSuite) TestAssumeRole_PolicyNotFound() {
	req := &iamv1.AssumeRoleRequest{
		RoleArn:         "arn:aws:iam::123456789012:role/NonExistentRole",
		RoleSessionName: "TestSession",
		DurationSeconds: 3600,
	}

	resp, err := suite.stsService.AssumeRole(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
}

// TestRefreshToken_Success 测试刷新令牌成功
func (suite *STSServiceIntegrationTestSuite) TestRefreshToken_Success() {
	// 先创建一个临时凭证
	getReq := &iamv1.GetSessionTokenRequest{
		DurationSeconds: 3600,
	}
	getResp, err := suite.stsService.GetSessionToken(context.Background(), getReq)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), getResp)

	// 刷新令牌
	refreshReq := &iamv1.RefreshTokenRequest{
		SessionToken:    getResp.Credentials.SessionToken,
		DurationSeconds: 7200,
	}

	refreshResp, err := suite.stsService.RefreshToken(context.Background(), refreshReq)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), refreshResp)
	assert.NotNil(suite.T(), refreshResp.Credentials)
	assert.NotEmpty(suite.T(), refreshResp.Credentials.AccessKeyId)
	assert.NotEmpty(suite.T(), refreshResp.Credentials.SecretAccessKey)
	assert.NotEmpty(suite.T(), refreshResp.Credentials.SessionToken)
}

// TestRefreshToken_CredentialNotFound 测试刷新令牌凭证不存在
func (suite *STSServiceIntegrationTestSuite) TestRefreshToken_CredentialNotFound() {
	req := &iamv1.RefreshTokenRequest{
		SessionToken:    "nonexistent-token",
		DurationSeconds: 7200,
	}

	resp, err := suite.stsService.RefreshToken(context.Background(), req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), resp)
}

// TestValidateTemporaryCredential_Success 测试验证临时凭证成功
func (suite *STSServiceIntegrationTestSuite) TestValidateTemporaryCredential_Success() {
	// 先创建一个临时凭证
	getReq := &iamv1.GetSessionTokenRequest{
		DurationSeconds: 3600,
	}
	getResp, err := suite.stsService.GetSessionToken(context.Background(), getReq)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), getResp)

	// 验证临时凭证
	result, err := suite.stsService.ValidateTemporaryCredential(
		getResp.Credentials.AccessKeyId,
		getResp.Credentials.SessionToken,
	)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), getResp.Credentials.AccessKeyId, result.AccessKeyID)
}

// TestValidateTemporaryCredential_NotFound 测试验证临时凭证不存在
func (suite *STSServiceIntegrationTestSuite) TestValidateTemporaryCredential_NotFound() {
	result, err := suite.stsService.ValidateTemporaryCredential(
		"AKIATEST123",
		"test-session-token",
	)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
}

// TestCleanupExpiredCredentials_Success 测试清理过期凭证成功
func (suite *STSServiceIntegrationTestSuite) TestCleanupExpiredCredentials_Success() {
	// 创建一个已过期的临时凭证
	expiredCred := model.NewSessionToken(
		suite.testUser.ID,
		"AKIAEXPIRED123",
		"expired-secret",
		"expired-session-token",
		900, // 15分钟
	)
	// 手动设置为已过期
	expiredCred.ExpiresAt = time.Now().Add(-time.Hour)
	err := suite.tempCredStore.Create(expiredCred, suite.cfg.Middleware.MasterKey)
	assert.NoError(suite.T(), err)

	// 清理过期凭证
	count, err := suite.stsService.CleanupExpiredCredentials()

	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), count, int64(1))
}

// TestGetStore 测试获取存储接口
func (suite *STSServiceIntegrationTestSuite) TestGetStore() {
	store := suite.stsService.GetStore()
	assert.NotNil(suite.T(), store)
	assert.Equal(suite.T(), suite.tempCredStore, store)
}

// TestSTSServiceIntegrationTestSuite 运行集成测试套件
func TestSTSServiceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(STSServiceIntegrationTestSuite))
}

// TestNewSTSService 测试创建STS服务
func TestNewSTSService(t *testing.T) {
	// 使用vgo-kit加载配置文件
	v, err := vgoconfig.LoadConfig("../config/config.yaml")
	if err != nil {
		t.Skipf("跳过测试，无法加载配置文件: %v", err)
		return
	}

	// 反序列化配置到结构体
	cfg := &config.AppConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		t.Skipf("跳过测试，无法解析配置文件: %v", err)
		return
	}

	// 使用vgo-kit初始化数据库连接
	dbStore, err := db.NewPostgresStore(cfg.Database.DSN)
	if err != nil {
		t.Skipf("跳过测试，无法连接到数据库: %v", err)
		return
	}
	defer dbStore.Close()

	// 初始化存储层
	userStore := store.NewUserStore(dbStore.Session)
	policyStore := store.NewPolicyStore(dbStore.Session)
	tempCredStore := store.NewTemporaryCredentialStore(dbStore.Session)

	stsService := service.NewSTSService(tempCredStore, userStore, policyStore, cfg.Middleware.MasterKey)

	assert.NotNil(t, stsService)
	assert.Equal(t, tempCredStore, stsService.GetStore())
}