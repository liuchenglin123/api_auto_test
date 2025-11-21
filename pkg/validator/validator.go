package validator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"api_auto_test/pkg/client"
	"api_auto_test/pkg/config"
)

// ValidationResult 验证结果
type ValidationResult struct {
	Passed   bool
	Errors   []ValidationError
	Warnings []string
}

// ValidationError 验证错误
type ValidationError struct {
	Field    string
	Expected interface{}
	Actual   interface{}
	Message  string
}

// Validator 验证器
type Validator struct {
	expectation config.ResponseExpectation
}

// NewValidator 创建验证器
func NewValidator(expectation config.ResponseExpectation) *Validator {
	return &Validator{
		expectation: expectation,
	}
}

// Validate 执行验证
func (v *Validator) Validate(resp *client.Response) *ValidationResult {
	result := &ValidationResult{
		Passed:   true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]string, 0),
	}

	// 验证状态码
	if v.expectation.StatusCode != 0 {
		if resp.StatusCode != v.expectation.StatusCode {
			result.Passed = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    "StatusCode",
				Expected: v.expectation.StatusCode,
				Actual:   resp.StatusCode,
				Message:  fmt.Sprintf("Expected status code %d, got %d", v.expectation.StatusCode, resp.StatusCode),
			})
		}
	}

	// 验证Headers
	v.validateHeaders(resp, result)

	// 验证Body包含内容
	v.validateBodyContains(resp, result)

	// 验证Body不包含内容
	v.validateBodyExcludes(resp, result)

	// 验证Body字段
	v.validateBodyFields(resp, result)

	// 执行自定义验证器
	v.executeCustomValidators(resp, result)

	return result
}

// validateHeaders 验证响应头
func (v *Validator) validateHeaders(resp *client.Response, result *ValidationResult) {
	for key, expectedValue := range v.expectation.Headers {
		actualValue := resp.Headers.Get(key)
		if actualValue != expectedValue {
			result.Passed = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("Header[%s]", key),
				Expected: expectedValue,
				Actual:   actualValue,
				Message:  fmt.Sprintf("Expected header %s=%s, got %s", key, expectedValue, actualValue),
			})
		}
	}
}

// validateBodyContains 验证响应体包含指定内容
func (v *Validator) validateBodyContains(resp *client.Response, result *ValidationResult) {
	bodyStr := string(resp.Body)
	for _, content := range v.expectation.BodyContains {
		if !strings.Contains(bodyStr, content) {
			result.Passed = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    "Body",
				Expected: fmt.Sprintf("contains '%s'", content),
				Actual:   "not found",
				Message:  fmt.Sprintf("Response body should contain '%s'", content),
			})
		}
	}
}

// validateBodyExcludes 验证响应体不包含指定内容
func (v *Validator) validateBodyExcludes(resp *client.Response, result *ValidationResult) {
	bodyStr := string(resp.Body)
	for _, content := range v.expectation.BodyExcludes {
		if strings.Contains(bodyStr, content) {
			result.Passed = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    "Body",
				Expected: fmt.Sprintf("excludes '%s'", content),
				Actual:   "found",
				Message:  fmt.Sprintf("Response body should not contain '%s'", content),
			})
		}
	}
}

// validateBodyFields 验证响应体字段
func (v *Validator) validateBodyFields(resp *client.Response, result *ValidationResult) {
	if len(v.expectation.Body) == 0 {
		return
	}

	if resp.BodyJSON == nil {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "Body",
			Message: "Expected JSON response, but got non-JSON content",
		})
		return
	}

	for field, expectedValue := range v.expectation.Body {
		actualValue := getJSONField(resp.BodyJSON, field)
		if !compareValues(expectedValue, actualValue) {
			result.Passed = false
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("Body.%s", field),
				Expected: expectedValue,
				Actual:   actualValue,
				Message:  fmt.Sprintf("Field '%s': expected %v, got %v", field, expectedValue, actualValue),
			})
		}
	}
}

// executeCustomValidators 执行自定义验证器
func (v *Validator) executeCustomValidators(resp *client.Response, result *ValidationResult) {
	for _, validator := range v.expectation.Validators {
		if err := v.executeValidator(validator, resp); err != nil {
			result.Passed = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   validator.Field,
				Message: err.Error(),
			})
		}
	}
}

// executeValidator 执行单个验证器
func (v *Validator) executeValidator(validator config.Validator, resp *client.Response) error {
	// 获取字段值
	fieldValue := getJSONField(resp.BodyJSON, validator.Field)

	// 确定期望值（支持value和expect两种写法）
	expectedValue := validator.Value
	if expectedValue == nil {
		expectedValue = validator.Expect
	}

	switch strings.ToLower(validator.Type) {
	case "equals", "equal", "eq":
		if !compareValues(expectedValue, fieldValue) {
			return fmt.Errorf("expected %v, got %v", expectedValue, fieldValue)
		}
	case "contains":
		fieldStr := fmt.Sprintf("%v", fieldValue)
		expectedStr := fmt.Sprintf("%v", expectedValue)
		if !strings.Contains(fieldStr, expectedStr) {
			return fmt.Errorf("expected to contain '%s', got '%s'", expectedStr, fieldStr)
		}
	case "regex", "regexp":
		fieldStr := fmt.Sprintf("%v", fieldValue)
		pattern := fmt.Sprintf("%v", expectedValue)
		matched, err := regexp.MatchString(pattern, fieldStr)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		if !matched {
			return fmt.Errorf("value '%s' does not match pattern '%s'", fieldStr, pattern)
		}
	case "not_empty", "notempty":
		if fieldValue == nil || fieldValue == "" {
			return fmt.Errorf("field should not be empty")
		}
	case "type":
		expectedType := fmt.Sprintf("%v", expectedValue)
		if fieldValue == nil {
			return fmt.Errorf("expected type %s, got nil", expectedType)
		}

		actualTypeObj := reflect.TypeOf(fieldValue)
		actualType := actualTypeObj.String()
		actualKind := actualTypeObj.Kind().String()

		// 归一化类型字符串，去除空格
		normalizedActualType := strings.ReplaceAll(actualType, " ", "")
		normalizedExpectedType := strings.ReplaceAll(expectedType, " ", "")

		// 支持多种匹配方式：
		// 1. 完整类型名匹配（去除空格后）
		// 2. Kind 匹配（如 "slice", "map", "string" 等）
		// 3. 包含匹配（兼容性）
		if normalizedActualType == normalizedExpectedType ||
			strings.EqualFold(actualKind, expectedType) ||
			strings.Contains(normalizedActualType, normalizedExpectedType) {
			return nil
		}

		return fmt.Errorf("expected type %s, got %s", expectedType, actualType)
	default:
		return fmt.Errorf("unknown validator type: %s", validator.Type)
	}

	return nil
}

// getJSONField 获取JSON字段值（支持嵌套路径，如 "data.user.id"）
func getJSONField(data map[string]interface{}, path string) interface{} {
	if data == nil {
		return nil
	}

	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}

	return current
}

// compareValues 比较两个值是否相等
func compareValues(expected, actual interface{}) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}

	// 尝试JSON序列化比较（处理map和slice）
	expectedJSON, err1 := json.Marshal(expected)
	actualJSON, err2 := json.Marshal(actual)
	if err1 == nil && err2 == nil {
		return string(expectedJSON) == string(actualJSON)
	}

	// 直接比较
	return reflect.DeepEqual(expected, actual)
}
