package hooks

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"crypto"

	"github.com/dop251/goja"
)

// JSHook 实现BeforeRequestHook和AsyncBeforeRequestHook接口，用于执行JavaScript预请求脚本
// 可以用于灵活地处理请求体、添加请求头等操作
type JSHook struct {
	ScriptPath    string        // JavaScript脚本文件路径
	ScriptContent string        // JavaScript脚本内容（优先级高于ScriptPath）
	IsAsync       bool          // 是否异步执行
	Timeout       time.Duration // 脚本执行超时时间
}

// NewJSHook 创建一个新的JavaScript钩子
// 参数:
// - scriptPath: JavaScript脚本文件路径
// - isAsync: 是否异步执行
// - timeoutSeconds: 脚本执行超时时间（秒）
func NewJSHook(scriptPath string, isAsync bool, timeoutSeconds int) *JSHook {
	return &JSHook{
		ScriptPath: scriptPath,
		IsAsync:    isAsync,
		Timeout:    time.Duration(timeoutSeconds) * time.Second,
	}
}

// NewJSHookFromFile 从文件创建JavaScript钩子
// 这个函数会检查文件是否存在，但不会验证文件内容的有效性
// 参数:
// - scriptPath: JavaScript脚本文件路径
// - isAsync: 是否异步执行
// - timeoutSeconds: 脚本执行超时时间（秒）
func NewJSHookFromFile(scriptPath string, isAsync bool, timeoutSeconds int) (*JSHook, error) {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("脚本文件不存在: %s", scriptPath)
	}
	return NewJSHook(scriptPath, isAsync, timeoutSeconds), nil
}

// NewJSHookFromString 从字符串内容创建JavaScript钩子
// 推荐在脚本较小或动态生成脚本内容时使用此方法
// 参数:
// - scriptContent: JavaScript脚本内容
// - isAsync: 是否异步执行
// - timeoutSeconds: 脚本执行超时时间（秒）
func NewJSHookFromString(scriptContent string, isAsync bool, timeoutSeconds int) (*JSHook, error) {
	hook := &JSHook{
		ScriptContent: scriptContent,
		IsAsync:       isAsync,
		Timeout:       time.Duration(timeoutSeconds) * time.Second,
	}
	return hook, nil
}

// Before 在请求发送前执行JavaScript脚本
// 如果钩子配置为异步模式，此方法会同步等待异步执行完成，但仍会阻塞直到结果返回或超时
// 实现BeforeRequestHook接口
func (h *JSHook) Before(req *http.Request) (*http.Request, error) {
	if h.IsAsync {
		reqChan, errChan := h.BeforeAsync(req)
		select {
		case modifiedReq := <-reqChan:
			return modifiedReq, nil
		case err := <-errChan:
			return req, err
		case <-time.After(h.Timeout):
			return req, fmt.Errorf("脚本执行超时")
		}
	}

	// 同步执行脚本
	return h.executeScript(req)
}

// BeforeAsync 异步执行JavaScript脚本
// 返回两个通道，一个用于获取处理后的请求，一个用于获取可能发生的错误
// 实现AsyncBeforeRequestHook接口
func (h *JSHook) BeforeAsync(req *http.Request) (chan *http.Request, chan error) {
	reqChan := make(chan *http.Request, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedReq, err := h.executeScript(req)
		if err != nil {
			errChan <- err
			close(reqChan)
			close(errChan)
			return
		}
		reqChan <- modifiedReq
		close(reqChan)
		close(errChan)
	}()

	return reqChan, errChan
}

