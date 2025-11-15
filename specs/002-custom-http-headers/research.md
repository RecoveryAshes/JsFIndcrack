# 技术调研: 自定义HTTP请求头

**功能**: 002-custom-http-headers
**日期**: 2025-11-15
**目的**: 解决技术选型和最佳实践问题,为Phase 1设计提供依据

## 调研任务

### 1. Go语言YAML配置文件解析最佳实践

**问题**: 如何可靠地解析YAML配置文件,处理格式错误并提供详细错误信息?

**调研结果**:

**选择方案**: **spf13/viper** (项目已引入)

**理由**:
- **已集成**: 项目go.mod中已包含 `github.com/spf13/viper v1.20.0-alpha.6`
- **功能完善**: 支持YAML/JSON/TOML多格式,自动类型转换,环境变量覆盖
- **错误处理**: 提供详细的解析错误信息,包含行号和原因
- **社区成熟**: 21k+ stars, cobra配套工具,广泛用于CLI工具

**替代方案对比**:
| 方案 | 优点 | 缺点 | 选择原因 |
|------|------|------|----------|
| gopkg.in/yaml.v3 | 轻量,标准库风格 | 需手动实现配置管理逻辑 | viper已引入且功能更丰富 |
| koanf | 高性能,插件化 | 学习曲线陡,过度设计 | 本需求无需如此复杂 |
| viper | 功能完善,生态好 | 依赖较多 | ✅ 已引入,无额外成本 |

**最佳实践**:
```go
// 1. 设置配置文件路径和名称
viper.SetConfigName("headers")
viper.SetConfigType("yaml")
viper.AddConfigPath("./configs")      // 项目配置目录
viper.AddConfigPath("$HOME/.jsfindcrack") // 用户主目录 (备选)

// 2. 读取配置并处理错误
if err := viper.ReadInConfig(); err != nil {
    if _, ok := err.(viper.ConfigFileNotFoundError); ok {
        // 配置文件不存在 - 生成默认模板
        generateDefaultConfig()
    } else {
        // 解析错误 - 返回详细错误 (包含行号)
        return fmt.Errorf("配置文件解析失败: %w", err)
    }
}

// 3. 绑定到结构体 (类型安全)
var config HeaderConfig
if err := viper.Unmarshal(&config); err != nil {
    return fmt.Errorf("配置绑定失败: %w", err)
}
```

**错误定位**:
viper底层使用 `gopkg.in/yaml.v3`,解析错误会包含行号,示例:
```
yaml: line 5: mapping values are not allowed in this context
```

---

### 2. HTTP头部验证规范

**问题**: 如何验证HTTP头部名称和值的合法性,符合RFC 7230/7231标准?

**调研结果**:

**选择方案**: **Go标准库 net/http + 自定义正则验证**

**HTTP头部命名规范** (RFC 7230 Section 3.2):
- **字段名** (field-name):
  - 允许字符: `a-zA-Z0-9`、连字符 `-`
  - 不区分大小写 (但惯例使用 `Camel-Case`)
  - 正则表达式: `^[A-Za-z0-9-]+$`

- **字段值** (field-value):
  - 允许字符: 可打印ASCII + 空格/制表符
  - 不允许控制字符 (0x00-0x1F, 0x7F)
  - 长度限制: <8KB (大多数服务器)
  - 正则表达式: `^[\x20-\x7E\t]*$`

**实现方案**:
```go
import (
    "net/http"
    "regexp"
)

var (
    // HTTP头部名称验证 (RFC 7230)
    headerNameRegex = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

    // HTTP头部值验证 (可打印ASCII + 空格/制表符)
    headerValueRegex = regexp.MustCompile(`^[\x20-\x7E\t]*$`)

    // 最大头部值长度 (8KB)
    maxHeaderValueLen = 8192
)

func ValidateHeader(name, value string) error {
    // 1. 验证名称
    if !headerNameRegex.MatchString(name) {
        return fmt.Errorf("非法头部名称 '%s': 只允许字母、数字和连字符", name)
    }

    // 2. 验证值长度
    if len(value) > maxHeaderValueLen {
        return fmt.Errorf("头部值过长: %d 字节 (最大 %d)", len(value), maxHeaderValueLen)
    }

    // 3. 验证值字符
    if !headerValueRegex.MatchString(value) {
        return fmt.Errorf("头部值包含非法字符 (仅允许可打印ASCII)")
    }

    return nil
}
```

