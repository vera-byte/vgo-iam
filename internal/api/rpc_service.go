package api

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/vera-byte/vgo-iam/pkg/signature"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vera-byte/vgo-iam/internal/model"
	"github.com/vera-byte/vgo-iam/internal/policy"
	"github.com/vera-byte/vgo-iam/internal/service"
	"github.com/vera-byte/vgo-iam/internal/util"
	iamv1 "github.com/vera-byte/vgo-iam/pkg/proto"
	vgokit "github.com/vera-byte/vgo-kit"
	"github.com/vera-byte/vgo-kit/i18n"
)

type IAMServer struct {
	iamv1.UnimplementedIAMServer
	userService                 *service.UserService
	policyService               *service.PolicyService
	accessKeyService            *service.AccessKeyService
	developerVerificationService service.DeveloperVerificationService
	applicationService          service.ApplicationService
	stsService                  *service.STSService
	policyEngine                *policy.PolicyEngine
	masterKey                   []byte
	translator                  i18n.Translator
}

// AccessKeyService 返回accessKeyService
func (s *IAMServer) AccessKeyService() *service.AccessKeyService {
	return s.accessKeyService
}

// UserService 返回userService
func (s *IAMServer) UserService() *service.UserService {
	return s.userService
}

// PolicyService 返回policyService
func (s *IAMServer) PolicyService() *service.PolicyService {
	return s.policyService
}

// ApplicationService 返回applicationService
func (s *IAMServer) ApplicationService() service.ApplicationService {
	return s.applicationService
}

// STSService 返回stsService
func (s *IAMServer) STSService() *service.STSService {
	return s.stsService
}

func NewIAMServer(
	userService *service.UserService,
	policyService *service.PolicyService,
	accessKeyService *service.AccessKeyService,
	developerVerificationService service.DeveloperVerificationService,
	applicationService service.ApplicationService,
	stsService *service.STSService,
	policyEngine *policy.PolicyEngine,
	masterKey []byte,
	translator i18n.Translator,
) *IAMServer {
	return &IAMServer{
		userService:                 userService,
		policyService:               policyService,
		accessKeyService:            accessKeyService,
		developerVerificationService: developerVerificationService,
		applicationService:          applicationService,
		stsService:                  stsService,
		policyEngine:                policyEngine,
		masterKey:                   masterKey,
		translator:                  translator,
	}
}

func (s *IAMServer) CreateUser(ctx context.Context, req *iamv1.CreateUserRequest) (*iamv1.User, error) {

	vgokit.Log.Info("CreateUser request received", zap.String("username", req.Name))

	// 检查用户是否已存在
	existingUser, err := s.userService.GetUser(ctx, req.Name)
	if err == nil && existingUser != nil {
		return nil, s.translateError(ctx, codes.AlreadyExists, "error.user.already_exists", req.Name)
	}

	user, err := s.userService.CreateUser(ctx, req.Name, req.DisplayName, req.Email)
	if err != nil {
		vgokit.Log.Error("Failed to create user", zap.Error(err))
		return nil, s.translateError(ctx, codes.Internal, "error.user.create_failed", err)
	}

	vgokit.Log.Info("User created successfully", zap.String("username", user.Name), zap.Int64("user_id", user.ID))
	return convertUserToProto(user), nil
}

func (s *IAMServer) GetUser(ctx context.Context, req *iamv1.GetUserRequest) (*iamv1.User, error) {
	user, err := s.userService.GetUser(ctx, req.Name)
	if err != nil {
		return nil, s.translateError(ctx, codes.NotFound, "error.user.not_found", req.Name)
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
		return nil, s.translateError(ctx, codes.Internal, "error.policy.create_failed", err)
	}
	return convertPolicyToProto(policy), nil
}

func (s *IAMServer) AttachUserPolicy(ctx context.Context, req *iamv1.AttachUserPolicyRequest) (*iamv1.AttachUserPolicyResponse, error) {
	if err := s.userService.AttachPolicy(ctx, req.UserName, req.PolicyName); err != nil {
		return nil, s.translateError(ctx, codes.Internal, "error.policy.attach_failed", err)
	}
	return &iamv1.AttachUserPolicyResponse{Success: true}, nil
}

