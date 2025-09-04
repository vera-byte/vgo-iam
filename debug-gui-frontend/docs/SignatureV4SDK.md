# VGO-IAM 签名V4 TypeScript SDK

本文档介绍如何使用VGO-IAM的TypeScript签名SDK，该SDK完全基于后端Go语言实现转换而来，确保前后端签名算法的一致性。

## 概述

签名V4 SDK提供了完整的签名生成、验证和请求头处理功能，支持VGO-IAM的HMAC-SHA256签名认证机制。

## 核心功能

### 1. 签名生成 (`signV4`)

```typescript
import { signV4, type SignV4Result } from '@/utils/signatureV4SDK';

const result: SignV4Result = signV4(
  'your-access-key-id',
  'your-secret-key',
  'request-data' // 通常GET请求为空字符串
);

console.log(result);
// {
//   secretKey: 'your-secret-key',
//   accessKeyID: 'your-access-key-id',
//   signature: 'base64-encoded-signature',
//   timestamp: 1756982600
// }
```

### 2. 签名验证 (`verifySignV4`)

```typescript
import { verifySignV4 } from '@/utils/signatureV4SDK';

const result = verifySignV4(
  'signature-to-verify',
  'request-data',
  '1756982600', // 时间戳字符串
  'your-secret-key'
);

console.log(result);
// { valid: true } 或 { valid: false, error: 'error message' }
```

### 3. 生成请求头 (`generateRequestHeaders`)

```typescript
import { generateRequestHeaders } from '@/utils/signatureV4SDK';

const headers = generateRequestHeaders(
  'your-access-key-id',
  'your-secret-key',
  '' // GET请求的request-data通常为空
);

console.log(headers);
// {
//   'access-key-id': 'your-access-key-id',
//   'signature': 'generated-signature',
//   'x-iam-date': '1756982600',
//   'request-data': ''
// }
```

### 4. 解析请求头 (`parseRequestHeaders`)

```typescript
import { parseRequestHeaders } from '@/utils/signatureV4SDK';

const headers = {
  'Authorization': 'IAM-HMAC-SHA256 Credential=access-key/date/region/service/request, Signature=signature',
  'X-IAM-Date': '1756982600'
};

const parsed = parseRequestHeaders(headers);
console.log(parsed);
// {
//   accessKeyID: 'access-key',
//   signature: 'signature',
//   timestamp: '1756982600'
// }
```

## 完整的API调用示例

```typescript
import axios from 'axios';
import { generateRequestHeaders } from '@/utils/signatureV4SDK';

async function callAPI() {
  const accessKeyID = 'Ga0rTSg3NSyoOkFUx9jg';
  const secretKey = 'poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk';
  const requestData = ''; // GET请求
  
  try {
    // 生成签名头
    const signatureHeaders = generateRequestHeaders(accessKeyID, secretKey, requestData);
    
    // 发送请求
    const response = await axios.get('http://localhost:50052/v1/dashboard/stats', {
      headers: {
        'Content-Type': 'application/json',
        ...signatureHeaders
      }
    });
    
    console.log('API响应:', response.data);
    return response.data;
  } catch (error) {
    console.error('API调用失败:', error);
    throw error;
  }
}
```

## 签名算法详解

### 算法流程

1. **时间戳验证**: 检查请求时间戳是否在允许范围内（±5分钟）
2. **构建待签名字符串**: 格式为 `IAM-HMAC-SHA256\n{timestamp}\n{SHA256(requestData)}`
3. **密钥派生**: 通过多层HMAC-SHA256派生最终签名密钥
4. **签名计算**: 使用派生密钥对待签名字符串进行HMAC-SHA256计算
5. **Base64编码**: 将签名结果进行Base64编码

### 密钥派生过程

```
dateKey = HMAC-SHA256("IAM" + secretKey, date)
regionKey = HMAC-SHA256(dateKey, "default")
serviceKey = HMAC-SHA256(regionKey, "iam")
signingKey = HMAC-SHA256(serviceKey, "request")
```

### 待签名字符串格式

```
IAM-HMAC-SHA256
{unix_timestamp}
{base64(SHA256(requestData))}
```

## 错误处理

### 常见错误类型

1. **时间戳错误**: `Invalid timestamp` - 时间戳格式不正确
2. **请求过期**: `Request expired` - 请求时间超出允许范围
3. **签名不匹配**: `valid: false` - 计算的签名与提供的签名不匹配

### 调试建议

1. **检查时间同步**: 确保客户端和服务器时间同步
2. **验证密钥**: 确认AccessKeyID和SecretKey正确
3. **检查请求数据**: 确保requestData与实际请求内容一致
4. **查看日志**: 使用示例代码输出详细的签名信息进行调试

## 与后端Go实现的对应关系

| Go函数 | TypeScript函数 | 功能 |
|--------|----------------|------|
| `VerifySignV4` | `verifySignV4` | 验证签名 |
| `SignV4` | `signV4` | 生成签名 |
| `buildStringToSign` | `buildStringToSign` | 构建待签名字符串 |
| `calculateSignature` | `calculateSignature` | 计算签名 |
| `verifyTimestamp` | `verifyTimestamp` | 验证时间戳 |
| `hmacSha256` | `hmacSha256` | HMAC-SHA256计算 |
| `sha256Hash` | `sha256Hash` | SHA256哈希 |

## 示例代码

完整的使用示例请参考 `src/examples/signatureSDKExample.ts` 文件，其中包含了各种使用场景的详细示例。

## 注意事项

1. **时间同步**: 确保客户端时间与服务器时间同步，允许误差为±5分钟
2. **密钥安全**: 不要在客户端代码中硬编码密钥，应从安全的配置中获取
3. **请求数据**: GET请求的requestData通常为空字符串
4. **错误处理**: 始终检查签名验证的结果和可能的错误信息
5. **调试模式**: 在开发环境中可以启用详细日志来调试签名问题

## 依赖项

- `crypto-js`: 用于加密算法实现
- `@types/crypto-js`: TypeScript类型定义

确保在项目中已安装这些依赖：

```bash
pnpm add crypto-js
pnpm add -D @types/crypto-js
```