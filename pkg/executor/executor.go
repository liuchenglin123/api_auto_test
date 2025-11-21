package executor

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"api_auto_test/pkg/client"
	"api_auto_test/pkg/config"
	"api_auto_test/pkg/validator"
)

// TestResult 单个测试结果
type TestResult struct {
	Name        string
	Description string
	Version     string
	Passed      bool
	Skipped     bool   // 是否被跳过
	SkipReason  string // 跳过原因
	Duration    time.Duration
	StatusCode  int
	Request     config.RequestConfig
	Response    *client.Response
	Validation  *validator.ValidationResult
	Error       error
	RetryCount  int
	ExecutedAt  time.Time
}

// TestReport 测试报告
type TestReport struct {
	TotalTests     int
	PassedTests    int
	FailedTests    int
	SkippedTests   int // 跳过的测试数量
	Duration       time.Duration
	Results        []TestResult
	StartTime      time.Time
	EndTime        time.Time
	Version        string
	BaseURL        string
	ConfigFileName string // 配置文件名称（不含路径）
}

// Executor 测试执行器
type Executor struct {
	client  *client.HTTPClient
	config  *config.TestConfig
	results map[string]*TestResult // 存储已执行的测试结果，用于依赖查询
	mu      sync.RWMutex           // 保护 results 的并发访问
}

// NewExecutor 创建测试执行器
func NewExecutor(cfg *config.TestConfig) (*Executor, error) {
	httpClient, err := client.NewHTTPClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Executor{
		client:  httpClient,
		config:  cfg,
		results: make(map[string]*TestResult),
	}, nil
}

// Execute 执行所有测试
func (e *Executor) Execute() *TestReport {
	startTime := time.Now()

	report := &TestReport{
		Results:   make([]TestResult, 0),
		StartTime: startTime,
		Version:   e.config.Version,
		BaseURL:   e.config.BaseURL,
	}

	// 按权重排序 APIs（权重高的在前）
	sortedAPIs := e.sortAPIsByWeight()

	// 按拓扑顺序执行（考虑依赖关系）
	executionOrder := e.resolveExecutionOrder(sortedAPIs)

	for _, apiTest := range executionOrder {
		// 检查依赖是否已成功执行
		if apiTest.DependsOn != "" {
			depResult := e.getResult(apiTest.DependsOn)
			if depResult == nil {
				// 依赖接口未执行
				result := TestResult{
					Name:        apiTest.Name,
					Description: apiTest.Description,
					Version:     apiTest.Version,
					Request:     apiTest.Request,
					ExecutedAt:  time.Now(),
					Passed:      false,
					Skipped:     true,
					SkipReason:  fmt.Sprintf("依赖接口 '%s' 未找到或未执行", apiTest.DependsOn),
				}
				report.Results = append(report.Results, result)
				report.SkippedTests++
				report.TotalTests++
				e.storeResult(&result)
				continue
			}
			if !depResult.Passed || depResult.Skipped {
				// 依赖接口执行失败或被跳过，需要跟踪依赖链找到根本原因
				rootCause := e.findRootCause(apiTest.DependsOn)
				skipReason := fmt.Sprintf("依赖接口 '%s' %s", apiTest.DependsOn, e.getDependencyFailureReason(depResult))
				if rootCause != "" && rootCause != apiTest.DependsOn {
					skipReason = fmt.Sprintf("依赖接口 '%s' %s（根本原因：接口 '%s' 执行失败）",
						apiTest.DependsOn, e.getDependencyFailureReason(depResult), rootCause)
				}

				result := TestResult{
					Name:        apiTest.Name,
					Description: apiTest.Description,
					Version:     apiTest.Version,
					Request:     apiTest.Request,
					ExecutedAt:  time.Now(),
					Passed:      false,
					Skipped:     true,
					SkipReason:  skipReason,
				}
				report.Results = append(report.Results, result)
				report.SkippedTests++
				report.TotalTests++
				e.storeResult(&result)
				continue
			}
		}

		// 替换请求中的变量
		processedTest := e.replaceVariables(apiTest)

		result := e.executeAPITest(processedTest)
		e.storeResult(&result)
		report.Results = append(report.Results, result)

		if result.Passed {
			report.PassedTests++
		} else {
			report.FailedTests++
		}
		report.TotalTests++
	}

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(startTime)

	return report
}

