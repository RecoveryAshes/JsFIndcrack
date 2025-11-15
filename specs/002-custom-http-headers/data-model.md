# 数据模型设计: 自定义HTTP请求头

**功能**: 002-custom-http-headers
**日期**: 2025-11-15
**基于**: research.md技术调研结果

## 核心实体

### 1. HeaderConfig (头部配置)

**用途**: 表示从YAML配置文件加载的HTTP头部配置

**字段**:
```go
// HeaderConfig 表示headers.yaml配置文件的结构
type HeaderConfig struct {
    // Headers 存储所有自定义HTTP头部 (键值对)
    // 键: 头部名称 (如 "User-Agent")
    // 值: 头部值 (如 "Mozilla/5.0...")
    Headers map[string]string `mapstructure:"headers" yaml:"headers"`
}
```

**验证规则**:
- `Headers` map不能为nil (可以为空map)
- 每个键(头部名称)必须匹配 `^[A-Za-z0-9-]+$`
- 每个值(头部值)长度 ≤ 8192字节
- 每个值必须匹配 `^[\x20-\x7E\t]*$` (可打印ASCII)
- 配置文件总大小 ≤ 1MB

**YAML示例**:
```yaml
headers:
  User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"
  Accept-Language: "zh-CN,zh;q=0.9"
  Referer: "https://www.google.com/"
```

**关系**:
- 被 `HeaderManager` 使用 (组合关系)
- 由 `viper` 库解析生成

---

### 2. CliHeaders (命令行头部)

**用途**: 表示通过命令行参数传递的HTTP头部

**字段**:
```go
// CliHeaders 表示命令行传递的头部列表
// 每个字符串格式为 "Name: Value"
type CliHeaders []string
```

**验证规则**:
- 每个字符串必须包含且仅包含一个冒号 `:`
- 冒号前为头部名称,冒号后为头部值
- 名称和值验证规则同 `HeaderConfig`

**示例**:
```go
cliHeaders := CliHeaders{
    "Authorization: Bearer abc123",
    "X-API-Key: xyz789",
}
```

**解析方法**:
```go
// Parse 将字符串列表解析为 http.Header
func (ch CliHeaders) Parse() (http.Header, error) {
    result := make(http.Header)
    for i, s := range ch {
        name, value, err := parseHeaderString(s)
        if err != nil {
            return nil, fmt.Errorf("参数 --header 第%d项格式错误: %w", i+1, err)
        }
        result.Set(name, value)
    }
    return result, nil
}
```

---

### 3. HeaderManager (头部管理器)

**用途**: 核心业务逻辑,负责加载、合并、验证HTTP头部

**字段**:
```go
// HeaderManager 管理HTTP请求头部的生命周期
type HeaderManager struct {
    // configFile 配置文件路径 (如 "configs/headers.yaml")
    configFile string

    // defaults 系统默认头部 (硬编码)
    defaults http.Header

    // config 从配置文件加载的头部
    config http.Header

    // cli 从命令行参数解析的头部
    cli http.Header

    // validator 头部验证器
    validator *HeaderValidator
}
```

**方法**:
```go
// NewHeaderManager 创建头部管理器
func NewHeaderManager(configFile string, cliHeaders []string) (*HeaderManager, error)

// LoadConfig 加载配置文件,首次运行时自动生成模板
func (hm *HeaderManager) LoadConfig() error

// Validate 验证所有头部的合法性
func (hm *HeaderManager) Validate() error

// GetMergedHeaders 按优先级合并头部 (default < config < cli)
func (hm *HeaderManager) GetMergedHeaders() http.Header

// GetSafeHeaders 返回脱敏后的头部 (用于日志)
func (hm *HeaderManager) GetSafeHeaders() map[string]string
```

**状态转换**:
```
[初始化] → LoadConfig() → [配置已加载]
              ↓
          Validate() → [验证通过] → GetMergedHeaders() → [可用]
              ↓
          [验证失败] → 返回错误
```

---

### 4. HeaderValidator (头部验证器)

**用途**: 验证HTTP头部名称和值的合法性

**字段**:
```go
// HeaderValidator 验证HTTP头部是否符合RFC 7230规范
type HeaderValidator struct {
    // nameRegex 验证头部名称 (字母数字连字符)
    nameRegex *regexp.Regexp

    // valueRegex 验证头部值 (可打印ASCII)
    valueRegex *regexp.Regexp

    // maxValueLength 头部值最大长度 (字节)
    maxValueLength int

    // forbiddenHeaders 禁止用户配置的头部 (如Host, Content-Length)
    forbiddenHeaders map[string]bool
}
```

**方法**:
```go
// NewHeaderValidator 创建验证器
func NewHeaderValidator() *HeaderValidator

// ValidateName 验证头部名称
func (hv *HeaderValidator) ValidateName(name string) error

// ValidateValue 验证头部值
func (hv *HeaderValidator) ValidateValue(value string) error

// ValidateHeader 验证头部名称+值
func (hv *HeaderValidator) ValidateHeader(name, value string) error

// IsForbidden 检查头部是否被禁止
func (hv *HeaderValidator) IsForbidden(name string) bool
```

**验证规则**:
- 名称正则: `^[A-Za-z0-9-]+$`
- 值正则: `^[\x20-\x7E\t]*$`
- 值长度: ≤ 8192字节
- 禁止头部: `Host`, `Content-Length`, `Transfer-Encoding`, `Connection`

---

## 数据流

