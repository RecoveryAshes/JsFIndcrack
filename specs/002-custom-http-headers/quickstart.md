# 快速入门: 自定义HTTP请求头

**功能**: 002-custom-http-headers
**目标用户**: JsFIndcrack用户和开发者
**预计阅读时间**: 5分钟

## 概述

此功能允许您自定义爬虫发送HTTP请求时携带的请求头部,支持两种配置方式:

1. **配置文件** - 设置通用头部 (如User-Agent、Referer),持久化保存
2. **命令行参数** - 传入临时头部 (如认证Token),适合敏感信息

**优先级**: 命令行参数 > 配置文件 > 系统默认

---

## 快速开始 (5分钟)

### 步骤1: 生成配置文件模板

首次运行程序时,会自动生成默认配置文件:

```bash
# 运行任意命令,程序会自动创建 configs/headers.yaml
jsfindcrack -u https://example.com

# 或显式验证配置 (生成模板但不爬取)
jsfindcrack --validate-config
```

**生成的文件位置**: `configs/headers.yaml`

---

### 步骤2: 编辑配置文件

打开 `configs/headers.yaml`,根据注释修改头部:

```yaml
# 示例配置
headers:
  # 模拟Chrome浏览器
  User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"

  # 设置来源页面
  Referer: "https://www.google.com/"

  # 指定接受的语言
  Accept-Language: "zh-CN,zh;q=0.9"
```

**修改说明**:
- **保留注释**: 帮助您理解每个头部的用途
- **取消注释**: 移除行首的 `#` 启用某个头部
- **添加自定义**: 直接添加新的键值对

---

### 步骤3: 验证配置

在实际爬取前,验证配置文件是否正确:

```bash
jsfindcrack --validate-config
```

**输出示例** (成功):
```
✓ 配置文件加载成功: configs/headers.yaml
✓ 所有头部验证通过
✓ 共配置 3 个自定义头部:
  - User-Agent: Mozilla/5.0...
  - Referer: https://www.google.com/
  - Accept-Language: zh-CN,zh;q=0.9
```

**输出示例** (失败):
```
✗ 配置文件解析失败: configs/headers.yaml
  错误: yaml: line 5: mapping values are not allowed in this context
  建议: 检查第5行是否缺少空格或引号
```

---

### 步骤4: 使用配置爬取

配置文件生效后,所有HTTP请求都会携带自定义头部:

```bash
# 使用配置文件中的头部
jsfindcrack -u https://example.com

# 查看实际发送的头部 (调试模式)
jsfindcrack -u https://example.com --log-level debug
```

---

## 命令行参数方式

### 基本用法

对于临时需求或敏感信息,使用 `--header` (或 `-H`) 参数:

```bash
# 单个头部
jsfindcrack -u https://example.com --header "User-Agent: MyBot/1.0"

# 多个头部 (多次使用--header)
jsfindcrack -u https://example.com \
  --header "User-Agent: MyBot/1.0" \
  --header "X-Custom-Header: value"
```

### 认证头部示例

```bash
# Bearer Token认证
jsfindcrack -u https://api.example.com \
  --header "Authorization: Bearer YOUR_TOKEN_HERE"

# API Key认证
jsfindcrack -u https://api.example.com \
  --header "X-API-Key: YOUR_API_KEY_HERE"

# 多种认证方式组合
jsfindcrack -u https://api.example.com \
  --header "Authorization: Bearer TOKEN" \
  --header "X-API-Key: KEY" \
  --header "X-Request-ID: 12345"
```

---

## 优先级规则

### 覆盖配置文件

命令行参数会覆盖配置文件中的同名头部:

**configs/headers.yaml**:
```yaml
headers:
  User-Agent: "Chrome/120.0.0.0"
```

**命令行**:
```bash
jsfindcrack -u https://example.com --header "User-Agent: Firefox/119.0"
```

**实际发送**: `User-Agent: Firefox/119.0` (命令行优先)

### 完整优先级

```
命令行参数 (--header)
    ↓ 覆盖
配置文件 (configs/headers.yaml)
    ↓ 覆盖
系统默认 (内置User-Agent等)
```

---

## 常见场景

### 场景1: 绕过基础反爬虫

**目标**: 模拟真实浏览器,避免被识别为爬虫

**配置** (`configs/headers.yaml`):
```yaml
headers:
  User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
  Accept-Language: "zh-CN,zh;q=0.9,en;q=0.8"
  Accept-Encoding: "gzip, deflate, br"
  Referer: "https://www.google.com/"
```

---

### 场景2: API认证爬取

**目标**: 访问需要Bearer Token的API

**命令行**:
```bash
jsfindcrack -u https://api.example.com/data \
  --header "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**为什么不用配置文件?**
- Token可能频繁变化
- 避免敏感信息泄露 (配置文件可能被提交到Git)

---

### 场景3: 自定义业务头部

**目标**: 携带业务相关的自定义头部

**混合使用**:
```yaml
# configs/headers.yaml (通用头部)
headers:
  User-Agent: "MyApp/1.0"
  X-App-Version: "1.0.0"
