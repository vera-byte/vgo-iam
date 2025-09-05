package controller

import (
	"context"
	"testing"
	"time"

	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSTSController 测试STS控制器创建
// 功能: 验证STS控制器实例创建是否正确
// 参数: t - 测试实例
// 返回值: 无
func TestNewSTSController(t *testing.T) {
	controller := NewSTSController()
	assert.NotNil(t, controller, "STS控制器实例不应为空")
}

// TestSTSController_AssumeRole 测试角色假设功能
// 功能: 验证AssumeRole方法的基本功能
// 参数: t - 测试实例
// 返回值: 无
func TestSTSController_AssumeRole(t *testing.T) {
	controller := NewSTSController()
	ctx := context.Background()

	tests := []struct {
		name    string
		request *iamv1.AssumeRoleRequest
		wantErr bool
	}{
		{
			name: "有效的角色假设请求",
			request: &iamv1.AssumeRoleRequest{
				RoleArn:         "arn:aws:iam::123456789012:role/TestRole",
				RoleSessionName: "test-session",
				DurationSeconds: 3600,
			},
			wantErr: false,
		},
		{
			name: "空的角色ARN",
			request: &iamv1.AssumeRoleRequest{
				RoleArn:         "",
				RoleSessionName: "test-session",
				DurationSeconds: 3600,
			},
			wantErr: true,
		},
		{
			name: "空的会话名称",
			request: &iamv1.AssumeRoleRequest{
				RoleArn:         "arn:aws:iam::123456789012:role/TestRole",
				RoleSessionName: "",
				DurationSeconds: 3600,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := controller.AssumeRole(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err, "期望返回错误")
				assert.Nil(t, resp, "错误情况下响应应为空")
			} else {
				assert.NoError(t, err, "不应返回错误")
				assert.NotNil(t, resp, "响应不应为空")
				assert.NotNil(t, resp.Credentials, "临时凭证不应为空")
				assert.NotNil(t, resp.AssumedRoleUser, "假设角色用户信息不应为空")

				// 验证凭证格式
				assert.True(t, len(resp.Credentials.AccessKeyId) > 0, "访问密钥ID不应为空")
				assert.True(t, len(resp.Credentials.SecretAccessKey) > 0, "秘密访问密钥不应为空")
				assert.True(t, len(resp.Credentials.SessionToken) > 0, "会话令牌不应为空")
				assert.NotNil(t, resp.Credentials.Expiration, "过期时间不应为空")

				// 验证过期时间在未来
				expiration := resp.Credentials.Expiration.AsTime()
				assert.True(t, expiration.After(time.Now()), "过期时间应在未来")
			}
		})
	}
}

// TestSTSController_GetSessionToken 测试获取会话令牌功能
// 功能: 验证GetSessionToken方法的基本功能
// 参数: t - 测试实例
// 返回值: 无
func TestSTSController_GetSessionToken(t *testing.T) {
	controller := NewSTSController()
	ctx := context.Background()

	request := &iamv1.GetSessionTokenRequest{
		DurationSeconds: 3600,
	}

	resp, err := controller.GetSessionToken(ctx, request)

	assert.NoError(t, err, "不应返回错误")
	assert.NotNil(t, resp, "响应不应为空")
	assert.NotNil(t, resp.Credentials, "临时凭证不应为空")

	// 验证凭证格式
	assert.True(t, len(resp.Credentials.AccessKeyId) > 0, "访问密钥ID不应为空")
	assert.True(t, len(resp.Credentials.SecretAccessKey) > 0, "秘密访问密钥不应为空")
	assert.True(t, len(resp.Credentials.SessionToken) > 0, "会话令牌不应为空")
	assert.NotNil(t, resp.Credentials.Expiration, "过期时间不应为空")

	// 验证过期时间在未来
	expiration := resp.Credentials.Expiration.AsTime()
	assert.True(t, expiration.After(time.Now()), "过期时间应在未来")
}

