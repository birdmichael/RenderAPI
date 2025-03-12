// RenderAPI 主程序入口
// 提供命令行工具功能，用于发送模板驱动的HTTP请求
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/birdmichael/RenderAPI/pkg/client"
	"github.com/birdmichael/RenderAPI/pkg/config"
)

func main() {
	// 定义命令行参数
	baseURL := flag.String("url", "", "API基础URL")
	templateFile := flag.String("template", "", "模板文件路径")
	dataFile := flag.String("data", "", "数据文件路径")
	configFile := flag.String("config", "", "配置文件路径")
	token := flag.String("token", "", "认证令牌")
	timeout := flag.Int("timeout", 30, "请求超时时间(秒)")
	verbose := flag.Bool("verbose", false, "启用详细日志")
	scriptFile := flag.String("script", "", "JavaScript脚本文件路径")
	method := flag.String("method", "GET", "HTTP方法(不使用模板时)")
	path := flag.String("path", "", "API路径(不使用模板时)")
	output := flag.String("output", "", "保存响应到文件")
	rawData := flag.String("raw", "", "原始请求数据(JSON格式)")

	// 解析命令行参数
	flag.Parse()

	if *baseURL == "" {
		fmt.Println("错误: 必须指定API基础URL")
		flag.Usage()
		os.Exit(1)
	}

	// 加载配置
	var cfg *config.Config
	var err error
	if *configFile != "" {
		cfg, err = config.LoadConfig(*configFile)
		if err != nil {
			fmt.Printf("加载配置文件失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg = config.DefaultConfig()
		cfg.BaseURL = *baseURL
		cfg.Timeout = *timeout
		cfg.EnableLogging = *verbose
	}

	// 创建客户端
	c := client.NewClient(cfg.BaseURL, cfg.GetTimeout())

	// 设置默认头部
	for key, value := range cfg.DefaultHeaders {
		c.SetHeader(key, value)
	}

	// 添加认证令牌
	if *token != "" {
		// 此处应该使用hooks.NewAuthHook，但暂时使用自定义钩子替代
		c.AddBeforeHook(&authHook{token: *token})
	} else if cfg.AuthToken != "" {
		c.AddBeforeHook(&authHook{token: cfg.AuthToken})
	}

	// 添加脚本钩子
	if *scriptFile != "" {
		// 此处应该使用hooks.NewScriptHookFromFile，但暂时使用简单逻辑替代
		fmt.Printf("注意: 添加脚本文件 %s\n", *scriptFile)
	}

	// 添加日志钩子
	if *verbose || cfg.EnableLogging {
		c.AddBeforeHook(&loggingHook{})
		c.AddAfterHook(&responseLogHook{})
	}

	// 处理请求
	var resp *http.Response
	ctx := context.Background()

	if *templateFile != "" {
		// 使用模板文件
		if *dataFile != "" {
			fmt.Println("使用模板和数据文件发送请求...")
			resp, err = c.ExecuteTemplateWithDataFile(ctx, *templateFile, *dataFile)
		} else if *rawData != "" {
			// 解析原始数据
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(*rawData), &data); err != nil {
				fmt.Printf("解析JSON数据失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("使用模板和提供的数据发送请求...")
			resp, err = c.ExecuteTemplateFile(ctx, *templateFile, data)
		} else {
			fmt.Println("错误: 使用模板文件时必须提供数据文件或原始数据")
			flag.Usage()
			os.Exit(1)
		}
	} else if *path != "" {
		// 使用原始HTTP方法
		fullPath := cfg.BaseURL + *path
		fmt.Printf("发送 %s 请求到 %s...\n", *method, fullPath)

		switch *method {
		case "GET":
			resp, err = c.Get(*path)
		case "POST":
			var body []byte
			if *rawData != "" {
				body = []byte(*rawData)
			}
			resp, err = c.Post(*path, body)
		case "PUT":
			var body []byte
			if *rawData != "" {
				body = []byte(*rawData)
			}
			resp, err = c.Put(*path, body)
		case "DELETE":
			resp, err = c.Delete(*path)
		default:
			fmt.Printf("不支持的HTTP方法: %s\n", *method)
			os.Exit(1)
		}
	} else {
		fmt.Println("错误: 必须指定模板文件或API路径")
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		os.Exit(1)
	}

	// 处理响应
	defer resp.Body.Close()
	fmt.Printf("状态码: %d\n", resp.StatusCode)

	// 读取响应体
	responseBody, err := readResponseBody(resp)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		os.Exit(1)
	}

	// 保存响应
	if *output != "" {
		err := os.WriteFile(*output, []byte(responseBody), 0644)
		if err != nil {
			fmt.Printf("保存响应到文件失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("响应已保存到文件: %s\n", *output)
	} else {
		// 尝试美化JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(responseBody), &jsonData); err == nil {
			prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
			if err == nil {
				fmt.Println("响应内容:")
				fmt.Println(string(prettyJSON))
				return
			}
		}

		// 如果不是JSON，直接输出
		fmt.Println("响应内容:")
		fmt.Println(responseBody)
	}
}

// 读取响应体
func readResponseBody(resp *http.Response) (string, error) {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 重置响应体，以便后续可能的处理
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes), nil
}

// 自定义认证钩子
type authHook struct {
	token string
}

func (h *authHook) Before(req *http.Request) (*http.Request, error) {
	req.Header.Set("Authorization", "Bearer "+h.token)
	return req, nil
}

// 自定义日志钩子
type loggingHook struct{}

func (h *loggingHook) Before(req *http.Request) (*http.Request, error) {
	fmt.Printf("发送 %s 请求到 %s\n", req.Method, req.URL.String())
	return req, nil
}

// 响应日志钩子
type responseLogHook struct{}

func (h *responseLogHook) After(resp *http.Response) (*http.Response, error) {
	fmt.Printf("收到响应: 状态码 %d\n", resp.StatusCode)
	return resp, nil
}
