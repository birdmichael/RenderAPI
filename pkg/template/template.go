// Package template 提供模板处理功能，支持模板渲染和缓存
package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"text/template"
)

// Engine 提供模板处理功能
type Engine struct {
	templates map[string]*template.Template
	mutex     sync.RWMutex      // 添加读写锁保证并发安全
	funcs     template.FuncMap  // 添加自定义函数映射
	cache     map[string][]byte // 添加结果缓存，提高性能
}

// NewEngine 创建一个新的模板引擎，并初始化内置函数
func NewEngine() *Engine {
	engine := &Engine{
		templates: make(map[string]*template.Template),
		funcs:     make(template.FuncMap),
		cache:     make(map[string][]byte),
	}

	// 初始化内置函数
	engine.registerBuiltinFunctions()

	return engine
}

// AddFunc 添加自定义模板函数
func (e *Engine) AddFunc(name string, fn interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.funcs[name] = fn
}

// AddTemplate 添加模板
func (e *Engine) AddTemplate(name, tmplStr string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// 创建带有自定义函数的模板
	tmpl := template.New(name).Funcs(e.funcs)

	// 解析模板
	parsedTmpl, err := tmpl.Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	// 存储模板
	e.templates[name] = parsedTmpl

	// 清除此模板的缓存
	delete(e.cache, name)

	return nil
}

// GetTemplate 获取模板
func (e *Engine) GetTemplate(name string) (*template.Template, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	tmpl, exists := e.templates[name]
	return tmpl, exists
}

// HasTemplate 检查模板是否存在
func (e *Engine) HasTemplate(name string) bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	_, exists := e.templates[name]
	return exists
}

// RemoveTemplate 删除模板
func (e *Engine) RemoveTemplate(name string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	delete(e.templates, name)
	delete(e.cache, name)
}

// Execute 执行模板并返回渲染后的内容
func (e *Engine) Execute(name string, data interface{}) (string, error) {
	tmpl, exists := e.GetTemplate(name)
	if !exists {
		return "", fmt.Errorf("找不到模板: %s", name)
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("执行模板失败: %w", err)
	}

	return buf.String(), nil
}

// RenderJSONTemplate 渲染JSON模板
func (e *Engine) RenderJSONTemplate(name string, data interface{}) ([]byte, error) {
	e.mutex.RLock()
	// 检查缓存
	cacheKey := fmt.Sprintf("%s_%p", name, data) // 根据模板名和数据指针生成缓存键
	cachedResult, hasCached := e.cache[cacheKey]
	e.mutex.RUnlock()

	// 如果有缓存且同一数据对象，直接返回（避免重复计算）
	if hasCached {
		return cachedResult, nil
	}

	// 渲染模板
	renderedJSON, err := e.Execute(name, data)
	if err != nil {
		return nil, err
	}

	// 验证生成的JSON是否有效
	var jsonObj interface{}
	err = json.Unmarshal([]byte(renderedJSON), &jsonObj)
	if err != nil {
		return nil, fmt.Errorf("生成的JSON无效: %w\n原始内容: %s", err, renderedJSON)
	}

	result := []byte(renderedJSON)

	// 存入缓存
	e.mutex.Lock()
	e.cache[cacheKey] = result
	e.mutex.Unlock()

	return result, nil
}

// ParseAndRenderJSON 解析并直接渲染JSON模板
func (e *Engine) ParseAndRenderJSON(templateStr string, data interface{}) ([]byte, error) {
	// 生成临时模板名称，避免冲突
	tmplName := fmt.Sprintf("temp_template_%p", &templateStr)

	// 添加临时模板
	err := e.AddTemplate(tmplName, templateStr)
	if err != nil {
		return nil, err
	}

	// 渲染并获取结果
	result, err := e.RenderJSONTemplate(tmplName, data)

	// 清理临时模板
	e.RemoveTemplate(tmplName)

	return result, err
}

// FormatJSON 格式化JSON字符串
func (e *Engine) FormatJSON(jsonBytes []byte) ([]byte, error) {
	var temp interface{}

	// 解析JSON
	if err := json.Unmarshal(jsonBytes, &temp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 重新格式化
	formatted, err := json.MarshalIndent(temp, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("格式化JSON失败: %w", err)
	}

	return formatted, nil
}

// ValidateJSON 验证JSON是否有效
func (e *Engine) ValidateJSON(jsonBytes []byte) error {
	var temp interface{}
	if err := json.Unmarshal(jsonBytes, &temp); err != nil {
		return fmt.Errorf("JSON验证失败: %w", err)
	}
	return nil
}

// ClearCache 清除结果缓存
func (e *Engine) ClearCache() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.cache = make(map[string][]byte)
}
