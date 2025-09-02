package test

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"testing"

	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	"github.com/vera-byte/vgo-iam/pkg/signature"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	AccessKeyID     = "Ga0rTSg3NSyoOkFUx9jg"
	SecretAccessKey = "poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk"
)

// 客户端
func NewTestIAMClient(t *testing.T) (client iamv1.IAMClient, conn *grpc.ClientConn, err error) {
	conn, err = grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("无法连接到服务器: %v", err)
		return nil, nil, err
	}
	// 创建客户端
	return iamv1.NewIAMClient(conn), conn, nil
}

// 验证AccessKey
func TestVerifyAccessKey(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	baseReqData := []byte{}
	ctx, signer := SignV4Ctx(context.Background(), AccessKeyID, SecretAccessKey, baseReqData)
	t.Logf("signer: %+v", signer)
	_, err = client.VerifyAccessKey(ctx, nil)

	if err != nil {
		t.Fatalf("验证访问密钥失败: %v", err)
	}
}

// 签名上下文
func SignV4Ctx(ctx context.Context, accessKeyID, secretAccessKey string, req interface{}) (context.Context, signature.SignV4Result) {
	reqData, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("序列化请求数据失败: %v", err)
	}
	signer := signature.SignV4(accessKeyID, secretAccessKey, string(reqData))
	md := metadata.Pairs(
		"access-key-id", signer.AccessKeyID,
		"signature", signer.Signature,
		"x-iam-date", strconv.FormatInt(signer.Timestamp, 10),
		"request-data", string(reqData),
	)
	return metadata.NewOutgoingContext(ctx, md), signer
}

