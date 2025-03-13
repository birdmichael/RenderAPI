package hooks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"
)

// CommandHook 命令行执行钩子
type CommandHook struct {
	Command string
	Timeout time.Duration
	IsAsync bool
}

// NewCommandHook 创建一个新的命令行执行钩子
func NewCommandHook(command string, timeoutSeconds int, isAsync bool) *CommandHook {
	return &CommandHook{
		Command: command,
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		IsAsync: isAsync,
	}
}

// Before 执行命令行命令处理请求
func (h *CommandHook) Before(req *http.Request) (*http.Request, error) {
	if h.IsAsync {
		reqChan, errChan := h.BeforeAsync(req)
		select {
		case modifiedReq := <-reqChan:
			return modifiedReq, nil
		case err := <-errChan:
			return req, err
		case <-time.After(h.Timeout):
			return req, fmt.Errorf("命令执行超时")
		}
	}

	// 同步执行命令
	return h.executeCommand(req)
}

// BeforeAsync 异步执行命令行命令处理请求
func (h *CommandHook) BeforeAsync(req *http.Request) (chan *http.Request, chan error) {
	reqChan := make(chan *http.Request, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedReq, err := h.executeCommand(req)
		if err != nil {
			errChan <- err
			return
		}
		reqChan <- modifiedReq
	}()

	return reqChan, errChan
}

// executeCommand 执行命令行命令
func (h *CommandHook) executeCommand(req *http.Request) (*http.Request, error) {
	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), h.Timeout)
	defer cancel()

	// 准备命令
	cmd := exec.CommandContext(ctx, "sh", "-c", h.Command)

	// 如果有请求体，通过stdin传递
	if req.Body != nil {
		bodyBytes, err := ReadRequestBody(req)
		if err != nil {
			return req, fmt.Errorf("读取请求体失败: %w", err)
		}

		// 将请求体传递给命令的标准输入
		cmd.Stdin = bytes.NewBuffer(bodyBytes)
	}

	// 捕获标准输出和错误
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		return req, fmt.Errorf("命令执行失败: %v, stderr: %s", err, stderr.String())
	}

	// 如果有输出，使用它来替换请求体
	if stdout.Len() > 0 {
		updatedReq, err := ReplaceRequestBody(req, stdout.Bytes())
		if err != nil {
			return req, fmt.Errorf("更新请求体失败: %w", err)
		}
		return updatedReq, nil
	}

	return req, nil
}

// CommandResponseHook 命令行执行响应钩子
type CommandResponseHook struct {
	Command string
	Timeout time.Duration
	IsAsync bool
}

// NewCommandResponseHook 创建一个新的命令行执行响应钩子
func NewCommandResponseHook(command string, timeoutSeconds int, isAsync bool) *CommandResponseHook {
	return &CommandResponseHook{
		Command: command,
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		IsAsync: isAsync,
	}
}

// After 执行命令行命令处理响应
func (h *CommandResponseHook) After(resp *http.Response) (*http.Response, error) {
	if h.IsAsync {
		respChan, errChan := h.AfterAsync(resp)
		select {
		case modifiedResp := <-respChan:
			return modifiedResp, nil
		case err := <-errChan:
			return resp, err
		case <-time.After(h.Timeout):
			return resp, fmt.Errorf("命令执行超时")
		}
	}

	// 同步执行命令
	return h.executeCommand(resp)
}

// AfterAsync 异步执行命令行命令处理响应
func (h *CommandResponseHook) AfterAsync(resp *http.Response) (chan *http.Response, chan error) {
	respChan := make(chan *http.Response, 1)
	errChan := make(chan error, 1)

	go func() {
		modifiedResp, err := h.executeCommand(resp)
		if err != nil {
			errChan <- err
			return
		}
		respChan <- modifiedResp
	}()

	return respChan, errChan
}

// executeCommand 执行命令行命令处理响应
func (h *CommandResponseHook) executeCommand(resp *http.Response) (*http.Response, error) {
	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), h.Timeout)
	defer cancel()

	// 准备命令
	cmd := exec.CommandContext(ctx, "sh", "-c", h.Command)

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, fmt.Errorf("读取响应体失败: %w", err)
	}
	resp.Body.Close()

	// 将响应体传递给命令的标准输入
	cmd.Stdin = bytes.NewBuffer(bodyBytes)

	// 捕获标准输出和错误
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		// 恢复原始响应体
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		return resp, fmt.Errorf("命令执行失败: %v, stderr: %s", err, stderr.String())
	}

	// 如果有输出，使用它来替换响应体
	if stdout.Len() > 0 {
		resp.Body = io.NopCloser(bytes.NewBuffer(stdout.Bytes()))
		// 更新内容长度
		resp.ContentLength = int64(stdout.Len())
		// 删除Content-Length头，让Transport重新计算
		resp.Header.Del("Content-Length")
		return resp, nil
	}

	// 恢复原始响应体
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return resp, nil
}
