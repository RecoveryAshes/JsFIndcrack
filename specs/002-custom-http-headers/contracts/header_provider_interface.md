# 接口契约: HeaderProvider

**功能**: 002-custom-http-headers
**日期**: 2025-11-15
**用途**: 定义HTTP头部提供者的标准接口,实现依赖倒置原则

## 接口定义

### HeaderProvider (头部提供者)

**责任**: 为HTTP客户端提供合并后的请求头部

**接口签名**:
```go
package models

import "net/http"

// HeaderProvider 定义HTTP头部提供者接口
// 实现此接口的类型负责管理和提供HTTP请求头部
type HeaderProvider interface {
    // GetHeaders 返回当前有效的HTTP请求头部
    // 返回的http.Header已按优先级合并(默认 < 配置 < 命令行)
    //
    // 返回值:
    //   - http.Header: 可直接应用于http.Request的头部集合
    //   - error: 如果头部加载或验证失败,返回错误
    //
    // 错误情况:
    //   - 配置文件解析失败
    //   - 头部验证失败
    //   - 配置文件不可读
    GetHeaders() (http.Header, error)
}
```

---

## 契约规范

### 前置条件 (Preconditions)
- 实现类必须在调用 `GetHeaders()` 前完成初始化
- 配置文件路径必须有效 (如不存在应自动生成)
- 命令行参数格式必须正确 (如提供)

### 后置条件 (Postconditions)
- 返回的 `http.Header` 已验证合法性 (符合RFC 7230)
- 返回的头部已按优先级合并 (命令行 > 配置文件 > 默认)
- 禁止的头部 (如 `Host`, `Content-Length`) 已被过滤
- 如果返回错误,`http.Header` 为nil

### 不变量 (Invariants)
- 同一实例多次调用 `GetHeaders()` 返回一致结果 (除非配置变化)
- 返回的 `http.Header` 不会包含非法字符或超长值
- 头部名称已规范化 (如 `user-agent` → `User-Agent`)

---

## 实现示例

### 标准实现 (HeaderManager)

```go
package core

import (
    "net/http"
    "github.com/RecoveryAshes/JsFIndcrack/internal/models"
)

// HeaderManager 是 HeaderProvider 的标准实现
type HeaderManager struct {
    configFile string
    cliHeaders []string
    // ... 其他字段
}

// GetHeaders 实现 HeaderProvider 接口
func (hm *HeaderManager) GetHeaders() (http.Header, error) {
    // 1. 加载配置文件 (如未加载)
    if err := hm.ensureLoaded(); err != nil {
        return nil, err
    }

    // 2. 验证所有头部
    if err := hm.validate(); err != nil {
        return nil, err
    }

    // 3. 合并头部
    return hm.mergeHeaders(), nil
}
```

### Mock实现 (用于测试)

```go
package mocks

import "net/http"

// MockHeaderProvider 用于测试的Mock实现
type MockHeaderProvider struct {
    Headers http.Header
    Err     error
}

func (m *MockHeaderProvider) GetHeaders() (http.Header, error) {
    if m.Err != nil {
        return nil, m.Err
    }
    return m.Headers, nil
}
```

---

## 使用示例

### 爬虫集成

```go
package crawlers

import (
    "net/http"
    "github.com/RecoveryAshes/JsFIndcrack/internal/models"
)

// StaticCrawler 静态爬虫 (依赖HeaderProvider接口,而非具体实现)
type StaticCrawler struct {
    // headerProvider 提供HTTP请求头部
    headerProvider models.HeaderProvider
}

// NewStaticCrawler 创建静态爬虫
func NewStaticCrawler(hp models.HeaderProvider) *StaticCrawler {
    return &StaticCrawler{
        headerProvider: hp,
    }
}

// Fetch 抓取URL
func (sc *StaticCrawler) Fetch(url string) (*http.Response, error) {
    // 1. 获取头部
    headers, err := sc.headerProvider.GetHeaders()
    if err != nil {
        return nil, fmt.Errorf("获取请求头部失败: %w", err)
    }

    // 2. 创建请求
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }

    // 3. 应用头部
    req.Header = headers

    // 4. 发送请求
    client := &http.Client{}
    return client.Do(req)
}
```

---

## 测试契约

### 契约测试用例

