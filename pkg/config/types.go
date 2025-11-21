package config

import (
	"time"
)

// TestConfig 测试配置
type TestConfig struct {
	BaseURL     string            `yaml:"base_url"`
	Version     string            `yaml:"version"`
	Certificate CertConfig        `yaml:"certificate"`
	Timeout     time.Duration     `yaml:"timeout"`
	Headers     map[string]string `yaml:"headers"`
	APIs        []APITest         `yaml:"apis"`
}

// CertConfig 证书配置
type CertConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// APITest 接口测试定义
type APITest struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Version     string              `yaml:"version"`    // 支持特定版本
	Versions    []string            `yaml:"versions"`   // 支持多版本
	Weight      int                 `yaml:"weight"`     // 权重，数字越大优先级越高，默认为0
	DependsOn   string              `yaml:"depends_on"` // 依赖的接口名称，该接口会在依赖接口执行成功后才执行
	Request     RequestConfig       `yaml:"request"`
	Response    ResponseExpectation `yaml:"response"`
	RetryPolicy RetryPolicy         `yaml:"retry_policy"`
}

// RequestConfig 请求配置
type RequestConfig struct {
	Method     string                 `yaml:"method"`
	Path       string                 `yaml:"path"`
	Headers    map[string]string      `yaml:"headers"`
	Query      map[string]interface{} `yaml:"query"`
	Body       interface{}            `yaml:"body"`
	BodySchema map[string]string      `yaml:"body_schema"` // 请求体字段类型约束: int, string, bool, float, array, object
}

// ResponseExpectation 响应预期
type ResponseExpectation struct {
	StatusCode   int                    `yaml:"status_code"`
	Headers      map[string]string      `yaml:"headers"`
	Body         map[string]interface{} `yaml:"body"`
	BodyContains []string               `yaml:"body_contains"`
	BodyExcludes []string               `yaml:"body_excludes"`
	JSONSchema   string                 `yaml:"json_schema"`
	Validators   []Validator            `yaml:"validators"`
}

// Validator 验证器配置
type Validator struct {
	Type   string      `yaml:"type"`   // equals, contains, regex, custom
	Field  string      `yaml:"field"`  // JSON路径，如 "data.user.id"
	Value  interface{} `yaml:"value"`  // 期望值
	Expect interface{} `yaml:"expect"` // 期望值（别名）
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries int           `yaml:"max_retries"`
	Interval   time.Duration `yaml:"interval"`
}
