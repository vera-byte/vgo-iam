package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	TestAccessKeyID     = "Ga0rTSg3NSyoOkFUx9jg"
	TestSecretAccessKey = "poh7b4bQi9fwXfIPXVGMzF0qiqaf9gDI9drEXtpk"
)

// addAuthHeaders 为HTTP请求添加认证头
func addAuthHeaders(req *http.Request, requestData string) {
	_, s := SignV4Ctx(req.Context(), TestAccessKeyID, TestSecretAccessKey, nil)

	req.Header.Add("x-iam-date", fmt.Sprintf("%d", s.Timestamp))
	req.Header.Add("access-key-id", s.AccessKeyID)
	req.Header.Add("signature", s.Signature)
	req.Header.Add("request-data", requestData)
}

// TestGetDashboardActivitiesHTTPGateway 测试 GetDashboardActivities gRPC Gateway HTTP 接口
// 该测试验证通过 HTTP 调用 gRPC Gateway 接口是否能正确返回仪表板活动数据
func TestGetDashboardActivitiesHTTPGateway(t *testing.T) {
	// 服务器地址 - 假设服务运行在 localhost:50052
	baseURL := "http://localhost:50052"

	// 测试用例
	tests := []struct {
		name       string
		path       string
		expectCode int
	}{
		{
			name:       "默认请求",
			path:       "/v1/dashboard/activities",
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 构建完整URL
			url := baseURL + tt.path

			// 创建HTTP客户端
			client := &http.Client{
				Timeout: 10 * time.Second,
			}
			// 创建请求
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			// 添加认证头
			addAuthHeaders(req, "{}")

			// 发送请求
			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("无法连接到服务器 %s: %v (请确保服务正在运行)", url, err)
				return
			}
			defer resp.Body.Close()

			// 检查状态码
			if resp.StatusCode != tt.expectCode {
				t.Errorf("期望状态码 %d，实际得到 %d", tt.expectCode, resp.StatusCode)
			}

			// 读取响应体
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("读取响应体失败: %v", err)
			}

			// 解析JSON响应
			var response map[string]interface{}
			if err := json.Unmarshal(body, &response); err != nil {
				t.Fatalf("解析JSON响应失败: %v\n响应内容: %s", err, string(body))
			}

			// 检查响应结构
			if activities, exists := response["activities"]; exists {
				activitiesList, ok := activities.([]interface{})
				if !ok {
					t.Error("activities 字段不是数组格式")
					return
				}

				t.Logf("成功获取到 %d 条活动记录", len(activitiesList))

				// 如果有活动记录，检查第一条记录的格式
				if len(activitiesList) > 0 {
					firstActivity, ok := activitiesList[0].(map[string]interface{})
					if !ok {
						t.Error("活动记录格式不正确")
						return
					}

					// 检查必要字段
					requiredFields := []string{"type", "description", "timestamp"}
					for _, field := range requiredFields {
						if _, exists := firstActivity[field]; !exists {
							t.Errorf("活动记录缺少必要字段: %s", field)
						}
					}

					t.Logf("第一条活动记录: %+v", firstActivity)
				}
			} else {
				t.Error("响应中缺少 activities 字段")
			}

			// 打印完整响应用于调试
			t.Logf("完整响应: %s", string(body))
		})
	}
}

