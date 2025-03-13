// Package client 提供HTTP客户端功能，支持模板驱动的请求
package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/birdmichael/RenderAPI/pkg/hooks"
	"github.com/birdmichael/RenderAPI/pkg/template"
)

// CachedResponse 缓存的响应
type CachedResponse struct {
	Response   *http.Response
	Body       []byte
	ExpireTime time.Time
}

// Client 提供HTTP请求功能
type Client struct {
	client         *http.Client
	baseURL        string
	headers        map[string]string
	beforeHook     []hooks.BeforeRequestHook
	afterHook      []hooks.AfterResponseHook
	templateEngine *template.Engine
	cache          map[string]*CachedResponse // 缓存
	cacheMutex     sync.RWMutex               // 缓存锁
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
		cache:          make(map[string]*CachedResponse),
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

// AddJSHookFromFile 从文件添加JavaScript钩子
func (c *Client) AddJSHookFromFile(scriptFile string, isAsync bool, timeoutSeconds int) error {
	hook, err := hooks.NewJSHookFromFile(scriptFile, isAsync, timeoutSeconds)
	if err != nil {
		return err
	}
	c.AddBeforeHook(hook)
	return nil
}

// AddJSHookFromString 从字符串添加JavaScript钩子
func (c *Client) AddJSHookFromString(scriptContent string, isAsync bool, timeoutSeconds int) error {
	hook, err := hooks.NewJSHookFromString(scriptContent, isAsync, timeoutSeconds)
	if err != nil {
		return err
	}
	c.AddBeforeHook(hook)
	return nil
}

// AddCommandHook 添加命令行执行钩子
func (c *Client) AddCommandHook(command string, isAsync bool, timeoutSeconds int) {
	hook := hooks.NewCommandHook(command, timeoutSeconds, isAsync)
	c.AddBeforeHook(hook)
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

// ExecuteTemplateWithDataFile 使用模板文件和数据文件执行请求
func (c *Client) ExecuteTemplateWithDataFile(ctx context.Context, templateFile, dataFile string) (*http.Response, error) {
	// 加载模板文件
	tmplContent, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("读取模板文件失败: %w", err)
	}

	// 加载数据文件
	dataContent, err := os.ReadFile(dataFile)
	if err != nil {
		return nil, fmt.Errorf("读取数据文件失败: %w", err)
	}

	// 解析数据
	var data interface{}
	if err := json.Unmarshal(dataContent, &data); err != nil {
		return nil, fmt.Errorf("解析数据文件失败: %w", err)
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
			Timeout int               `json:"timeout"`
		} `json:"request"`
		Body        map[string]interface{} `json:"body"`
		BeforeHooks []hooks.HookDefinition `json:"beforeHooks"`
		AfterHooks  []hooks.HookDefinition `json:"afterHooks"`
		Caching     struct {
			Enabled    bool   `json:"enabled"`
			TTL        int    `json:"ttl"`
			KeyPattern string `json:"keyPattern"`
		} `json:"caching"`
		Retry struct {
			Enabled       bool `json:"enabled"`
			MaxAttempts   int  `json:"maxAttempts"`
			InitialDelay  int  `json:"initialDelay"`
			BackoffFactor int  `json:"backoffFactor"`
		} `json:"retry"`
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
		// 使用模板引擎渲染头部值
		if err := c.templateEngine.AddTemplate(templateID+"_header_"+key, value); err != nil {
			return nil, fmt.Errorf("添加头部模板失败: %w", err)
		}
		renderedValue, err := c.templateEngine.Execute(templateID+"_header_"+key, data)
		if err != nil {
			return nil, fmt.Errorf("渲染请求头值失败: %w", err)
		}
		req.Header.Set(key, renderedValue)
	}

	// 设置Content-Type（如果未指定）
	if req.Header.Get("Content-Type") == "" && (method == "POST" || method == "PUT" || method == "PATCH") {
		req.Header.Set("Content-Type", "application/json")
	}

	// 处理模板中定义的前置钩子
	for _, hookDef := range tmplDef.BeforeHooks {
		hook, err := hooks.CreateHookFromDefinition(&hookDef)
		if err != nil {
			return nil, fmt.Errorf("创建请求前钩子失败: %w", err)
		}

		// 根据接口类型添加钩子
		beforeHook, ok := hook.(hooks.BeforeRequestHook)
		if !ok {
			return nil, fmt.Errorf("钩子类型不是请求前钩子: %T", hook)
		}

		// 执行请求前钩子
		req, err = beforeHook.Before(req)
		if err != nil {
			return nil, fmt.Errorf("执行请求前钩子失败: %w", err)
		}
	}

	// 应用全局钩子（在模板钩子之后应用，可以覆盖模板钩子的设置）
	for _, hook := range c.beforeHook {
		req, err = hook.Before(req)
		if err != nil {
			return nil, fmt.Errorf("执行请求前钩子失败: %w", err)
		}
	}

	// 设置超时
	clientCopy := *c.client
	if tmplDef.Request.Timeout > 0 {
		clientCopy.Timeout = time.Duration(tmplDef.Request.Timeout) * time.Second
	}

	// 处理缓存逻辑
	if tmplDef.Caching.Enabled {
		// 读取请求体
		var reqBodyBytes []byte
		if req.Body != nil {
			reqBodyBytes, _ = hooks.ReadRequestBody(req)
			// 重新设置请求体
			req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
		}

		// 生成缓存键
		cacheKey := tmplDef.Caching.KeyPattern
		if cacheKey == "" {
			// 使用请求URL和正文作为缓存键
			cacheKey = req.URL.String()
			if len(reqBodyBytes) > 0 {
				bodyHash := fmt.Sprintf("%x", sha256.Sum256(reqBodyBytes))
				cacheKey = cacheKey + ":" + bodyHash
			}
		} else {
			// 使用模板渲染缓存键模式
			renderedKey, err := c.templateEngine.Execute(templateID+"_cache_key", data)
			if err == nil && renderedKey != "" {
				cacheKey = renderedKey
			}
		}

		// 检查缓存
		cachedResp, cachedBody, found := c.getFromCache(req, reqBodyBytes)
		if found {
			// 重新设置响应体
			cachedResp.Body = io.NopCloser(bytes.NewReader(cachedBody))

			// 应用响应后钩子
			for _, hook := range c.afterHook {
				cachedResp, err = hook.After(cachedResp)
				if err != nil {
					return nil, fmt.Errorf("执行响应后钩子失败: %w", err)
				}
			}
			return cachedResp, nil
		}
	}

	// 发送请求并处理重试逻辑
	var resp *http.Response
	if tmplDef.Retry.Enabled && tmplDef.Retry.MaxAttempts > 0 {
		resp, err = c.doWithRetry(req, &clientCopy, tmplDef.Retry.MaxAttempts,
			tmplDef.Retry.InitialDelay, tmplDef.Retry.BackoffFactor)
	} else {
		resp, err = clientCopy.Do(req)
	}

	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}

	// 处理模板中定义的后置钩子
	for _, hookDef := range tmplDef.AfterHooks {
		hook, err := hooks.CreateHookFromDefinition(&hookDef)
		if err != nil {
			return nil, fmt.Errorf("创建响应后钩子失败: %w", err)
		}

		// 根据接口类型添加钩子
		afterHook, ok := hook.(hooks.AfterResponseHook)
		if !ok {
			return nil, fmt.Errorf("钩子类型不是响应后钩子: %T", hook)
		}

		// 执行响应后钩子
		resp, err = afterHook.After(resp)
		if err != nil {
			return nil, fmt.Errorf("执行响应后钩子失败: %w", err)
		}
	}

	// 应用全局响应后钩子
	for _, hook := range c.afterHook {
		resp, err = hook.After(resp)
		if err != nil {
			return nil, fmt.Errorf("执行响应后钩子失败: %w", err)
		}
	}

	// 处理缓存保存
	if tmplDef.Caching.Enabled && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// 读取请求体
		var reqBodyBytes []byte
		if req.Body != nil {
			reqBodyBytes, _ = hooks.ReadRequestBody(req)
		}

		// 读取响应体
		respBodyBytes, err := ReadResponseBody(resp)
		if err == nil {
			// 重新设置响应体
			resp.Body = io.NopCloser(bytes.NewReader(respBodyBytes))

			// 保存到缓存
			c.saveToCache(req, reqBodyBytes, resp, respBodyBytes, time.Duration(tmplDef.Caching.TTL)*time.Second)
		}
	}

	return resp, nil
}