```
[用户] → YAML配置文件 → viper → HeaderConfig → HeaderManager
                                                     ↓
[用户] → 命令行参数 → CliHeaders → Parse → http.Header → HeaderManager
                                                     ↓
                                            GetMergedHeaders
                                                     ↓
                                            [HTTP客户端]
```

**步骤说明**:
1. `HeaderManager` 初始化时加载配置文件 (`LoadConfig`)
2. 配置文件通过 viper 解析为 `HeaderConfig`
3. 命令行参数通过 `CliHeaders.Parse()` 解析为 `http.Header`
4. `HeaderValidator` 验证所有头部
5. `GetMergedHeaders()` 按优先级合并: 默认 → 配置 → 命令行
6. HTTP客户端获取最终头部并发送请求

---

## 错误模型

### ValidationError (验证错误)

**用途**: 表示头部验证失败的详细信息

```go
// ValidationError 头部验证错误
type ValidationError struct {
    // Field 出错的字段 ("name" 或 "value")
    Field string

    // HeaderName 头部名称
    HeaderName string

    // Reason 错误原因
    Reason string

    // Suggestion 修复建议 (可选)
    Suggestion string
}

func (e *ValidationError) Error() string {
    msg := fmt.Sprintf("头部验证失败 [%s]: %s", e.HeaderName, e.Reason)
    if e.Suggestion != "" {
        msg += fmt.Sprintf(" (建议: %s)", e.Suggestion)
    }
    return msg
}
```

**示例**:
```go
&ValidationError{
    Field:      "name",
    HeaderName: "User Agent",  // 包含空格,非法
    Reason:     "头部名称包含非法字符 (仅允许字母、数字和连字符)",
    Suggestion: "将 'User Agent' 改为 'User-Agent'",
}
// 输出: 头部验证失败 [User Agent]: 头部名称包含非法字符... (建议: 将'User Agent'改为'User-Agent')
```

### ConfigError (配置错误)

**用途**: 表示配置文件解析失败

```go
// ConfigError 配置文件错误
type ConfigError struct {
    // FilePath 配置文件路径
    FilePath string

    // Cause 底层错误 (如viper.ConfigParseError)
    Cause error
}

func (e *ConfigError) Error() string {
    return fmt.Sprintf("配置文件错误 [%s]: %v", e.FilePath, e.Cause)
}
```

---

## 常量定义

```go
const (
    // DefaultConfigFile 默认配置文件路径
    DefaultConfigFile = "configs/headers.yaml"

    // MaxHeaderValueLength HTTP头部值最大长度 (8KB)
    MaxHeaderValueLength = 8192

    // MaxConfigFileSize 配置文件最大大小 (1MB)
    MaxConfigFileSize = 1 * 1024 * 1024

    // DefaultUserAgent 默认User-Agent
    DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
        "AppleWebKit/537.36 (KHTML, like Gecko) " +
        "Chrome/120.0.0.0 Safari/537.36"
)

var (
    // SensitiveKeywords 敏感头部名称关键字 (用于脱敏)
    SensitiveKeywords = []string{
        "authorization",
        "token",
        "key",
        "secret",
        "password",
        "credential",
        "api-key",
    }

    // ForbiddenHeaders 禁止用户配置的头部 (由HTTP客户端管理)
    ForbiddenHeaders = []string{
        "Host",
        "Content-Length",
        "Transfer-Encoding",
        "Connection",
    }
)
```

---

## 数据完整性约束

| 约束 | 验证时机 | 失败行为 |
|------|----------|----------|
| 配置文件存在性 | `LoadConfig()` | 自动生成模板 |
| 配置文件大小 ≤ 1MB | `LoadConfig()` | 返回错误,拒绝加载 |
| YAML格式正确 | `LoadConfig()` | 返回错误+行号 |
| 头部名称合法 | `Validate()` | 返回ValidationError |
| 头部值合法 | `Validate()` | 返回ValidationError |
| 头部值长度 ≤ 8KB | `Validate()` | 返回ValidationError |
| 非禁止头部 | `Validate()` | 返回ValidationError+建议 |
| 命令行格式正确 | `CliHeaders.Parse()` | 返回错误+示例 |

---

## 使用示例

```go
// 1. 创建头部管理器
cliHeaders := []string{
    "Authorization: Bearer abc123",
    "X-Custom: value",
}

manager, err := NewHeaderManager("configs/headers.yaml", cliHeaders)
if err != nil {
    log.Fatal(err)
}

// 2. 加载配置
if err := manager.LoadConfig(); err != nil {
    log.Fatal(err)
}

// 3. 验证
if err := manager.Validate(); err != nil {
    log.Fatal(err)
}

// 4. 获取合并后的头部
headers := manager.GetMergedHeaders()

// 5. 日志输出 (脱敏)
log.Info().Interface("headers", manager.GetSafeHeaders()).Msg("使用头部")

// 6. 应用到HTTP请求
req, _ := http.NewRequest("GET", "https://example.com", nil)
req.Header = headers
```

---

## 未来扩展点

1. **配置热更新** (P3功能):
   - 添加 `Watch()` 方法监听配置文件变化
   - 添加 `Reload()` 方法重新加载配置

2. **头部模板** (可选):
   - 支持预定义模板 (如 `--profile chrome`, `--profile firefox`)
   - 从 `configs/profiles/` 目录加载

3. **环境变量覆盖** (可选):
   - 支持 `JSFINDCRACK_HEADER_USER_AGENT` 环境变量
   - 优先级: 命令行 > 环境变量 > 配置文件 > 默认
