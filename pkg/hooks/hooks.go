// Package hooks 提供请求和响应钩子功能
package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// BeforeRequestHook 定义请求前钩子接口
type BeforeRequestHook interface {
	Before(req *http.Request) (*http.Request, error)
}

// AfterResponseHook 定义响应后钩子接口
type AfterResponseHook interface {
	After(resp *http.Response) (*http.Response, error)
}

// LoggingHook 日志记录钩子
type LoggingHook struct{}

// Before 记录请求信息
func (h *LoggingHook) Before(req *http.Request) (*http.Request, error) {
	fmt.Printf("正在发送 %s 请求到 %s\n", req.Method, req.URL.String())
	return req, nil
}

// AuthHook 认证钩子
type AuthHook struct {
	Token string
}

// Before 添加认证信息
func (h *AuthHook) Before(req *http.Request) (*http.Request, error) {
	req.Header.Set("Authorization", "Bearer "+h.Token)
	return req, nil
}

// ResponseLogHook 响应日志钩子
type ResponseLogHook struct{}

// After 记录响应信息
func (h *ResponseLogHook) After(resp *http.Response) (*http.Response, error) {
	fmt.Printf("收到响应: 状态码 %d\n", resp.StatusCode)
	return resp, nil
}

// CustomHook 自定义钩子实现
type CustomHook struct {
	BeforeFn func(req *http.Request) (*http.Request, error)
	AfterFn  func(resp *http.Response) (*http.Response, error)
}

// Before 执行自定义前置操作
func (h *CustomHook) Before(req *http.Request) (*http.Request, error) {
	if h.BeforeFn != nil {
		return h.BeforeFn(req)
	}
	return req, nil
}

// After 执行自定义后置操作
func (h *CustomHook) After(resp *http.Response) (*http.Response, error) {
	if h.AfterFn != nil {
		return h.AfterFn(resp)
	}
	return resp, nil
}

// FieldTransformHook 字段转换钩子
type FieldTransformHook struct{}

// Before 在请求前转换JSON字段
func (h *FieldTransformHook) Before(req *http.Request) (*http.Request, error) {
	// 只处理POST和PUT请求
	if req.Method != http.MethodPost && req.Method != http.MethodPut {
		return req, nil
	}

	// 读取请求体
	if req.Body == nil {
		return req, nil
	}

	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败: %w", err)
	}

	// 解析JSON
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// 如果不是JSON，直接返回原始请求
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		return req, nil
	}

	// 字段转换
	if userValue, ok := data["user"]; ok {
		data["phone"] = userValue
		delete(data, "user")
	}

	// 重新编码为JSON
	newBody, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("JSON编码失败: %w", err)
	}

	// 更新请求
	req.Body = io.NopCloser(bytes.NewBuffer(newBody))
	req.ContentLength = int64(len(newBody))

	return req, nil
}

// NewScriptHookFromFile 从文件创建脚本钩子
func NewScriptHookFromFile(scriptFile string) (BeforeRequestHook, error) {
	if _, err := os.Stat(scriptFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("脚本文件不存在: %s", scriptFile)
	}
	return NewScriptHook(scriptFile), nil
}
