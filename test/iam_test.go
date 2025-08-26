package test

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"testing"

	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	"github.com/vera-byte/vgo-iam/pkg/signature"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	AccessKeyID     = "rqMXIc39E32rUNHbXmgE"
	SecretAccessKey = "CcYKlMCFIMdTGPqeEZ2YdG59hLuM40PfyFoZbKRW"
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

// 为用户创建accesskey
func TestCreateUserAccessKey(t *testing.T) {
	client, conn, err := NewTestIAMClient(t)
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}

	req := &iamv1.CreateAccessKeyRequest{
		UserName: "testuser",
	}
	defer conn.Close()

	ctx, _ := SignV4Ctx(t.Context(), AccessKeyID, SecretAccessKey, req)

	access, err := client.CreateAccessKey(ctx, req)
	if err != nil {
		t.Fatalf("创建访问密钥失败: %v", err)
	}

	if access == nil {
		t.Fatalf("为admin用户创建访问密钥失败，admin用户可能不存在")
	}
	t.Logf("admin访问密钥创建成功: %+v", access)
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