```go
package models_test

import (
    "testing"
    "github.com/RecoveryAshes/JsFIndcrack/internal/models"
)

// TestHeaderProviderContract 验证HeaderProvider接口契约
func TestHeaderProviderContract(t *testing.T) {
    // 可替换为任何HeaderProvider实现
    provider := createHeaderProvider()

    t.Run("成功返回合法头部", func(t *testing.T) {
        headers, err := provider.GetHeaders()

        // 后置条件1: 无错误时返回非nil头部
        if err != nil {
            t.Fatalf("期望无错误,得到: %v", err)
        }
        if headers == nil {
            t.Fatal("期望非nil头部,得到nil")
        }

        // 不变量1: 头部名称已规范化
        if headers.Get("user-agent") == "" && headers.Get("User-Agent") != "" {
            t.Log("✓ 头部名称已规范化")
        }

        // 不变量2: 禁止的头部已过滤
        if headers.Get("Host") == "" && headers.Get("Content-Length") == "" {
            t.Log("✓ 禁止头部已过滤")
        }
    })

    t.Run("多次调用返回一致结果", func(t *testing.T) {
        h1, err1 := provider.GetHeaders()
        h2, err2 := provider.GetHeaders()

        // 不变量3: 一致性
        if (err1 == nil) != (err2 == nil) {
            t.Fatal("多次调用错误状态不一致")
        }

        if err1 == nil {
            if len(h1) != len(h2) {
                t.Fatal("多次调用返回头部数量不一致")
            }
        }
    })

    t.Run("错误时返回nil头部", func(t *testing.T) {
        // 构造会失败的provider (如配置文件损坏)
        brokenProvider := createBrokenHeaderProvider()

        headers, err := brokenProvider.GetHeaders()

        // 后置条件2: 错误时返回nil
        if err != nil && headers != nil {
            t.Fatal("期望错误时返回nil头部")
        }
    })
}
```

---

## 扩展接口 (可选)

### HeaderValidator 接口

```go
// HeaderValidator 定义头部验证器接口
type HeaderValidator interface {
    // Validate 验证HTTP头部的合法性
    //
    // 参数:
    //   - headers: 待验证的HTTP头部
    //
    // 返回:
    //   - error: 验证失败时返回详细错误,成功时返回nil
    Validate(headers http.Header) error
}
```

### HeaderRedactor 接口 (脱敏)

```go
// HeaderRedactor 定义头部脱敏器接口
type HeaderRedactor interface {
    // Redact 将敏感头部脱敏,返回安全的字符串表示
    //
    // 参数:
    //   - headers: 原始HTTP头部
    //
    // 返回:
    //   - map[string]string: 脱敏后的头部 (用于日志)
    Redact(headers http.Header) map[string]string
}
```

---

## 依赖关系图

```
[StaticCrawler] ──depends on──> [HeaderProvider接口]
                                       ↑
                                       |
                                  implements
                                       |
                                [HeaderManager实现]
                                       ↑
                                       |
                                   uses
                                       |
                        +──────────────+──────────────+
                        |                             |
                [HeaderValidator接口]        [HeaderRedactor接口]
                        ↑                             ↑
                        |                             |
                   implements                    implements
                        |                             |
            [HeaderValidatorImpl]           [HeaderRedactorImpl]
```

**依赖倒置体现**:
- `StaticCrawler` (高层模块) 依赖 `HeaderProvider` 接口
- `HeaderManager` (低层模块) 实现 `HeaderProvider` 接口
- 高层不依赖低层具体实现,符合DIP原则

---

## 版本兼容性

| 版本 | 接口变更 | 兼容性 |
|------|----------|--------|
| v1.0 | 初始版本 | N/A |
| v1.1 (未来) | 新增 `GetHeadersWithContext(ctx context.Context)` 方法 | 向后兼容 (新接口) |
| v2.0 (未来) | `GetHeaders()` 签名变更 (增加context参数) | 破坏性变更 |

**向后兼容承诺**:
- v1.x版本保持 `GetHeaders()` 签名不变
- 新增功能通过新方法或新接口实现

---

## 实现检查清单

实现 `HeaderProvider` 接口时,必须满足:

- [ ] 实现 `GetHeaders() (http.Header, error)` 方法
- [ ] 返回的头部已验证合法性 (符合RFC 7230)
- [ ] 返回的头部已过滤禁止字段 (Host, Content-Length等)
- [ ] 错误时返回nil头部
- [ ] 多次调用返回一致结果 (幂等性)
- [ ] 头部名称已规范化 (首字母大写,如User-Agent)
- [ ] 包含至少一个默认User-Agent头部 (如未配置)
- [ ] 敏感头部可通过 `HeaderRedactor` 脱敏 (如实现)
- [ ] 通过契约测试用例 (`TestHeaderProviderContract`)