// executeScript 执行JavaScript脚本并处理请求
// 这是内部方法，用于实际执行JS代码并处理请求
func (h *JSHook) executeScript(req *http.Request) (*http.Request, error) {
	// 获取脚本内容
	scriptContent, err := h.getScriptContent()
	if err != nil {
		return req, err
	}

	// 创建JavaScript运行时
	vm := goja.New()

	// 设置JavaScript环境
	if err := h.setupJSEnvironment(vm); err != nil {
		return req, err
	}

	// 执行脚本
	if _, err := vm.RunString(string(scriptContent)); err != nil {
		return req, fmt.Errorf("执行脚本失败: %w", err)
	}

	// 如果没有请求体，直接返回
	if req.Body == nil {
		return req, nil
	}

	// 处理请求体
	return h.processRequestWithJS(vm, req)
}

// getScriptContent 获取脚本内容，优先使用直接提供的内容，其次从文件读取
func (h *JSHook) getScriptContent() ([]byte, error) {
	// 优先使用ScriptContent
	if h.ScriptContent != "" {
		return []byte(h.ScriptContent), nil
	}

	// 其次使用ScriptPath
	if h.ScriptPath != "" {
		content, err := os.ReadFile(h.ScriptPath)
		if err != nil {
			return nil, fmt.Errorf("读取脚本文件失败: %w", err)
		}
		return content, nil
	}

	return nil, fmt.Errorf("未提供脚本内容或脚本路径")
}

// setupJSEnvironment 设置JavaScript运行环境，添加控制台日志和RSA加密等功能
func (h *JSHook) setupJSEnvironment(vm *goja.Runtime) error {
	// 添加console.log实现
	console := make(map[string]interface{})
	console["log"] = func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Printf("[JS] %v\n", args)
		return goja.Undefined()
	}
	vm.Set("console", console)

	// 添加RSA加密函数
	vm.Set("rsaEncryptGo", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return vm.ToValue("错误: 缺少参数")
		}

		text := call.Arguments[0].String()
		pemKey := call.Arguments[1].String()

		encryptedB64, err := RSAEncrypt(text, pemKey)
		if err != nil {
			return vm.ToValue("错误: " + err.Error())
		}

		return vm.ToValue(encryptedB64)
	})

	return nil
}

// processRequestWithJS 使用JS处理请求
// 将HTTP请求转换为JavaScript对象，调用JS函数处理，再转回HTTP请求
func (h *JSHook) processRequestWithJS(vm *goja.Runtime, req *http.Request) (*http.Request, error) {
	// 读取请求体
	bodyBytes, err := ReadRequestBody(req)
	if err != nil {
		return req, fmt.Errorf("读取请求体失败: %w", err)
	}

	// 解析JSON请求体
	var requestBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
		return req, fmt.Errorf("解析请求体失败: %w", err)
	}

	// 获取请求头
	headers := make(map[string]string)
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// 准备JavaScript请求对象
	jsRequest := map[string]interface{}{
		"body":    requestBody,
		"headers": headers,
		"method":  req.Method,
		"url":     req.URL.String(),
	}

	// 调用JavaScript处理函数
	processRequestFn, ok := goja.AssertFunction(vm.Get("processRequest"))
	if !ok {
		return req, fmt.Errorf("脚本中未找到processRequest函数")
	}

	// 执行处理函数
	result, err := processRequestFn(goja.Undefined(), vm.ToValue(jsRequest))
	if err != nil {
		return req, fmt.Errorf("执行processRequest函数失败: %w", err)
	}

	// 处理JavaScript返回的结果
	return h.handleProcessedRequest(req, result)
}

// getRequestHeaders 获取请求头，返回键值对形式的Map
func getRequestHeaders(req *http.Request) map[string]string {
	headers := make(map[string]string)
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return headers
}

