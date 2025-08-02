package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/vera-byte/vgo-iam/internal/auth"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
)

func main() {
	// 连接到gRPC服务器
	conn, err := grpc.Dial("localhost:8899", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("无法连接到服务器: %v", err)
	}
	defer conn.Close()

	// 创建客户端
	client := iamv1.NewIAMClient(conn)

	// 1. 首先创建一个用户
	createUser(client)

	// 2. 为用户创建访问密钥
	accessKey := createAccessKey(client)
	if accessKey == nil {
		log.Fatalf("创建访问密钥失败")
	}

	// 3. 使用访问密钥认证并测试其他API
	ctx := context.Background()
	getUserWithAuth(ctx, client, accessKey)
}

func createUser(client iamv1.IAMClient) string {
	// 创建用户请求
	req := &iamv1.CreateUserRequest{
		Name:        "testuser",
		DisplayName: "Test User",
		Email:       "test@example.com",
	}

	// 发送请求
	resp, err := client.CreateUser(context.Background(), req)
	if err != nil {
		log.Fatalf("创建用户失败: %v", err)
	}

	log.Printf("创建用户成功: %+v", resp)
	return resp.Name
}

func createAccessKey(client iamv1.IAMClient) *iamv1.AccessKey {
	// 创建访问密钥请求
	req := &iamv1.CreateAccessKeyRequest{
		UserName: "testuser",
	}

	// 发送请求
	resp, err := client.CreateAccessKey(context.Background(), req)
	if err != nil {
		log.Fatalf("创建访问密钥失败: %v", err)
	}

	log.Printf("创建访问密钥成功: AccessKeyId=%s, SecretAccessKey=%s", resp.AccessKeyId, resp.SecretAccessKey)
	return resp
}

func getUserWithAuth(ctx context.Context, client iamv1.IAMClient, ak *iamv1.AccessKey) {
	// 准备请求数据
	req := &iamv1.GetUserRequest{
		Name: "testuser",
	}

	// 序列化请求数据
	reqData, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("序列化请求数据失败: %v", err)
	}

	// 生成时间戳，使用与服务端相同的格式
	timestamp := time.Now().Format("20060102T150405Z")

	// 构建待签字符串
	stringToSign := auth.BuildStringToSign(timestamp, string(reqData))

	// 计算签名
	signature := auth.CalculateSignature(stringToSign, ak.SecretAccessKey, timestamp)

	// 添加认证元数据
	md := metadata.Pairs(
		"access-key-id", ak.AccessKeyId,
		"signature", signature,
		"x-iam-date", timestamp,
		"request-data", string(reqData),
	)
	ctx = metadata.NewOutgoingContext(ctx, md)

	// 发送请求
	resp, err := client.GetUser(ctx, req)
	if err != nil {
		log.Fatalf("使用认证获取用户失败: %v", err)
	}

	log.Printf("使用认证获取用户成功: %+v", resp)
}
