package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/birdmichael/RenderAPI/pkg/client"
	"github.com/birdmichael/RenderAPI/pkg/config"
)

// 自定义认证钩子
type testAuthHook struct {
	token string
}

func (h *testAuthHook) Before(req *http.Request) (*http.Request, error) {
	req.Header.Set("Authorization", "Bearer "+h.token)
	return req, nil
}

// 自定义响应日志钩子
type testResponseLogHook struct{}

func (h *testResponseLogHook) After(resp *http.Response) (*http.Response, error) {
	// 实际实现中可能会记录响应状态码等信息
	return resp, nil
}

// 自定义日志钩子
type testLoggingHook struct{}

func (h *testLoggingHook) Before(req *http.Request) (*http.Request, error) {
	// 实际实现中可能会记录请求URL等信息
	return req, nil
}

// 创建测试认证钩子
func newTestAuthHook(token string) *testAuthHook {
	return &testAuthHook{token: token}
}

// 创建测试响应日志钩子
func newTestResponseLogHook() *testResponseLogHook {
	return &testResponseLogHook{}
}

// 创建测试日志钩子
func newTestLoggingHook() *testLoggingHook {
	return &testLoggingHook{}
}

// TestIntegration 集成测试所有组件
func TestIntegration(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置响应头
		w.Header().Set("Content-Type", "application/json")

		// 检查认证头
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Logf("认证头错误: %s", authHeader)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"未授权访问"}`))
			return
		}

		// 根据路径返回不同响应
		switch r.URL.Path {
		case "/api/data", "/users", "/api/users":
			// 正常数据响应
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"success","data":[{"id":1,"name":"测试项目"},{"id":2,"name":"示例项目"}]}`))
		case "/api/error", "/error":
			// 服务器错误响应
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"服务器内部错误"}`))
		default:
			// 路径不存在
			t.Logf("未处理的请求路径: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"路径不存在"}`))
		}
	}))
	defer server.Close()

	// 创建临时目录用于测试文件
	tempDir, err := os.MkdirTemp("", "render-api-integration-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试模板文件
	templatePath := filepath.Join(tempDir, "test-template.json")
	templateContent := `{
		"method": "GET",
		"url": "{{.BaseURL}}/api/users",
		"headers": {
			"Authorization": "Bearer {{.Token}}",
			"Accept": "application/json"
		}
	}`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("创建模板文件失败: %v", err)
	}

	// 创建测试数据文件
	dataPath := filepath.Join(tempDir, "test-data.json")
	dataContent := fmt.Sprintf(`{
		"BaseURL": "%s",
		"Token": "test-token"
	}`, server.URL)
	if err := os.WriteFile(dataPath, []byte(dataContent), 0644); err != nil {
		t.Fatalf("创建数据文件失败: %v", err)
	}

	// 创建配置
	cfg := &config.Config{
		BaseURL: server.URL,
		DefaultHeaders: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "RenderAPI-Test",
		},
		Timeout:       5,
		EnableLogging: true,
		AuthToken:     "test-token",
	}

	// 创建客户端
	c := client.NewClient(cfg.BaseURL, cfg.GetTimeout())

	// 设置默认头部
	for key, value := range cfg.DefaultHeaders {
		c.SetHeader(key, value)
	}

	// 添加钩子
	c.AddBeforeHook(newTestAuthHook(cfg.AuthToken))
	c.AddAfterHook(newTestResponseLogHook())

	// 测试使用模板文件和数据文件发送请求
	t.Run("TemplateWithDataFile", func(t *testing.T) {
		resp, err := c.ExecuteTemplateWithDataFile(context.Background(), templatePath, dataPath)
		if err != nil {
			t.Fatalf("执行模板失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("状态码错误，期望: %d, 实际: %d", http.StatusOK, resp.StatusCode)
		}

		// 读取并检查响应内容
		responseData, err := client.ReadResponseBody(resp)
		if err != nil {
			t.Fatalf("读取响应失败: %v", err)
		}

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(responseData), &jsonData); err != nil {
			t.Fatalf("解析JSON失败: %v", err)
		}

		status, ok := jsonData["status"].(string)
		if !ok || status != "success" {
			t.Errorf("响应状态错误，期望: %s, 实际: %v", "success", status)
		}
	})

	// 测试错误处理
	t.Run("ErrorHandling", func(t *testing.T) {
		// 创建错误模板
		errorTemplatePath := filepath.Join(tempDir, "error-template.json")
		errorTemplateContent := `{
			"method": "GET",
			"url": "{{.BaseURL}}/api/error",
			"headers": {
				"Authorization": "Bearer {{.Token}}"
			}
		}`
		if err := os.WriteFile(errorTemplatePath, []byte(errorTemplateContent), 0644); err != nil {
			t.Fatalf("创建错误模板文件失败: %v", err)
		}

		resp, err := c.ExecuteTemplateWithDataFile(context.Background(), errorTemplatePath, dataPath)
		if err != nil {
			t.Fatalf("执行错误模板失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("错误状态码错误，期望: %d, 实际: %d", http.StatusInternalServerError, resp.StatusCode)
		}

		responseData, err := client.ReadResponseBody(resp)
		if err != nil {
			t.Fatalf("读取错误响应失败: %v", err)
		}

		var errorData map[string]interface{}
		if err := json.Unmarshal([]byte(responseData), &errorData); err != nil {
			t.Fatalf("解析错误JSON失败: %v", err)
		}

		errorMsg, ok := errorData["error"].(string)
		if !ok || errorMsg != "服务器内部错误" {
			t.Errorf("错误消息错误，期望: %s, 实际: %v", "服务器内部错误", errorMsg)
		}
	})
}

// ExampleClientUsage 提供了一个使用RenderAPI客户端的完整示例
func ExampleClientUsage() {
	// 创建配置
	cfg := config.DefaultConfig()
	cfg.BaseURL = "https://api.example.com"
	cfg.AuthToken = "your-auth-token"

	// 创建客户端
	c := client.NewClient(cfg.BaseURL, cfg.GetTimeout())

	// 设置默认头部
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("User-Agent", "RenderAPI-Example")

	// 添加认证钩子
	c.AddBeforeHook(newTestAuthHook(cfg.AuthToken))

	// 添加日志钩子
	c.AddBeforeHook(newTestLoggingHook())
	c.AddAfterHook(newTestResponseLogHook())

	// 创建一个模板引擎实例
	engine := c.GetTemplateEngine()

	// 添加一个自定义函数
	engine.AddFunc("formatDate", func(t time.Time) string {
		return t.Format("2006-01-02")
	})

	// 使用JSON模板发送请求示例
	templateJSON := `{
		"method": "POST",
		"url": "{{.BaseURL}}/api/users",
		"headers": {
			"X-Custom-Header": "custom-value"
		},
		"body": {
			"name": "{{.Name}}",
			"email": "{{.Email}}",
			"registrationDate": "{{formatDate .RegisterDate}}"
		}
	}`

	data := map[string]interface{}{
		"BaseURL":      "https://api.example.com",
		"Name":         "张三",
		"Email":        "zhangsan@example.com",
		"RegisterDate": time.Now(),
	}

	// 假设这里是实际使用，由于这是一个示例，我们只是打印而不实际执行
	_ = templateJSON
	_ = data
	fmt.Println("示例客户端使用完成")

	// Output:
	// 示例客户端使用完成
}