// handleProcessedRequest 处理JavaScript返回的请求对象
// 将JS对象转换回HTTP请求，包括处理请求体和请求头
func (h *JSHook) handleProcessedRequest(req *http.Request, result goja.Value) (*http.Request, error) {
	// 获取处理后的请求对象
	processedRequest, ok := result.Export().(map[string]interface{})
	if !ok {
		return req, fmt.Errorf("无法解析处理后的请求对象")
	}

	// 提取处理后的请求体
	processedBody, ok := processedRequest["body"].(map[string]interface{})
	if !ok {
		return req, fmt.Errorf("无法解析处理后的请求体")
	}

	// 处理请求头
	fmt.Println("处理JS返回的请求头:")
	if headers, ok := processedRequest["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if strVal, ok := v.(string); ok {
				req.Header.Set(k, strVal)
				fmt.Printf("[JS-DEBUG] 设置请求头 %s: %s\n", k, strVal)
			}
		}
	}

	// 打印最终的请求头
	fmt.Println("JS处理后的所有请求头:")
	for k, v := range req.Header {
		fmt.Printf("%s: %v\n", k, v)
	}

	// 将处理后的请求体重新序列化为JSON
	newBodyBytes, err := json.Marshal(processedBody)
	if err != nil {
		return req, fmt.Errorf("序列化处理后的请求体失败: %w", err)
	}

	// 更新请求体
	return ReplaceRequestBody(req, newBodyBytes)
}

// JSResponseHook JavaScript响应钩子，用于在接收到响应后执行JavaScript处理
type JSResponseHook struct {
	ScriptPath    string        // JavaScript脚本文件路径
	ScriptContent string        // JavaScript脚本内容
	IsAsync       bool          // 是否异步执行
	Timeout       time.Duration // 脚本执行超时时间
}

// NewJSResponseHook 创建一个新的JavaScript响应钩子
func NewJSResponseHook(scriptPath string, isAsync bool, timeoutSeconds int) *JSResponseHook {
	return &JSResponseHook{
		ScriptPath: scriptPath,
		IsAsync:    isAsync,
		Timeout:    time.Duration(timeoutSeconds) * time.Second,
	}
}

// NewJSResponseHookFromFile 从文件创建JavaScript响应钩子
// 参数:
// - scriptPath: JavaScript脚本文件路径
// - isAsync: 是否异步执行
// - timeoutSeconds: 脚本执行超时时间（秒）
func NewJSResponseHookFromFile(scriptPath string, isAsync bool, timeoutSeconds int) (*JSResponseHook, error) {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("脚本文件不存在: %s", scriptPath)
	}
	return NewJSResponseHook(scriptPath, isAsync, timeoutSeconds), nil
}

// NewJSResponseHookFromString 从字符串内容创建JavaScript响应钩子
// 参数:
// - scriptContent: JavaScript脚本内容
// - isAsync: 是否异步执行
// - timeoutSeconds: 脚本执行超时时间（秒）
func NewJSResponseHookFromString(scriptContent string, isAsync bool, timeoutSeconds int) (*JSResponseHook, error) {
	hook := &JSResponseHook{
		ScriptContent: scriptContent,
		IsAsync:       isAsync,
		Timeout:       time.Duration(timeoutSeconds) * time.Second,
	}
	return hook, nil
}

// After 在响应接收后执行JavaScript脚本
// 实现AfterResponseHook接口
func (h *JSResponseHook) After(resp *http.Response) (*http.Response, error) {
	if h.IsAsync {
		respChan, errChan := h.AfterAsync(resp)
		select {
		case modifiedResp := <-respChan:
			return modifiedResp, nil
		case err := <-errChan:
			return resp, err
		case <-time.After(h.Timeout):
			return resp, fmt.Errorf("脚本执行超时")
		}
	}

	// 同步执行脚本
	return h.executeScript(resp)
}

// AfterAsync 异步执行JavaScript脚本
// 实现AsyncAfterResponseHook接口
func (h *JSResponseHook) AfterAsync(resp *http.Response) (chan *http.Response, chan error) {
	respChan := make(chan *http.Response, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedResp, err := h.executeScript(resp)
		if err != nil {
			errChan <- err
			close(respChan)
			close(errChan)
			return
		}
		respChan <- modifiedResp
		close(respChan)
		close(errChan)
	}()

	return respChan, errChan
}

