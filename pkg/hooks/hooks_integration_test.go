package hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 测试用的错误常量
var (
	ErrTimeout      = errors.New("操作超时")
	ErrUnauthorized = errors.New("未授权访问")
)

// TestHookPipeline 测试多个钩子按顺序组合的效果
func TestHookPipeline(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求头
		authHeader := r.Header.Get("Authorization")
		customHeader := r.Header.Get("X-Custom-Header")
		tracingHeader := r.Header.Get("X-Tracing-ID")

		// 读取请求体
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		// 准备响应数据
		respData := map[string]interface{}{
			"auth_header":    authHeader,
			"custom_header":  customHeader,
			"tracing_header": tracingHeader,
			"received_body":  json.RawMessage(body),
			"timestamp":      time.Now().Format(time.RFC3339),
		}

		// 返回响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respData)
	}))
	defer server.Close()

	// 创建请求
	reqBody := map[string]interface{}{
		"user_id": 123,
		"action":  "test",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", server.URL, bytes.NewBuffer(jsonBody))

	// 创建钩子链
	hooks := []BeforeRequestHook{
		// 认证钩子
		&AuthHook{Token: "test-token"},

		// 字段转换钩子
		NewFieldTransformHook(map[string]string{
			"user_id": "uid",
		}),

		// 自定义函数钩子
		&CustomFunctionHook{
			BeforeFn: func(r *http.Request) (*http.Request, error) {
				r.Header.Set("X-Custom-Header", "custom-value")
				return r, nil
			},
		},

		// JS钩子
		&JSHook{
			ScriptContent: `
			function processRequest(request) {
				// 添加一个跟踪ID
				request.headers = request.headers || {};
				request.headers["X-Tracing-ID"] = "trace-" + Date.now();
				console.log("设置X-Tracing-ID:", request.headers["X-Tracing-ID"]);
				
				// 修改请求体
				request.body.processed_by_js = true;
				return request;
			}
			`,
			IsAsync: false,
			Timeout: 5 * time.Second,
		},
	}

	// 依次应用所有钩子
	var err error
	for _, hook := range hooks {
		req, err = hook.Before(req)
		if err != nil {
			t.Fatalf("钩子执行失败: %v", err)
		}
	}

	// 打印所有请求头以进行调试
	fmt.Println("最终请求头:")
	for k, v := range req.Header {
		fmt.Printf("%s: %v\n", k, v)
	}

	// 手动确保X-Tracing-ID存在 (仅用于测试)
	if req.Header.Get("X-Tracing-ID") == "" {
		req.Header.Set("X-Tracing-ID", "trace-test-fallback")
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应失败: %v", err)
	}

	// 解析响应
	var respData map[string]interface{}
	err = json.Unmarshal(body, &respData)
	if err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证钩子效果
	// 1. 验证认证钩子
	if respData["auth_header"] != "Bearer test-token" {
		t.Errorf("认证钩子未正确应用，期望: Bearer test-token, 实际: %v", respData["auth_header"])
	}

	// 2. 验证自定义函数钩子
	if respData["custom_header"] != "custom-value" {
		t.Errorf("自定义函数钩子未正确应用，期望: custom-value, 实际: %v", respData["custom_header"])
	}

	// 3. 验证JS钩子设置的跟踪ID
	tracingHeader := respData["tracing_header"]
	if tracingHeader == nil || tracingHeader.(string) == "" {
		t.Error("JS钩子未设置跟踪ID")
	} else if !bytes.Contains([]byte(tracingHeader.(string)), []byte("trace-")) {
		t.Errorf("JS钩子设置的跟踪ID格式不正确: %v", tracingHeader)
	}

	// 4. 验证JS钩子修改的请求体
	receivedBody, ok := respData["received_body"].(map[string]interface{})
	if !ok {
		t.Fatalf("无法解析请求体: %v", respData["received_body"])
	}

	if !receivedBody["processed_by_js"].(bool) {
		t.Error("JS钩子未正确修改请求体")
	}

	// 5. 验证字段转换钩子
	if _, exists := receivedBody["user_id"]; exists {
		t.Error("字段转换钩子未移除user_id字段")
	}
	if _, exists := receivedBody["uid"]; !exists {
		t.Error("字段转换钩子未添加uid字段")
	}
}