// doWithRetry 执行带有重试逻辑的请求
func (c *Client) doWithRetry(req *http.Request, client *http.Client, maxAttempts, initialDelay, backoffFactor int) (*http.Response, error) {
	var resp *http.Response
	var err error
	delay := initialDelay

	// 如果没有设置适当的值，使用默认值
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	if initialDelay <= 0 {
		initialDelay = 1000 // 1秒
	}
	if backoffFactor <= 0 {
		backoffFactor = 2
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// 创建请求体的副本
		reqCopy := c.cloneRequest(req)
		resp, err = client.Do(reqCopy)

		// 成功或不可恢复的错误，直接返回
		if err == nil || !c.isRetryableError(err) {
			return resp, err
		}

		// 最后一次尝试失败，直接返回错误
		if attempt == maxAttempts-1 {
			return nil, fmt.Errorf("最大重试次数(%d)已用尽: %w", maxAttempts, err)
		}

		// 等待一段时间后重试
		time.Sleep(time.Duration(delay) * time.Millisecond)

		// 计算下一次延迟（指数退避）
		delay *= backoffFactor
	}

	return resp, err
}

// cloneRequest 创建请求的深度副本
func (c *Client) cloneRequest(req *http.Request) *http.Request {
	// 创建新的上下文，保持原始超时设置
	reqCopy := req.Clone(req.Context())

	// 如果有请求体，需要重新读取和设置
	if req.Body != nil {
		// 读取原始请求体
		bodyBytes, err := hooks.ReadRequestBody(req)
		if err != nil {
			// 如果读取失败，返回无请求体的副本
			reqCopy.Body = nil
			reqCopy.ContentLength = 0
			return reqCopy
		}

		// 为副本设置相同的请求体
		reqCopy.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		reqCopy.ContentLength = int64(len(bodyBytes))
	}

	return reqCopy
}

