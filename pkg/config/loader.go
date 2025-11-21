package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Loader 配置加载器
type Loader struct {
	configPath string
}

// NewLoader 创建配置加载器
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
	}
}

// Load 加载配置文件
func (l *Loader) Load() (*TestConfig, error) {
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config TestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// LoadWithVersion 加载配置并过滤指定版本的API测试
func (l *Loader) LoadWithVersion(version string) (*TestConfig, error) {
	config, err := l.Load()
	if err != nil {
		return nil, err
	}

	// 如果指定了版本，则覆盖配置中的版本
	if version != "" {
		config.Version = version
	}

	// 过滤适用于当前版本的API测试
	filteredAPIs := make([]APITest, 0)
	for _, api := range config.APIs {
		if l.isVersionMatch(api, config.Version) {
			filteredAPIs = append(filteredAPIs, api)
		}
	}
	config.APIs = filteredAPIs

	return config, nil
}

// isVersionMatch 检查API测试是否适用于指定版本
func (l *Loader) isVersionMatch(api APITest, targetVersion string) bool {
	// 如果API没有指定版本，则适用于所有版本
	if api.Version == "" && len(api.Versions) == 0 {
		return true
	}

	// 检查单个版本匹配
	if api.Version != "" && api.Version == targetVersion {
		return true
	}

	// 检查多版本匹配
	for _, v := range api.Versions {
		if v == targetVersion {
			return true
		}
	}

	return false
}

// MergeConfig 合并运行时配置（支持命令行参数覆盖）
func MergeConfig(base *TestConfig, baseURL, certFile, keyFile, caFile, version string) *TestConfig {
	if baseURL != "" {
		base.BaseURL = baseURL
	}
	if version != "" {
		base.Version = version
	}
	if certFile != "" {
		base.Certificate.CertFile = certFile
	}
	if keyFile != "" {
		base.Certificate.KeyFile = keyFile
	}
	if caFile != "" {
		base.Certificate.CAFile = caFile
	}
	return base
}
