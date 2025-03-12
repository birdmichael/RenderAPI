// Package config 提供配置文件管理功能
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config 存储应用程序配置
type Config struct {
	BaseURL             string            `json:"base_url"`
	DefaultHeaders      map[string]string `json:"default_headers"`
	Timeout             int               `json:"timeout"`
	EnableLogging       bool              `json:"enable_logging"`
	AuthToken           string            `json:"auth_token"`
	TemplatesFolderPath string            `json:"templates_folder_path"`
}

// LoadConfig 从文件加载配置
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

// GetTimeout 获取超时时间
func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// SaveConfig 保存配置到文件
func (c *Config) SaveConfig(filePath string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// DefaultConfig 创建默认配置
func DefaultConfig() *Config {
	return &Config{
		BaseURL: "http://localhost:8080",
		DefaultHeaders: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "RenderAPI/1.0",
		},
		Timeout:       30,
		EnableLogging: true,
	}
}
