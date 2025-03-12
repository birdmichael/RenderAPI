// Advanced example - 展示RenderAPI的高级功能
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/birdmichael/RenderAPI/internal/utils"
	"github.com/birdmichael/RenderAPI/pkg/client"
	"github.com/birdmichael/RenderAPI/pkg/hooks"
)

// 自定义钩子
type CustomHook struct{}

func (h *CustomHook) Before(req *http.Request) (*http.Request, error) {
	fmt.Println("[自定义钩子] 请求前处理")
	// 添加自定义请求头
	req.Header.Set("X-Custom-Header", "custom-value")
	return req, nil
}

// 添加钩子
func setupHooks(c *client.Client) {
	// 添加日志钩子
	c.AddBeforeHook(&hooks.LoggingHook{})
	c.AddAfterHook(&hooks.ResponseLogHook{})

	// 添加自定义钩子
	c.AddBeforeHook(&CustomHook{})

	// 添加字段转换钩子（将"user"字段转换为"phone"）
	c.AddBeforeHook(&hooks.FieldTransformHook{})
}

func main() {
	// 创建配置
	baseURL := "https://httpbin.org"
	timeout := 30 * time.Second

	// 创建客户端
	apiClient := client.NewClient(baseURL, timeout)

	// 设置默认头
	apiClient.SetHeader("User-Agent", "RenderAPI-Advanced-Example")
	apiClient.SetHeader("Content-Type", "application/json")

	// 设置钩子
	setupHooks(apiClient)

	// 创建和使用上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("\n=== 1. 从文件加载模板和数据 ===")
	// 准备示例文件路径
	exPath, err := os.Executable()
	if err != nil {
		fmt.Printf("获取可执行文件路径失败: %v\n", err)
		exPath = "."
	}

	exDir := filepath.Dir(exPath)
	templateFile := filepath.Join(exDir, "template.json")
	dataFile := filepath.Join(exDir, "data.json")

	// 创建示例文件
	createExampleFiles(templateFile, dataFile)

	// 执行模板
	resp, err := apiClient.ExecuteTemplateWithDataFile(ctx, templateFile, dataFile)
	if err != nil {
		fmt.Printf("执行模板失败: %v\n", err)
	} else {
		body, _ := utils.ReadResponseBody(resp)
		fmt.Printf("\n状态码: %d\n", resp.StatusCode)
		prettyBody, _ := utils.PrettyJSON(body)
		fmt.Printf("响应体:\n%s\n", string(prettyBody))
	}

	fmt.Println("\n=== 2. 使用字段转换钩子 ===")
	// 准备请求体，其中包含"user"字段（会被钩子转换为"phone"）
	jsonBody := []byte(`{
		"user": "13800138000",
		"password": "secret123"
	}`)

	resp, err = apiClient.Post("/post", jsonBody)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
	} else {
		body, _ := utils.ReadResponseBody(resp)
		fmt.Printf("\n状态码: %d\n", resp.StatusCode)
		prettyBody, _ := utils.PrettyJSON(body)
		fmt.Printf("响应体:\n%s\n", string(prettyBody))
	}

	// 清理示例文件
	os.Remove(templateFile)
	os.Remove(dataFile)
}

// 创建示例文件
func createExampleFiles(templateFile, dataFile string) {
	// 模板文件
	templateContent := `{
		"request": {
			"method": "POST",
			"path": "/post",
			"headers": {
				"Content-Type": "application/json",
				"X-Template-Header": "{{.header_value}}"
			}
		},
		"body": {
			"username": "{{.username}}",
			"email": "{{.email}}",
			"age": {{.age}},
			"is_active": {{.is_active}},
			"registration_date": "{{.registration_date}}",
			"interests": [{{range $index, $item := .interests}}{{if $index}}, {{end}}"{{$item}}"{{end}}]
		}
	}`

	// 数据文件
	dataContent := `{
		"username": "测试用户",
		"email": "test@example.com",
		"age": 28,
		"is_active": true,
		"registration_date": "2023-05-15",
		"interests": ["编程", "阅读", "旅行"],
		"header_value": "template-header-value"
	}`

	// 写入文件
	os.WriteFile(templateFile, []byte(templateContent), 0644)
	os.WriteFile(dataFile, []byte(dataContent), 0644)
}