**Go标准库支持**:
`net/http`包的 `Header` 类型已内置规范化 (如 `content-type` → `Content-Type`),因此:
- 使用 `http.Header` 存储头部 (自动规范化)
- 手动验证名称和值的合法性

**禁止的头部** (安全考虑):
某些头部由HTTP客户端自动管理,不应由用户配置:
```go
var forbiddenHeaders = map[string]bool{
    "Host":              true, // 由URL决定
    "Content-Length":    true, // 由请求体决定
    "Transfer-Encoding": true, // 由客户端管理
    "Connection":        true, // 由客户端管理
}
```

---

### 3. 命令行参数多次传递的最佳实践

**问题**: 如何实现 `--header "A: 1" --header "B: 2"` 多次传递同一参数?

**调研结果**:

**选择方案**: **cobra.StringSlice** (项目已使用cobra)

**实现方式**:
```go
var cliHeaders []string

// root.go中定义flag
rootCmd.PersistentFlags().StringSliceVarP(
    &cliHeaders,
    "header",      // 长参数名
    "H",           // 短参数名
    []string{},    // 默认值
    "自定义HTTP头部,格式: 'Name: Value',可多次指定",
)

// 使用示例:
// jsfindcrack -u https://example.com -H "User-Agent: Bot" -H "Authorization: Bearer token"
```

**解析头部字符串**:
```go
func ParseHeaderString(s string) (name, value string, err error) {
    parts := strings.SplitN(s, ":", 2)
    if len(parts) != 2 {
        return "", "", fmt.Errorf("格式错误: 缺少冒号分隔符,应为 'Name: Value'")
    }

    name = strings.TrimSpace(parts[0])
    value = strings.TrimSpace(parts[1])

    if name == "" {
        return "", "", fmt.Errorf("头部名称不能为空")
    }

    return name, value, nil
}
```

**cobra替代方案**:
- `StringArray`: 每次传递一个独立值,适合多次 `--header`
- `StringSlice`: 支持逗号分隔+多次传递,更灵活 (推荐)

选择 `StringSlice` 理由: 同时支持 `--header "A:1,B:2"` 和 `--header "A:1" --header "B:2"`

---

### 4. 配置文件自动生成模板设计

**问题**: 首次运行时如何生成易理解、带注释的YAML模板?

**调研结果**:

**模板设计**:
```yaml
# JsFIndcrack HTTP请求头配置文件
# 生成时间: 2025-11-15
# 说明: 此文件用于配置爬虫发送HTTP请求时携带的自定义头部

# ========================================
# 通用HTTP头部配置
# ========================================
# 所有HTTP请求都会携带以下头部 (除非被命令行参数覆盖)

headers:
  # 用户代理字符串 - 模拟浏览器身份,避免被反爬虫识别
  # 示例: Chrome 120浏览器
  User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

  # 来源页面 - 某些网站会检查Referer头部
  # 留空表示不发送此头部
  # Referer: "https://www.google.com/"

  # 接受的语言 - 影响网站返回的内容语言
  Accept-Language: "zh-CN,zh;q=0.9,en;q=0.8"

  # 接受的内容类型
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"

  # 接受的编码方式
  Accept-Encoding: "gzip, deflate, br"

# ========================================
# 认证头部配置 (可选)
# ========================================
# 警告: 此配置文件可能包含敏感信息,请勿提交到版本控制系统
#      建议通过命令行参数 --header 传递认证信息

# 认证示例 (取消注释以启用):
# headers:
#   Authorization: "Bearer YOUR_API_TOKEN_HERE"
#   X-API-Key: "YOUR_API_KEY_HERE"

# ========================================
# 自定义头部
# ========================================
# 您可以添加任意自定义头部,格式:
# headers:
#   Custom-Header-Name: "Custom Header Value"

# 注意事项:
# 1. 头部名称只允许字母、数字和连字符
# 2. 头部值不能包含控制字符
# 3. 单个头部值最大长度为8KB
# 4. 命令行参数 --header 优先级高于此配置文件
# 5. 使用 jsfindcrack --validate-config 验证配置正确性
```