// ExecuteConcurrent 并发执行所有测试
func (e *Executor) ExecuteConcurrent(maxConcurrency int) *TestReport {
	startTime := time.Now()

	report := &TestReport{
		Results:   make([]TestResult, 0),
		StartTime: startTime,
		Version:   e.config.Version,
		BaseURL:   e.config.BaseURL,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, maxConcurrency)

	for _, apiTest := range e.config.APIs {
		wg.Add(1)
		go func(test config.APITest) {
			defer wg.Done()

			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result := e.executeAPITest(test)

			mu.Lock()
			report.Results = append(report.Results, result)
			if result.Passed {
				report.PassedTests++
			} else {
				report.FailedTests++
			}
			report.TotalTests++
			mu.Unlock()
		}(apiTest)
	}

	wg.Wait()
	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(startTime)

	return report
}

// executeAPITest 执行单个API测试
func (e *Executor) executeAPITest(apiTest config.APITest) TestResult {
	result := TestResult{
		Name:        apiTest.Name,
		Description: apiTest.Description,
		Version:     apiTest.Version,
		Request:     apiTest.Request,
		ExecutedAt:  time.Now(),
		RetryCount:  0,
	}

	maxRetries := apiTest.RetryPolicy.MaxRetries
	retryInterval := apiTest.RetryPolicy.Interval

	// 默认不重试
	if maxRetries == 0 {
		maxRetries = 1
	} else {
		maxRetries++ // 加上首次尝试
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			result.RetryCount++
			if retryInterval > 0 {
				time.Sleep(retryInterval)
			}
		}

		// 发送请求
		startTime := time.Now()
		resp, err := e.client.Do(apiTest.Request)
		duration := time.Since(startTime)

		result.Duration = duration

		if err != nil {
			lastErr = err
			continue // 重试
		}

		result.Response = resp
		result.StatusCode = resp.StatusCode

		// 验证响应
		v := validator.NewValidator(apiTest.Response)
		validationResult := v.Validate(resp)
		result.Validation = validationResult

		if validationResult.Passed {
			result.Passed = true
			return result
		}

		// 如果验证失败且有重试次数，继续重试
		if attempt < maxRetries-1 {
			continue
		}
	}

	// 所有重试都失败
	if lastErr != nil {
		result.Error = lastErr
	}
	result.Passed = false

	return result
}

// ExecuteByName 按名称执行指定的测试
func (e *Executor) ExecuteByName(name string) (*TestResult, error) {
	for _, apiTest := range e.config.APIs {
		if apiTest.Name == name {
			result := e.executeAPITest(apiTest)
			return &result, nil
		}
	}
	return nil, fmt.Errorf("test '%s' not found", name)
}

// GetTestNames 获取所有测试名称
func (e *Executor) GetTestNames() []string {
	names := make([]string, 0, len(e.config.APIs))
	for _, apiTest := range e.config.APIs {
		names = append(names, apiTest.Name)
	}
	return names
}

// sortAPIsByWeight 按权重排序 APIs（权重高的在前）
func (e *Executor) sortAPIsByWeight() []config.APITest {
	sorted := make([]config.APITest, len(e.config.APIs))
	copy(sorted, e.config.APIs)

	sort.SliceStable(sorted, func(i, j int) bool {
		// 权重高的排在前面（降序）
		if sorted[i].Weight != sorted[j].Weight {
			return sorted[i].Weight > sorted[j].Weight
		}
		// 权重相同时保持原有顺序
		return false
	})

	return sorted
}

// resolveExecutionOrder 解析执行顺序（考虑依赖关系）
// 使用拓扑排序确保依赖的接口先执行
func (e *Executor) resolveExecutionOrder(apis []config.APITest) []config.APITest {
	// 构建名称到索引的映射
	nameToIndex := make(map[string]int)
	for i, api := range apis {
		nameToIndex[api.Name] = i
	}

	// 拓扑排序
	visited := make(map[int]bool)
	visiting := make(map[int]bool)
	result := make([]config.APITest, 0, len(apis))

	var visit func(int) bool
	visit = func(idx int) bool {
		if visited[idx] {
			return true
		}
		if visiting[idx] {
			// 检测到循环依赖
			return false
		}

		visiting[idx] = true

		// 先访问依赖
		api := apis[idx]
		if api.DependsOn != "" {
			if depIdx, exists := nameToIndex[api.DependsOn]; exists {
				if !visit(depIdx) {
					return false
				}
			}
		}

		visiting[idx] = false
		visited[idx] = true
		result = append(result, api)
		return true
	}

	for i := range apis {
		if !visited[i] {
			if !visit(i) {
				// 检测到循环依赖，回退到原顺序
				return apis
			}
		}
	}

	return result
}