func (s *IAMServer) CreateAccessKey(ctx context.Context, req *iamv1.CreateAccessKeyRequest) (*iamv1.AccessKey, error) {
	user, err := s.userService.GetUser(ctx, req.UserName)
	if err != nil {
		return nil, s.translateError(ctx, codes.NotFound, "error.user.not_found", req.UserName)
	}

	var ak *model.AccessKey
	
	// 如果指定了应用ID，创建应用专用访问密钥
	if req.AppId > 0 {
		ak, err = s.accessKeyService.CreateAccessKeyForApp(ctx, user.Name, req.AppId, req.Description)
	} else {
		// 否则创建通用访问密钥（已废弃，但保持向后兼容）
		ak, err = s.accessKeyService.CreateAccessKey(ctx, user.Name)
	}
	
	if err != nil {
		return nil, s.translateError(ctx, codes.Internal, "error.access_key.create_failed", err)
	}

	response := &iamv1.AccessKey{
		AccessKeyId:     ak.AccessKeyID,
		SecretAccessKey: ak.SecretAccessKey,
		Status:          ak.Status,
		UserName:        user.Name,
		Description:     ak.Description,
		CreatedAt:       convertTimeToTimestamp(ak.CreatedAt),
	}
	
	// 如果有应用ID，添加到响应中
	if ak.AppID != nil {
		response.AppId = *ak.AppID
	}
	
	return response, nil
}