// executeScript 执行JavaScript脚本并处理响应
// 这是内部方法，用于实际执行JS代码并处理响应
func (h *JSResponseHook) executeScript(resp *http.Response) (*http.Response, error) {
	// 获取脚本内容
	scriptContent, err := h.getScriptContent()
	if err != nil {
		return resp, err
	}

	// 创建JavaScript运行时
	vm := goja.New()

	// 设置JavaScript环境
	if err := h.setupJSEnvironment(vm); err != nil {
		return resp, err
	}

	// 执行脚本
	if _, err := vm.RunString(string(scriptContent)); err != nil {
		return resp, fmt.Errorf("执行脚本失败: %w", err)
	}

	// 如果没有响应体，直接返回
	if resp.Body == nil {
		return resp, nil
	}

	// 处理响应体
	return h.processResponseWithJS(vm, resp)
}

// getScriptContent 获取脚本内容
// 优先使用ScriptContent，其次使用ScriptPath
func (h *JSResponseHook) getScriptContent() ([]byte, error) {
	if h.ScriptContent != "" {
		return []byte(h.ScriptContent), nil
	}

	if h.ScriptPath != "" {
		content, err := os.ReadFile(h.ScriptPath)
		if err != nil {
			return nil, fmt.Errorf("读取脚本文件失败: %w", err)
		}
		return content, nil
	}

	return nil, fmt.Errorf("未提供脚本内容或脚本路径")
}

// setupJSEnvironment 设置JavaScript运行环境
// 添加控制台日志等功能
func (h *JSResponseHook) setupJSEnvironment(vm *goja.Runtime) error {
	// 添加console.log实现
	console := make(map[string]interface{})
	console["log"] = func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Printf("[JS] %v\n", args)
		return goja.Undefined()
	}
	vm.Set("console", console)

	return nil
}

// processResponseWithJS 使用JS处理响应
// 将HTTP响应转换为JavaScript对象，调用JS函数处理，再转回HTTP响应
func (h *JSResponseHook) processResponseWithJS(vm *goja.Runtime, resp *http.Response) (*http.Response, error) {
	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, fmt.Errorf("读取响应体失败: %w", err)
	}
	resp.Body.Close()

	// 解析响应体 (尝试解析为JSON，如果失败则保留原始内容)
	var responseBody interface{}
	if err := json.Unmarshal(bodyBytes, &responseBody); err != nil {
		// 如果不是JSON，使用原始内容
		responseBody = string(bodyBytes)
	}

	// 准备JavaScript响应对象
	jsResponse := map[string]interface{}{
		"body":    responseBody,
		"status":  resp.StatusCode,
		"headers": getResponseHeaders(resp),
	}

	// 记录原始状态码，用于调试
	originalStatusCode := resp.StatusCode
	fmt.Printf("[DEBUG] 原始状态码: %d\n", originalStatusCode)

	// 调用JavaScript处理函数
	processResponseFn, ok := goja.AssertFunction(vm.Get("processResponse"))
	if !ok {
		// 恢复原始响应体
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		return resp, fmt.Errorf("脚本中未找到processResponse函数")
	}

	// 执行处理函数
	result, err := processResponseFn(goja.Undefined(), vm.ToValue(jsResponse))
	if err != nil {
		// 恢复原始响应体
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		return resp, fmt.Errorf("执行processResponse函数失败: %w", err)
	}

	// 输出处理后的响应对象，用于调试
	fmt.Printf("[DEBUG] 处理后的响应对象: %+v\n", result.Export())

	// 处理JavaScript返回的结果
	return h.handleProcessedResponse(resp, result, bodyBytes)
}