```

```bash
# 命令行 (临时业务头部)
jsfindcrack -u https://api.example.com \
  --header "X-Request-ID: req-12345" \
  --header "X-Trace-ID: trace-67890"
```

---

## 安全注意事项

### 敏感信息保护

**自动脱敏**: 日志输出时,认证头部会自动脱敏

**配置文件日志**:
```
INFO  加载配置文件头部:
  User-Agent: Mozilla/5.0...
  Authorization: Bearer ***  ← 自动脱敏
```

**哪些头部会被脱敏?**
包含以下关键字的头部:
- `Authorization`
- `Token`
- `Key`
- `Secret`
- `Password`
- `Credential`

### 配置文件安全

**建议做法**:
```bash
# 将configs/目录添加到.gitignore,避免敏感信息泄露
echo "configs/headers.yaml" >> .gitignore
```

**不建议做法**:
```yaml
# ✗ 不要在配置文件中硬编码Token
headers:
  Authorization: "Bearer my-secret-token"  # 危险!
```

**推荐做法**:
```bash
# ✓ 通过命令行参数或环境变量传递
export TOKEN="my-secret-token"
jsfindcrack -u https://example.com --header "Authorization: Bearer $TOKEN"
```

---

## 常见问题

### Q1: 为什么首次运行没有生成配置文件?

**A**: 检查以下情况:
1. 当前目录是否有写权限?
   ```bash
   ls -ld . configs/
   ```
2. 是否指定了其他配置路径?
   ```bash
   # 默认路径是 ./configs/headers.yaml
   ```

---

### Q2: 配置文件格式错误怎么办?

**A**: 运行 `--validate-config` 查看详细错误:
```bash
jsfindcrack --validate-config
```

**常见错误**:
- **缺少空格**: `User-Agent:"value"` → `User-Agent: "value"`
- **缺少引号**: `User-Agent: Mozilla 5.0` → `User-Agent: "Mozilla 5.0"`
- **缩进错误**: YAML使用2空格缩进,不要用Tab

---

### Q3: 如何查看实际发送的头部?

**A**: 开启调试日志:
```bash
jsfindcrack -u https://example.com --log-level debug
```

**输出示例**:
```
DEBUG 最终HTTP头部:
  User-Agent: Mozilla/5.0...
  Referer: https://www.google.com/
  Authorization: Bearer ***
  X-Custom: value
```

---

### Q4: 某些头部不生效?

**A**: 某些头部由HTTP客户端自动管理,无法自定义:

**禁止配置的头部**:
- `Host` (由URL决定)
- `Content-Length` (由请求体决定)
- `Transfer-Encoding` (由客户端管理)
- `Connection` (由客户端管理)

**验证时会报错**:
```
✗ 头部验证失败 [Host]: 此头部由HTTP客户端自动管理,不允许自定义
  建议: 移除 'Host' 头部配置
```

---

### Q5: 命令行参数格式是什么?

**A**: 格式为 `"Name: Value"` (冒号后有空格)

**正确**:
```bash
--header "User-Agent: MyBot/1.0"
--header "X-Custom: value"
```

**错误**:
```bash
--header "User-Agent:MyBot/1.0"     # ✗ 冒号后缺少空格
--header "User-Agent = MyBot/1.0"   # ✗ 使用等号而非冒号
--header User-Agent: MyBot/1.0      # ✗ 缺少引号 (包含空格时)
```

---

## 下一步

- **实施开发**: 参考 [plan.md](plan.md) 了解技术设计
- **数据模型**: 参考 [data-model.md](data-model.md) 了解内部结构
- **接口契约**: 参考 [contracts/](contracts/) 了解接口定义
- **提交反馈**: 如有问题,请提交Issue

---

## 配置文件完整示例

```yaml
# JsFIndcrack HTTP请求头配置文件
# 生成时间: 2025-11-15

headers:
  # ========== 浏览器模拟 ==========
  User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"

  Accept-Language: "zh-CN,zh;q=0.9,en;q=0.8"

  Accept-Encoding: "gzip, deflate, br"

  # ========== 来源控制 ==========
  # Referer: "https://www.google.com/"

  # ========== 自定义业务头部 ==========
  # X-App-Name: "JsFIndcrack"
  # X-App-Version: "1.0.0"

  # ========== 认证 (不建议在配置文件中) ==========
  # Authorization: "Bearer YOUR_TOKEN"
  # X-API-Key: "YOUR_KEY"

# 使用说明:
# 1. 取消注释 (移除#) 以启用某个头部
# 2. 修改值以自定义头部内容
# 3. 添加新行以配置额外头部
# 4. 运行 jsfindcrack --validate-config 验证配置
```
