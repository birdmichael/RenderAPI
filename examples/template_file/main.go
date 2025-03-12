// 这是导入main包，但实际运行需要链接目标项目的主要实现文件
// 通过build.sh将相关文件链接到一起编译
package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// 这个示例展示了如何使用新的文件传入设计
// 通过直接读取模板文件和数据文件来执行HTTP请求
func main() {
	// 创建HTTP客户端
	client := NewHTTPClient("https://httpbin.org", 30*time.Second)

	// 设置默认请求头
	client.SetHeader("X-Custom-Header", "CustomValue")

	// 添加日志钩子
	client.AddBeforeHook(&LoggingHook{})
	client.AddAfterHook(&ResponseLogHook{})

	fmt.Println("=== 示例1: 使用模板文件和数据文件 ===")

	// 使用模板文件和数据文件
	resp, err := client.ExecuteTemplateWithDataFile(
		context.Background(),
		"post_template.json",
		"post_data.json",
	)
	if err != nil {
		log.Fatalf("请求失败: %v", err)
	}

	// 读取响应
	body, err := ReadResponseBody(resp)
	if err != nil {
		log.Fatalf("读取响应失败: %v", err)
	}

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应体: %s\n", string(body))

	fmt.Println("\n=== 示例2: 使用直接的JSON模板字符串 ===")

	// 使用JSON模板字符串
	templateJSON := `{
		"request": {
			"method": "POST",
			"baseURL": "https://httpbin.org",
			"path": "/post",
			"headers": {
				"Content-Type": "application/json"
			}
		},
		"body": {
			"name": "{{.Name}}",
			"email": "{{.Email}}",
			"message": "{{.Message}}"
		}
	}`

	// 定义数据
	data := map[string]interface{}{
		"Name":    "李四",
		"Email":   "lisi@example.com",
		"Message": "这是一条测试消息",
	}

	// 执行模板
	resp2, err := client.ExecuteTemplateJSON(
		context.Background(),
		templateJSON,
		data,
	)
	if err != nil {
		log.Fatalf("请求失败: %v", err)
	}

	// 读取响应
	body2, err := ReadResponseBody(resp2)
	if err != nil {
		log.Fatalf("读取响应失败: %v", err)
	}

	fmt.Printf("状态码: %d\n", resp2.StatusCode)
	fmt.Printf("响应体: %s\n", string(body2))
}
