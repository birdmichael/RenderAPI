package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CustomFunctionHook 自定义钩子实现
type CustomFunctionHook struct {
	BeforeFn func(req *http.Request) (*http.Request, error)
	AfterFn  func(resp *http.Response) (*http.Response, error)
}

// Before 执行自定义前置操作
func (h *CustomFunctionHook) Before(req *http.Request) (*http.Request, error) {
	if h.BeforeFn != nil {
		return h.BeforeFn(req)
	}
	return req, nil
}

// BeforeAsync 异步执行自定义前置操作
func (h *CustomFunctionHook) BeforeAsync(req *http.Request) (chan *http.Request, chan error) {
	reqChan := make(chan *http.Request, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedReq, err := h.Before(req)
		if err != nil {
			errChan <- err
			return
		}
		reqChan <- modifiedReq
	}()

	return reqChan, errChan
}

// After 执行自定义后置操作
func (h *CustomFunctionHook) After(resp *http.Response) (*http.Response, error) {
	if h.AfterFn != nil {
		return h.AfterFn(resp)
	}
	return resp, nil
}

// AfterAsync 异步执行自定义后置操作
func (h *CustomFunctionHook) AfterAsync(resp *http.Response) (chan *http.Response, chan error) {
	respChan := make(chan *http.Response, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedResp, err := h.After(resp)
		if err != nil {
			errChan <- err
			return
		}
		respChan <- modifiedResp
	}()

	return respChan, errChan
}

// NewCustomFunctionHook 创建新的自定义钩子
func NewCustomFunctionHook(
	beforeFn func(req *http.Request) (*http.Request, error),
	afterFn func(resp *http.Response) (*http.Response, error),
) *CustomFunctionHook {
	return &CustomFunctionHook{
		BeforeFn: beforeFn,
		AfterFn:  afterFn,
	}
}

// LoggingHook 日志记录钩子
type LoggingHook struct{}

// Before 记录请求信息
func (h *LoggingHook) Before(req *http.Request) (*http.Request, error) {
	fmt.Printf("正在发送 %s 请求到 %s\n", req.Method, req.URL.String())
	return req, nil
}

// BeforeAsync 异步记录请求信息
func (h *LoggingHook) BeforeAsync(req *http.Request) (chan *http.Request, chan error) {
	reqChan := make(chan *http.Request, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedReq, err := h.Before(req)
		if err != nil {
			errChan <- err
			return
		}
		reqChan <- modifiedReq
	}()

	return reqChan, errChan
}

// NewLoggingHook 创建新的日志钩子
func NewLoggingHook() *LoggingHook {
	return &LoggingHook{}
}

// ResponseLogHook 响应日志钩子
type ResponseLogHook struct{}

// After 记录响应信息
func (h *ResponseLogHook) After(resp *http.Response) (*http.Response, error) {
	fmt.Printf("收到响应: 状态码 %d\n", resp.StatusCode)
	return resp, nil
}

// AfterAsync 异步记录响应信息
func (h *ResponseLogHook) AfterAsync(resp *http.Response) (chan *http.Response, chan error) {
	respChan := make(chan *http.Response, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedResp, err := h.After(resp)
		if err != nil {
			errChan <- err
			return
		}
		respChan <- modifiedResp
	}()

	return respChan, errChan
}

// NewResponseLogHook 创建新的响应日志钩子
func NewResponseLogHook() *ResponseLogHook {
	return &ResponseLogHook{}
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

// BeforeAsync 异步添加认证信息
func (h *AuthHook) BeforeAsync(req *http.Request) (chan *http.Request, chan error) {
	reqChan := make(chan *http.Request, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedReq, err := h.Before(req)
		if err != nil {
			errChan <- err
			return
		}
		reqChan <- modifiedReq
	}()

	return reqChan, errChan
}

// NewAuthHook 创建新的认证钩子
func NewAuthHook(token string) *AuthHook {
	return &AuthHook{
		Token: token,
	}
}

// FieldTransformHook 字段转换钩子
type FieldTransformHook struct {
	TransformMap map[string]string // 源字段到目标字段的映射
}

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

	bodyBytes, err := ReadRequestBody(req)
	if err != nil {
		return nil, err
	}

	// 解析JSON
	data, err := parseJSONBody(bodyBytes)
	if err != nil {
		// 如果不是JSON，直接返回原始请求
		req.Body = createBodyReader(bodyBytes)
		return req, nil
	}

	// 应用字段转换
	transformed := false
	for srcField, destField := range h.TransformMap {
		if val, ok := data[srcField]; ok {
			data[destField] = val
			delete(data, srcField)
			transformed = true
		}
	}

	// 只在有转换时重新编码
	if transformed {
		newBody, err := encodeJSONBody(data)
		if err != nil {
			return nil, err
		}

		// 更新请求体
		return ReplaceRequestBody(req, newBody)
	}

	return req, nil
}

// BeforeAsync 异步在请求前转换JSON字段
func (h *FieldTransformHook) BeforeAsync(req *http.Request) (chan *http.Request, chan error) {
	reqChan := make(chan *http.Request, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedReq, err := h.Before(req)
		if err != nil {
			errChan <- err
			return
		}
		reqChan <- modifiedReq
	}()

	return reqChan, errChan
}

// NewFieldTransformHook 创建新的字段转换钩子
func NewFieldTransformHook(transformMap map[string]string) *FieldTransformHook {
	return &FieldTransformHook{
		TransformMap: transformMap,
	}
}

// 辅助函数：解析JSON
func parseJSONBody(body []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	return data, err
}

// 辅助函数：创建请求体reader
func createBodyReader(body []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(body))
}

// 辅助函数：编码JSON
func encodeJSONBody(data map[string]interface{}) ([]byte, error) {
	return json.Marshal(data)
}
