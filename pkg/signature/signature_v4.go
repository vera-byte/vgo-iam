package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	AuthHeaderPrefix = "IAM-HMAC-SHA256"
	TimeFormat       = time.RFC3339
)

// VerifySignatureV4 验证V4签名
func VerifySignV4(signature, requestData, timestamp, secretKey string) (bool, error) {
	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, err
	}

	// 1. 验证时间窗口 (通常为±15分钟)
	if !verifyTimestamp(timestampInt) {
		return false, fmt.Errorf("request expired")
	}

	// 2. 构建待签字符串
	stringToSign := buildStringToSign(timestampInt, requestData)

	// 3. 计算签名
	computedSignature := calculateSignature(stringToSign, secretKey, timestampInt)

	// 4. 比较签名
	return computedSignature == signature, nil
}

type SignV4Result struct {
	SecretKey   string
	AccessKeyID string
	Signature   string
	Timestamp   int64
}

// Sign 签名V4
func SignV4(accessKeyID, secretKey, requestData string) SignV4Result {

	timestamp := time.Now().Unix()

	// 1. 构建待签字符串
	stringToSign := buildStringToSign(timestamp, requestData)

	// 2. 计算签名
	return SignV4Result{
		SecretKey:   secretKey,
		AccessKeyID: accessKeyID,
		Signature:   calculateSignature(stringToSign, secretKey, timestamp),
		Timestamp:   timestamp,
	}
}

// verifyTimestamp 验证时间戳是否在允许范围内
func verifyTimestamp(timestamp int64) bool {

	// 转换为时间
	t := time.Unix(timestamp, 0)

	// 检查时间戳是否在允许范围内（±5分钟）
	now := time.Now()
	diff := now.Sub(t)
	return diff.Abs() <= 5*time.Minute

}

// BuildStringToSign 构建待签名字符串
func buildStringToSign(timestamp int64, requestData string) string {
	return fmt.Sprintf("%s\n%d\n%s",
		AuthHeaderPrefix,
		timestamp,
		sha256Hash(requestData))
}

// CalculateSignature 计算签名
func calculateSignature(stringToSign, secretKey string, timestamp int64) string {
	// 步骤1: 从时间戳中提取日期
	date := time.Unix(timestamp, 0).Format("20060102")

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
