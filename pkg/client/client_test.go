package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestServer 创建一个测试HTTP服务器
func setupTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置通用头部
		w.Header().Set("Content-Type", "application/json")

		// 检查请求路径和方法
		switch {
		case r.URL.Path == "/api/users" && r.Method == "GET":
			// 返回用户列表
			response := `{
				"status": "success",
				"data": [
					{"id": 1, "name": "用户1", "email": "user1@example.com"},
					{"id": 2, "name": "用户2", "email": "user2@example.com"}
				]
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))

		case r.URL.Path == "/api/users" && r.Method == "POST":
			// 解析请求体
			var requestBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "无效的请求体"}`))
				return
			}

			// 返回创建成功的用户信息
			response := fmt.Sprintf(`{
				"status": "success",
				"data": {
					"id": 3,
					"name": "%s",
					"email": "%s",
					"created_at": "2023-01-01T12:00:00Z"
				}
			}`, requestBody["name"], requestBody["email"])
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(response))

		case r.URL.Path == "/api/users/1" && r.Method == "PUT":
			// 更新用户信息
			var requestBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "无效的请求体"}`))
				return
			}

			// 返回更新后的用户信息
			response := fmt.Sprintf(`{
				"status": "success",
				"data": {
					"id": 1,
					"name": "%s",
					"email": "%s",
					"updated_at": "2023-01-02T12:00:00Z"
				}
			}`, requestBody["name"], requestBody["email"])
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))

		case r.URL.Path == "/api/users/1" && r.Method == "DELETE":
			// 删除用户
			response := `{
				"status": "success",
				"message": "用户已成功删除"
			}`
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))

		case r.URL.Path == "/error":
			// 返回服务器错误
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "内部服务器错误"}`))

		default:
			// 未知路径返回404
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "未找到请求的资源"}`))
		}
	}))
}

// TestNewClient 测试客户端创建
func TestNewClient(t *testing.T) {
	client := NewClient("https://example.com", 30*time.Second)
	if client == nil {
		t.Fatal("创建客户端失败")
	}

	if client.baseURL != "https://example.com" {
		t.Errorf("基础URL错误，期望: %s, 实际: %s", "https://example.com", client.baseURL)
	}

	if client.client.Timeout != 30*time.Second {
		t.Errorf("超时设置错误，期望: %s, 实际: %s", 30*time.Second, client.client.Timeout)
	}
}

// TestHTTPMethods 测试HTTP方法
func TestHTTPMethods(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)

	// 测试GET请求
	t.Run("GET请求", func(t *testing.T) {
		resp, err := client.Get("/users")
		if err != nil {
			t.Fatalf("GET请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("响应状态码错误，期望: %d, 实际: %d", http.StatusOK, resp.StatusCode)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if response["method"] != "GET" {
			t.Errorf("请求方法错误，期望: %s, 实际: %s", "GET", response["method"])
		}

		if response["path"] != "/users" {
			t.Errorf("请求路径错误，期望: %s, 实际: %s", "/users", response["path"])
		}
	})

	// 测试POST请求
	t.Run("POST请求", func(t *testing.T) {
		data := []byte(`{"name": "张三", "age": 30}`)
		resp, err := client.Post("/users", data)
		if err != nil {
			t.Fatalf("POST请求失败: %v", err)
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if response["method"] != "POST" {
			t.Errorf("请求方法错误，期望: %s, 实际: %s", "POST", response["method"])
		}

		json, ok := response["json"].(map[string]interface{})
		if !ok {
			t.Fatalf("响应中缺少JSON数据")
		}

		if json["name"] != "张三" || json["age"] != float64(30) {
			t.Errorf("请求体数据错误，期望: %v, 实际: %v", map[string]interface{}{"name": "张三", "age": float64(30)}, json)
		}
	})

	// 测试PUT请求
	t.Run("PUT请求", func(t *testing.T) {
		data := []byte(`{"name": "李四", "age": 25}`)
		resp, err := client.Put("/users/1", data)
		if err != nil {
			t.Fatalf("PUT请求失败: %v", err)
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if response["method"] != "PUT" {
			t.Errorf("请求方法错误，期望: %s, 实际: %s", "PUT", response["method"])
		}
	})

	// 测试DELETE请求
	t.Run("DELETE请求", func(t *testing.T) {
		resp, err := client.Delete("/users/1")
		if err != nil {
			t.Fatalf("DELETE请求失败: %v", err)
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if response["method"] != "DELETE" {
			t.Errorf("请求方法错误，期望: %s, 实际: %s", "DELETE", response["method"])
		}
	})

	// 测试错误处理
	t.Run("错误处理", func(t *testing.T) {
		resp, err := client.Get("/error")
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("响应状态码错误，期望: %d, 实际: %d", http.StatusInternalServerError, resp.StatusCode)
		}
	})
}

// TestTemplateExecution 测试模板执行
func TestTemplateExecution(t *testing.T) {
	// 设置测试服务器
	server := setupTestServer()
	defer server.Close()

	c := NewClient(server.URL, 5*time.Second)

	t.Run("JSON模板执行", func(t *testing.T) {
		// 使用正确的JSON格式
		templateJSON := `{
			"method": "GET",
			"url": "{{.BaseURL}}/api/users",
			"headers": {
				"Accept": "application/json"
			}
		}`

		data := map[string]interface{}{
			"BaseURL": server.URL,
		}

		resp, err := c.ExecuteTemplateJSON(context.Background(), templateJSON, data)
		if err != nil {
			t.Fatalf("执行模板失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("状态码错误，期望: %d, 实际: %d", http.StatusOK, resp.StatusCode)
		}

		// 读取并检查响应内容
		responseData, err := ReadResponseBody(resp)
		if err != nil {
			t.Fatalf("读取响应失败: %v", err)
		}

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(responseData), &jsonData); err != nil {
			t.Fatalf("解析JSON失败: %v", err)
		}

		// 验证响应内容
		if jsonData["status"] != "success" {
			t.Errorf("状态不正确，期望: %s, 实际: %v", "success", jsonData["status"])
		}
	})
}

// TestTemplateWithFiles 测试文件模板执行
func TestTemplateWithFiles(t *testing.T) {
	// 设置测试服务器
	server := setupTestServer()
	defer server.Close()

	// 创建客户端
	c := NewClient(server.URL, 5*time.Second)

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "template-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建模板文件
	templatePath := filepath.Join(tempDir, "test-template.json")
	templateContent := `{
		"method": "GET",
		"url": "{{.BaseURL}}/api/users",
		"headers": {
			"Accept": "application/json"
		}
	}`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("创建模板文件失败: %v", err)
	}

	// 创建数据文件
	dataPath := filepath.Join(tempDir, "test-data.json")
	dataContent := `{
		"BaseURL": "%s"
	}`
	dataContent = fmt.Sprintf(dataContent, server.URL)
	if err := os.WriteFile(dataPath, []byte(dataContent), 0644); err != nil {
		t.Fatalf("创建数据文件失败: %v", err)
	}

	t.Run("文件模板执行", func(t *testing.T) {
		resp, err := c.ExecuteTemplateWithDataFile(context.Background(), templatePath, dataPath)
		if err != nil {
			t.Fatalf("执行文件模板失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("状态码错误，期望: %d, 实际: %d", http.StatusOK, resp.StatusCode)
		}

		// 读取并检查响应内容
		responseData, err := ReadResponseBody(resp)
		if err != nil {
			t.Fatalf("读取响应失败: %v", err)
		}

		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(responseData), &jsonData); err != nil {
			t.Fatalf("解析JSON失败: %v", err)
		}

		// 验证响应内容
		if jsonData["status"] != "success" {
			t.Errorf("状态不正确，期望: %s, 实际: %v", "success", jsonData["status"])
		}
	})
}

// TestSetHeader 测试设置请求头
func TestSetHeader(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)
	client.SetHeader("X-Test-Header", "test-value")
	client.SetHeader("Authorization", "Bearer token123")

	resp, err := client.Get("/headers-test")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	headers, ok := response["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("响应中缺少headers数据")
	}

	// 检查X-Test-Header
	xTestHeader, ok := headers["X-Test-Header"].([]interface{})
	if !ok || len(xTestHeader) == 0 || xTestHeader[0] != "test-value" {
		t.Errorf("X-Test-Header设置错误")
	}

	// 检查Authorization
	authorization, ok := headers["Authorization"].([]interface{})
	if !ok || len(authorization) == 0 || authorization[0] != "Bearer token123" {
		t.Errorf("Authorization设置错误")
	}
}

// TestReadResponseBody 测试读取响应体
func TestReadResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "测试响应"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)
	resp, err := client.Get("/")
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	body, err := ReadResponseBody(resp)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}

	// 验证响应体内容
	if !strings.Contains(string(body), "测试响应") {
		t.Errorf("响应体内容错误，期望包含: %s, 实际: %s", "测试响应", string(body))
	}
}

// TestLoadDataFromFile 测试从文件加载数据
func TestLoadDataFromFile(t *testing.T) {
	// 创建临时数据文件
	tempFile, err := os.CreateTemp("", "data-*.json")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// 写入测试数据
	dataContent := `{
		"name": "测试数据",
		"value": 123,
		"nested": {
			"key": "value"
		}
	}`
	if _, err := tempFile.Write([]byte(dataContent)); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tempFile.Close()

	// 测试加载数据
	data, err := LoadDataFromFile(tempFile.Name())
	if err != nil {
		t.Fatalf("从文件加载数据失败: %v", err)
	}

	// 验证数据内容
	if data["name"] != "测试数据" || data["value"] != float64(123) {
		t.Errorf("加载的数据内容错误: %v", data)
	}

	// 验证嵌套数据
	nested, ok := data["nested"].(map[string]interface{})
	if !ok || nested["key"] != "value" {
		t.Errorf("嵌套数据内容错误: %v", nested)
	}
}
