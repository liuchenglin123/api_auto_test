package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"api_auto_test/pkg/config"
)

// HTTPClient HTTP客户端
type HTTPClient struct {
	client      *http.Client
	baseURL     string
	headers     map[string]string
	timeout     time.Duration
	certificate *config.CertConfig
}

// Response HTTP响应封装
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	BodyJSON   map[string]interface{}
	Duration   time.Duration
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(cfg *config.TestConfig) (*HTTPClient, error) {
	client := &HTTPClient{
		baseURL: cfg.BaseURL,
		headers: cfg.Headers,
		timeout: cfg.Timeout,
	}

	if client.timeout == 0 {
		client.timeout = 30 * time.Second
	}

	// 配置TLS证书
	tlsConfig, err := client.loadTLSConfig(&cfg.Certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	// 创建HTTP客户端
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client.client = &http.Client{
		Timeout:   client.timeout,
		Transport: transport,
	}

	return client, nil
}

// loadTLSConfig 加载TLS配置
func (c *HTTPClient) loadTLSConfig(certConfig *config.CertConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{}

	// 如果没有配置证书，返回默认配置
	if certConfig.CertFile == "" && certConfig.CAFile == "" {
		return tlsConfig, nil
	}

	// 加��客户端证书
	if certConfig.CertFile != "" && certConfig.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(certConfig.CertFile, certConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 加载CA证书
	if certConfig.CAFile != "" {
		caCert, err := os.ReadFile(certConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// Do 执行HTTP请求
func (c *HTTPClient) Do(reqConfig config.RequestConfig) (*Response, error) {
	startTime := time.Now()

	// 验证请求体类型（如果配置了 body_schema）
	if len(reqConfig.BodySchema) > 0 && reqConfig.Body != nil {
		if err := validateBodySchema(reqConfig.Body, reqConfig.BodySchema); err != nil {
			return nil, fmt.Errorf("body schema validation failed: %w", err)
		}
	}

	// 构建完整URL
	fullURL, err := c.buildURL(reqConfig.Path, reqConfig.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// 构建请求体
	var bodyReader io.Reader
	if reqConfig.Body != nil {
		bodyBytes, err := json.Marshal(reqConfig.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		// 调试：打印实际发送的请求体
		fmt.Printf("[DEBUG] Request Body: %s\n", string(bodyBytes))
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// 创建HTTP请求
	req, err := http.NewRequest(strings.ToUpper(reqConfig.Method), fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置Headers
	c.setHeaders(req, reqConfig.Headers)

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 解析JSON响应
	var bodyJSON map[string]interface{}
	if len(respBody) > 0 && resp.Header.Get("Content-Type") != "" &&
		(strings.Contains(resp.Header.Get("Content-Type"), "application/json") ||
			strings.Contains(resp.Header.Get("Content-Type"), "text/json")) {
		_ = json.Unmarshal(respBody, &bodyJSON)
	}

	duration := time.Since(startTime)

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
		BodyJSON:   bodyJSON,
		Duration:   duration,
	}, nil
}

// buildURL 构建完整URL
func (c *HTTPClient) buildURL(path string, query map[string]interface{}) (string, error) {
	baseURL := strings.TrimRight(c.baseURL, "/")
	path = strings.TrimLeft(path, "/")
	fullURL := fmt.Sprintf("%s/%s", baseURL, path)

	if len(query) == 0 {
		return fullURL, nil
	}

	// 添加查询参数
	u, err := url.Parse(fullURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	for key, value := range query {
		q.Add(key, fmt.Sprintf("%v", value))
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// setHeaders 设置请求头
func (c *HTTPClient) setHeaders(req *http.Request, customHeaders map[string]string) {
	// 设置全局Headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// 设置自定义Headers（会覆盖全局Headers）
	for key, value := range customHeaders {
		req.Header.Set(key, value)
	}

	// 设置默认Content-Type
	if req.Header.Get("Content-Type") == "" && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
}

// validateBodySchema 验��请求体字段类型
func validateBodySchema(body interface{}, schema map[string]string) error {
	// 将 body 转换为 map[string]interface{}
	bodyMap, ok := body.(map[string]interface{})
	if !ok {
		// 尝试通过 JSON 编解码转换
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body for validation: %w", err)
		}
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			return fmt.Errorf("failed to unmarshal body for validation: %w", err)
		}
	}

	// 验证每个字段的类型
	for field, expectedType := range schema {
		value, exists := getNestedValue(bodyMap, field)
		if !exists {
			return fmt.Errorf("field '%s' not found in request body", field)
		}

		if err := validateFieldType(field, value, expectedType); err != nil {
			return err
		}
	}

	return nil
}

// getNestedValue 获取嵌套字段的值（支持点号分隔的路径，如 "extend.source"）
func getNestedValue(data map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}

		value, exists := currentMap[part]
		if !exists {
			return nil, false
		}
		current = value
	}

	return current, true
}

// validateFieldType 验证单个字段的类型
func validateFieldType(field string, value interface{}, expectedType string) error {
	if value == nil {
		return fmt.Errorf("field '%s' is nil, expected type '%s'", field, expectedType)
	}

	actualType := getValueType(value)

	// 类型映射和兼容性检查
	switch expectedType {
	case "int":
		// Go 的 JSON 解析默认将数字解析为 float64
		// 需要检查是否为整数值的 float64
		if actualType == "float64" {
			if floatVal, ok := value.(float64); ok {
				if floatVal == float64(int64(floatVal)) {
					return nil // 是整数值
				}
			}
		}
		if actualType != "int" && actualType != "int64" && actualType != "int32" {
			return fmt.Errorf("field '%s' has type '%s', expected 'int'", field, actualType)
		}
	case "float", "float64":
		if actualType != "float64" && actualType != "float32" {
			return fmt.Errorf("field '%s' has type '%s', expected 'float'", field, actualType)
		}
	case "string":
		if actualType != "string" {
			return fmt.Errorf("field '%s' has type '%s', expected 'string'", field, actualType)
		}
	case "bool", "boolean":
		if actualType != "bool" {
			return fmt.Errorf("field '%s' has type '%s', expected 'bool'", field, actualType)
		}
	case "array", "slice":
		if actualType != "slice" {
			return fmt.Errorf("field '%s' has type '%s', expected 'array/slice'", field, actualType)
		}
	case "object", "map":
		if actualType != "map" {
			return fmt.Errorf("field '%s' has type '%s', expected 'object/map'", field, actualType)
		}
	default:
		return fmt.Errorf("unsupported type '%s' for field '%s'", expectedType, field)
	}

	return nil
}

// getValueType 获取值的类型名称
func getValueType(value interface{}) string {
	if value == nil {
		return "nil"
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float64"
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Slice, reflect.Array:
		return "slice"
	case reflect.Map:
		return "map"
	default:
		return v.Kind().String()
	}
}