**生成逻辑**:
```go
const defaultHeaderTemplate = `...` // 上述YAML模板

func EnsureConfigExists(configPath string) error {
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        // 创建目录
        dir := filepath.Dir(configPath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("无法创建配置目录: %w", err)
        }

        // 写入模板
        if err := os.WriteFile(configPath, []byte(defaultHeaderTemplate), 0644); err != nil {
            return fmt.Errorf("无法生成配置文件: %w", err)
        }

        log.Info().Str("path", configPath).Msg("已生成默认配置文件模板")
    }
    return nil
}
```

---

### 5. 敏感信息脱敏策略

**问题**: 如何自动识别并脱敏日志中的认证头部?

**调研结果**:

**识别规则**:
根据头部名称关键字判断是否敏感:
```go
var sensitiveHeaderKeywords = []string{
    "authorization",
    "token",
    "key",
    "secret",
    "password",
    "credential",
    "api-key",
    "x-api-key",
}

func IsSensitiveHeader(name string) bool {
    nameLower := strings.ToLower(name)
    for _, keyword := range sensitiveHeaderKeywords {
        if strings.Contains(nameLower, keyword) {
            return true
        }
    }
    return false
}
```

**脱敏方式**:
```go
func RedactHeaderValue(name, value string) string {
    if !IsSensitiveHeader(name) {
        return value
    }

    // 策略1: 仅显示前缀 (适合Bearer Token)
    if strings.HasPrefix(value, "Bearer ") {
        return "Bearer ***"
    }

    // 策略2: 显示前4位+后4位 (适合API Key)
    if len(value) > 8 {
        return value[:4] + "***" + value[len(value)-4:]
    }

    // 策略3: 完全隐藏
    return "***"
}
```

**日志集成**:
项目使用 `zerolog`,可定义自定义类型:
```go
type SafeHeaders http.Header

func (h SafeHeaders) MarshalZerologObject(e *zerolog.Event) {
    for name, values := range h {
        if IsSensitiveHeader(name) {
            e.Str(name, RedactHeaderValue(name, values[0]))
        } else {
            e.Strs(name, values)
        }
    }
}

// 使用:
log.Info().Object("headers", SafeHeaders(headers)).Msg("发送请求")
```

---

### 6. 头部合并优先级实现

**问题**: 如何正确实现"命令行 > 配置文件 > 系统默认"优先级?

**调研结果**:

**实现方案**:
```go
type HeaderManager struct {
    defaults   http.Header  // 系统默认头部
    config     http.Header  // 配置文件头部
    cli        http.Header  // 命令行头部
}

// GetMergedHeaders 按优先级合并头部
func (hm *HeaderManager) GetMergedHeaders() http.Header {
    result := make(http.Header)

    // 1. 首先应用默认头部
    for name, values := range hm.defaults {
        result[name] = values
    }

    // 2. 配置文件覆盖默认
    for name, values := range hm.config {
        result[name] = values
    }

    // 3. 命令行覆盖配置文件
    for name, values := range hm.cli {
        result[name] = values
    }

    return result
}
```

**系统默认头部**:
```go
func GetDefaultHeaders() http.Header {
    return http.Header{
        "User-Agent": []string{
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
            "AppleWebKit/537.36 (KHTML, like Gecko) " +
            "Chrome/120.0.0.0 Safari/537.36",
        },
        "Accept": []string{"*/*"},
        "Accept-Encoding": []string{"gzip, deflate, br"},
    }
}
```

---

## 技术决策总结

| 决策点 | 选择方案 | 理由 |
|--------|----------|------|
| 配置文件格式 | YAML | 易读,支持注释,viper原生支持 |
| 配置解析库 | spf13/viper | 项目已引入,功能完善,错误信息详细 |
| 命令行解析 | cobra.StringSlice | 项目已使用cobra,支持多次传递 |
| 头部验证 | 自定义正则+标准库 | 符合RFC 7230,无额外依赖 |
| 配置文件路径 | `./configs/headers.yaml` | 项目根目录,避免污染用户工作目录 |
| 敏感信息脱敏 | 关键字匹配+部分显示 | 平衡安全性和可调试性 |
| 头部合并 | 三层覆盖机制 | 默认→配置→命令行,符合直觉 |

## 未解决问题

无 - 所有技术问题均已有明确方案。

## 下一步

进入Phase 1设计阶段,生成:
1. data-model.md - 数据结构定义
2. contracts/ - HeaderProvider接口契约
3. quickstart.md - 用户快速入门指南
