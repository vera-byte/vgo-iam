package api

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vera-byte/vgo-iam/internal/auth"
	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/util"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
)

type IAMServer struct {
	iamv1.UnimplementedIAMServer
	userService      *service.UserService
	policyService    *service.PolicyService
	accessKeyService *service.AccessKeyService
	policyEngine     *policy.PolicyEngine
	masterKey        []byte
}

func NewIAMServer(
	userService *service.UserService,
	policyService *service.PolicyService,
	accessKeyService *service.AccessKeyService,
	policyEngine *policy.PolicyEngine,
	masterKey []byte,
) *IAMServer {
	return &IAMServer{
		userService:      userService,
		policyService:    policyService,
		accessKeyService: accessKeyService,
		policyEngine:     policyEngine,
		masterKey:        masterKey,
	}
}

func (s *IAMServer) CreateUser(ctx context.Context, req *iamv1.CreateUserRequest) (*iamv1.User, error) {
	reqID := util.GenerateRequestID()
	logger := util.WithRequestID(util.Logger, reqID)
	logger.Info("CreateUser request received", zap.String("username", req.Name))

	user, err := s.userService.CreateUser(ctx, req.Name, req.DisplayName, req.Email)
	if err != nil {
		logger.Error("Failed to create user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	logger.Info("User created successfully", zap.String("username", user.Name), zap.Int("user_id", user.ID))
	return convertUserToProto(user), nil
}

func (s *IAMServer) GetUser(ctx context.Context, req *iamv1.GetUserRequest) (*iamv1.User, error) {
	user, err := s.userService.GetUser(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}
	return convertUserToProto(user), nil
}

func (s *IAMServer) CreatePolicy(ctx context.Context, req *iamv1.CreatePolicyRequest) (*iamv1.Policy, error) {
	// 验证策略文档
	if !util.ValidatePolicyDocument(req.PolicyDocument) {
		return nil, status.Error(codes.InvalidArgument, "invalid policy document")
	}

	policy, err := s.policyService.CreatePolicy(ctx, req.Name, req.Description, req.PolicyDocument)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create policy: %v", err)
	}
	return convertPolicyToProto(policy), nil
}

func (s *IAMServer) AttachUserPolicy(ctx context.Context, req *iamv1.AttachUserPolicyRequest) (*iamv1.AttachUserPolicyResponse, error) {
	if err := s.userService.AttachPolicy(ctx, req.UserName, req.PolicyName); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to attach policy: %v", err)
	}
	return &iamv1.AttachUserPolicyResponse{Success: true}, nil
}

func (s *IAMServer) CreateAccessKey(ctx context.Context, req *iamv1.CreateAccessKeyRequest) (*iamv1.AccessKey, error) {
	user, err := s.userService.GetUser(ctx, req.UserName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	ak, err := s.accessKeyService.CreateAccessKey(ctx, user.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create access key: %v", err)
	}

	return &iamv1.AccessKey{
		AccessKeyId:     ak.AccessKeyID,
		SecretAccessKey: ak.SecretAccessKey,
		Status:          ak.Status,
		UserName:        user.Name,
		CreatedAt:       convertTimeToTimestamp(ak.CreatedAt),
	}, nil
}

func (s *IAMServer) ListAccessKeys(ctx context.Context, req *iamv1.ListAccessKeysRequest) (*iamv1.ListAccessKeysResponse, error) {
	user, err := s.userService.GetUser(ctx, req.UserName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	keys, err := s.accessKeyService.ListAccessKeys(ctx, user.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list access keys: %v", err)
	}

	resp := &iamv1.ListAccessKeysResponse{}
	for _, key := range keys {
		resp.AccessKeys = append(resp.AccessKeys, &iamv1.AccessKey{
			AccessKeyId: key.AccessKeyID,
			Status:      key.Status,
			UserName:    user.Name,
			CreatedAt:   convertTimeToTimestamp(key.CreatedAt),
			UpdatedAt:   convertTimeToTimestamp(key.UpdatedAt),
		})
	}
	return resp, nil
}

func (s *IAMServer) UpdateAccessKeyStatus(ctx context.Context, req *iamv1.UpdateAccessKeyStatusRequest) (*iamv1.AccessKey, error) {
	// 参数验证
	if req.Status != "active" && req.Status != "inactive" {
		return nil, status.Error(codes.InvalidArgument, "status must be either 'active' or 'inactive'")
	}

	// 先获取访问密钥信息
	ak, err := s.accessKeyService.GetAccessKey(ctx, req.AccessKeyId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "access key not found: %v", err)
	}

	// 调用服务层更新状态
	updatedKey, err := s.accessKeyService.UpdateStatus(ctx, req.AccessKeyId, req.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update access key status: %v", err)
	}

	// 获取关联用户信息
	user, err := s.userService.GetUser(ctx, ak.UserName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "associated user not found: %v", err)
	}

	// 构造返回响应
	return &iamv1.AccessKey{
		AccessKeyId: updatedKey.AccessKeyID,
		Status:      updatedKey.Status,
		UserName:    user.Name,
		UpdatedAt:   convertTimeToTimestamp(updatedKey.UpdatedAt),
	}, nil
}

func (s *IAMServer) VerifyAccessKey(ctx context.Context, req *iamv1.VerifyRequest) (*iamv1.VerifyResponse, error) {
	// 1. 获取访问密钥
	ak, err := s.accessKeyService.GetAccessKey(ctx, req.AccessKeyId)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid access key")
	}

	// 2. 验证密钥状态
	if ak.Status != "active" {
		return nil, status.Errorf(codes.PermissionDenied, "access key is inactive")
	}

	// 3. 验证签名
	valid, err := auth.VerifySignatureV4(req.Signature, req.RequestData, req.Timestamp, ak.SecretAccessKey)
	if err != nil || !valid {
		return nil, status.Errorf(codes.Unauthenticated, "signature verification failed")
	}

	// 4. 获取用户名
	user, err := s.userService.GetUser(ctx, ak.UserName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	return &iamv1.VerifyResponse{
		Valid:    true,
		UserName: user.Name,
	}, nil
}

func (s *IAMServer) CheckPermission(ctx context.Context, req *iamv1.CheckPermissionRequest) (*iamv1.CheckPermissionResponse, error) {
	user, err := s.userService.GetUser(ctx, req.UserName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	allowed, err := s.policyEngine.Evaluate(user, req.Action, req.Resource)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "permission check failed")
	}

	return &iamv1.CheckPermissionResponse{Allowed: allowed}, nil
}

// 辅助函数：转换时间到Timestamp
func convertTimeToTimestamp(t time.Time) *timestamppb.Timestamp {
	ts, _ := ptypes.TimestampProto(t)
	return ts
}

// 辅助函数：转换User到proto格式
func convertUserToProto(user *model.User) *iamv1.User {
	return &iamv1.User{
		Id:          int64(user.ID),
		Name:        user.Name,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		CreatedAt:   convertTimeToTimestamp(user.CreatedAt),
		UpdatedAt:   convertTimeToTimestamp(user.UpdatedAt),
	}
}

// 辅助函数：转换Policy到proto格式
func convertPolicyToProto(policy *model.Policy) *iamv1.Policy {
	return &iamv1.Policy{
		Id:             int64(policy.ID),
		Name:           policy.Name,
		Description:    policy.Description,
		PolicyDocument: policy.PolicyDocument,
		CreatedAt:      convertTimeToTimestamp(policy.CreatedAt),
		UpdatedAt:      convertTimeToTimestamp(policy.UpdatedAt),
	}
}
