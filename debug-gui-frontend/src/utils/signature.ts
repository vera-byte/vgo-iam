import CryptoJS from 'crypto-js'
import dayjs from 'dayjs'

// 签名配置
interface SignatureConfig {
  accessKeyId: string
  secretAccessKey: string
}

// 从localStorage获取签名配置
export function getSignatureConfig(): SignatureConfig | null {
  const config = localStorage.getItem('signature-config')
  if (!config) return null
  return JSON.parse(config)
}

// 设置签名配置
export function setSignatureConfig(config: SignatureConfig): void {
  localStorage.setItem('signature-config', JSON.stringify(config))
}

// 清除签名配置
export function clearSignatureConfig(): void {
  localStorage.removeItem('signature-config')
}

// 生成签名 - 使用与后端一致的V4签名算法
function generateSignature(
  method: string,
  path: string,
  timestamp: string,
  requestData: string,
  secretKey: string
): string {
  // 将时间戳转换为Unix时间戳
  const timestampUnix = parseInt(timestamp)
  
  // 构建待签名字符串 (与后端保持一致)
  const authHeaderPrefix = 'IAM-HMAC-SHA256'
  // 使用Base64编码，与后端保持一致
  const hashedRequestData = CryptoJS.SHA256(requestData).toString(CryptoJS.enc.Base64)
  const stringToSign = `${authHeaderPrefix}\n${timestampUnix}\n${hashedRequestData}`
  
  // 计算V4签名
  const signature = calculateV4Signature(stringToSign, secretKey, timestampUnix)
  
  return signature
}

// 计算V4签名 - 与后端算法完全一致
function calculateV4Signature(stringToSign: string, secretKey: string, timestamp: number): string {
  // 步骤1: 从时间戳中提取日期
  const date = dayjs.unix(timestamp).format('YYYYMMDD')
  
  // 步骤2: 派生签名密钥 (与Go代码保持一致)
  // 注意：CryptoJS.HmacSHA256(message, key) 与 Go的 hmacSha256(key, data) 参数顺序相反
  const dateKey = CryptoJS.HmacSHA256(date, 'IAM' + secretKey)
  const regionKey = CryptoJS.HmacSHA256('default', dateKey)
  const serviceKey = CryptoJS.HmacSHA256('iam', regionKey)
  const signingKey = CryptoJS.HmacSHA256('request', serviceKey)
  
  // 步骤3: 计算签名，使用Base64编码与后端保持一致
  const signature = CryptoJS.HmacSHA256(stringToSign, signingKey).toString(CryptoJS.enc.Base64)
  
  return signature
}

// 为请求添加签名
export function signRequest(config: any): any {
  const signatureConfig = getSignatureConfig()
  
  // 如果没有配置签名信息，直接返回原配置
  if (!signatureConfig) {
    console.warn('未配置签名信息，请先设置AccessKey和SecretKey')
    return config
  }
  
  const { accessKeyId, secretAccessKey } = signatureConfig
  
  // 生成Unix时间戳
  const timestampUnix = Math.floor(Date.now() / 1000)
  
  // 获取请求方法和路径
  const method = (config.method || 'GET').toUpperCase()
  const path = config.url || '/'
  
  // 获取请求数据
  let requestData = ''
  if (config.data) {
    requestData = typeof config.data === 'string' ? config.data : JSON.stringify(config.data)
  } else {
    // 对于GET请求或没有请求体的请求，使用空对象作为requestData
    // 这样可以避免后端CombinedAuthInterceptor检查requestData为空字符串的问题
    requestData = '{}'
  }
  
  // 生成签名
  const signature = generateSignature(method, path, timestampUnix.toString(), requestData, secretAccessKey)
  
  // 添加签名头部
  config.headers = {
    ...config.headers,
    'access-key-id': accessKeyId,
    'signature': signature,
    'x-iam-date': timestampUnix.toString(),
    'request-data': requestData || ''
  }
  
  return config
}

// 验证签名配置
export function validateSignatureConfig(): boolean {
  const config = getSignatureConfig()
  return !!(config?.accessKeyId && config?.secretAccessKey)
}