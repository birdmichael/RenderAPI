package hooks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	// 测试异步方法
	reqChan, errChan := hook.BeforeAsync(req)
	select {
	case modifiedReq := <-reqChan:
		if modifiedReq != req {
			t.Error("LoggingHook不应在异步模式下修改请求对象")
		}
	case err := <-errChan:
		t.Fatalf("异步执行钩子失败: %v", err)
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

	// 测试异步方法
	reqChan, errChan := hook.BeforeAsync(req)
	select {
	case modifiedReq := <-reqChan:
		authHeader := modifiedReq.Header.Get("Authorization")
		if authHeader != expectedHeader {
			t.Errorf("异步认证头不正确，期望: %s, 实际: %s", expectedHeader, authHeader)
		}
	case err := <-errChan:
		t.Fatalf("异步执行钩子失败: %v", err)
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

	// 测试异步方法
	respChan, errChan := hook.AfterAsync(resp)
	select {
	case modifiedResp := <-respChan:
		if modifiedResp != resp {
			t.Error("ResponseLogHook不应在异步模式下修改响应对象")
		}
	case err := <-errChan:
		t.Fatalf("异步执行钩子失败: %v", err)
	}
}

// TestCustomFunctionHook 测试自定义钩子
func TestCustomFunctionHook(t *testing.T) {
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
	hook := &CustomFunctionHook{
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
	emptyHook := &CustomFunctionHook{}
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

	// 测试异步方法
	reqChan, errChan := hook.BeforeAsync(req)
	select {
	case modifiedReq := <-reqChan:
		customHeader := modifiedReq.Header.Get("X-Custom")
		if customHeader != "before-value" {
			t.Errorf("异步自定义钩子未正确设置请求头，期望: %s, 实际: %s",
				"before-value", customHeader)
		}
	case err := <-errChan:
		t.Fatalf("异步执行钩子失败: %v", err)
	}

	respChan, errChan := hook.AfterAsync(resp)
	select {
	case modifiedResp := <-respChan:
		if modifiedResp != resp {
			t.Error("异步自定义After钩子未正确处理响应")
		}
	case err := <-errChan:
		t.Fatalf("异步执行钩子失败: %v", err)
	}
}

// TestFieldTransformHook 测试字段转换钩子
func TestFieldTransformHook(t *testing.T) {
	transformMap := map[string]string{
		"user": "phone",
	}
	hook := NewFieldTransformHook(transformMap)

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

			// 测试异步方法
			// 为异步测试准备新的请求对象，因为前面的已经读取过body了
			asyncReq, _ := http.NewRequest(tc.method, "https://example.com",
				bytes.NewBufferString(tc.body))

			reqChan, errChan := hook.BeforeAsync(asyncReq)
			select {
			case modifiedReq := <-reqChan:
				// 读取修改后的请求体
				body, _ := io.ReadAll(modifiedReq.Body)
				modifiedReq.Body.Close()

				// 规范化JSON以便比较
				var bodyObj map[string]interface{}
				json.Unmarshal(body, &bodyObj)
				normalizedBody, _ := json.Marshal(bodyObj)

				if string(normalizedBody) != string(normalizedExpected) {
					t.Errorf("异步请求体未正确转换，\n期望: %s\n实际: %s",
						string(normalizedExpected), string(normalizedBody))
				}
			case err := <-errChan:
				t.Fatalf("异步执行钩子失败: %v", err)
			}
		})
	}
}