// storeResult 存储测试结果
func (e *Executor) storeResult(result *TestResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.results[result.Name] = result
}

// getResult 获取已执行的测试结果
func (e *Executor) getResult(name string) *TestResult {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.results[name]
}

// findRootCause 查找依赖链中的根本原因（最初失败的接口）
func (e *Executor) findRootCause(testName string) string {
	visited := make(map[string]bool)
	current := testName

	for {
		if visited[current] {
			// 检测到循环依赖，返回当前节点
			return current
		}
		visited[current] = true

		result := e.getResult(current)
		if result == nil {
			return current
		}

		// 如果这个接口是真正失败的（不是被跳过），那它就是根本原因
		if !result.Passed && !result.Skipped {
			return current
		}

		// 如果这个接口是被跳过的，继续查找它的依赖
		if result.Skipped {
			// 尝试从配置中找到它依赖的接口
			dependsOn := e.findDependsOn(current)
			if dependsOn == "" {
				return current
			}
			current = dependsOn
		} else {
			// 接口通过了，返回当前节点
			return current
		}
	}
}

// findDependsOn 查找指定接口的依赖
func (e *Executor) findDependsOn(testName string) string {
	for _, api := range e.config.APIs {
		if api.Name == testName {
			return api.DependsOn
		}
	}
	return ""
}

// getDependencyFailureReason 获取依赖失败的原因描述
func (e *Executor) getDependencyFailureReason(result *TestResult) string {
	if result.Skipped {
		return "被跳过"
	}
	return "执行失败"
}

// replaceVariables 替换请求中的变量
// 支持格式：
//   - {{接口名称.request.字段路径}}，引用请求数据，例如 {{创建部门.request.name}}
//   - {{接口名称.response.字段路径}}，引用响应数据，例如 {{创建部门.response.data.id}}
//   - {{接口名称.字段路径}}，默认引用响应数据（向后兼容），例如 {{创建部门.data.id}}
//   - {{$random.type}}，例如 {{$random.name}}, {{$random.string.10}}
func (e *Executor) replaceVariables(apiTest config.APITest) config.APITest {
	// 正则表达式匹配 {{name.field.path}} 或 {{$random.type}}
	varPattern := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	// 辅助函数：提取单个变量的值（保持原始类型）
	extractValue := func(varPath string) (interface{}, bool) {
		varPath = strings.TrimSpace(varPath)

		// 检查是否是随机值占位符
		if strings.HasPrefix(varPath, "$random") {
			randomValue := e.generateRandomValue(varPath)
			if randomValue != "" {
				return randomValue, true
			}
			return nil, false
		}

		// 处理接口返回值引用
		parts := strings.SplitN(varPath, ".", 2)
		if len(parts) < 1 {
			return nil, false
		}

		testName := parts[0]
		var fieldPath string
		if len(parts) == 2 {
			fieldPath = parts[1]
		}

		// 获取依赖接口的结果
		depResult := e.getResult(testName)
		if depResult == nil {
			return nil, false
		}

		// 判断是引用请求数据还是响应数据
		var sourceData interface{}
		if strings.HasPrefix(fieldPath, "request.") {
			// 引用请求数据
			fieldPath = strings.TrimPrefix(fieldPath, "request.")
			sourceData = depResult.Request.Body
		} else if strings.HasPrefix(fieldPath, "response.") {
			// 引用响应数据
			fieldPath = strings.TrimPrefix(fieldPath, "response.")
			if depResult.Response == nil {
				return nil, false
			}
			sourceData = depResult.Response.BodyJSON
		} else {
			// 默认引用响应数据（向后兼容）
			if depResult.Response == nil {
				return nil, false
			}
			sourceData = depResult.Response.BodyJSON
		}

		// 从数据源中提取字段值
		var value interface{}
		if fieldPath == "" {
			value = sourceData
		} else {
			value = e.extractFieldValue(sourceData, fieldPath)
		}

		return value, value != nil
	}

	// 辅助函数：替换字符串中的变量（返回字符串）
	replaceInString := func(s string) string {
		return varPattern.ReplaceAllStringFunc(s, func(match string) string {
			varPath := strings.Trim(match, "{}")
			value, ok := extractValue(varPath)
			if ok && value != nil {
				return fmt.Sprintf("%v", value)
			}
			return match // 保持原样
		})
	}

	// 辅助函数：递归替换 interface{} 中的变量
	var replaceInInterface func(interface{}) interface{}
	replaceInInterface = func(v interface{}) interface{} {
		switch val := v.(type) {
		case string:
			// 检查字符串是否完全是一个变量引用
			trimmed := strings.TrimSpace(val)
			matches := varPattern.FindAllStringSubmatch(trimmed, -1)

			// 如果整个字符串就是一个单独的变量，返回原始类型的值
			if len(matches) == 1 && trimmed == matches[0][0] {
				varPath := matches[0][1]
				if value, ok := extractValue(varPath); ok {
					// JSON 解析数字默认为 float64，如果是整数则转换为 int64
					if f, ok := value.(float64); ok {
						if f == float64(int64(f)) {
							return int64(f)
						}
					}
					return value // 返回原���类型
				}
			}

			// 否则作为字符串处理（可能包含多个变量或混合文本）
			return replaceInString(val)

		case map[string]interface{}:
			result := make(map[string]interface{})
			for k, v := range val {
				result[k] = replaceInInterface(v)
			}
			return result
		case []interface{}:
			result := make([]interface{}, len(val))
			for i, item := range val {
				result[i] = replaceInInterface(item)
			}
			return result
		default:
			return val
		}
	}

	// 创建副本以避免修改原始配置
	processedTest := apiTest

	// 替换 Path
	processedTest.Request.Path = replaceInString(apiTest.Request.Path)

	// 替换 Query 参数
	if apiTest.Request.Query != nil {
		processedTest.Request.Query = replaceInInterface(apiTest.Request.Query).(map[string]interface{})
	}

	// 替换 Body
	if apiTest.Request.Body != nil {
		processedTest.Request.Body = replaceInInterface(apiTest.Request.Body)

		// 如果配置了 body_schema，根据 schema 转换字段类型
		if len(apiTest.Request.BodySchema) > 0 {
			processedTest.Request.Body = e.convertBodyToSchemaTypes(processedTest.Request.Body, apiTest.Request.BodySchema)
		}
	}

	// 替换 Headers
	if apiTest.Request.Headers != nil {
		processedHeaders := make(map[string]string)
		for k, v := range apiTest.Request.Headers {
			processedHeaders[k] = replaceInString(v)
		}
		processedTest.Request.Headers = processedHeaders
	}

	return processedTest
}