// handleProcessedResponse 处理JavaScript处理后的响应
// 将JS对象转换回HTTP响应，包括处理状态码、响应头和响应体
func (h *JSResponseHook) handleProcessedResponse(resp *http.Response, result goja.Value, originalBody []byte) (*http.Response, error) {
	// 获取处理后的响应
	processedResponse, ok := result.Export().(map[string]interface{})
	if !ok {
		// 恢复原始响应体
		resp.Body = io.NopCloser(bytes.NewBuffer(originalBody))
		return resp, fmt.Errorf("无法解析处理后的响应对象")
	}

	// 处理状态码 - 支持多种数值类型
	if status, ok := processedResponse["status"].(float64); ok {
		resp.StatusCode = int(status)
		fmt.Printf("[DEBUG] 设置状态码为 %d (从float64)\n", int(status))
	} else if status, ok := processedResponse["status"].(int64); ok {
		resp.StatusCode = int(status)
		fmt.Printf("[DEBUG] 设置状态码为 %d (从int64)\n", int(status))
	} else if status, ok := processedResponse["status"].(int); ok {
		resp.StatusCode = status
		fmt.Printf("[DEBUG] 设置状态码为 %d (从int)\n", status)
	}

	// 处理头部 - 支持两种常见的头部格式
	if headers, ok := processedResponse["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if strVal, ok := v.(string); ok {
				resp.Header.Set(k, strVal)
				fmt.Printf("[DEBUG] 设置头部 %s: %s\n", k, strVal)
			}
		}
	} else if headers, ok := processedResponse["headers"].(map[string]string); ok {
		for k, v := range headers {
			resp.Header.Set(k, v)
			fmt.Printf("[DEBUG] 设置头部 %s: %s\n", k, v)
		}
	}

	// 处理响应体
	if body, exists := processedResponse["body"]; exists {
		return h.setResponseBody(resp, body)
	}

	// 如果没有修改响应体，恢复原始响应体
	resp.Body = io.NopCloser(bytes.NewBuffer(originalBody))
	return resp, nil
}

// setResponseBody 设置新的响应体
// 根据body的类型(字符串或其他)设置新的响应体
func (h *JSResponseHook) setResponseBody(resp *http.Response, body interface{}) (*http.Response, error) {
	var newBodyBytes []byte
	var err error

	// 根据类型处理响应体
	switch bodyVal := body.(type) {
	case string:
		newBodyBytes = []byte(bodyVal)
	default:
		// 否则，尝试序列化为JSON
		newBodyBytes, err = json.Marshal(bodyVal)
		if err != nil {
			return resp, fmt.Errorf("序列化处理后的响应体失败: %w", err)
		}
	}

	// 设置新的响应体
	resp.Body = io.NopCloser(bytes.NewBuffer(newBodyBytes))
	// 更新内容长度
	resp.ContentLength = int64(len(newBodyBytes))
	// 删除Content-Length头，让Transport重新计算
	resp.Header.Del("Content-Length")

	return resp, nil
}

// getResponseHeaders 获取响应头，返回键值对形式的Map
func getResponseHeaders(resp *http.Response) map[string]string {
	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return headers
}

// RSAEncrypt 使用RSA-OAEP算法加密文本
// 此函数可在JavaScript中通过rsaEncryptGo函数调用
// 参数:
// - text: 要加密的文本
// - publicKeyPEM: PEM格式的RSA公钥
// 返回:
// - 加密后的Base64编码字符串和可能的错误
func RSAEncrypt(text string, publicKeyPEM string) (string, error) {
	// 解析PEM格式的公钥
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", fmt.Errorf("无法解析PEM格式的公钥")
	}

	// 解析公钥
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("解析公钥失败: %w", err)
	}

	// 转换为RSA公钥
	rsaPublicKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("不是有效的RSA公钥")
	}

	// 使用RSA-OAEP加密数据，使用SHA-256哈希函数
	encryptedBytes, err := rsa.EncryptOAEP(
		crypto.SHA256.New(),
		rand.Reader,
		rsaPublicKey,
		[]byte(text),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("RSA-OAEP加密失败: %w", err)
	}

	// 返回Base64编码的加密结果
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}
