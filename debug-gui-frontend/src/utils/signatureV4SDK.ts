import CryptoJS from 'crypto-js';

/**
 * 签名算法常量
 */
const AUTH_HEADER_PREFIX = 'IAM-HMAC-SHA256';
const TIME_FORMAT = 'RFC3339';

/**
 * 签名结果接口
 */
export interface SignV4Result {
  secretKey: string;
  accessKeyID: string;
  signature: string;
  timestamp: number;
}

/**
 * 验证V4签名
 * @param signature 待验证的签名
 * @param requestData 请求数据
 * @param timestamp 时间戳字符串
 * @param secretKey 密钥
 * @returns 验证结果和错误信息
 */
export function verifySignV4(
  signature: string,
  requestData: string,
  timestamp: string,
  secretKey: string
): { valid: boolean; error?: string } {
  const timestampInt = parseInt(timestamp, 10);
  if (isNaN(timestampInt)) {
    return { valid: false, error: 'Invalid timestamp' };
  }

  // 1. 验证时间窗口 (通常为±5分钟)
  if (!verifyTimestamp(timestampInt)) {
    return { valid: false, error: 'Request expired' };
  }

  // 2. 构建待签字符串
  const stringToSign = buildStringToSign(timestampInt, requestData);

  // 3. 计算签名
  const computedSignature = calculateSignature(stringToSign, secretKey, timestampInt);

  // 4. 比较签名
  return { valid: computedSignature === signature };
}

/**
 * 签名V4
 * @param accessKeyID 访问密钥ID
 * @param secretKey 密钥
 * @param requestData 请求数据
 * @returns 签名结果
 */
export function signV4(
  accessKeyID: string,
  secretKey: string,
  requestData: string
): SignV4Result {
  const timestamp = Math.floor(Date.now() / 1000);

  // 1. 构建待签字符串
  const stringToSign = buildStringToSign(timestamp, requestData);

  // 2. 计算签名
  const signature = calculateSignature(stringToSign, secretKey, timestamp);

  return {
    secretKey,
    accessKeyID,
    signature,
    timestamp,
  };
}

/**
 * 验证时间戳是否在允许范围内
 * @param timestamp Unix时间戳
 * @returns 是否在允许范围内
 */
function verifyTimestamp(timestamp: number): boolean {
  // 转换为时间
  const t = new Date(timestamp * 1000);

  // 检查时间戳是否在允许范围内（±5分钟）
  const now = new Date();
  const diff = Math.abs(now.getTime() - t.getTime());
  return diff <= 5 * 60 * 1000; // 5分钟 = 5 * 60 * 1000毫秒
}

/**
 * 构建待签名字符串
 * @param timestamp Unix时间戳
 * @param requestData 请求数据
 * @returns 待签名字符串
 */
function buildStringToSign(timestamp: number, requestData: string): string {
  return `${AUTH_HEADER_PREFIX}\n${timestamp}\n${sha256Hash(requestData)}`;
}

/**
 * 计算签名
 * @param stringToSign 待签名字符串
 * @param secretKey 密钥
 * @param timestamp Unix时间戳
 * @returns Base64编码的签名
 */
function calculateSignature(
  stringToSign: string,
  secretKey: string,
  timestamp: number
): string {
  // 步骤1: 从时间戳中提取日期
  const date = new Date(timestamp * 1000)
    .toISOString()
    .slice(0, 10)
    .replace(/-/g, ''); // YYYYMMDD格式

  // 步骤2: 派生签名密钥 (与Go代码保持一致)
  // Go代码: hmacSha256(key, data) 返回 []byte，然后用 string([]byte) 转换
  // TypeScript需要模拟这个过程
  const dateKey = CryptoJS.HmacSHA256(date, `IAM${secretKey}`);
  const regionKey = CryptoJS.HmacSHA256('default', dateKey);
  const serviceKey = CryptoJS.HmacSHA256('iam', regionKey);
  const signingKey = CryptoJS.HmacSHA256('request', serviceKey);

  // 步骤3: 计算签名
  const signature = CryptoJS.HmacSHA256(stringToSign, signingKey);
  return CryptoJS.enc.Base64.stringify(signature);
}

// 注意：以下函数已被内联到calculateSignature中以确保与Go代码的一致性
// 不再需要单独的hmacSha256和hmacSha256Base64函数

/**
 * 计算SHA256哈希
 * @param data 数据
 * @returns Base64编码的哈希值
 */
function sha256Hash(data: string): string {
  const hash = CryptoJS.SHA256(data);
  return CryptoJS.enc.Base64.stringify(hash);
}

/**
 * 解析HTTP请求中的签名信息（用于调试）
 * @param headers 请求头对象
 * @returns 解析出的签名信息
 */
export function parseRequestHeaders(headers: Record<string, string>): {
  accessKeyID?: string;
  signature?: string;
  signedHeaders?: string;
  timestamp?: string;
} {
  const result: {
    accessKeyID?: string;
    signature?: string;
    signedHeaders?: string;
    timestamp?: string;
  } = {};

  // 从Authorization头解析
  const authHeader = headers['Authorization'] || headers['authorization'];
  if (authHeader) {
    const parts = authHeader.split(', ');
    for (const part of parts) {
      if (part.startsWith('Credential=')) {
        const credParts = part.replace('Credential=', '').split('/');
        if (credParts.length > 0) {
          result.accessKeyID = credParts[0];
        }
      } else if (part.startsWith('Signature=')) {
        result.signature = part.replace('Signature=', '');
      } else if (part.startsWith('SignedHeaders=')) {
        result.signedHeaders = part.replace('SignedHeaders=', '');
      }
    }
  }

  // 从专用头获取时间戳
  result.timestamp = headers['X-IAM-Date'] || headers['x-iam-date'];

  return result;
}

/**
 * 生成用于API调用的请求头
 * @param accessKeyID 访问密钥ID
 * @param secretKey 密钥
 * @param requestData 请求数据（通常为空字符串用于GET请求）
 * @returns 包含签名信息的请求头
 */
export function generateRequestHeaders(
  accessKeyID: string,
  secretKey: string,
  requestData: string = ''
): Record<string, string> {
  const signResult = signV4(accessKeyID, secretKey, requestData);

  return {
    'access-key-id': accessKeyID,
    'signature': signResult.signature,
    'x-iam-date': signResult.timestamp.toString(),
    'request-data': requestData,
  };
}