// Basic example - 展示RenderAPI的基本用法
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/birdmichael/RenderAPI/pkg/client"
)

func main() {
	// 创建新的API客户端
	apiClient := client.NewClient("https://httpbin.org", 30)

	// 添加默认请求头
	apiClient.SetHeader("User-Agent", "RenderAPI-Example")
	apiClient.SetHeader("Content-Type", "application/json")

	fmt.Println("=== 发送GET请求 ===")
	resp, err := apiClient.Get("/get")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		os.Exit(1)
	}

	body, err := client.ReadResponseBody(resp)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应体:\n%s\n\n", string(body))

	// 发送POST请求
	fmt.Println("=== 发送POST请求 ===")
	postData := []byte(`{"name": "test", "value": 123}`)

	resp, err = apiClient.Post("/post", postData)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		os.Exit(1)
	}

	body, err = client.ReadResponseBody(resp)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应体:\n%s\n\n", string(body))

	// 使用模板渲染
	fmt.Println("=== 使用模板渲染 ===")
	templateJSON := `{
		"request": {
			"method": "POST",
			"url": "https://httpbin.org",
			"path": "/post",
			"headers": {
				"Content-Type": "application/json"
			}
		},
		"body": {
			"name": "{{.name}}",
			"value": {{.value}},
			"items": [{{range $index, $item := .items}}{{if $index}}, {{end}}"{{$item}}"{{end}}]
		}
	}`

	templateData := map[string]interface{}{
		"name":  "template-test",
		"value": 456,
		"items": []string{"item1", "item2", "item3"},
	}

	ctx := context.Background()
	resp, err = apiClient.ExecuteTemplateJSON(ctx, templateJSON, templateData)
	if err != nil {
		fmt.Printf("执行模板失败: %v\n", err)
		os.Exit(1)
	}

	body, err = client.ReadResponseBody(resp)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应体:\n%s\n", string(body))
}
