// Package client 提供HTTP客户端功能，支持模板驱动的请求
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/birdmichael/RenderAPI/pkg/hooks"
	"github.com/birdmichael/RenderAPI/pkg/template"
)

// Client 提供HTTP请求功能
type Client struct {
	client         *http.Client
	baseURL        string
	headers        map[string]string
	beforeHook     []hooks.BeforeRequestHook
	afterHook      []hooks.AfterResponseHook
	templateEngine *template.Engine
}

// NewClient 创建一个新的HTTP客户端
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL:        baseURL,
		headers:        make(map[string]string),
		templateEngine: template.NewEngine(),
	}
}

// SetHeader 设置HTTP请求头
func (c *Client) SetHeader(key, value string) {
	c.headers[key] = value
}

// AddBeforeHook 添加请求前钩子
func (c *Client) AddBeforeHook(hook hooks.BeforeRequestHook) {
	c.beforeHook = append(c.beforeHook, hook)
}

// AddAfterHook 添加响应后钩子
func (c *Client) AddAfterHook(hook hooks.AfterResponseHook) {
	c.afterHook = append(c.afterHook, hook)
}

// AddScriptHookFromFile 从文件添加脚本钩子
func (c *Client) AddScriptHookFromFile(scriptFile string) error {
	hook, err := hooks.NewScriptHookFromFile(scriptFile)
	if err != nil {
		return err
	}
	c.AddBeforeHook(hook)
	return nil
}

// GetTemplateEngine 获取模板引擎
func (c *Client) GetTemplateEngine() *template.Engine {
	return c.templateEngine
}

// ExecuteTemplateFile 使用模板文件执行请求
func (c *Client) ExecuteTemplateFile(ctx context.Context, templateFile string, data interface{}) (*http.Response, error) {
	// 加载模板文件
	tmplContent, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("读取模板文件失败: %w", err)
	}

	return c.ExecuteTemplateJSON(ctx, string(tmplContent), data)
}

// ExecuteTemplateJSON 使用JSON字符串模板执行请求
func (c *Client) ExecuteTemplateJSON(ctx context.Context, templateJSON string, data interface{}) (*http.Response, error) {
	// 解析模板定义
	var tmplDef struct {
		Request struct {
			Method  string            `json:"method"`
			BaseURL string            `json:"baseURL"`
			Path    string            `json:"path"`
			Headers map[string]string `json:"headers"`
		} `json:"request"`
		Body map[string]interface{} `json:"body"`
	}

	if err := json.Unmarshal([]byte(templateJSON), &tmplDef); err != nil {
		return nil, fmt.Errorf("解析模板定义失败: %w", err)
	}

	// 生成唯一模板ID
	templateID := fmt.Sprintf("template_%d", time.Now().UnixNano())

	// 添加正文模板
	bodyTemplate, err := json.Marshal(tmplDef.Body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体模板失败: %w", err)
	}

	if err := c.templateEngine.AddTemplate(templateID, string(bodyTemplate)); err != nil {
		return nil, fmt.Errorf("添加请求体模板失败: %w", err)
	}

	// 渲染请求体
	renderedBody, err := c.templateEngine.RenderJSONTemplate(templateID, data)
	if err != nil {
		return nil, fmt.Errorf("渲染请求体失败: %w", err)
	}

	// 确定URL和路径
	baseURL := c.baseURL
	if tmplDef.Request.BaseURL != "" {
		baseURL = tmplDef.Request.BaseURL
	}

	// 发送请求
	method := tmplDef.Request.Method
	if method == "" {
		method = "GET"
	}

	// 合并请求头
	headers := make(map[string]string)
	for k, v := range c.headers {
		headers[k] = v
	}
	for k, v := range tmplDef.Request.Headers {
		headers[k] = v
	}

	// 创建请求对象
	req, err := http.NewRequestWithContext(
		ctx,
		method,
		baseURL+tmplDef.Request.Path,
		bytes.NewReader(renderedBody),
	)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 执行前置钩子
	for _, hook := range c.beforeHook {
		req, err = hook.Before(req)
		if err != nil {
			return nil, fmt.Errorf("前置钩子执行失败: %w", err)
		}
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	// 执行后置钩子
	for _, hook := range c.afterHook {
		resp, err = hook.After(resp)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("后置钩子执行失败: %w", err)
		}
	}

	return resp, nil
}

// ExecuteTemplateWithDataFile 使用模板文件和数据文件执行请求
func (c *Client) ExecuteTemplateWithDataFile(ctx context.Context, templateFile, dataFile string) (*http.Response, error) {
	// 加载数据文件
	data, err := LoadDataFromFile(dataFile)
	if err != nil {
		return nil, err
	}

	// 执行模板
	return c.ExecuteTemplateFile(ctx, templateFile, data)
}

// LoadDataFromFile 从文件加载数据
func LoadDataFromFile(dataFile string) (map[string]interface{}, error) {
	content, err := os.ReadFile(dataFile)
	if err != nil {
		return nil, fmt.Errorf("读取数据文件失败: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("解析数据JSON失败: %w", err)
	}

	return data, nil
}

// Request 发送HTTP请求
func (c *Client) Request(method, path string, body []byte) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// 执行前置钩子
	for _, hook := range c.beforeHook {
		req, err = hook.Before(req)
		if err != nil {
			return nil, fmt.Errorf("前置钩子执行失败: %w", err)
		}
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	// 执行后置钩子
	for _, hook := range c.afterHook {
		resp, err = hook.After(resp)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("后置钩子执行失败: %w", err)
		}
	}

	return resp, nil
}

// Get 发送GET请求
func (c *Client) Get(path string) (*http.Response, error) {
	return c.Request(http.MethodGet, path, nil)
}

// Post 发送POST请求
func (c *Client) Post(path string, body []byte) (*http.Response, error) {
	return c.Request(http.MethodPost, path, body)
}

// Put 发送PUT请求
func (c *Client) Put(path string, body []byte) (*http.Response, error) {
	return c.Request(http.MethodPut, path, body)
}

// Delete 发送DELETE请求
func (c *Client) Delete(path string) (*http.Response, error) {
	return c.Request(http.MethodDelete, path, nil)
}

// ReadResponseBody 读取响应主体
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Response 封装HTTP响应
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// NewResponseFromHTTP 从http.Response创建Response
func NewResponseFromHTTP(resp *http.Response) (*Response, error) {
	body, err := ReadResponseBody(resp)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	}, nil
}

// String 返回响应体的字符串表示
func (r *Response) String() string {
	return string(r.Body)
}

// JSON 返回响应体的格式化JSON字符串
func (r *Response) JSON() (string, error) {
	var data interface{}
	if err := json.Unmarshal(r.Body, &data); err != nil {
		return "", err
	}

	formattedJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formattedJSON), nil
}