func (s *IAMServer) ListAccessKeys(ctx context.Context, req *iamv1.ListAccessKeysRequest) (*iamv1.ListAccessKeysResponse, error) {
	user, err := s.userService.GetUser(ctx, req.UserName)
	if err != nil {
		return nil, s.translateError(ctx, codes.NotFound, "error.user.not_found", req.UserName)
	}

	keys, err := s.accessKeyService.ListAccessKeys(ctx, user.Name)
	if err != nil {
		return nil, s.translateError(ctx, codes.Internal, "error.access_key.list_failed", err)
	}

	resp := &iamv1.ListAccessKeysResponse{}
	for _, key := range keys {
		// 如果请求中指定了应用ID，只返回该应用的访问密钥
		if req.AppId > 0 {
			if key.AppID == nil || *key.AppID != req.AppId {
				continue
			}
		}
		
		accessKey := &iamv1.AccessKey{
			AccessKeyId: key.AccessKeyID,
			Status:      key.Status,
			UserName:    user.Name,
			Description: key.Description,
			CreatedAt:   convertTimeToTimestamp(key.CreatedAt),
			UpdatedAt:   convertTimeToTimestamp(key.UpdatedAt),
		}
		
		// 如果有应用ID，添加到响应中
		if key.AppID != nil {
			accessKey.AppId = *key.AppID
		}
		
		resp.AccessKeys = append(resp.AccessKeys, accessKey)
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
		return nil, s.translateError(ctx, codes.NotFound, "error.access_key.not_found", req.AccessKeyId)
	}

	// 调用服务层更新状态
	updatedKey, err := s.accessKeyService.UpdateStatus(ctx, req.AccessKeyId, req.Status)
	if err != nil {
		return nil, s.translateError(ctx, codes.Internal, "error.access_key.update_failed", err)
	}

	// 获取关联用户信息
	user, err := s.userService.GetUser(ctx, ak.UserName)
	if err != nil {
		return nil, s.translateError(ctx, codes.NotFound, "error.user.not_found", ak.UserName)
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
	var (
		AccessKeyId = vgokit.GetMetadataValue(ctx, "access-key-id")
		sign        = vgokit.GetMetadataValue(ctx, "signature")
		timestamp   = vgokit.GetMetadataValue(ctx, "x-iam-date")
		requestData = vgokit.GetMetadataValue(ctx, "request-data")
	)
	// 1. 获取访问密钥
	ak, err := s.accessKeyService.GetAccessKey(ctx, AccessKeyId)
	if err != nil {
		return nil, s.translateError(ctx, codes.NotFound, "error.access_key.invalid")
	}

	// 2. 验证密钥状态
	if ak.Status != "active" {
		return nil, s.translateError(ctx, codes.InvalidArgument, "error.access_key.inactive")
	}

	// 3. 验证签名
	valid, err := signature.VerifySignV4(sign, requestData, timestamp, ak.SecretAccessKey)
	if err != nil || !valid {
		return nil, s.translateError(ctx, codes.Unauthenticated, "error.auth.signature_verification_failed")
	}

	// 4. 获取用户名
	user, err := s.userService.GetUser(ctx, ak.UserName)
	if err != nil {
		return nil, s.translateError(ctx, codes.NotFound, "error.user.not_found", ak.UserName)
	}

	return &iamv1.VerifyResponse{
		Valid:    true,
		UserName: user.Name,
	}, nil
}

func (s *IAMServer) CheckPermission(ctx context.Context, req *iamv1.CheckPermissionRequest) (*iamv1.CheckPermissionResponse, error) {
	user, err := s.userService.GetUser(ctx, req.UserName)
	if err != nil {
		return nil, s.translateError(ctx, codes.NotFound, "error.user.not_found", req.UserName)
	}

	allowed, err := s.policyEngine.Evaluate(user, req.Action, req.Resource)
	if err != nil {
		return nil, s.translateError(ctx, codes.Internal, "error.permission.check_failed", err)
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

// SubmitDeveloperVerification 提交开发者认证
func (s *IAMServer) SubmitDeveloperVerification(ctx context.Context, req *iamv1.SubmitDeveloperVerificationRequest) (*iamv1.DeveloperVerification, error) {
	// 从context中获取用户信息（假设已通过认证中间件设置）
	// 这里暂时使用固定用户ID，实际应该从认证上下文获取
	userID := int64(1) // TODO: 从认证上下文获取用户ID

	// 转换开发者类型
	developerType, err := convertDeveloperType(req.DeveloperType)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid developer type: %v", err)
	}

	// 构建服务层请求
	serviceReq := &service.SubmitVerificationRequest{
		UserID:        userID,
		DeveloperType: developerType,
	}

	// 根据开发者类型设置相应字段
	if developerType == model.DeveloperTypeIndividual {
		serviceReq.RealName = &req.RealName
		serviceReq.IDCardNumber = &req.IdCardNumber
		serviceReq.IDCardFrontURL = &req.IdCardFrontUrl
		serviceReq.IDCardBackURL = &req.IdCardBackUrl
	} else if developerType == model.DeveloperTypeEnterprise {
		serviceReq.CompanyName = &req.CompanyName
		serviceReq.BusinessLicenseNumber = &req.BusinessLicenseNumber
		serviceReq.BusinessLicenseURL = &req.BusinessLicenseUrl
		serviceReq.LegalRepresentative = &req.LegalRepresentative
		serviceReq.CompanyAddress = &req.CompanyAddress
	}

	// 调用服务层
	verification, err := s.developerVerificationService.SubmitVerification(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to submit verification: %v", err)
	}

	// 转换为proto消息
	return convertVerificationToProto(verification, "user"), nil
}

// GetDeveloperVerification 获取开发者认证信息
func (s *IAMServer) GetDeveloperVerification(ctx context.Context, req *iamv1.GetDeveloperVerificationRequest) (*iamv1.DeveloperVerification, error) {
	// 根据用户名获取用户信息
	user, err := s.userService.GetUser(ctx, req.UserName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	// 转换开发者类型
	developerType, err := convertDeveloperType(req.DeveloperType)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid developer type: %v", err)
	}

	// 调用服务层
	verification, err := s.developerVerificationService.GetVerification(ctx, user.ID, developerType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get verification: %v", err)
	}

	if verification == nil {
		return nil, status.Errorf(codes.NotFound, "verification not found")
	}

	// 转换为proto消息
	return convertVerificationToProto(verification, user.Name), nil
}

// ListDeveloperVerifications 获取开发者认证列表
func (s *IAMServer) ListDeveloperVerifications(ctx context.Context, req *iamv1.ListDeveloperVerificationsRequest) (*iamv1.ListDeveloperVerificationsResponse, error) {
	// 转换状态
	verificationStatus, err := convertVerificationStatus(req.Status)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status: %v", err)
	}

	// 调用服务层
	verifications, total, err := s.developerVerificationService.ListVerifications(ctx, verificationStatus, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list verifications: %v", err)
	}

	// 转换为proto消息
	protoVerifications := make([]*iamv1.DeveloperVerification, len(verifications))
	for i, v := range verifications {
		protoVerifications[i] = convertVerificationToProto(v, "user")
	}

	return &iamv1.ListDeveloperVerificationsResponse{
		Verifications: protoVerifications,
		Total:         int32(total),
		Page:          req.Page,
		PageSize:      req.PageSize,
	}, nil
}

// ReviewDeveloperVerification 审核开发者认证
func (s *IAMServer) ReviewDeveloperVerification(ctx context.Context, req *iamv1.ReviewDeveloperVerificationRequest) (*iamv1.DeveloperVerification, error) {
	// 转换状态
	verificationStatus, err := convertVerificationStatus(req.Status)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status: %v", err)
	}

	// 构建服务层请求
	serviceReq := &service.ReviewVerificationRequest{
		VerificationID: req.VerificationId,
		ReviewerID:     1, // TODO: 从认证上下文获取审核员ID
		Status:         verificationStatus,
		Comment:        req.ReviewComment,
	}

	// 调用服务层
	err = s.developerVerificationService.ReviewVerification(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to review verification: %v", err)
	}

	// 获取更新后的认证信息 - 需要先实现GetVerificationByID方法
	// verification, err := s.developerVerificationService.GetVerificationByID(ctx, req.VerificationId)
	// if err != nil {
	//	return nil, status.Errorf(codes.Internal, "failed to get updated verification: %v", err)
	// }

	// 暂时返回空的认证信息
	return &iamv1.DeveloperVerification{}, nil
}