// isRetryableError 判断错误是否可重试
func (c *Client) isRetryableError(err error) bool {
	// 网络连接错误通常是可重试的
	if err != nil {
		// 检查常见的临时网络错误
		// 这些错误通常是因为网络故障、服务器过载等暂时性问题
		errMsg := err.Error()

		// 常见的可重试错误模式
		retryablePatterns := []string{
			"connection refused",
			"connection reset",
			"timeout",
			"temporary failure",
			"EOF",
			"i/o timeout",
			"too many open files",
			"no such host",
		}

		for _, pattern := range retryablePatterns {
			if strings.Contains(strings.ToLower(errMsg), pattern) {
				return true
			}
		}
	}

	return false
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

// generateCacheKey 生成缓存键
func (c *Client) generateCacheKey(req *http.Request, body []byte) string {
	h := sha256.New()
	io.WriteString(h, req.URL.String())
	h.Write(body)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// getFromCache 从缓存中获取响应
func (c *Client) getFromCache(req *http.Request, body []byte) (*http.Response, []byte, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	key := c.generateCacheKey(req, body)
	if cached, ok := c.cache[key]; ok {
		if time.Now().Before(cached.ExpireTime) {
			// 复制响应以确保安全返回
			respCopy := *cached.Response
			bodyCopy := make([]byte, len(cached.Body))
			copy(bodyCopy, cached.Body)
			return &respCopy, bodyCopy, true
		}
		// 缓存已过期，删除
		delete(c.cache, key)
	}
	return nil, nil, false
}

// saveToCache 保存响应到缓存
func (c *Client) saveToCache(req *http.Request, reqBody []byte, resp *http.Response, respBody []byte, duration time.Duration) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// 只缓存成功的响应
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		key := c.generateCacheKey(req, reqBody)
		c.cache[key] = &CachedResponse{
			Response:   resp,
			Body:       respBody,
			ExpireTime: time.Now().Add(duration),
		}
	}
}