// TestJSHook 测试从文件创建JavaScript钩子
func TestJSHook(t *testing.T) {
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
	// 直接使用请求体对象（不需要JSON.parse）
	request.body.modified = true;
	request.body.timestamp = "2023-01-01T00:00:00Z";
	return request;
}
`
	err = os.WriteFile(validScriptPath, []byte(validScriptContent), 0644)
	if err != nil {
		t.Fatalf("写入脚本文件失败: %v", err)
	}

	// 测试有效脚本文件
	t.Run("有效脚本文件", func(t *testing.T) {
		hook, err := NewJSHookFromFile(validScriptPath, false, 30)
		if err != nil {
			t.Fatalf("从有效文件创建脚本钩子失败: %v", err)
		}

		if hook == nil {
			t.Fatal("钩子不应为nil")
		}

		// 测试执行钩子
		req, _ := http.NewRequest("POST", "https://example.com/api",
			bytes.NewBufferString(`{"name":"test"}`))

		modifiedReq, err := hook.Before(req)
		if err != nil {
			t.Fatalf("执行JS钩子失败: %v", err)
		}

		// 验证请求体已被修改
		body, _ := io.ReadAll(modifiedReq.Body)
		modifiedReq.Body.Close()

		var bodyObj map[string]interface{}
		err = json.Unmarshal(body, &bodyObj)
		if err != nil {
			t.Fatalf("解析修改后的请求体失败: %v", err)
		}

		if modified, ok := bodyObj["modified"].(bool); !ok || !modified {
			t.Error("JS钩子未正确修改请求体中的modified字段")
		}

		if timestamp, ok := bodyObj["timestamp"].(string); !ok || timestamp != "2023-01-01T00:00:00Z" {
			t.Error("JS钩子未正确添加timestamp字段")
		}
	})

	// 测试不存在的脚本文件
	t.Run("不存在的脚本文件", func(t *testing.T) {
		_, err := NewJSHookFromFile(filepath.Join(tempDir, "non-existent.js"), false, 30)
		if err == nil {
			t.Fatal("应该检测到不存在的脚本文件")
		}

		if !strings.Contains(err.Error(), "不存在") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})

	// 测试从字符串创建钩子
	t.Run("从字符串创建钩子", func(t *testing.T) {
		scriptContent := `
function processRequest(request) {
	request.body.fromString = true;
	return request;
}
`
		hook, err := NewJSHookFromString(scriptContent, false, 30)
		if err != nil {
			t.Fatalf("从字符串创建JS钩子失败: %v", err)
		}

		req, _ := http.NewRequest("POST", "https://example.com/api",
			bytes.NewBufferString(`{"name":"test"}`))

		modifiedReq, err := hook.Before(req)
		if err != nil {
			t.Fatalf("执行从字符串创建的JS钩子失败: %v", err)
		}

		body, _ := io.ReadAll(modifiedReq.Body)
		modifiedReq.Body.Close()

		var bodyObj map[string]interface{}
		err = json.Unmarshal(body, &bodyObj)
		if err != nil {
			t.Fatalf("解析修改后的请求体失败: %v", err)
		}

		if fromString, ok := bodyObj["fromString"].(bool); !ok || !fromString {
			t.Error("从字符串创建的JS钩子未正确修改请求体")
		}
	})
}

// TestJSHookErrors 测试JS钩子的错误处理
func TestJSHookErrors(t *testing.T) {
	// 测试无效的JavaScript代码
	t.Run("无效的JavaScript代码", func(t *testing.T) {
		invalidScript := `