// CreateApplication 创建应用
func (s *IAMServer) CreateApplication(ctx context.Context, req *iamv1.CreateApplicationRequest) (*iamv1.Application, error) {
	// 从context中获取用户信息
	userID := int64(1) // TODO: 从认证上下文获取用户ID

	// 构建服务层请求
	serviceReq := &service.CreateApplicationRequest{
		UserID:         userID,
		AppName:        req.AppName,
		AppDescription: req.AppDescription,
		AppType:        req.AppType,
		CallbackURLs:   req.CallbackUrls,
		AllowedOrigins: req.AllowedOrigins,
	}

	// 调用服务层
	app, err := s.applicationService.CreateApplication(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create application: %v", err)
	}

	// 转换为proto消息
	return convertApplicationToProto(app, "user"), nil
}

// GetApplication 获取应用信息
func (s *IAMServer) GetApplication(ctx context.Context, req *iamv1.GetApplicationRequest) (*iamv1.Application, error) {
	// 调用服务层
	app, err := s.applicationService.GetApplication(ctx, req.AppId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get application: %v", err)
	}

	if app == nil {
		return nil, status.Errorf(codes.NotFound, "application not found")
	}

	// 转换为proto消息
	return convertApplicationToProto(app, "user"), nil
}

// ListApplications 获取应用列表
func (s *IAMServer) ListApplications(ctx context.Context, req *iamv1.ListApplicationsRequest) (*iamv1.ListApplicationsResponse, error) {
	// 根据用户名获取用户信息
	user, err := s.userService.GetUser(ctx, req.UserName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	// 转换状态
	appStatus, err := convertAppStatus(req.Status)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status: %v", err)
	}

	// 调用服务层
	apps, total, err := s.applicationService.ListApplications(ctx, user.ID, appStatus, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list applications: %v", err)
	}

	// 转换为proto消息
	protoApps := make([]*iamv1.Application, len(apps))
	for i, app := range apps {
		protoApps[i] = convertApplicationToProto(app, user.Name)
	}

	return &iamv1.ListApplicationsResponse{
		Applications: protoApps,
		Total:        int32(total),
		Page:         req.Page,
		PageSize:     req.PageSize,
	}, nil
}

// UpdateApplication 更新应用信息
func (s *IAMServer) UpdateApplication(ctx context.Context, req *iamv1.UpdateApplicationRequest) (*iamv1.Application, error) {
	// 从context中获取用户信息
	userID := int64(1) // TODO: 从认证上下文获取用户ID

	// 构建服务层请求
	serviceReq := &service.UpdateApplicationRequest{
		ID:             req.AppId,
		UserID:         userID,
		AppName:        req.AppName,
		AppDescription: req.AppDescription,
		AppType:        req.AppType,
		CallbackURLs:   req.CallbackUrls,
		AllowedOrigins: req.AllowedOrigins,
	}

	// 调用服务层
	err := s.applicationService.UpdateApplication(ctx, serviceReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update application: %v", err)
	}

	// 获取更新后的应用信息
	app, err := s.applicationService.GetApplication(ctx, req.AppId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated application: %v", err)
	}

	// 转换为proto消息
	return convertApplicationToProto(app, "user"), nil
}