// extractFieldValue 从响应体中提取字段值
// 支持点号分隔的路径，例如 "data.user.id"
// 支持数组索引，例如 "data[0].id" 或 "items[0].children[1].name"
func (e *Executor) extractFieldValue(body interface{}, fieldPath string) interface{} {
	// 解析路径，支持数组索引 [index]
	parts := e.parseFieldPath(fieldPath)
	current := body

	for _, part := range parts {
		// 检查是否是数组索引访问
		if part.isArray {
			// 先访问字段名（如果有）
			if part.name != "" {
				switch v := current.(type) {
				case map[string]interface{}:
					var exists bool
					current, exists = v[part.name]
					if !exists {
						return nil
					}
				default:
					// 尝试将其他类型转换为 map
					data, err := json.Marshal(current)
					if err != nil {
						return nil
					}
					var m map[string]interface{}
					if err := json.Unmarshal(data, &m); err != nil {
						return nil
					}
					var exists bool
					current, exists = m[part.name]
					if !exists {
						return nil
					}
				}
			}

			// 然后访问数组索引
			switch arr := current.(type) {
			case []interface{}:
				if part.index >= 0 && part.index < len(arr) {
					current = arr[part.index]
				} else {
					return nil // 索引越界
				}
			default:
				// 尝试转换为数组
				data, err := json.Marshal(current)
				if err != nil {
					return nil
				}
				var slice []interface{}
				if err := json.Unmarshal(data, &slice); err != nil {
					return nil
				}
				if part.index >= 0 && part.index < len(slice) {
					current = slice[part.index]
				} else {
					return nil
				}
			}
		} else {
			// 普通的字段访问
			switch v := current.(type) {
			case map[string]interface{}:
				var exists bool
				current, exists = v[part.name]
				if !exists {
					return nil
				}
			default:
				// 尝试将其他类型转换为 map
				data, err := json.Marshal(current)
				if err != nil {
					return nil
				}
				var m map[string]interface{}
				if err := json.Unmarshal(data, &m); err != nil {
					return nil
				}
				var exists bool
				current, exists = m[part.name]
				if !exists {
					return nil
				}
			}
		}
	}

	return current
}