function processRequest(request) {
	// 这是一个语法错误
	if (true {
		return request;
	}
}
`
		hook, err := NewJSHookFromString(invalidScript, false, 30)
		if err != nil {
			t.Fatalf("创建JS钩子失败: %v", err)
		}

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		_, err = hook.Before(req)

		if err == nil {
			t.Error("应该检测到无效的JavaScript代码")
		} else if !strings.Contains(err.Error(), "执行脚本失败") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})

	// 测试缺少processRequest函数
	t.Run("缺少processRequest函数", func(t *testing.T) {
		scriptWithoutFunction := `
// 这个脚本没有定义processRequest函数
const x = 10;
console.log("Hello, world!");
`
		hook, err := NewJSHookFromString(scriptWithoutFunction, false, 30)
		if err != nil {
			t.Fatalf("创建JS钩子失败: %v", err)
		}

		req, _ := http.NewRequest("GET", "https://example.com", bytes.NewBufferString(`{"test":"value"}`))
		_, err = hook.Before(req)

		if err == nil {
			t.Error("应该检测到缺少processRequest函数")
		} else if !strings.Contains(err.Error(), "未找到processRequest函数") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})

	// 测试processRequest函数返回非对象值
	t.Run("processRequest返回非对象值", func(t *testing.T) {
		invalidReturnScript := `
function processRequest(request) {
	// 返回字符串而不是对象
	return "这不是一个有效的请求对象";
}
`
		hook, err := NewJSHookFromString(invalidReturnScript, false, 30)
		if err != nil {
			t.Fatalf("创建JS钩子失败: %v", err)
		}

		req, _ := http.NewRequest("POST", "https://example.com", bytes.NewBufferString(`{"test":"value"}`))
		_, err = hook.Before(req)

		if err == nil {
			t.Error("应该检测到无效的返回值")
		} else if !strings.Contains(err.Error(), "无法解析处理后的请求对象") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})
}

// TestCommandHook 测试命令行钩子
func TestCommandHook(t *testing.T) {
	// 跳过在非Unix环境中的测试
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("跳过测试: 无法找到sh命令")
	}

	// 测试简单的命令行钩子
	t.Run("简单命令钩子", func(t *testing.T) {
		// 创建一个命令行钩子，该命令将请求体中的name转为大写
		hook := NewCommandHook(`jq '.name = (.name | ascii_upcase)'`, 5, false)

		req, _ := http.NewRequest("POST", "https://example.com/api",
			bytes.NewBufferString(`{"name":"test","id":123}`))

		modifiedReq, err := hook.Before(req)
		if err != nil {
			t.Fatalf("执行命令行钩子失败: %v", err)
		}

		body, _ := io.ReadAll(modifiedReq.Body)
		modifiedReq.Body.Close()

		var bodyObj map[string]interface{}
		json.Unmarshal(body, &bodyObj)

		if name, ok := bodyObj["name"].(string); !ok || name != "TEST" {
			t.Errorf("命令行钩子未正确修改name字段，期望: TEST, 实际: %v", bodyObj["name"])
		}
	})

	// 测试异步命令行钩子
	t.Run("异步命令钩子", func(t *testing.T) {
		hook := NewCommandHook(`sleep 0.1 && jq '.async = true'`, 5, true)

		req, _ := http.NewRequest("POST", "https://example.com/api",
			bytes.NewBufferString(`{"name":"test"}`))

		reqChan, errChan := hook.BeforeAsync(req)

		select {
		case modifiedReq := <-reqChan:
			body, _ := io.ReadAll(modifiedReq.Body)
			modifiedReq.Body.Close()

			var bodyObj map[string]interface{}
			json.Unmarshal(body, &bodyObj)

			if async, ok := bodyObj["async"].(bool); !ok || !async {
				t.Error("异步命令行钩子未正确修改请求体")
			}
		case err := <-errChan:
			t.Fatalf("异步执行命令行钩子失败: %v", err)
		case <-time.After(6 * time.Second):
			t.Fatal("异步命令行钩子执行超时")
		}
	})

	// 测试命令行超时
	t.Run("命令超时", func(t *testing.T) {
		hook := NewCommandHook(`sleep 3`, 1, false)

		req, _ := http.NewRequest("GET", "https://example.com/api", nil)

		_, err := hook.Before(req)
		if err == nil {
			t.Fatal("命令应该因超时而失败")
		}

		if !strings.Contains(strings.ToLower(err.Error()), "timeout") &&
			!strings.Contains(strings.ToLower(err.Error()), "killed") {
			t.Errorf("错误消息不包含超时信息: %v", err)
		}
	})
}

// TestCommandHookErrors 测试命令行钩子的错误处理
func TestCommandHookErrors(t *testing.T) {
	// 测试无效的命令
	t.Run("无效的命令", func(t *testing.T) {
		hook := NewCommandHook("non_existent_command", 5, false)

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		_, err := hook.Before(req)

		if err == nil {
			t.Error("应该检测到无效的命令")
		}

		if !strings.Contains(err.Error(), "命令执行失败") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})

	// 测试命令执行错误
	t.Run("命令执行错误", func(t *testing.T) {
		// 使用会产生错误的jq命令
		hook := NewCommandHook(`jq '.invalid syntax'`, 5, false)

		req, _ := http.NewRequest("POST", "https://example.com",
			bytes.NewBufferString(`{"name":"test"}`))

		_, err := hook.Before(req)

		if err == nil {
			t.Error("应该检测到命令执行错误")
		}

		if !strings.Contains(err.Error(), "命令执行失败") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})

	// 测试异步命令错误处理
	t.Run("异步命令错误", func(t *testing.T) {
		hook := NewCommandHook("non_existent_command", 5, true)

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		_, errChan := hook.BeforeAsync(req)

		select {
		case err := <-errChan:
			if !strings.Contains(err.Error(), "命令执行失败") {
				t.Errorf("错误消息不正确: %v", err)
			}
		case <-time.After(6 * time.Second):
			t.Fatal("异步命令错误处理超时")
		}
	})
}

// TestCommandResponseHook 测试响应命令行钩子
func TestCommandResponseHook(t *testing.T) {
	// 跳过在非Unix环境中的测试
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("跳过测试: 无法找到sh命令")
	}

	// 测试响应处理
	t.Run("处理响应", func(t *testing.T) {
		hook := NewCommandResponseHook(`jq '.processed = true'`, 5, false)

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}

		modifiedResp, err := hook.After(resp)
		if err != nil {
			t.Fatalf("执行响应命令行钩子失败: %v", err)
		}

		body, _ := io.ReadAll(modifiedResp.Body)
		modifiedResp.Body.Close()

		var bodyObj map[string]interface{}
		json.Unmarshal(body, &bodyObj)

		if processed, ok := bodyObj["processed"].(bool); !ok || !processed {
			t.Error("响应命令行钩子未正确修改响应体")
		}
	})

	// 测试异步响应处理
	t.Run("异步处理响应", func(t *testing.T) {
		hook := NewCommandResponseHook(`sleep 0.1 && jq '.async_processed = true'`, 5, true)

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}

		respChan, errChan := hook.AfterAsync(resp)

		select {
		case modifiedResp := <-respChan:
			body, _ := io.ReadAll(modifiedResp.Body)
			modifiedResp.Body.Close()

			var bodyObj map[string]interface{}
			json.Unmarshal(body, &bodyObj)

			if asyncProcessed, ok := bodyObj["async_processed"].(bool); !ok || !asyncProcessed {
				t.Error("异步响应命令行钩子未正确修改响应体")
			}
		case err := <-errChan:
			t.Fatalf("异步执行响应命令行钩子失败: %v", err)
		case <-time.After(6 * time.Second):
			t.Fatal("异步响应命令行钩子执行超时")
		}
	})
}

// TestJSResponseHook 测试JavaScript响应钩子
func TestJSResponseHook(t *testing.T) {
	// 创建临时脚本文件
	tempDir, err := os.MkdirTemp("", "js-response-hook-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建有效响应脚本文件
	validScriptPath := filepath.Join(tempDir, "valid-response-script.js")
	validScriptContent := `
function processResponse(response) {
	// 添加调试输出
	console.log("处理前的响应对象:", JSON.stringify(response));
	console.log("状态码类型:", typeof response.status);
	
	// 直接使用响应体对象（不需要JSON.parse）
	response.body.js_processed = true;
	response.body.timestamp = "2023-01-01T00:00:00Z";
	// 修改状态码（确保是数字类型）
	response.status = Number(201);
	// 添加头部
	response.headers["X-Processed-By"] = "JSHook";
	
	// 再次调试输出
	console.log("处理后的响应对象:", JSON.stringify(response));
	console.log("处理后状态码类型:", typeof response.status);
	
	return response;
}
`
	err = os.WriteFile(validScriptPath, []byte(validScriptContent), 0644)
	if err != nil {
		t.Fatalf("写入响应脚本文件失败: %v", err)
	}

	// 测试从文件创建响应钩子
	t.Run("从文件创建响应钩子", func(t *testing.T) {
		hook, err := NewJSResponseHookFromFile(validScriptPath, false, 30)
		if err != nil {
			t.Fatalf("从文件创建JS响应钩子失败: %v", err)
		}

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}

		modifiedResp, err := hook.After(resp)
		if err != nil {
			t.Fatalf("执行JS响应钩子失败: %v", err)
		}

		// 验证状态码已修改
		if modifiedResp.StatusCode != 201 {
			t.Errorf("JS响应钩子未修改状态码，期望: 201, 实际: %d", modifiedResp.StatusCode)
		}

		// 验证头部已添加
		if modifiedResp.Header.Get("X-Processed-By") != "JSHook" {
			t.Error("JS响应钩子未添加请求头")
		}

		// 验证响应体已修改
		body, _ := io.ReadAll(modifiedResp.Body)
		modifiedResp.Body.Close()

		var bodyObj map[string]interface{}
		err = json.Unmarshal(body, &bodyObj)
		if err != nil {
			t.Fatalf("解析修改后的响应体失败: %v", err)
		}

		if jsProcessed, ok := bodyObj["js_processed"].(bool); !ok || !jsProcessed {
			t.Error("JS响应钩子未正确标记处理过的响应")
		}

		if timestamp, ok := bodyObj["timestamp"].(string); !ok || timestamp != "2023-01-01T00:00:00Z" {
			t.Error("JS响应钩子未正确添加timestamp字段")
		}
	})

	// 测试从字符串创建响应钩子
	t.Run("从字符串创建响应钩子", func(t *testing.T) {
		scriptContent := `
function processResponse(response) {
	// 调试输出
	console.log("字符串创建钩子 - 响应对象:", JSON.stringify(response));
	console.log("字符串钩子 - 状态码类型:", typeof response.status);
	
	response.body.from_string = true;
	// 设置状态码和头部（确保是数字类型）
	response.status = Number(202);
	response.headers["X-String-Hook"] = "测试";
	
	console.log("字符串钩子 - 处理后状态码类型:", typeof response.status);
	return response;
}
`
		hook, err := NewJSResponseHookFromString(scriptContent, false, 30)
		if err != nil {
			t.Fatalf("从字符串创建JS响应钩子失败: %v", err)
		}

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}

		modifiedResp, err := hook.After(resp)
		if err != nil {
			t.Fatalf("执行从字符串创建的JS响应钩子失败: %v", err)
		}

		// 验证状态码
		if modifiedResp.StatusCode != 202 {
			t.Errorf("字符串创建的JS响应钩子未修改状态码，期望: 202, 实际: %d", modifiedResp.StatusCode)
		}

		// 验证头部
		if modifiedResp.Header.Get("X-String-Hook") != "测试" {
			t.Error("字符串创建的JS响应钩子未添加头部")
		}

		body, _ := io.ReadAll(modifiedResp.Body)
		modifiedResp.Body.Close()

		var bodyObj map[string]interface{}
		err = json.Unmarshal(body, &bodyObj)
		if err != nil {
			t.Fatalf("解析修改后的响应体失败: %v", err)
		}

		if fromString, ok := bodyObj["from_string"].(bool); !ok || !fromString {
			t.Error("从字符串创建的JS响应钩子未正确修改响应体")
		}
	})

	// 测试异步执行
	t.Run("异步执行响应钩子", func(t *testing.T) {
		hook, err := NewJSResponseHookFromString(`
function processResponse(response) {
	// 调试输出
	console.log("异步响应钩子 - 响应对象:", JSON.stringify(response));
	console.log("异步钩子 - 状态码类型:", typeof response.status);
	
	response.body.async_processed = true;
	// 设置状态码和头部（确保是数字类型）
	response.status = Number(203);
	response.headers["X-Async-Hook"] = "异步";
	
	console.log("异步钩子 - 处理后状态码类型:", typeof response.status);
	return response;
}
`, true, 30)
		if err != nil {
			t.Fatalf("创建异步JS响应钩子失败: %v", err)
		}

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}

		respChan, errChan := hook.AfterAsync(resp)

		select {
		case modifiedResp := <-respChan:
			// 验证状态码
			if modifiedResp.StatusCode != 203 {
				t.Errorf("异步JS响应钩子未修改状态码，期望: 203, 实际: %d", modifiedResp.StatusCode)
			}

			// 验证头部
			if modifiedResp.Header.Get("X-Async-Hook") != "异步" {
				t.Error("异步JS响应钩子未添加头部")
			}

			body, _ := io.ReadAll(modifiedResp.Body)
			modifiedResp.Body.Close()

			var bodyObj map[string]interface{}
			err = json.Unmarshal(body, &bodyObj)
			if err != nil {
				t.Fatalf("解析修改后的响应体失败: %v", err)
			}

			if asyncProcessed, ok := bodyObj["async_processed"].(bool); !ok || !asyncProcessed {
				t.Error("异步JS响应钩子未正确修改响应体")
			}
		case err := <-errChan:
			t.Fatalf("异步执行JS响应钩子失败: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("异步JS响应钩子执行超时")
		}
	})
}

// TestJSResponseHookErrors 测试JS响应钩子的错误处理
func TestJSResponseHookErrors(t *testing.T) {
	// 测试无效的JavaScript代码
	t.Run("无效的JavaScript代码", func(t *testing.T) {
		invalidScript := `
function processResponse(response) {
	// 这是一个语法错误
	if (true {
		return response;
	}
}
`
		hook, err := NewJSResponseHookFromString(invalidScript, false, 30)
		if err != nil {
			t.Fatalf("创建JS响应钩子失败: %v", err)
		}

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}

		_, err = hook.After(resp)

		if err == nil {
			t.Error("应该检测到无效的JavaScript代码")
		} else if !strings.Contains(err.Error(), "执行脚本失败") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})

	// 测试缺少processResponse函数
	t.Run("缺少processResponse函数", func(t *testing.T) {
		scriptWithoutFunction := `
// 这个脚本没有定义processResponse函数
const x = 10;
console.log("Hello, world!");
`
		hook, err := NewJSResponseHookFromString(scriptWithoutFunction, false, 30)
		if err != nil {
			t.Fatalf("创建JS响应钩子失败: %v", err)
		}

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
			Header:     make(http.Header),
		}

		_, err = hook.After(resp)

		if err == nil {
			t.Error("应该检测到缺少processResponse函数")
		} else if !strings.Contains(err.Error(), "未找到processResponse函数") {
			t.Errorf("错误消息不正确: %v", err)
		}
	})
}

// TestHookChaining 测试钩子链式调用
func TestHookChaining(t *testing.T) {
	// 创建多个钩子
	authHook := &AuthHook{Token: "test-token"}
	loggingHook := &LoggingHook{}

	// 创建自定义函数钩子
	customHook := &CustomFunctionHook{
		BeforeFn: func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Custom", "custom-value")
			return req, nil
		},
	}

	// 创建请求
	req, _ := http.NewRequest("GET", "https://example.com/api", nil)

	// 按顺序应用钩子
	var err error
	req, err = authHook.Before(req)
	if err != nil {
		t.Fatalf("执行认证钩子失败: %v", err)
	}

	req, err = loggingHook.Before(req)
	if err != nil {
		t.Fatalf("执行日志钩子失败: %v", err)
	}

	req, err = customHook.Before(req)
	if err != nil {
		t.Fatalf("执行自定义钩子失败: %v", err)
	}

	// 验证所有钩子都已应用
	authHeader := req.Header.Get("Authorization")
	if authHeader != "Bearer test-token" {
		t.Errorf("认证钩子未正确应用，期望: Bearer test-token, 实际: %s", authHeader)
	}

	customHeader := req.Header.Get("X-Custom")
	if customHeader != "custom-value" {
		t.Errorf("自定义钩子未正确应用，期望: custom-value, 实际: %s", customHeader)
	}
}

// TestAsyncHookChaining 测试异步钩子链式调用
func TestAsyncHookChaining(t *testing.T) {
	// 创建多个异步钩子
	authHook := &AuthHook{Token: "test-token"}

	// 创建自定义函数钩子
	customHook := &CustomFunctionHook{
		BeforeFn: func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Custom", "custom-value")
			return req, nil
		},
	}

	// 创建请求
	req, _ := http.NewRequest("GET", "https://example.com/api", nil)

	// 异步应用第一个钩子
	reqChan1, errChan1 := authHook.BeforeAsync(req)

	var modifiedReq *http.Request
	select {
	case modifiedReq = <-reqChan1:
		// 继续处理
	case err := <-errChan1:
		t.Fatalf("异步执行认证钩子失败: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("异步执行认证钩子超时")
	}

	// 异步应用第二个钩子
	reqChan2, errChan2 := customHook.BeforeAsync(modifiedReq)

	select {
	case modifiedReq = <-reqChan2:
		// 验证所有钩子都已应用
		authHeader := modifiedReq.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("异步认证钩子未正确应用，期望: Bearer test-token, 实际: %s", authHeader)
		}

		customHeader := modifiedReq.Header.Get("X-Custom")
		if customHeader != "custom-value" {
			t.Errorf("异步自定义钩子未正确应用，期望: custom-value, 实际: %s", customHeader)
		}
	case err := <-errChan2:
		t.Fatalf("异步执行自定义钩子失败: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("异步执行自定义钩子超时")
	}
}