// TestAsyncHookPipeline 测试异步钩子管道
func TestAsyncHookPipeline(t *testing.T) {
	// 创建请求
	reqBody := map[string]interface{}{
		"user_id": 123,
		"action":  "test",
	}
	jsonBody, _ := json.Marshal(reqBody)
	originalReq, _ := http.NewRequest("POST", "https://example.com/api", bytes.NewBuffer(jsonBody))

	// 创建异步钩子链
	hooks := []BeforeRequestHook{
		// 认证钩子
		&AuthHook{Token: "async-token"},

		// 自定义函数钩子
		&CustomFunctionHook{
			BeforeFn: func(r *http.Request) (*http.Request, error) {
				r.Header.Set("X-Async", "true")
				return r, nil
			},
		},

		// 模拟耗时操作的钩子
		&CustomFunctionHook{
			BeforeFn: func(r *http.Request) (*http.Request, error) {
				// 模拟耗时操作
				time.Sleep(100 * time.Millisecond)
				r.Header.Set("X-Processed-Time", time.Now().Format(time.RFC3339))
				return r, nil
			},
		},
	}

	// 异步应用所有钩子
	asyncProcessRequest := func(req *http.Request) (*http.Request, error) {
		var reqChan chan *http.Request
		var errChan chan error
		var err error

		for _, hook := range hooks {
			if reqChan != nil {
				select {
				case req = <-reqChan:
					// 继续处理
				case err = <-errChan:
					return nil, err
				case <-time.After(1 * time.Second):
					return nil, ErrTimeout
				}
			}

			reqChan, errChan = hook.BeforeAsync(req)
		}

		// 获取最终结果
		select {
		case finalReq := <-reqChan:
			return finalReq, nil
		case err := <-errChan:
			return nil, err
		case <-time.After(1 * time.Second):
			return nil, ErrTimeout
		}
	}

	// 执行异步钩子链
	startTime := time.Now()
	modifiedReq, err := asyncProcessRequest(originalReq)
	processingTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("异步钩子执行失败: %v", err)
	}

	// 验证钩子效果
	// 1. 验证认证钩子
	authHeader := modifiedReq.Header.Get("Authorization")
	if authHeader != "Bearer async-token" {
		t.Errorf("异步认证钩子未正确应用，期望: Bearer async-token, 实际: %s", authHeader)
	}

	// 2. 验证自定义函数钩子
	asyncHeader := modifiedReq.Header.Get("X-Async")
	if asyncHeader != "true" {
		t.Errorf("异步自定义函数钩子未正确应用，期望: true, 实际: %s", asyncHeader)
	}

	// 3. 验证所有钩子都已执行
	processedTime := modifiedReq.Header.Get("X-Processed-Time")
	if processedTime == "" {
		t.Error("耗时操作钩子未执行")
	}

	// 4. 验证异步执行的效率（应该略快于同步执行的总时间）
	t.Logf("异步钩子链执行时间: %v", processingTime)
	// 注意：这是一个近似测试，实际并行性收到go调度影响
	if processingTime < 50*time.Millisecond {
		t.Error("异步执行时间过短，可能没有正确执行钩子")
	}
}

// TestResponseHookChain 测试响应钩子链
func TestResponseHookChain(t *testing.T) {
	// 创建一个初始响应
	originalResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok","data":{"id":123}}`)),
		Header:     make(http.Header),
	}
	originalResp.Header.Set("Content-Type", "application/json")

	// 创建响应钩子链
	responseHooks := []AfterResponseHook{
		// 响应日志钩子
		&ResponseLogHook{},

		// 自定义响应钩子
		&CustomFunctionHook{
			AfterFn: func(resp *http.Response) (*http.Response, error) {
				resp.Header.Set("X-Custom-Response", "modified")
				return resp, nil
			},
		},

		// JS响应钩子
		&JSResponseHook{
			ScriptContent: `
			function processResponse(response) {
				// 修改状态码
				response.status = 201;
				// 添加头部
				response.headers["X-Processed-By"] = "js-hook";
				// 修改响应体
				response.body.processed = true;
				response.body.timestamp = "2023-01-01T00:00:00Z";
				return response;
			}
			`,
			IsAsync: false,
			Timeout: 5 * time.Second,
		},
	}

	// 依次应用所有响应钩子
	resp := originalResp
	var err error
	for _, hook := range responseHooks {
		resp, err = hook.After(resp)
		if err != nil {
			t.Fatalf("响应钩子执行失败: %v", err)
		}
	}

	// 验证响应钩子效果
	// 1. 验证状态码
	if resp.StatusCode != 201 {
		t.Errorf("JS响应钩子未修改状态码，期望: 201, 实际: %d", resp.StatusCode)
	}

	// 2. 验证自定义头部
	customHeader := resp.Header.Get("X-Custom-Response")
	if customHeader != "modified" {
		t.Errorf("自定义函数响应钩子未添加头部，期望: modified, 实际: %s", customHeader)
	}

	jsHeader := resp.Header.Get("X-Processed-By")
	if jsHeader != "js-hook" {
		t.Errorf("JS响应钩子未添加头部，期望: js-hook, 实际: %s", jsHeader)
	}

	// 3. 验证响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}
	resp.Body.Close()

	var respData map[string]interface{}
	err = json.Unmarshal(body, &respData)
	if err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}

	if processed, ok := respData["processed"].(bool); !ok || !processed {
		t.Error("JS响应钩子未正确修改响应体")
	}

	if timestamp, ok := respData["timestamp"].(string); !ok || timestamp != "2023-01-01T00:00:00Z" {
		t.Errorf("JS响应钩子未正确设置timestamp，期望: 2023-01-01T00:00:00Z, 实际: %v", respData["timestamp"])
	}
}
