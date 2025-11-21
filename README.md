# Auto Test - API 自动化测试工具

一个功能强大、易于扩展的 Go 语言 API 自动化测试工具，支持多版本、TLS 证书认证、灵活的验证规则和详细的测试报告。

## 特性

- ✅ **多版本支持**: 通过配置文件管理不同版本的 API 测试
- ✅ **TLS 证书认证**: 支持客户端证书、服务器 CA 证书
- ✅ **请求体类型约束**: 支持在发送请求前验证请求参数类型
- ✅ **灵活的验证规则**:
  - 状态码验证
  - 响应头验证
  - Body 内容验证（包含/不包含）
  - JSON 字段验证（支持嵌套路径）
  - 正则表达式验证
  - 自定义验证器
- ✅ **重试机制**: 支持配置重试次数和重试间隔
- ✅ **并发执行**: 支持并发执行测试用例
- ✅ **多种报告格式**: 控制台、JSON、HTML
- ✅ **美观的代码结构**: 模块化设计，易于扩展

## 项目结构

```
auto_test/
├── cmd/
│   └── auto_test/          # 主程序入口
│       └── main.go
├── pkg/
│   ├── client/             # HTTP 客户端（支持 TLS）
│   ├── config/             # 配置管理和加载
│   ├── executor/           # 测试执行引擎
│   ├── validator/          # 响应验证器
│   └── report/             # 测试报告生成器
├── testdata/               # 测试配置文件
│   └── api_tests.yaml      # API 测试配置示例
├── certs/                  # 证书目录
└── go.mod
```

## 快速开始

### 安装依赖

```bash
go mod download
```

### 构建

```bash
go build -o api_auto_test cmd/api_auto_test/main.go
```

### 运行测试

```bash
# 使用默认配置文件
./api_auto_test

# 指定配置文件
./api_auto_test -config testdata/api_tests.yaml

# 指定 URL 和版本
./api_auto_test -url https://api.example.com -version v2

# 使用 TLS 证书
./api_auto_test -cert certs/client.crt -key certs/client.key -ca certs/ca.crt

# 并发执行测试
./api_auto_test -concurrent -workers 10

# 生成 HTML 报告
./api_auto_test -format html -output report.html

# 生成 JSON 报告
./api_auto_test -format json -output report.json

# 列出所有测试
./api_auto_test -list

# 运行指定测试
./api_auto_test -test "获取用户列表"
```

## 配置文件示例

```yaml
# 基础 URL
base_url: https://api.example.com

# API 版本
version: v1

# TLS 证书配置
certificate:
  cert_file: certs/client.crt
  key_file: certs/client.key
  ca_file: certs/ca.crt

# 超时设置
timeout: 30s

# 全局请求头
headers:
  Content-Type: application/json
  User-Agent: AutoTestTool/1.0

# API 测试列表
apis:
  - name: 获取用户列表
    description: 测试用户列表接口
    version: v1  # 仅在 v1 版本执行
    request:
      method: GET
      path: /api/users
      query:
        page: 1
        limit: 10
    response:
      status_code: 200
      body:
        success: true
      validators:
        - type: not_empty
          field: data
        - type: type
          field: data
          value: slice

  - name: 创建用户
    description: 测试创建用户接口
    versions: [v1, v2]  # 在 v1 和 v2 版本都执行
    request:
      method: POST
      path: /api/users
      body:
        username: testuser
        email: test@example.com
        age: 25
      body_schema:  # 请求体参数类型约束（可选）
        username: string
        email: string
        age: int
    response:
      status_code: 201
      validators:
        - type: equals
          field: data.username
          value: testuser
    retry_policy:
      max_retries: 3
      interval: 1s
```

## 请求体类型约束（body_schema）

可以通过 `body_schema` 字段对请求体参数进行类型约束，在发送请求前自动验证参数类型。

### 支持的类型

| 类型 | 说明 | 示例值 |
|------|------|--------|
| `int` | 整数 | `123` |
| `float` / `float64` | 浮点数 | `19.99` |
| `string` | 字符串 | `"test"` |
| `bool` / `boolean` | 布尔值 | `true` / `false` |
| `array` / `slice` | 数组 | `[1, 2, 3]` |
| `object` / `map` | 对象 | `{"key": "value"}` |

### 嵌套字段支持

使用点号（`.`）分隔符来验证嵌套对象中的字段类型：

```yaml
request:
  method: POST
  path: /api/departments
  body:
    name: 测试部门
    parent_id: 1
    extend:
      source: iam
      data: {}
  body_schema:
    name: string
    parent_id: int
    extend: object
    extend.source: string  # 嵌套字段
    extend.data: object
```

### 验证行为