// TestSTSController_RefreshToken 测试令牌刷新功能
// 功能: 验证RefreshToken方法的基本功能
// 参数: t - 测试实例
// 返回值: 无
func TestSTSController_RefreshToken(t *testing.T) {
	controller := NewSTSController()
	ctx := context.Background()

	tests := []struct {
		name    string
		request *iamv1.RefreshTokenRequest
		wantErr bool
	}{
		{
			name: "有效的刷新请求",
			request: &iamv1.RefreshTokenRequest{
				SessionToken:    "valid-session-token",
				DurationSeconds: 7200,
			},
			wantErr: false,
		},
		{
			name: "空的会话令牌",
			request: &iamv1.RefreshTokenRequest{
				SessionToken:    "",
				DurationSeconds: 7200,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := controller.RefreshToken(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err, "期望返回错误")
				assert.Nil(t, resp, "错误情况下响应应为空")
			} else {
				assert.NoError(t, err, "不应返回错误")
				assert.NotNil(t, resp, "响应不应为空")
				assert.NotNil(t, resp.Credentials, "临时凭证不应为空")

				// 验证凭证格式
				assert.True(t, len(resp.Credentials.AccessKeyId) > 0, "访问密钥ID不应为空")
				assert.True(t, len(resp.Credentials.SecretAccessKey) > 0, "秘密访问密钥不应为空")
				assert.Equal(t, tt.request.SessionToken, resp.Credentials.SessionToken, "会话令牌应保持一致")
				assert.NotNil(t, resp.Credentials.Expiration, "过期时间不应为空")
			}
		})
	}
}

// TestSTSController_RevokeToken 测试令牌撤销功能
// 功能: 验证RevokeToken方法的基本功能
// 参数: t - 测试实例
// 返回值: 无
func TestSTSController_RevokeToken(t *testing.T) {
	controller := NewSTSController()
	ctx := context.Background()

	tests := []struct {
		name    string
		request *iamv1.RevokeTokenRequest
		wantErr bool
	}{
		{
			name: "有效的撤销请求",
			request: &iamv1.RevokeTokenRequest{
				SessionToken: "valid-session-token",
			},
			wantErr: false,
		},
		{
			name: "空的会话令牌",
			request: &iamv1.RevokeTokenRequest{
				SessionToken: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := controller.RevokeToken(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err, "期望返回错误")
				assert.Nil(t, resp, "错误情况下响应应为空")
			} else {
				assert.NoError(t, err, "不应返回错误")
				assert.NotNil(t, resp, "响应不应为空")
				assert.True(t, resp.Success, "撤销应该成功")
				assert.NotEmpty(t, resp.Message, "应该有响应消息")
			}
		})
	}
}

// TestGenerateRandomString 测试随机字符串生成功能
// 功能: 验证generateRandomString函数的正确性
// 参数: t - 测试实例
// 返回值: 无
func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"长度为0", 0},
		{"长度为1", 1},
		{"长度为10", 10},
		{"长度为32", 32},
		{"长度为64", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomString(tt.length)
			assert.Equal(t, tt.length, len(result), "生成的字符串长度应该匹配")

			// 验证字符串只包含预期的字符
			if tt.length > 0 {
				for _, char := range result {
					assert.True(t, 
						(char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9'),
						"字符应该是大写字母或数字")
				}
			}
		})
	}

	// 测试多次生成的结果应该不同
	result1 := generateRandomString(16)
	result2 := generateRandomString(16)
	assert.NotEqual(t, result1, result2, "多次生成的随机字符串应该不同")
}

// BenchmarkGenerateRandomString 性能测试随机字符串生成
// 功能: 测试generateRandomString函数的性能
// 参数: b - 性能测试实例
// 返回值: 无
func BenchmarkGenerateRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateRandomString(32)
	}
}

// BenchmarkAssumeRole 性能测试角色假设
// 功能: 测试AssumeRole方法的性能
// 参数: b - 性能测试实例
// 返回值: 无
func BenchmarkAssumeRole(b *testing.B) {
	controller := NewSTSController()
	ctx := context.Background()
	request := &iamv1.AssumeRoleRequest{
		RoleArn:         "arn:aws:iam::123456789012:role/TestRole",
		RoleSessionName: "benchmark-session",
		DurationSeconds: 3600,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := controller.AssumeRole(ctx, request)
		require.NoError(b, err)
	}
}