// DeleteApplication 删除应用
func (s *IAMServer) DeleteApplication(ctx context.Context, req *iamv1.DeleteApplicationRequest) (*iamv1.DeleteApplicationResponse, error) {
	// 从context中获取用户信息
	userID := int64(1) // TODO: 从认证上下文获取用户ID

	// 调用服务层
	err := s.applicationService.DeleteApplication(ctx, req.AppId, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete application: %v", err)
	}

	return &iamv1.DeleteApplicationResponse{
		Success: true,
	}, nil
}

// 辅助函数：转换开发者类型
func convertDeveloperType(protoType string) (model.DeveloperType, error) {
	switch protoType {
	case "individual":
		return model.DeveloperTypeIndividual, nil
	case "enterprise":
		return model.DeveloperTypeEnterprise, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid developer type: %s", protoType)
	}
}

// 辅助函数：转换认证状态
func convertVerificationStatus(protoStatus string) (model.VerificationStatus, error) {
	switch protoStatus {
	case "pending":
		return model.VerificationStatusPending, nil
	case "approved":
		return model.VerificationStatusApproved, nil
	case "rejected":
		return model.VerificationStatusRejected, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid verification status: %s", protoStatus)
	}
}

// 辅助函数：转换应用状态
func convertAppStatus(protoStatus string) (model.AppStatus, error) {
	switch protoStatus {
	case "active":
		return model.AppStatusActive, nil
	case "inactive":
		return model.AppStatusInactive, nil
	case "suspended":
		return model.AppStatusSuspended, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid app status: %s", protoStatus)
	}
}

// 辅助函数：转换认证信息为proto消息
func convertVerificationToProto(v *model.DeveloperVerification, userName string) *iamv1.DeveloperVerification {
	proto := &iamv1.DeveloperVerification{
		Id:            v.ID,
		UserName:      userName,
		DeveloperType: string(v.DeveloperType),
		Status:        string(v.Status),
		CreatedAt:     convertTimeToTimestamp(v.CreatedAt),
		UpdatedAt:     convertTimeToTimestamp(v.UpdatedAt),
	}

	// 设置个人开发者信息
	if v.RealName != nil {
		proto.RealName = *v.RealName
	}
	if v.IDCardNumber != nil {
		proto.IdCardNumber = *v.IDCardNumber
	}
	if v.IDCardFrontURL != nil {
		proto.IdCardFrontUrl = *v.IDCardFrontURL
	}
	if v.IDCardBackURL != nil {
		proto.IdCardBackUrl = *v.IDCardBackURL
	}

	// 设置企业开发者信息
	if v.CompanyName != nil {
		proto.CompanyName = *v.CompanyName
	}
	if v.BusinessLicenseNumber != nil {
		proto.BusinessLicenseNumber = *v.BusinessLicenseNumber
	}
	if v.BusinessLicenseURL != nil {
		proto.BusinessLicenseUrl = *v.BusinessLicenseURL
	}
	if v.LegalRepresentative != nil {
		proto.LegalRepresentative = *v.LegalRepresentative
	}
	if v.CompanyAddress != nil {
		proto.CompanyAddress = *v.CompanyAddress
	}

	// 设置审核信息
	if v.ReviewComment != nil {
		proto.ReviewComment = *v.ReviewComment
	}
	if v.ReviewedAt != nil {
		proto.ReviewedAt = convertTimeToTimestamp(*v.ReviewedAt)
	}

	return proto
}

// 辅助函数：转换应用信息为proto消息
func convertApplicationToProto(app *model.Application, userName string) *iamv1.Application {
	return &iamv1.Application{
		Id:             app.ID,
		UserName:       userName,
		AppName:        app.AppName,
		AppDescription: app.AppDescription,
		AppType:        string(app.AppType),
		AppIconUrl:     app.AppIconURL,
		AppWebsite:     app.AppWebsite,
		CallbackUrls:   []string(app.CallbackURLs),
		AllowedOrigins: []string(app.AllowedOrigins),
		Status:         string(app.Status),
		CreatedAt:      convertTimeToTimestamp(app.CreatedAt),
		UpdatedAt:      convertTimeToTimestamp(app.UpdatedAt),
	}
}