// TestGetDashboardActivitiesHTTPWithStandardResponse 测试标准响应格式
// 该测试验证如果服务使用了标准响应包装，响应格式是否正确
func TestGetDashboardActivitiesHTTPWithStandardResponse(t *testing.T) {
	// 服务器地址
	baseURL := "http://localhost:50052"
	url := baseURL + "/v1/dashboard/activities?limit=10"

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送GET请求
	resp, err := client.Get(url)
	if err != nil {
		t.Skipf("无法连接到服务器 %s: %v (请确保服务正在运行)", url, err)
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}

	// 尝试解析为标准响应格式
	var standardResp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &standardResp); err == nil {
		// 如果能解析为标准响应格式
		t.Log("检测到标准响应格式")

		// 检查标准响应字段
		if standardResp.Code != 0 {
			t.Errorf("期望响应码 0，实际得到 %d，消息: %s", standardResp.Code, standardResp.Message)
		}

		if standardResp.Message != "success" {
			t.Errorf("期望消息 'success'，实际得到 '%s'", standardResp.Message)
		}

		// 检查数据部分
		if standardResp.Data == nil {
			t.Error("标准响应中数据为空")
			return
		}

		// 将数据转换为 map 检查结构
		dataMap, ok := standardResp.Data.(map[string]interface{})
		if !ok {
			t.Error("标准响应数据格式不正确")
			return
		}

		if activities, exists := dataMap["activities"]; exists {
			activitiesList, ok := activities.([]interface{})
			if !ok {
				t.Error("activities 字段不是数组格式")
				return
			}
			t.Logf("标准响应格式中获取到 %d 条活动记录", len(activitiesList))
		} else {
			t.Error("标准响应数据中缺少 activities 字段")
		}
	} else {
		// 如果不是标准响应格式，尝试直接解析
		t.Log("检测到原始 gRPC Gateway 响应格式")

		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("解析原始响应失败: %v\n响应内容: %s", err, string(body))
		}

		if activities, exists := response["activities"]; exists {
			activitiesList, ok := activities.([]interface{})
			if !ok {
				t.Error("activities 字段不是数组格式")
				return
			}
			t.Logf("原始响应格式中获取到 %d 条活动记录", len(activitiesList))
		} else {
			t.Error("原始响应中缺少 activities 字段")
		}
	}

	t.Logf("完整响应: %s", string(body))
}

// TestGetDashboardActivitiesHTTPHeaders 测试HTTP头部
// 该测试验证响应头部是否正确设置
func TestGetDashboardActivitiesHTTPHeaders(t *testing.T) {
	// 服务器地址
	baseURL := "http://localhost:50052"
	url := baseURL + "/v1/dashboard/activities"

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送GET请求
	resp, err := client.Get(url)
	if err != nil {
		t.Skipf("无法连接到服务器 %s: %v (请确保服务正在运行)", url, err)
		return
	}
	defer resp.Body.Close()

	// 检查Content-Type头部
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		t.Error("响应缺少 Content-Type 头部")
	} else if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		t.Errorf("期望 Content-Type 为 application/json，实际得到 %s", contentType)
	}

	// 检查CORS头部（如果配置了的话）
	if corsOrigin := resp.Header.Get("Access-Control-Allow-Origin"); corsOrigin != "" {
		t.Logf("检测到CORS配置: %s", corsOrigin)
	}

	// 打印所有响应头部用于调试
	t.Log("响应头部:")
	for name, values := range resp.Header {
		for _, value := range values {
			t.Logf("  %s: %s", name, value)
		}
	}
}

// BenchmarkGetDashboardActivitiesHTTP 性能基准测试
// 该测试用于评估 HTTP 接口的性能
func BenchmarkGetDashboardActivitiesHTTP(b *testing.B) {
	// 服务器地址
	baseURL := "http://localhost:50052"
	url := baseURL + "/v1/dashboard/activities?limit=10"

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 先测试一次连接
	resp, err := client.Get(url)
	if err != nil {
		b.Skipf("无法连接到服务器 %s: %v (请确保服务正在运行)", url, err)
		return
	}
	resp.Body.Close()

	// 重置计时器
	b.ResetTimer()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(url)
		if err != nil {
			b.Errorf("请求失败: %v", err)
			continue
		}

		// 读取响应体以确保完整处理
		_, err = io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			b.Errorf("读取响应失败: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			b.Errorf("期望状态码 %d，实际得到 %d", http.StatusOK, resp.StatusCode)
		}
	}
}