// 为应用创建访问密钥（新的必需方式）
func TestCreateUserAccessKey(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	// 现在所有访问密钥都必须关联到应用ID
	req := &iamv1.CreateAccessKeyRequest{
		UserName:    "testuser",
		AppId:       1, // 必须指定应用ID
		Description: "Test User Access Key",
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	access, err := client.CreateAccessKey(ctx, req)
	if err != nil {
		t.Fatalf("创建访问密钥失败: %v", err)
	}

	if access == nil {
		t.Fatalf("为用户创建访问密钥失败")
	}
	
	// 验证返回的访问密钥包含应用ID
	if access.AppId != 1 {
		t.Fatalf("访问密钥应该关联到应用ID 1，但得到: %d", access.AppId)
	}
	
	t.Logf("用户访问密钥创建成功: %+v", access)
}

// 创建用户
func TestCreateUserWithAuth(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()
	req := &iamv1.CreateUserRequest{
		Name:        "testuser",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.CreateUser(ctx, req)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	t.Logf("创建用户成功: %+v", resp)
}

// 提交开发者认证
func TestSubmitDeveloperVerification(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	// 测试个人开发者认证
	req := &iamv1.SubmitDeveloperVerificationRequest{
		DeveloperType:  "individual",
		RealName:       "John Doe",
		IdCardNumber:   "110101199001011234",
		IdCardFrontUrl: "https://example.com/id_front.jpg",
		IdCardBackUrl:  "https://example.com/id_back.jpg",
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.SubmitDeveloperVerification(ctx, req)
	if err != nil {
		// 如果已经存在认证记录，这是预期的行为
		if strings.Contains(err.Error(), "verification already exists") {
			t.Logf("个人开发者认证已存在，跳过测试: %v", err)
			return
		}
		t.Fatalf("提交开发者认证失败: %v", err)
	}

	t.Logf("提交开发者认证成功: %+v", resp)

	// 立即审核通过认证
	reviewReq := &iamv1.ReviewDeveloperVerificationRequest{
		VerificationId: resp.Id,
		Status:         "approved",
		ReviewComment:  "Test approval",
	}

	reviewCtx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, reviewReq)
	_, err = client.ReviewDeveloperVerification(reviewCtx, reviewReq)
	if err != nil {
		t.Logf("审核认证失败，但继续测试: %v", err)
	} else {
		t.Logf("审核认证成功")
	}
}

// 提交企业开发者认证
func TestSubmitEnterpriseDeveloperVerification(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	// 测试企业开发者认证
	req := &iamv1.SubmitDeveloperVerificationRequest{
		DeveloperType:         "enterprise",
		CompanyName:           "Test Technology Co Ltd",
		BusinessLicenseNumber: "91110000123456789X",
		BusinessLicenseUrl:    "https://example.com/license.jpg",
		LegalRepresentative:   "Jane Smith",
		CompanyAddress:        "123 Test Street Beijing",
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.SubmitDeveloperVerification(ctx, req)
	if err != nil {
		// 如果已经存在认证记录，这是预期的行为
		if strings.Contains(err.Error(), "verification already exists") {
			t.Logf("企业开发者认证已存在，跳过测试: %v", err)
			return
		}
		t.Fatalf("提交企业开发者认证失败: %v", err)
	}

	t.Logf("提交企业开发者认证成功: %+v", resp)

	// 立即审核通过认证
	reviewReq := &iamv1.ReviewDeveloperVerificationRequest{
		VerificationId: resp.Id,
		Status:         "approved",
		ReviewComment:  "Test approval for enterprise",
	}

	reviewCtx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, reviewReq)
	_, err = client.ReviewDeveloperVerification(reviewCtx, reviewReq)
	if err != nil {
		t.Logf("审核企业认证失败，但继续测试: %v", err)
	} else {
		t.Logf("审核企业认证成功")
	}
}

// 获取开发者认证信息
func TestGetDeveloperVerification(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	req := &iamv1.GetDeveloperVerificationRequest{
		UserName:      "testuser",
		DeveloperType: "individual",
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.GetDeveloperVerification(ctx, req)
	if err != nil {
		t.Fatalf("获取开发者认证信息失败: %v", err)
	}

	t.Logf("获取开发者认证信息成功: %+v", resp)
}

// 创建应用
func TestCreateApplication(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	req := &iamv1.CreateApplicationRequest{
		AppName:        "Test Application",
		AppDescription: "This is a test application",
		AppType:        "web",
		AppIconUrl:     "https://example.com/icon.png",
		AppWebsite:     "https://example.com",
		CallbackUrls:   []string{"https://example.com/callback"},
		AllowedOrigins: []string{"https://example.com"},
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.CreateApplication(ctx, req)
	if err != nil {
		t.Fatalf("创建应用失败: %v", err)
	}

	t.Logf("创建应用成功: %+v", resp)
}

// 获取应用信息
func TestGetApplication(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	req := &iamv1.GetApplicationRequest{
		AppId: 1, // 假设应用ID为1
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.GetApplication(ctx, req)
	if err != nil {
		t.Fatalf("获取应用信息失败: %v", err)
	}

	t.Logf("获取应用信息成功: %+v", resp)
}

// 列出应用
func TestListApplications(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	req := &iamv1.ListApplicationsRequest{
		UserName: "testuser",
		Status:   "active",
		Page:     1,
		PageSize: 10,
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.ListApplications(ctx, req)
	if err != nil {
		t.Fatalf("列出应用失败: %v", err)
	}

	t.Logf("列出应用成功: %+v", resp)
}

// 更新应用
func TestUpdateApplication(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	req := &iamv1.UpdateApplicationRequest{
		AppId:          1,
		AppName:        "Updated Test Application",
		AppDescription: "This is an updated test application",
		AppType:        "mobile",
		AppIconUrl:     "https://example.com/new_icon.png",
		AppWebsite:     "https://newexample.com",
		CallbackUrls:   []string{"https://newexample.com/callback"},
		AllowedOrigins: []string{"https://newexample.com"},
		Status:         "active",
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.UpdateApplication(ctx, req)
	if err != nil {
		t.Fatalf("更新应用失败: %v", err)
	}

	t.Logf("更新应用成功: %+v", resp)
}

// 删除应用
func TestDeleteApplication(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	req := &iamv1.DeleteApplicationRequest{
		AppId: 1, // 假设应用ID为1
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.DeleteApplication(ctx, req)
	if err != nil {
		t.Fatalf("删除应用失败: %v", err)
	}

	t.Logf("删除应用成功: %+v", resp)
}

// 为应用创建访问密钥（第二个测试）
func TestCreateAccessKeyForApp(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	defer conn.Close()

	// 创建第二个应用的访问密钥
	req := &iamv1.CreateAccessKeyRequest{
		UserName:    "testuser",
		AppId:       1, // 关联的应用ID
		Description: "Second Test App Access Key",
	}

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	resp, err := client.CreateAccessKey(ctx, req)
	if err != nil {
		t.Fatalf("为应用创建访问密钥失败: %v", err)
	}

	// 验证返回的访问密钥包含正确的应用ID和描述
	if resp.AppId != 1 {
		t.Fatalf("访问密钥应该关联到应用ID 1，但得到: %d", resp.AppId)
	}
	
	if resp.Description != "Second Test App Access Key" {
		t.Fatalf("访问密钥描述不匹配，期望: %s，得到: %s", "Second Test App Access Key", resp.Description)
	}

	t.Logf("为应用创建访问密钥成功: %+v", resp)
}