// 辅助函数：处理国际化错误
// ctx: 上下文，用于获取语言信息
// code: gRPC状态码
// key: 翻译键
// args: 翻译参数
// 返回: gRPC错误
func (s *IAMServer) translateError(ctx context.Context, code codes.Code, key string, args ...interface{}) error {
	// 设置翻译器的语言（从上下文获取）
	s.translator.SetLanguage(i18n.GetLanguageFromContext(ctx))
	message := s.translator.Translate(key, args...)
	return status.Error(code, message)
}

// 辅助函数：获取国际化消息
// ctx: 上下文，用于获取语言信息
// key: 翻译键
// args: 翻译参数
// 返回: 翻译后的消息
func (s *IAMServer) translateMessage(ctx context.Context, key string, args ...interface{}) string {
	// 设置翻译器的语言（从上下文获取）
	s.translator.SetLanguage(i18n.GetLanguageFromContext(ctx))
	return s.translator.Translate(key, args...)
}

// STS相关的RPC方法实现

// GetSessionToken 获取会话令牌
// ctx: 上下文
// req: 获取会话令牌请求
// 返回: 获取会话令牌响应和错误信息
func (s *IAMServer) GetSessionToken(ctx context.Context, req *iamv1.GetSessionTokenRequest) (*iamv1.GetSessionTokenResponse, error) {
	vgokit.Log.Info("GetSessionToken request received", zap.Int32("duration_seconds", req.DurationSeconds))

	// 调用STS服务
	resp, err := s.stsService.GetSessionToken(ctx, req)
	if err != nil {
		vgokit.Log.Error("Failed to get session token", zap.Error(err))
		return nil, s.translateError(ctx, codes.Internal, "error.sts.get_session_token_failed", err)
	}

	vgokit.Log.Info("Session token created successfully", zap.String("access_key_id", resp.Credentials.AccessKeyId))
	return resp, nil
}

// AssumeRole 扮演角色
// ctx: 上下文
// req: 扮演角色请求
// 返回: 扮演角色响应和错误信息
func (s *IAMServer) AssumeRole(ctx context.Context, req *iamv1.AssumeRoleRequest) (*iamv1.AssumeRoleResponse, error) {
	vgokit.Log.Info("AssumeRole request received", 
		zap.String("role_arn", req.RoleArn),
		zap.String("role_session_name", req.RoleSessionName))

	// 调用STS服务
	resp, err := s.stsService.AssumeRole(ctx, req)
	if err != nil {
		vgokit.Log.Error("Failed to assume role", zap.Error(err))
		return nil, s.translateError(ctx, codes.Internal, "error.sts.assume_role_failed", err)
	}

	vgokit.Log.Info("Role assumed successfully", 
		zap.String("access_key_id", resp.Credentials.AccessKeyId),
		zap.String("assumed_role_id", resp.AssumedRoleUser.AssumedRoleId))
	return resp, nil
}

// RefreshToken 刷新令牌
// ctx: 上下文
// req: 刷新令牌请求
// 返回: 刷新令牌响应和错误信息
func (s *IAMServer) RefreshToken(ctx context.Context, req *iamv1.RefreshTokenRequest) (*iamv1.RefreshTokenResponse, error) {
	vgokit.Log.Info("RefreshToken request received", zap.String("session_token", req.SessionToken))

	// 调用STS服务
	resp, err := s.stsService.RefreshToken(ctx, req)
	if err != nil {
		vgokit.Log.Error("Failed to refresh token", zap.Error(err))
		return nil, s.translateError(ctx, codes.Internal, "error.sts.refresh_token_failed", err)
	}

	vgokit.Log.Info("Token refreshed successfully", zap.String("access_key_id", resp.Credentials.AccessKeyId))
	return resp, nil
}

