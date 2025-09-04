# VGO-IAM Debug GUI 安全性文档

## 签名认证机制

### 当前实现

VGO-IAM 使用基于 HMAC-SHA256 的签名认证机制，类似于 AWS Signature Version 4，确保请求的完整性和身份验证。

#### 签名算法流程

1. **时间戳验证**
   - 请求时间戳必须在当前时间的 ±5 分钟内
   - 防止重放攻击

2. **待签名字符串构建**
   ```
   StringToSign = AuthHeaderPrefix + "\n" + 
                  Timestamp + "\n" + 
                  SHA256(RequestData)
   ```

3. **签名密钥派生**
   ```
   DateKey = HMAC-SHA256("IAM" + SecretKey, Date)
   RegionKey = HMAC-SHA256(DateKey, "default")
   ServiceKey = HMAC-SHA256(RegionKey, "iam")
   SigningKey = HMAC-SHA256(ServiceKey, "request")
   ```

4. **最终签名计算**
   ```
   Signature = Base64(HMAC-SHA256(SigningKey, StringToSign))
   ```

### 请求头格式

前端发送请求时需要包含以下头部：

- `access-key-id`: 访问密钥ID
- `signature`: 计算得出的签名
- `x-iam-date`: 请求时间戳（ISO 8601格式）
- `request-data`: 请求数据的JSON字符串

### 安全特性

#### 1. 时间窗口验证
- **目的**: 防止重放攻击
- **实现**: 请求时间戳必须在服务器时间的 ±5 分钟内
- **建议**: 客户端应确保系统时间准确

#### 2. 请求数据完整性
- **目的**: 确保请求数据未被篡改
- **实现**: 将请求数据的 SHA256 哈希值包含在签名中
- **覆盖**: 包括 HTTP 方法、路径、查询参数和请求体

#### 3. 密钥派生
- **目的**: 增强安全性，避免直接使用原始密钥
- **实现**: 使用多层 HMAC 计算派生最终签名密钥
- **优势**: 即使签名泄露，也难以反推原始密钥

#### 4. Base64 编码
- **目的**: 确保签名在 HTTP 传输中的安全性
- **实现**: 所有哈希值和签名都使用 Base64 编码

## 前端安全实现

### 签名配置管理

```typescript
// 安全存储访问密钥
interface SignatureConfig {
  accessKeyId: string
  secretAccessKey: string
}

// 存储在 localStorage（生产环境建议使用更安全的存储方式）
function setSignatureConfig(config: SignatureConfig): void {
  localStorage.setItem('signature-config', JSON.stringify(config))
}
```

### 请求签名流程

```typescript
export function signRequest(config: AxiosRequestConfig): AxiosRequestConfig {
  const signatureConfig = getSignatureConfig()
  if (!signatureConfig) return config
  
  const { accessKeyId, secretAccessKey } = signatureConfig
  const timestamp = dayjs().format('YYYY-MM-DDTHH:mm:ss[Z]')
  const method = (config.method || 'GET').toUpperCase()
  const path = config.url || '/'
  const requestData = config.data ? JSON.stringify(config.data) : ''
  
  const signature = generateSignature(method, path, timestamp, requestData, secretAccessKey)
  
  config.headers = {
    ...config.headers,
    'access-key-id': accessKeyId,
    'signature': signature,
    'x-iam-date': timestamp,
    'request-data': requestData
  }
  
  return config
}
```

## 安全建议

### 1. 密钥管理
- **生产环境**: 不要在前端代码中硬编码密钥
- **存储**: 使用安全的密钥存储方案（如 Web Crypto API）
- **轮换**: 定期轮换访问密钥
- **权限**: 遵循最小权限原则

### 2. 传输安全
- **HTTPS**: 生产环境必须使用 HTTPS
- **CORS**: 正确配置 CORS 策略
- **CSP**: 实施内容安全策略

### 3. 客户端安全
- **时间同步**: 确保客户端时间准确
- **错误处理**: 不要在错误信息中泄露敏感信息
- **日志**: 避免在客户端日志中记录敏感数据

### 4. 监控和审计
- **请求日志**: 记录所有 API 请求
- **异常检测**: 监控异常的请求模式
- **访问控制**: 实施基于角色的访问控制

## 潜在安全风险

### 1. 客户端存储
- **风险**: localStorage 可被恶意脚本访问
- **缓解**: 考虑使用 httpOnly cookies 或 Web Crypto API

### 2. 时间同步
- **风险**: 客户端时间不准确可能导致请求失败
- **缓解**: 提供时间同步检查功能

### 3. 重放攻击
- **风险**: 虽然有时间窗口限制，但仍存在短时间内的重放风险
- **缓解**: 考虑添加 nonce 机制

### 4. 中间人攻击
- **风险**: HTTP 传输可能被拦截
- **缓解**: 强制使用 HTTPS

## 合规性考虑

- **数据保护**: 遵循 GDPR、CCPA 等数据保护法规
- **审计要求**: 保留足够的审计日志
- **加密标准**: 使用行业标准的加密算法
- **访问控制**: 实施适当的身份验证和授权机制

## 更新和维护

- **定期审查**: 定期审查安全配置和实现
- **漏洞扫描**: 定期进行安全漏洞扫描
- **依赖更新**: 及时更新依赖库以修复安全漏洞
- **安全培训**: 为开发团队提供安全培训