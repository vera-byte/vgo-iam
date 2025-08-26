package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	"github.com/vera-byte/vgo-iam/pkg/signature"
)

func main() {
	// 连接到gRPC服务器
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("无法连接到服务器: %v", err)
	}
	defer conn.Close()

	// 创建客户端
	client := iamv1.NewIAMClient(conn)

	// 1. 为admin用户创建访问密钥（无需认证）
	// 注意：这需要admin用户已经存在，或者我们可以先创建admin用户
	// 由于CreateUser需要认证，我们需要先创建一个初始用户

	// 让我们先尝试为admin用户创建访问密钥
	adminAccessKey := createAccessKeyForAdmin(client, "admin")
	if adminAccessKey == nil {
		log.Fatalf("为admin用户创建访问密钥失败，admin用户可能不存在")
	}
	log.Printf("admin访问密钥创建成功: %+v", adminAccessKey)

	// 2. 使用admin访问密钥创建测试用户
	user := createUserWithAuth(client, adminAccessKey)
	if user == "" {
		log.Fatalf("创建测试用户失败")
	}
	log.Printf("测试用户创建成功: %+v", user)

	// 3. 使用admin访问密钥为测试用户创建访问密钥
	userAccessKey := createAccessKeyForUser(client, "testuser", adminAccessKey)
	if userAccessKey == nil {
		log.Fatalf("为测试用户创建访问密钥失败")
	}
	log.Printf("测试用户访问密钥创建成功: %+v", userAccessKey)

	// 4. 使用测试用户的访问密钥获取用户信息
	ctx := context.Background()
	getUserWithAuth(ctx, client, userAccessKey)
}

// 创建管理员用户
func createAccessKeyForAdmin(client iamv1.IAMClient, userName string) *iamv1.AccessKey {
	req := &iamv1.CreateAccessKeyRequest{
		UserName: userName,
	}

	// 创建访问密钥（无需认证）
	ctx := context.Background()
	resp, err := client.CreateAccessKey(ctx, req)
	if err != nil {
		log.Printf("创建访问密钥失败: %v", err)
		return nil
	}

	log.Printf("访问密钥创建成功: AccessKeyId=%s, SecretAccessKey=%s", resp.AccessKeyId, resp.SecretAccessKey)
	return resp
}

func createUserWithAuth(client iamv1.IAMClient, accessKey *iamv1.AccessKey) string {
	req := &iamv1.CreateUserRequest{
		Name:        "testuser",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}

	// 使用管理员访问密钥创建用户
	ctx := createAuthContext(accessKey, req)
	resp, err := client.CreateUser(ctx, req)
	if err != nil {
		log.Fatalf("创建用户失败: %v", err)
	}

	log.Printf("创建用户成功: %+v", resp)
	return "testuser"
}

func createAccessKeyForUser(client iamv1.IAMClient, userName string, accessKey *iamv1.AccessKey) *iamv1.AccessKey {
	req := &iamv1.CreateAccessKeyRequest{
		UserName: userName,
	}

	// 使用管理员访问密钥为用户创建访问密钥
	ctx := createAuthContext(accessKey, req)
	resp, err := client.CreateAccessKey(ctx, req)
	if err != nil {
		log.Fatalf("为用户创建访问密钥失败: %v", err)
	}

	log.Printf("为用户创建访问密钥成功: %+v", resp)
	return resp
}

func createAuthContext(accessKey *iamv1.AccessKey, req interface{}) context.Context {
	// 序列化请求数据
	reqData, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("序列化请求数据失败: %v", err)
	}

	signer := signature.SignV4(string(reqData), accessKey.AccessKeyId, accessKey.SecretAccessKey)

	// 添加认证元数据
	md := metadata.Pairs(
		"access-key-id", signer.AccessKeyID,
		"signature", signer.Signature,
		"x-iam-date", strconv.FormatInt(signer.Timestamp, 10),
		"request-data", string(reqData),
	)
	return metadata.NewOutgoingContext(context.Background(), md)
}

func getUserWithAuth(ctx context.Context, client iamv1.IAMClient, ak *iamv1.AccessKey) {
	req := &iamv1.GetUserRequest{
		Name: "testuser",
	}

	// 使用认证调用GetUser
	ctx = createAuthContext(ak, req)
	resp, err := client.GetUser(ctx, req)
	if err != nil {
		log.Fatalf("使用认证获取用户失败: %v", err)
	}

	log.Printf("使用认证获取用户成功: %+v", resp)
}