// RevokeToken 撤销令牌
// ctx: 上下文
// req: 撤销令牌请求
// 返回: 撤销令牌响应和错误信息
func (s *IAMServer) RevokeToken(ctx context.Context, req *iamv1.RevokeTokenRequest) (*iamv1.RevokeTokenResponse, error) {
	vgokit.Log.Info("RevokeToken request received", zap.String("session_token", req.SessionToken))

	// 调用STS服务
	resp, err := s.stsService.RevokeToken(ctx, req)
	if err != nil {
		vgokit.Log.Error("Failed to revoke token", zap.Error(err))
		return nil, s.translateError(ctx, codes.Internal, "error.sts.revoke_token_failed", err)
	}

	vgokit.Log.Info("Token revoked successfully")
	return resp, nil
}

// Dashboard相关方法实现

// GetDashboardStats 获取仪表板统计数据
func (s *IAMServer) GetDashboardStats(ctx context.Context, req *iamv1.DashboardStatsRequest) (*iamv1.DashboardStatsResponse, error) {
	vgokit.Log.Info("GetDashboardStats request received")

	// 获取用户总数
	usersCount, err := s.userService.GetUsersCount(ctx)
	if err != nil {
		vgokit.Log.Error("Failed to get users count", zap.Error(err))
		usersCount = 0
	}

	// 获取访问密钥总数
	accessKeysCount, err := s.accessKeyService.GetAccessKeysCount(ctx)
	if err != nil {
		vgokit.Log.Error("Failed to get access keys count", zap.Error(err))
		accessKeysCount = 0
	}

	// 获取策略总数
	policiesCount, err := s.policyService.GetPoliciesCount(ctx)
	if err != nil {
		vgokit.Log.Error("Failed to get policies count", zap.Error(err))
		policiesCount = 0
	}

	// 获取应用总数
	applicationsCount, err := s.applicationService.GetApplicationsCount(ctx)
	if err != nil {
		vgokit.Log.Error("Failed to get applications count", zap.Error(err))
		applicationsCount = 0
	}

	return &iamv1.DashboardStatsResponse{
		Users:        int32(usersCount),
		AccessKeys:   int32(accessKeysCount),
		Policies:     int32(policiesCount),
		Applications: int32(applicationsCount),
	}, nil
}

// GetDashboardStatus 获取系统状态
func (s *IAMServer) GetDashboardStatus(ctx context.Context, req *iamv1.DashboardStatusRequest) (*iamv1.DashboardStatusResponse, error) {
	vgokit.Log.Info("GetDashboardStatus request received")

	// 检查数据库连接状态
	databaseStatus := "connected"
	if err := s.userService.HealthCheck(ctx); err != nil {
		vgokit.Log.Error("Database health check failed", zap.Error(err))
		databaseStatus = "disconnected"
	}

	// 获取系统运行时间（简化实现）
	uptime := "running"

	// 获取版本信息
	version := "v1.0.0"

	return &iamv1.DashboardStatusResponse{
		ServiceStatus:  "healthy",
		DatabaseStatus: databaseStatus,
		Uptime:         uptime,
		Version:        version,
	}, nil
}

// GetDashboardActivities 获取最近活动
func (s *IAMServer) GetDashboardActivities(ctx context.Context, req *iamv1.DashboardActivitiesRequest) (*iamv1.DashboardActivitiesResponse, error) {
	vgokit.Log.Info("GetDashboardActivities request received")

	// 设置默认限制
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	// 这里返回模拟数据，实际项目中应该从数据库或日志系统获取
	activities := []*iamv1.Activity{
		{
			Id:          "1",
			Type:        "user_created",
			Description: "Created new user",
			User:        "admin",
			Timestamp:   timestamppb.New(time.Now().Add(-30 * time.Minute)),
		},
		{
			Id:          "2",
			Type:        "key_created",
			Description: "Created new access key",
			User:        "admin",
			Timestamp:   timestamppb.New(time.Now().Add(-2 * time.Hour)),
		},
		{
			Id:          "3",
			Type:        "policy_updated",
			Description: "Updated policy",
			User:        "admin",
			Timestamp:   timestamppb.New(time.Now().Add(-4 * time.Hour)),
		},
	}

	// 限制返回数量
	if int(limit) < len(activities) {
		activities = activities[:limit]
	}

	return &iamv1.DashboardActivitiesResponse{
		Activities: activities,
	}, nil
}
