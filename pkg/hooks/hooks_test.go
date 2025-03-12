package hooks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoggingHook 测试日志钩子
func TestLoggingHook(t *testing.T) {
	hook := &LoggingHook{}

	// 创建测试请求
	req, err := http.NewRequest("GET", "https://example.com/test", nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行钩子
	modifiedReq, err := hook.Before(req)
	if err != nil {
		t.Fatalf("执行钩子失败: %v", err)
	}

	// 验证请求未被修改
	if modifiedReq != req {
		t.Error("LoggingHook不应修改请求对象")
	}
}

// TestAuthHook 测试认证钩子
func TestAuthHook(t *testing.T) {
	// 创建带有令牌的钩子
	hook := &AuthHook{Token: "test-token-123"}

	// 创建测试请求
	req, err := http.NewRequest("GET", "https://example.com/api", nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}

	// 执行钩子
	modifiedReq, err := hook.Before(req)
	if err != nil {
		t.Fatalf("执行认证钩子失败: %v", err)
	}

	// 验证认证头已添加
	authHeader := modifiedReq.Header.Get("Authorization")
	expectedHeader := "Bearer test-token-123"

	if authHeader != expectedHeader {
		t.Errorf("认证头不正确，期望: %s, 实际: %s", expectedHeader, authHeader)
	}
}

// TestResponseLogHook 测试响应日志钩子
func TestResponseLogHook(t *testing.T) {
	hook := &ResponseLogHook{}

	// 创建测试响应
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"status": "success"}`)),
	}

	// 执行钩子
	modifiedResp, err := hook.After(resp)
	if err != nil {
		t.Fatalf("执行响应日志钩子失败: %v", err)
	}

	// 验证响应未被修改
	if modifiedResp != resp {
		t.Error("ResponseLogHook不应修改响应对象")
	}
}

// TestCustomHook 测试自定义钩子
func TestCustomHook(t *testing.T) {
	// 定义自定义函数
	beforeFunc := func(req *http.Request) (*http.Request, error) {
		req.Header.Set("X-Custom", "before-value")
		return req, nil
	}

	afterFunc := func(resp *http.Response) (*http.Response, error) {
		// 这是个测试函数，仅修改已有响应
		return resp, nil
	}

	// 创建自定义钩子
	hook := &CustomHook{
		BeforeFn: beforeFunc,
		AfterFn:  afterFunc,
	}

	// 测试Before函数
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	modifiedReq, err := hook.Before(req)
	if err != nil {
		t.Fatalf("执行自定义Before钩子失败: %v", err)
	}

	customHeader := modifiedReq.Header.Get("X-Custom")
	if customHeader != "before-value" {
		t.Errorf("自定义钩子未正确设置请求头，期望: %s, 实际: %s",
			"before-value", customHeader)
	}

	// 测试After函数
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"status": "ok"}`)),
	}

	modifiedResp, err := hook.After(resp)
	if err != nil {
		t.Fatalf("执行自定义After钩子失败: %v", err)
	}

	if modifiedResp != resp {
		t.Error("自定义After钩子未正确处理响应")
	}

	// 测试空钩子函数
	emptyHook := &CustomHook{}
	req2, _ := http.NewRequest("GET", "https://example.com", nil)
	modifiedReq2, _ := emptyHook.Before(req2)

	if modifiedReq2 != req2 {
		t.Error("空Before函数不应修改请求")
	}

	resp2 := &http.Response{StatusCode: http.StatusOK}
	modifiedResp2, _ := emptyHook.After(resp2)

	if modifiedResp2 != resp2 {
		t.Error("空After函数不应修改响应")
	}
}

// TestFieldTransformHook 测试字段转换钩子
func TestFieldTransformHook(t *testing.T) {
	hook := &FieldTransformHook{}

	testCases := []struct {
		name           string
		method         string
		body           string
		expectedChange bool
		expectedBody   string
	}{
		{
			name:           "POST请求有user字段",
			method:         "POST",
			body:           `{"user": "13800138000", "password": "secret"}`,
			expectedChange: true,
			expectedBody:   `{"password":"secret","phone":"13800138000"}`,
		},
		{
			name:           "PUT请求有user字段",
			method:         "PUT",
			body:           `{"user": "13900139000", "name": "测试用户"}`,
			expectedChange: true,
			expectedBody:   `{"name":"测试用户","phone":"13900139000"}`,
		},
		{
			name:           "POST请求没有user字段",
			method:         "POST",
			body:           `{"email": "test@example.com", "password": "secret"}`,
			expectedChange: false,
			expectedBody:   `{"email":"test@example.com","password":"secret"}`,
		},
		{
			name:           "GET请求不处理",
			method:         "GET",
			body:           `{"user": "13800138000"}`,
			expectedChange: false,
			expectedBody:   `{"user":"13800138000"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建请求
			req, _ := http.NewRequest(tc.method, "https://example.com",
				bytes.NewBufferString(tc.body))

			// 执行钩子
			modifiedReq, err := hook.Before(req)
			if err != nil {
				t.Fatalf("执行字段转换钩子失败: %v", err)
			}

			// 读取修改后的请求体
			body, _ := io.ReadAll(modifiedReq.Body)
			modifiedReq.Body.Close()

			// 规范化JSON以便比较（忽略字段顺序）
			var bodyObj, expectedObj map[string]interface{}
			json.Unmarshal(body, &bodyObj)
			json.Unmarshal([]byte(tc.expectedBody), &expectedObj)

			normalizedBody, _ := json.Marshal(bodyObj)
			normalizedExpected, _ := json.Marshal(expectedObj)

			// 验证结果
			if string(normalizedBody) != string(normalizedExpected) {
				t.Errorf("请求体未正确转换，\n期望: %s\n实际: %s",
					string(normalizedExpected), string(normalizedBody))
			}
		})
	}
}

// TestNewScriptHookFromFile 测试从文件创建脚本钩子
func TestNewScriptHookFromFile(t *testing.T) {
	// 创建临时脚本文件
	tempDir, err := os.MkdirTemp("", "script-hook-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建有效脚本文件
	validScriptPath := filepath.Join(tempDir, "valid-script.js")
	validScriptContent := `
function processRequest(request) {
	request.body.modified = true;
	return request;
}
`
	err = os.WriteFile(validScriptPath, []byte(validScriptContent), 0644)
	if err != nil {
		t.Fatalf("写入脚本文件失败: %v", err)
	}

	// 测试有效脚本文件
	t.Run("有效脚本文件", func(t *testing.T) {
		hook, err := NewScriptHookFromFile(validScriptPath)
		if err != nil {
			t.Fatalf("从有效文件创建脚本钩子失败: %v", err)
		}

		if hook == nil {
			t.Fatal("钩子不应为nil")
		}
	})

	// 测试不存在的脚本文件
	t.Run("不存在的脚本文件", func(t *testing.T) {
		_, err := NewScriptHookFromFile(filepath.Join(tempDir, "non-existent.js"))
		if err == nil {
			t.Fatal("应该检测到不存在的脚本文件")
		}

		if !strings.Contains(err.Error(), "不存在") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})
}
