// Package hooks 提供请求处理前后的钩子功能
package hooks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// 定义错误类型
var (
	ErrJSHookMissingSourceOrContent  = errors.New("JS钩子必须指定source或content")
	ErrCmdHookMissingSourceOrContent = errors.New("命令钩子必须指定source或content")
	ErrCustomHookNotSupported        = errors.New("自定义钩子不能通过模板创建，需要在代码中注册")
	ErrUnsupportedHookType           = errors.New("不支持的钩子类型")
)

// BeforeRequestHookFunc 请求前钩子函数
type BeforeRequestHookFunc func(*http.Request) (*http.Request, error)

// AfterResponseHookFunc 响应后钩子函数
type AfterResponseHookFunc func(*http.Response) (*http.Response, error)

// BeforeRequestHook 请求前钩子接口
type BeforeRequestHook interface {
	Before(req *http.Request) (*http.Request, error)
	BeforeAsync(req *http.Request) (chan *http.Request, chan error)
}

// AfterResponseHook 响应后钩子接口
type AfterResponseHook interface {
	After(resp *http.Response) (*http.Response, error)
	AfterAsync(resp *http.Response) (chan *http.Response, chan error)
}

// Hook 通用钩子接口
type Hook interface {
	GetConfig() *HookConfig
}

// HookConfig 钩子配置
type HookConfig struct {
	Type           string
	Name           string
	Async          bool
	TimeoutSeconds int
}

// HookDefinition 钩子定义
type HookDefinition struct {
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Script   string            `json:"script,omitempty"`
	Command  string            `json:"command,omitempty"`
	Function string            `json:"function,omitempty"`
	Config   map[string]string `json:"config,omitempty"`
	Async    bool              `json:"async,omitempty"`
	Timeout  int               `json:"timeout,omitempty"`
}

// ReadRequestBody 读取请求体内容并重置Body
func ReadRequestBody(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return []byte{}, nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}

	// 重置请求体，以便后续处理可以再次读取
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return bodyBytes, nil
}

// ReplaceRequestBody 替换请求的正文内容
func ReplaceRequestBody(req *http.Request, bodyBytes []byte) (*http.Request, error) {
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.ContentLength = int64(len(bodyBytes))
	return req, nil
}

// IsBodyJSON 检查请求体是否为JSON格式
func IsBodyJSON(req *http.Request) bool {
	contentType := req.Header.Get("Content-Type")
	return contentType == "application/json" || contentType == "application/json; charset=utf-8"
}

// ExecuteHookWithTimeout 带超时执行钩子
func ExecuteHookWithTimeout(ctx context.Context, hook func() error, timeoutSeconds int) error {
	if timeoutSeconds <= 0 {
		// 默认超时10秒
		timeoutSeconds = 10
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- hook()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return errors.New("hook execution timed out")
	}
}

// CreateHookFromDefinition 从定义创建钩子
func CreateHookFromDefinition(def *HookDefinition) (interface{}, error) {
	switch def.Type {
	case "js":
		return NewJSHookFromString(def.Script, def.Async, def.Timeout)
	case "command":
		return NewCommandHook(def.Command, def.Timeout, def.Async), nil
	case "function":
		return nil, fmt.Errorf("未实现的钩子类型: %s", def.Type)
	default:
		return nil, fmt.Errorf("未知的钩子类型: %s", def.Type)
	}
}