- 如果类型不匹配，请求将被拦截，不会发送到服务器
- 错误信息会明确指出哪个字段类型错误
- 如果未配置 `body_schema`，则不进行类型检查
- 支持 JSON 数字的智能检测（整数和浮点数）

### 示例

```yaml
apis:
  - name: 创建部门
    request:
      method: POST
      path: /api/departments
      body:
        name: 研发部
        parent_id: 1
        code: DEV001
        active: true
        tags: ["tech", "dev"]
      body_schema:
        name: string      # 验证 name 必须是字符串
        parent_id: int    # 验证 parent_id 必须是整数
        code: string
        active: bool      # 验证 active 必须是布尔值
        tags: array       # 验证 tags 必须是数组
```

错误示例（类型不匹配）：

```yaml
body:
  id: "abc"  # 错误：应该是整数
body_schema:
  id: int
# 输出: body schema validation failed: field 'id' has type 'string', expected 'int'
```

## 变量替换和依赖管理

本工具支持接口间的依赖关系和变量替换，可以在测试用例中引用其他接口的请求或响应数据。

### 接口依赖

使用 `depends_on` 字段指定依赖关系：

```yaml
- name: 创建部门
  request:
    method: POST
    path: /api/department
    body:
      name: "测试部门"

- name: 更新部门
  depends_on: 创建部门  # 只有创建部门成功后，才会执行此接口
  request:
    method: PATCH
    path: /api/department
```

### 变量替换语法

支持三种变量引用方式：

#### 1. 引用响应数据（推荐）

```yaml
{{接口名称.response.字段路径}}
```

示例：
```yaml
- name: 更新部门
  depends_on: 创建部门
  request:
    body:
      # 引用"创建部门"接口响应中的 data.id 字段
      id: "{{创建部门.response.data.id}}"
```

#### 2. 引用请求数据

```yaml
{{接口名称.request.字段路径}}
```

示例：
```yaml
- name: 验证创建
  depends_on: 创建部门
  request:
    body:
      # 引用"创建部门"实际发送的请求数据
      original_name: "{{创建部门.request.name}}"
```

#### 3. 默认引用（向后兼容）

```yaml
{{接口名称.字段路径}}
```

不使用 `request` 或 `response` 前缀时，默认引用响应数据：

```yaml
# 等同于 {{创建部门.response.data.id}}
id: "{{创建部门.data.id}}"
```

### 随机值生成

支持生成随机测试数据：

```yaml
body:
  name: "部门_{{$random.string.6}}"      # 随机6位字符串
  code: "CODE_{{$random.number.4}}"     # 随机4位数字
  email: "{{$random.email}}"            # 随机邮箱
  phone: "{{$random.phone}}"            # 随机手机号
  username: "{{$random.username}}"      # 随机用户名
  created_at: "{{$random.datetime}}"    # 当前日期时间
```

### 类型自动转换

配合 `body_schema` 使用时，变量替换后的值会自动转换为期望的类型：

```yaml
- name: 更新部门
  depends_on: 创建部门
  request:
    body:
      # 即使响应中 id 是 float64 或 string，也会转换为 int
      id: "{{创建部门.response.data.id}}"
    body_schema:
      id: int  # 自动类型转换
```

支持的转换：
- `int` ← string, float64, int64
- `float64` ← string, int, int64
- `string` ← 任何类型
- `bool` ← string ("true"/"false")

## 验证器类型

| 类型 | 说明 | 示例 |
|------|------|------|
| `equals` | 字段值相等 | `type: equals, field: status, value: success` |
| `contains` | 字段包含指定内容 | `type: contains, field: message, value: success` |
| `regex` | 正则表达式匹配 | `type: regex, field: email, value: ^[a-z]+@.*` |
| `not_empty` | 字段非空 | `type: not_empty, field: data.id` |
| `type` | 字段类型验证 | `type: type, field: count, value: float64` |

## 运行单元测试

本项目使用 Ginkgo 作为测试框架：

```bash
# 安装 Ginkgo
go install github.com/onsi/ginkgo/v2/ginkgo@latest

# 运行所有测试
ginkgo -r

# 运行特定包的测试
ginkgo pkg/validator
ginkgo pkg/config

# 查看详细输出
ginkgo -v -r
```

## 扩展指南

### 添加自定义验证器

在 `pkg/validator/validator.go` 的 `executeValidator` 方法中添加新的验证类型：

```go
case "my_custom_validator":
    // 实现自定义验证逻辑
    if !myCustomValidation(fieldValue, expectedValue) {
        return fmt.Errorf("custom validation failed")
    }
```

### 添加新的报告格式

在 `pkg/report/reporter.go` 中实现新的报告生成方法，然后在 `cmd/auto_test/main.go` 中添加对应的命令行参数处理。

## License

MIT License