// fieldPathPart 表示路径的一部分
type fieldPathPart struct {
	name    string // 字段名
	isArray bool   // 是否是数组索引
	index   int    // 数组索引
}

// parseFieldPath 解析字段路径，支持点号和数组索引
// 例如: "data.items[0].children[1].name"
// 返回: [{name:"data"}, {name:"items", isArray:true, index:0}, {name:"children", isArray:true, index:1}, {name:"name"}]
func (e *Executor) parseFieldPath(path string) []fieldPathPart {
	if path == "" {
		return nil
	}

	var parts []fieldPathPart
	var currentPart strings.Builder
	var inBracket bool
	var bracketContent strings.Builder

	for i, ch := range path {
		switch ch {
		case '.':
			if inBracket {
				bracketContent.WriteRune(ch)
			} else {
				// 处理当前累积的部分
				if currentPart.Len() > 0 {
					parts = append(parts, fieldPathPart{
						name:    currentPart.String(),
						isArray: false,
					})
					currentPart.Reset()
				}
			}
		case '[':
			inBracket = true
			// 保存字段名（如果有）
			if currentPart.Len() > 0 {
				// 字段名会在后面处理数组索引时保存
			}
		case ']':
			if inBracket {
				inBracket = false
				// 解析索引
				indexStr := bracketContent.String()
				index, err := strconv.Atoi(indexStr)
				if err == nil && index >= 0 {
					parts = append(parts, fieldPathPart{
						name:    currentPart.String(),
						isArray: true,
						index:   index,
					})
					currentPart.Reset()
				}
				bracketContent.Reset()
			}
		default:
			if inBracket {
				bracketContent.WriteRune(ch)
			} else {
				currentPart.WriteRune(ch)
			}
		}

		// 处理最后一个字符
		if i == len(path)-1 && currentPart.Len() > 0 {
			parts = append(parts, fieldPathPart{
				name:    currentPart.String(),
				isArray: false,
			})
		}
	}

	return parts
}

// convertBodyToSchemaTypes 根据 body_schema 转换字段类型
// 支持嵌套字段（点号分隔）
func (e *Executor) convertBodyToSchemaTypes(body interface{}, schema map[string]string) interface{} {
	bodyMap, ok := body.(map[string]interface{})
	if !ok {
		return body
	}

	// 创建副本
	result := make(map[string]interface{})
	for k, v := range bodyMap {
		result[k] = v
	}

	// 遍历 schema 中的每个字段，进行类型转换
	for fieldPath, expectedType := range schema {
		// 获取并转换字段值
		e.setNestedValue(result, fieldPath, expectedType)
	}

	return result
}

// setNestedValue 设置嵌套字段的值，并根据 expectedType 进行类型转换
func (e *Executor) setNestedValue(data map[string]interface{}, fieldPath string, expectedType string) {
	parts := strings.Split(fieldPath, ".")
	if len(parts) == 0 {
		return
	}

	// 如果是顶层字段
	if len(parts) == 1 {
		field := parts[0]
		if value, exists := data[field]; exists {
			data[field] = e.convertToSchemaType(value, expectedType)
		}
		return
	}

	// 处理嵌套字段
	current := data
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if nextMap, ok := current[part].(map[string]interface{}); ok {
			current = nextMap
		} else {
			// 如果中间路径不存在或不是 map，则无法设置
			return
		}
	}

	// 设置最后一个字段
	lastField := parts[len(parts)-1]
	if value, exists := current[lastField]; exists {
		current[lastField] = e.convertToSchemaType(value, expectedType)
	}
}

// convertToSchemaType 将值转换为指定的类型
func (e *Executor) convertToSchemaType(value interface{}, expectedType string) interface{} {
	if value == nil {
		return value
	}

	switch expectedType {
	case "int":
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}

	case "float", "float64":
		switch v := value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}

	case "string":
		return fmt.Sprintf("%v", value)

	case "bool", "boolean":
		switch v := value.(type) {
		case bool:
			return v
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}

	case "array", "slice":
		// 保持数组类型不变
		return value

	case "object", "map":
		// 保持对象类型不变
		return value
	}

	// 如果无法转换，返回原值
	return value
}

