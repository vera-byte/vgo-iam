/**
 * VGO-IAM 签名SDK使用示例
 * 本文件展示如何使用signatureV4SDK进行签名和验证
 */

import {
  signV4,
  verifySignV4,
  generateRequestHeaders,
  parseRequestHeaders,
  type SignV4Result
} from '../utils/signatureV4SDK';

/**
 * 示例：生成签名
 */
export function exampleGenerateSignature() {
  const accessKeyID = 'Ga0rTSg3NSyoOkFUx9jg';
  const secretKey = 'poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk';
  const requestData = ''; // GET请求通常为空

  // 生成签名
  const signResult: SignV4Result = signV4(accessKeyID, secretKey, requestData);

  console.log('签名结果:', {
    accessKeyID: signResult.accessKeyID,
    signature: signResult.signature,
    timestamp: signResult.timestamp,
    timestampISO: new Date(signResult.timestamp * 1000).toISOString()
  });

  return signResult;
}

/**
 * 示例：验证签名
 */
export function exampleVerifySignature() {
  const signature = 'aL3xhQkFXYx7Hh2lFj0R8p2i/LJQnx6QcSHrgwsokmU=';
  const requestData = '';
  const timestamp = '1756982600';
  const secretKey = 'poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk';

  // 验证签名
  const verifyResult = verifySignV4(signature, requestData, timestamp, secretKey);

  console.log('验证结果:', verifyResult);

  return verifyResult;
}

/**
 * 示例：生成API请求头
 */
export function exampleGenerateHeaders() {
  const accessKeyID = 'Ga0rTSg3NSyoOkFUx9jg';
  const secretKey = 'poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk';
  const requestData = ''; // GET请求

  // 生成请求头
  const headers = generateRequestHeaders(accessKeyID, secretKey, requestData);

  console.log('生成的请求头:', headers);

  return headers;
}

/**
 * 示例：解析请求头
 */
export function exampleParseHeaders() {
  const headers = {
    'Authorization': 'IAM-HMAC-SHA256 Credential=Ga0rTSg3NSyoOkFUx9jg/20250125/default/iam/request, SignedHeaders=host;x-iam-date, Signature=aL3xhQkFXYx7Hh2lFj0R8p2i/LJQnx6QcSHrgwsokmU=',
    'X-IAM-Date': '1756982600'
  };

  // 解析请求头
  const parsed = parseRequestHeaders(headers);

  console.log('解析的请求头信息:', parsed);

  return parsed;
}

/**
 * 示例：完整的API调用流程
 */
export async function exampleAPICall() {
  const accessKeyID = 'Ga0rTSg3NSyoOkFUx9jg';
  const secretKey = 'poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk';
  const requestData = '';
  const apiUrl = 'http://localhost:50052/v1/dashboard/stats';

  try {
    // 1. 生成签名头
    const headers = generateRequestHeaders(accessKeyID, secretKey, requestData);

    // 2. 添加其他必要的头
    const requestHeaders = {
      'Content-Type': 'application/json',
      ...headers
    };

    console.log('请求头:', requestHeaders);

    // 3. 发送请求（这里只是示例，实际使用时需要axios或fetch）
    const requestInfo = {
      method: 'GET',
      url: apiUrl,
      headers: requestHeaders
    };

    console.log('完整请求信息:', requestInfo);

    return requestInfo;
  } catch (error) {
    console.error('API调用失败:', error);
    throw error;
  }
}

/**
 * 示例：批量测试不同时间戳的签名
 */
export function exampleBatchSignature() {
  const accessKeyID = 'Ga0rTSg3NSyoOkFUx9jg';
  const secretKey = 'poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk';
  const requestData = '';

  const results = [];

  // 生成5个不同时间戳的签名
  for (let i = 0; i < 5; i++) {
    const signResult = signV4(accessKeyID, secretKey, requestData);
    results.push({
      index: i + 1,
      ...signResult,
      timestampISO: new Date(signResult.timestamp * 1000).toISOString()
    });

    // 等待1秒，确保时间戳不同
    // 注意：在实际使用中，不需要等待
  }

  console.log('批量签名结果:', results);

  return results;
}

/**
 * 运行所有示例
 */
export function runAllExamples() {
  console.log('=== VGO-IAM 签名SDK示例 ===');

  console.log('\n1. 生成签名示例:');
  exampleGenerateSignature();

  console.log('\n2. 验证签名示例:');
  exampleVerifySignature();

  console.log('\n3. 生成请求头示例:');
  exampleGenerateHeaders();

  console.log('\n4. 解析请求头示例:');
  exampleParseHeaders();

  console.log('\n5. 完整API调用示例:');
  exampleAPICall();

  console.log('\n6. 批量签名示例:');
  exampleBatchSignature();

  console.log('\n=== 示例运行完成 ===');
}