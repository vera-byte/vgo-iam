package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	authHeaderPrefix = "IAM-HMAC-SHA256"
	timeFormat       = "20060102T150405Z"
)

// VerifySignatureV4 验证V4签名
func VerifySignatureV4(signature, requestData, timestamp, secretKey string) (bool, error) {
	// 1. 验证时间窗口 (通常为±15分钟)
	t, err := time.Parse(timeFormat, timestamp)
	if err != nil {
		return false, fmt.Errorf("invalid timestamp format")
	}

	if time.Since(t).Abs() > 15*time.Minute {
		return false, fmt.Errorf("request expired")
	}

	// 2. 构建待签字符串
	stringToSign := buildStringToSign(timestamp, requestData)

	// 3. 计算签名
	computedSignature := calculateSignature(stringToSign, secretKey, timestamp)

	// 4. 比较签名
	return computedSignature == signature, nil
}

// buildStringToSign 构建待签名字符串
func buildStringToSign(timestamp, requestData string) string {
	return fmt.Sprintf("%s\n%s\n%s",
		authHeaderPrefix,
		timestamp,
		sha256Hash(requestData))
}

// calculateSignature 计算签名
func calculateSignature(stringToSign, secretKey, timestamp string) string {
	// 步骤1: 从时间戳中提取日期
	date := timestamp[0:8]

	// 步骤2: 派生签名密钥
	dateKey := hmacSha256("IAM"+secretKey, date)
	regionKey := hmacSha256(string(dateKey), "default")
	serviceKey := hmacSha256(string(regionKey), "iam")
	signingKey := hmacSha256(string(serviceKey), "request")

	// 步骤3: 计算签名
	return hmacSha256Hex(signingKey, stringToSign)
}

// hmacSha256 HMAC-SHA256计算
func hmacSha256(key, data string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return h.Sum(nil)
}

// hmacSha256Hex 计算HMAC-SHA256并返回16进制字符串
func hmacSha256Hex(key []byte, data string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// sha256Hash 计算SHA256哈希
func sha256Hash(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ParseRequest 解析HTTP请求中的签名信息
func ParseRequest(r *http.Request) (accessKeyID, signature, signedHeaders, timestamp string) {
	// 从Authorization头解析
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, ", ")
		for _, part := range parts {
			switch {
			case strings.HasPrefix(part, "Credential="):
				credParts := strings.Split(strings.TrimPrefix(part, "Credential="), "/")
				if len(credParts) > 0 {
					accessKeyID = credParts[0]
				}
			case strings.HasPrefix(part, "Signature="):
				signature = strings.TrimPrefix(part, "Signature=")
			case strings.HasPrefix(part, "SignedHeaders="):
				signedHeaders = strings.TrimPrefix(part, "SignedHeaders=")
			}
		}
	}

	// 从专用头获取时间戳
	timestamp = r.Header.Get("X-IAM-Date")

	return accessKeyID, signature, signedHeaders, timestamp
}