// generateRandomValue 生成随机值
// 支持的格式：
//   - {{$random.string}} 或 {{$random.string.8}} - 随机字符串（默认8位）
//   - {{$random.number}} 或 {{$random.number.6}} - 随机数字（默认6位）
//   - {{$random.uuid}} - UUID
//   - {{$random.timestamp}} - Unix时间戳
//   - {{$random.datetime}} - 日期时间格式
//   - {{$random.date}} - 日期格式
//   - {{$random.email}} - 随机邮箱
//   - {{$random.phone}} - 随机手机号
//   - {{$random.name}} - 随机中文名字
//   - {{$random.username}} - 随机用户名
func (e *Executor) generateRandomValue(randomType string) string {
	// 解析类型和参数
	parts := strings.Split(randomType, ".")
	if len(parts) < 2 {
		return ""
	}

	baseType := parts[1]
	param := ""
	if len(parts) >= 3 {
		param = parts[2]
	}

	switch baseType {
	case "string":
		length := 8
		if param != "" {
			if l, err := strconv.Atoi(param); err == nil && l > 0 {
				length = l
			}
		}
		return e.randomString(length)

	case "number":
		length := 6
		if param != "" {
			if l, err := strconv.Atoi(param); err == nil && l > 0 {
				length = l
			}
		}
		return e.randomNumber(length)

	case "uuid":
		return e.randomUUID()

	case "timestamp":
		return fmt.Sprintf("%d", time.Now().UnixNano()/1e6)

	case "datetime":
		return time.Now().Format("2006-01-02 15:04:05")

	case "date":
		return time.Now().Format("2006-01-02")

	case "email":
		return e.randomEmail()

	case "phone":
		return e.randomPhone()

	case "name":
		return e.randomChineseName()

	case "username":
		return e.randomUsername()

	default:
		return ""
	}
}

// randomString 生成随机字符串
func (e *Executor) randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// randomNumber 生成随机数字字符串
func (e *Executor) randomNumber(length int) string {
	const charset = "0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	// 确保第一位不为0
	if result[0] == '0' {
		n, _ := rand.Int(rand.Reader, big.NewInt(9))
		result[0] = charset[n.Int64()+1]
	}
	return string(result)
}

// randomUUID 生成UUID
func (e *Executor) randomUUID() string {
	uuid := make([]byte, 16)
	rand.Read(uuid)
	// 设置版本4和变体位
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(uuid[0:4]),
		hex.EncodeToString(uuid[4:6]),
		hex.EncodeToString(uuid[6:8]),
		hex.EncodeToString(uuid[8:10]),
		hex.EncodeToString(uuid[10:16]))
}

// randomEmail 生成随机邮箱
func (e *Executor) randomEmail() string {
	domains := []string{"test.com", "example.com", "demo.org", "mail.com"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(domains))))
	return fmt.Sprintf("%s@%s", e.randomString(8), domains[n.Int64()])
}

// randomPhone 生成随机手机号
func (e *Executor) randomPhone() string {
	prefixes := []string{"130", "131", "132", "133", "134", "135", "136", "137", "138", "139",
		"150", "151", "152", "153", "155", "156", "157", "158", "159",
		"180", "181", "182", "183", "184", "185", "186", "187", "188", "189"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(prefixes))))
	return prefixes[n.Int64()] + e.randomNumber(8)
}

// randomChineseName 生成随机中文名字
func (e *Executor) randomChineseName() string {
	surnames := []string{"张", "王", "李", "赵", "刘", "陈", "杨", "黄", "周", "吴",
		"徐", "孙", "马", "朱", "胡", "郭", "何", "林", "罗", "高"}
	names := []string{"伟", "芳", "娜", "秀英", "敏", "静", "强", "磊", "军", "洋",
		"勇", "艳", "杰", "娟", "涛", "明", "超", "秀兰", "霞", "平",
		"刚", "桂英", "文", "华", "建", "国", "志", "海", "云", "峰"}

	sn, _ := rand.Int(rand.Reader, big.NewInt(int64(len(surnames))))
	nn, _ := rand.Int(rand.Reader, big.NewInt(int64(len(names))))
	return surnames[sn.Int64()] + names[nn.Int64()]
}

// randomUsername 生成随机用户名
func (e *Executor) randomUsername() string {
	prefixes := []string{"user", "test", "dev", "admin", "guest", "demo"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(prefixes))))
	return fmt.Sprintf("%s_%s", prefixes[n.Int64()], e.randomString(6))
}
