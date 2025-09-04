# VGO-IAM gRPC Gateway API 文档

## 概述

VGO-IAM 通过 gRPC Gateway 提供 RESTful API 接口，所有请求都需要进行签名认证。

**基础URL**: `http://localhost:50052` (开发环境)

## 认证

所有 API 请求都需要包含以下认证头部：

```http
Content-Type: application/json
access-key-id: YOUR_ACCESS_KEY_ID
signature: CALCULATED_SIGNATURE
x-iam-date: 2024-01-15T10:30:00Z
request-data: {"key":"value"}
```

## 用户管理 API

### 创建用户

```http
POST /v1/users
```

**请求体**:
```json
{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "secure_password",
  "profile": {
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890"
  }
}
```

**响应**:
```json
{
  "id": "user_123456",
  "username": "john_doe",
  "email": "john@example.com",
  "status": "active",
  "profile": {
    "first_name": "John",
    "last_name": "Doe",
    "phone": "+1234567890"
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### 获取用户信息

```http
GET /v1/users/{user_id}
```

**路径参数**:
- `user_id`: 用户ID

**响应**: 同创建用户响应格式

### 用户列表

```http
GET /v1/users
```

**查询参数**:
- `page`: 页码 (默认: 1)
- `page_size`: 每页数量 (默认: 20)
- `status`: 用户状态过滤 (active/inactive)
- `search`: 搜索关键词

**响应**:
```json
{
  "users": [
    {
      "id": "user_123456",
      "username": "john_doe",
      "email": "john@example.com",
      "status": "active",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

### 更新用户

```http
PUT /v1/users/{user_id}
```

**请求体**: 同创建用户，但所有字段都是可选的

### 删除用户

```http
DELETE /v1/users/{user_id}
```

**响应**:
```json
{
  "message": "User deleted successfully"
}
```

## 访问密钥管理 API

### 创建访问密钥

```http
POST /v1/access-keys
```

**请求体**:
```json
{
  "user_id": "user_123456",
  "description": "API access for mobile app",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

**响应**:
```json
{
  "access_key_id": "AKIA1234567890ABCDEF",
  "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
  "user_id": "user_123456",
  "description": "API access for mobile app",
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

### 访问密钥列表

```http
GET /v1/access-keys
```

**查询参数**:
- `user_id`: 用户ID过滤
- `status`: 状态过滤 (active/inactive)
- `page`: 页码
- `page_size`: 每页数量

### 更新访问密钥状态

```http
PUT /v1/access-keys/{access_key_id}/status
```

**请求体**:
```json
{
  "status": "inactive"
}
```

### 验证访问密钥

```http
POST /v1/access-keys/verify
```

**请求体**:
```json
{
  "access_key_id": "AKIA1234567890ABCDEF",
  "signature": "calculated_signature",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_data": "request_payload"
}
```

## 策略管理 API

### 创建策略

```http
POST /v1/policies
```

**请求体**:
```json
{
  "name": "ReadOnlyAccess",
  "description": "Provides read-only access to all resources",
  "policy_document": {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": ["iam:Get*", "iam:List*"],
        "Resource": "*"
      }
    ]
  }
}
```

### 获取策略

```http
GET /v1/policies/{policy_id}
```

### 策略列表

```http
GET /v1/policies
```

### 更新策略

```http
PUT /v1/policies/{policy_id}
```

### 删除策略

```http
DELETE /v1/policies/{policy_id}
```

## 权限检查 API

### 检查权限

```http
POST /v1/permissions/check
```

**请求体**:
```json
{
  "user_id": "user_123456",
  "action": "iam:GetUser",
  "resource": "arn:iam::account:user/john_doe",
  "context": {
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0..."
  }
}
```

**响应**:
```json
{
  "allowed": true,
  "matched_policies": ["ReadOnlyAccess"],
  "reason": "Access granted by policy ReadOnlyAccess"
}
```

### 验证权限

```http
POST /v1/permissions/validate
```

类似权限检查，但提供更详细的验证信息。

## 应用管理 API

### 创建应用

```http
POST /v1/applications
```

**请求体**:
```json
{
  "name": "Mobile App",
  "description": "iOS and Android mobile application",
  "type": "mobile",
  "redirect_uris": ["myapp://callback"],
  "scopes": ["read", "write"]
}
```

### 获取应用

```http
GET /v1/applications/{app_id}
```

### 应用列表

```http
GET /v1/applications
```

### 更新应用

```http
PUT /v1/applications/{app_id}
```

### 删除应用

```http
DELETE /v1/applications/{app_id}
```

## 开发者认证 API

### 提交开发者认证

```http
POST /v1/developer-verification
```

**请求体**:
```json
{
  "user_id": "user_123456",
  "company_name": "Tech Corp",
  "contact_email": "contact@techcorp.com",
  "business_license": "license_document_url",
  "identity_document": "id_document_url",
  "description": "We are developing a fintech application"
}
```

### 获取认证信息

```http
GET /v1/developer-verification/{verification_id}
```

### 认证列表

```http
GET /v1/developer-verification
```

**查询参数**:
- `status`: 认证状态 (pending/approved/rejected)
- `user_id`: 用户ID过滤

### 审核认证

```http
PUT /v1/developer-verification/{verification_id}/review
```

**请求体**:
```json
{
  "status": "approved",
  "reviewer_notes": "All documents verified successfully"
}
```

## 系统监控 API

### 健康检查

```http
GET /v1/health
```

**响应**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "services": {
    "database": "healthy",
    "cache": "healthy",
    "grpc": "healthy"
  }
}
```

### 系统指标

```http
GET /v1/metrics
```

**响应**:
```json
{
  "requests_total": 12345,
  "requests_per_second": 25.5,
  "active_users": 150,
  "active_sessions": 89,
  "database_connections": 10,
  "memory_usage": "256MB",
  "cpu_usage": "15%"
}
```

## 配置管理 API

### 获取配置

```http
GET /v1/config
```

### 更新配置

```http
PUT /v1/config
```

**请求体**:
```json
{
  "session_timeout": 3600,
  "max_login_attempts": 5,
  "password_policy": {
    "min_length": 8,
    "require_uppercase": true,
    "require_lowercase": true,
    "require_numbers": true,
    "require_symbols": true
  }
}
```

## 错误响应

所有 API 在出错时都会返回统一的错误格式：

```json
{
  "code": 400,
  "message": "Invalid request parameters",
  "data": [
    {
      "field": "email",
      "message": "Invalid email format"
    }
  ]
}
```

### 常见错误码

- `400`: 请求参数错误
- `401`: 未授权（签名验证失败）
- `403`: 权限不足
- `404`: 资源不存在
- `409`: 资源冲突
- `429`: 请求频率限制
- `500`: 服务器内部错误

## 速率限制

- 默认限制：每分钟 1000 次请求
- 超出限制时返回 `429 Too Many Requests`
- 响应头包含限制信息：
  - `X-RateLimit-Limit`: 限制数量
  - `X-RateLimit-Remaining`: 剩余次数
  - `X-RateLimit-Reset`: 重置时间

## SDK 和工具

### JavaScript/TypeScript

```typescript
import { api } from './api'

// 创建用户
const user = await api.users.create({
  username: 'john_doe',
  email: 'john@example.com',
  password: 'secure_password'
})

// 获取用户列表
const users = await api.users.list({ page: 1, page_size: 20 })
```

### cURL 示例

```bash
# 创建用户
curl -X POST http://localhost:50052/v1/users \
  -H "Content-Type: application/json" \
  -H "access-key-id: YOUR_ACCESS_KEY_ID" \
  -H "signature: CALCULATED_SIGNATURE" \
  -H "x-iam-date: 2024-01-15T10:30:00Z" \
  -H "request-data: {\"username\":\"john_doe\"}" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com",
    "password": "secure_password"
  }'
```

## 版本控制

- 当前版本：v1
- API 版本通过 URL 路径指定：`/v1/...`
- 向后兼容性：新版本会保持向后兼容
- 废弃通知：废弃的 API 会提前通知

## 支持和反馈

- 文档更新：请查看项目 README
- 问题报告：请在 GitHub Issues 中提交
- 功能请求：请在 GitHub Discussions 中讨论