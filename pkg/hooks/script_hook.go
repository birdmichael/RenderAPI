package hooks

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"

	"crypto"

	"github.com/dop251/goja"
)

// ScriptHook 实现BeforeRequestHook接口，用于执行JavaScript预请求脚本
type ScriptHook struct {
	ScriptPath string // JavaScript脚本文件路径
}

// NewScriptHook 创建一个新的脚本钩子
func NewScriptHook(scriptPath string) *ScriptHook {
	return &ScriptHook{
		ScriptPath: scriptPath,
	}
}

// Before 在请求发送前执行JavaScript脚本
func (h *ScriptHook) Before(req *http.Request) (*http.Request, error) {
	// 读取脚本文件
	scriptContent, err := os.ReadFile(h.ScriptPath)
	if err != nil {
		return req, fmt.Errorf("读取脚本文件失败: %w", err)
	}

	// 创建JavaScript运行时
	vm := goja.New()

	// 添加console.log实现
	console := make(map[string]interface{})
	console["log"] = func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Printf("[JS] %v\n", args)
		return goja.Undefined()
	}
	vm.Set("console", console)

	// 添加实际的RSA加密函数
	vm.Set("rsaEncryptGo", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return vm.ToValue("错误: 缺少参数")
		}

		text := call.Arguments[0].String()
		pemKey := call.Arguments[1].String()

		encryptedB64, err := RSAEncrypt(text, pemKey)
		if err != nil {
			return vm.ToValue("错误: " + err.Error())
		}

		return vm.ToValue(encryptedB64)
	})

	// 执行脚本
	_, err = vm.RunString(string(scriptContent))
	if err != nil {
		return req, fmt.Errorf("执行脚本失败: %w", err)
	}

	// 准备请求体数据
	var requestBody map[string]interface{}
	if req.Body != nil {
		// 读取请求体
		bodyBytes, err := ReadRequestBody(req)
		if err != nil {
			return req, fmt.Errorf("读取请求体失败: %w", err)
		}

		// 解析JSON请求体
		err = json.Unmarshal(bodyBytes, &requestBody)
		if err != nil {
			return req, fmt.Errorf("解析请求体失败: %w", err)
		}

		// 准备JavaScript请求对象
		jsRequest := make(map[string]interface{})
		jsRequest["body"] = requestBody

		// 调用JavaScript预处理函数
		processRequestFn, ok := goja.AssertFunction(vm.Get("processRequest"))
		if !ok {
			return req, fmt.Errorf("脚本中未找到processRequest函数")
		}

		// 执行处理函数
		result, err := processRequestFn(goja.Undefined(), vm.ToValue(jsRequest))
		if err != nil {
			return req, fmt.Errorf("执行processRequest函数失败: %w", err)
		}

		// 获取处理后的请求体
		processedRequest, ok := result.Export().(map[string]interface{})
		if !ok {
			return req, fmt.Errorf("无法解析处理后的请求对象")
		}

		// 提取处理后的请求体
		processedBody, ok := processedRequest["body"].(map[string]interface{})
		if !ok {
			return req, fmt.Errorf("无法解析处理后的请求体")
		}

		// 将处理后的请求体重新序列化为JSON
		newBodyBytes, err := json.Marshal(processedBody)
		if err != nil {
			return req, fmt.Errorf("序列化处理后的请求体失败: %w", err)
		}

		// 更新请求体
		req, err = ReplaceRequestBody(req, newBodyBytes)
		if err != nil {
			return req, fmt.Errorf("更新请求体失败: %w", err)
		}
	}

	return req, nil
}

// ReadRequestBody 读取并返回请求体的字节切片，同时恢复请求体以便后续使用
func ReadRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}

	// 读取请求体内容
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	// 关闭原始Body
	req.Body.Close()

	// 恢复请求体
	req, err = ReplaceRequestBody(req, bodyBytes)
	if err != nil {
		return nil, err
	}

	return bodyBytes, nil
}

// CreateReadCloser 创建一个io.ReadCloser接口的实现
func CreateReadCloser(data []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(data))
}

// ReplaceRequestBody 替换请求的正文内容
func ReplaceRequestBody(req *http.Request, bodyBytes []byte) (*http.Request, error) {
	req.Body = CreateReadCloser(bodyBytes)
	req.ContentLength = int64(len(bodyBytes))
	return req, nil
}

// RSAEncrypt 使用RSA-OAEP算法加密文本
func RSAEncrypt(text string, publicKeyPEM string) (string, error) {
	// 解析PEM格式的公钥
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", fmt.Errorf("无法解析PEM格式的公钥")
	}

	// 解析公钥
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("解析公钥失败: %w", err)
	}

	// 转换为RSA公钥
	rsaPublicKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("不是有效的RSA公钥")
	}

	// 使用RSA-OAEP加密数据，使用SHA-256哈希函数
	encryptedBytes, err := rsa.EncryptOAEP(
		crypto.SHA256.New(),
		rand.Reader,
		rsaPublicKey,
		[]byte(text),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("RSA-OAEP加密失败: %w", err)
	}

	// 返回Base64编码的加密结果
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}
