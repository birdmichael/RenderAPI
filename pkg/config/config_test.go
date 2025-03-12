package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig应返回非nil配置")
	}

	// 检查默认值
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL默认值错误，期望: %s, 实际: %s",
			"http://localhost:8080", cfg.BaseURL)
	}

	if len(cfg.DefaultHeaders) != 2 {
		t.Errorf("DefaultHeaders数量错误，期望: %d, 实际: %d",
			2, len(cfg.DefaultHeaders))
	}

	if cfg.Timeout != 30 {
		t.Errorf("Timeout默认值错误，期望: %d, 实际: %d",
			30, cfg.Timeout)
	}

	if !cfg.EnableLogging {
		t.Error("EnableLogging默认值应为true")
	}
}

// TestGetTimeout 测试获取超时时间
func TestGetTimeout(t *testing.T) {
	cfg := &Config{Timeout: 45}

	timeout := cfg.GetTimeout()
	expected := 45 * time.Second

	if timeout != expected {
		t.Errorf("GetTimeout返回值错误，期望: %s, 实际: %s",
			expected, timeout)
	}
}

// TestSaveAndLoadConfig 测试配置的保存和加载
func TestSaveAndLoadConfig(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 准备测试配置
	originalCfg := &Config{
		BaseURL: "https://api.example.com",
		DefaultHeaders: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "RenderAPI-Test",
			"X-Custom":     "custom-value",
		},
		Timeout:             60,
		EnableLogging:       true,
		AuthToken:           "test-token-123",
		TemplatesFolderPath: "/templates",
	}

	// 保存配置
	configPath := filepath.Join(tempDir, "test-config.json")
	err = originalCfg.SaveConfig(configPath)
	if err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 加载配置
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证加载的配置
	if loadedCfg.BaseURL != originalCfg.BaseURL {
		t.Errorf("BaseURL不匹配，期望: %s, 实际: %s",
			originalCfg.BaseURL, loadedCfg.BaseURL)
	}

	if loadedCfg.Timeout != originalCfg.Timeout {
		t.Errorf("Timeout不匹配，期望: %d, 实际: %d",
			originalCfg.Timeout, loadedCfg.Timeout)
	}

	if loadedCfg.EnableLogging != originalCfg.EnableLogging {
		t.Errorf("EnableLogging不匹配，期望: %t, 实际: %t",
			originalCfg.EnableLogging, loadedCfg.EnableLogging)
	}

	if loadedCfg.AuthToken != originalCfg.AuthToken {
		t.Errorf("AuthToken不匹配，期望: %s, 实际: %s",
			originalCfg.AuthToken, loadedCfg.AuthToken)
	}

	if loadedCfg.TemplatesFolderPath != originalCfg.TemplatesFolderPath {
		t.Errorf("TemplatesFolderPath不匹配，期望: %s, 实际: %s",
			originalCfg.TemplatesFolderPath, loadedCfg.TemplatesFolderPath)
	}

	// 检查头部
	if len(loadedCfg.DefaultHeaders) != len(originalCfg.DefaultHeaders) {
		t.Errorf("DefaultHeaders长度不匹配，期望: %d, 实际: %d",
			len(originalCfg.DefaultHeaders), len(loadedCfg.DefaultHeaders))
	}

	for key, expectedValue := range originalCfg.DefaultHeaders {
		actualValue, exists := loadedCfg.DefaultHeaders[key]
		if !exists {
			t.Errorf("DefaultHeaders中缺少键 %s", key)
			continue
		}

		if actualValue != expectedValue {
			t.Errorf("DefaultHeaders[%s]不匹配，期望: %s, 实际: %s",
				key, expectedValue, actualValue)
		}
	}
}

// TestLoadConfigError 测试加载配置错误
func TestLoadConfigError(t *testing.T) {
	// 测试不存在的配置文件
	_, err := LoadConfig("non-existent-config.json")
	if err == nil {
		t.Error("应该检测到不存在的配置文件")
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "config-error-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建无效的配置文件
	invalidConfigPath := filepath.Join(tempDir, "invalid-config.json")
	err = os.WriteFile(invalidConfigPath, []byte(`{"base_url": "https://example.com", "invalid_json": {`), 0644)
	if err != nil {
		t.Fatalf("写入无效配置文件失败: %v", err)
	}

	// 加载无效的配置文件
	_, err = LoadConfig(invalidConfigPath)
	if err == nil {
		t.Error("应该检测到无效的配置文件")
	}
}
