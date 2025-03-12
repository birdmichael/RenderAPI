// Package utils 提供内部工具函数
package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// PrettyJSON 格式化JSON字符串
func PrettyJSON(data []byte) ([]byte, error) {
	var obj interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(obj, "", "  ")
}

// ReadResponseBody 读取HTTP响应体
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// LoadDataFromFile 从文件加载JSON数据
func LoadDataFromFile(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// SaveDataToFile 保存数据到文件
func SaveDataToFile(filePath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// LogHTTPRequest 记录HTTP请求信息
func LogHTTPRequest(req *http.Request, body []byte) {
	fmt.Printf("[请求] %s %s\n", req.Method, req.URL.String())

	if len(req.Header) > 0 {
		fmt.Println("请求头:")
		for k, v := range req.Header {
			fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
		}
	}

	if len(body) > 0 {
		fmt.Println("请求体:")
		prettyBody, err := PrettyJSON(body)
		if err == nil {
			fmt.Println(string(prettyBody))
		} else {
			fmt.Println(string(body))
		}
	}
}

// LogHTTPResponse 记录HTTP响应信息
func LogHTTPResponse(resp *http.Response, body []byte) {
	fmt.Printf("[响应] 状态码: %d\n", resp.StatusCode)

	if len(resp.Header) > 0 {
		fmt.Println("响应头:")
		for k, v := range resp.Header {
			fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
		}
	}

	if len(body) > 0 {
		fmt.Println("响应体:")
		prettyBody, err := PrettyJSON(body)
		if err == nil {
			fmt.Println(string(prettyBody))
		} else {
			fmt.Println(string(body))
		}
	}
}
