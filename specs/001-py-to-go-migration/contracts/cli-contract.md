# 命令行接口契约

**项目**: JsFIndcrack - Python到Go迁移
**分支**: 001-py-to-go-migration
**日期**: 2025-11-15
**版本**: 1.0

---

## 概述

本文档定义JsFIndcrack Go版本的命令行接口(CLI)契约,确保与Python版本100%兼容。由于本项目是CLI工具而非传统REST/GraphQL API,本契约定义命令行参数、选项、子命令、输出格式和退出码规范。

---

## 命令行接口规范

### 根命令

```
jsfindcrack - JavaScript文件爬取和反混淆工具
```

### 全局标志(Flags)

| 标志 | 简写 | 类型 | 默认值 | 必需 | 描述 |
|------|------|------|--------|------|------|
| `--url` | `-u` | string | - | 条件* | 目标网站URL |
| `--url-file` | `-f` | string | - | 条件* | URL列表文件路径 |
| `--depth` | `-d` | int | 2 | 否 | 爬取深度(1-10) |
| `--wait` | `-w` | int | 3 | 否 | 页面等待时间(秒,0-60) |
| `--threads` | `-t` | int | 2 | 否 | 静态爬取并行线程数(1-100) |
| `--playwright-tabs` | - | int | 4 | 否 | Playwright同时打开的标签页数量(1-20) |
| `--headless` | - | bool | true | 否 | Playwright无头模式运行 |
| `--no-headless` | - | bool | false | 否 | Playwright有头模式运行 |
| `--mode` | - | string | "all" | 否 | 爬取模式: static/dynamic/all |
| `--resume` | `-r` | bool | false | 否 | 从检查点恢复 |
| `--similarity` | - | bool | true | 否 | 启用智能相似度检测和去重 |
| `--similarity-threshold` | - | float64 | 0.8 | 否 | 相似度阈值(0.0-1.0) |
| `--similarity-workers` | - | int | CPU核心数 | 否 | 相似度分析并行工作线程数 |
| `--batch-delay` | - | int | 0 | 否 | 批量模式下URL之间的延迟时间(秒) |
| `--continue-on-error` | - | bool | false | 否 | 批量模式下遇到错误时继续处理下一个URL |
| `--config` | `-c` | string | - | 否 | 配置文件路径(YAML格式) |
| `--output` | `-o` | string | "output" | 否 | 输出目录 |
| `--log-level` | - | string | "INFO" | 否 | 日志级别: DEBUG/INFO/WARN/ERROR |
| `--version` | `-v` | bool | false | 否 | 显示版本信息 |
| `--help` | `-h` | bool | false | 否 | 显示帮助信息 |

*注: `--url` 和 `--url-file` 互斥,必须指定其中一个。

---

## 使用示例

### 1. 单URL爬取

#### 基本爬取
```bash
jsfindcrack -u https://example.com
```

#### 自定义参数
```bash
jsfindcrack -u https://example.com -d 3 -w 5 -t 4 --playwright-tabs 6
```

#### 仅静态爬取
```bash
jsfindcrack -u https://example.com --mode static
```

#### 仅动态爬取
```bash
jsfindcrack -u https://example.com --mode dynamic
```

#### 启用相似度检测
```bash
jsfindcrack -u https://example.com --similarity --similarity-threshold 0.8
```

#### 断点续爬
```bash
jsfindcrack -u https://example.com --resume
```

### 2. 批量URL爬取

#### 基本批量爬取
```bash
jsfindcrack -f urls.txt
```

#### 批量爬取,遇到错误继续
```bash
jsfindcrack -f urls.txt --continue-on-error
```

#### 批量爬取,设置延迟
```bash
jsfindcrack -f urls.txt --batch-delay 2 --continue-on-error
```

#### 批量爬取,自定义参数
```bash
jsfindcrack -f urls.txt -d 2 -t 4 --batch-delay 1 --continue-on-error --mode static
```

### 3. 配置文件使用

```bash
jsfindcrack -u https://example.com -c config.yaml
```

**配置文件格式(config.yaml)**:
```yaml
# 爬取配置
depth: 3
wait_time: 5
max_workers: 4
playwright_tabs: 6
headless: true
mode: all

# 相似度配置
similarity_enabled: true
similarity_threshold: 0.8
similarity_workers: 8

# 输出配置
output_dir: ./output
log_level: INFO
```

---

## 输入验证规则

### URL验证

```go
// ValidateURL 验证URL格式
func ValidateURL(url string) error {
    if url == "" {
        return fmt.Errorf("URL不能为空")
    }

    parsed, err := url.Parse(url)
    if err != nil {
        return fmt.Errorf("无效的URL格式: %w", err)
    }

    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return fmt.Errorf("URL必须使用HTTP或HTTPS协议")
    }

    if parsed.Host == "" {
        return fmt.Errorf("URL必须包含域名")
    }

    return nil
}
```

### 参数范围验证

```go
// ValidateFlags 验证命令行参数
func ValidateFlags(flags *Flags) error {
    // URL互斥检查
    if flags.URL == "" && flags.URLFile == "" {
        return fmt.Errorf("必须指定 --url 或 --url-file 参数")
    }
    if flags.URL != "" && flags.URLFile != "" {
        return fmt.Errorf("--url 和 --url-file 参数互斥,只能指定一个")
    }

    // 深度范围检查
    if flags.Depth < 1 || flags.Depth > 10 {
        return fmt.Errorf("--depth 必须在1-10之间,当前值: %d", flags.Depth)
    }

    // 等待时间检查
    if flags.WaitTime < 0 || flags.WaitTime > 60 {
        return fmt.Errorf("--wait 必须在0-60秒之间,当前值: %d", flags.WaitTime)
    }

    // 并发数检查
    if flags.Threads < 1 || flags.Threads > 100 {
        return fmt.Errorf("--threads 必须在1-100之间,当前值: %d", flags.Threads)
    }

    // 标签页数检查
    if flags.PlaywrightTabs < 1 || flags.PlaywrightTabs > 20 {
        return fmt.Errorf("--playwright-tabs 必须在1-20之间,当前值: %d", flags.PlaywrightTabs)
    }

    // 爬取模式检查
    validModes := map[string]bool{"static": true, "dynamic": true, "all": true}
    if !validModes[flags.Mode] {
        return fmt.Errorf("--mode 必须是 static, dynamic 或 all,当前值: %s", flags.Mode)
    }

    // 相似度阈值检查
    if flags.SimilarityThreshold < 0.0 || flags.SimilarityThreshold > 1.0 {
        return fmt.Errorf("--similarity-threshold 必须在0.0-1.0之间,当前值: %.2f", flags.SimilarityThreshold)
    }

    // 日志级别检查
    validLevels := map[string]bool{"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true}
    if !validLevels[strings.ToUpper(flags.LogLevel)] {
        return fmt.Errorf("--log-level 必须是 DEBUG, INFO, WARN 或 ERROR,当前值: %s", flags.LogLevel)
    }

    return nil
}
```

### URL文件格式验证

```go
// LoadURLsFromFile 从文件加载URL列表
func LoadURLsFromFile(filepath string) ([]string, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return nil, fmt.Errorf("无法打开文件: %w", err)
    }
    defer file.Close()

    var urls []string
    scanner := bufio.NewScanner(file)
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        line := strings.TrimSpace(scanner.Text())

        // 跳过空行和注释
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        // 验证URL
        if err := ValidateURL(line); err != nil {
            return nil, fmt.Errorf("第%d行URL无效: %w", lineNum, err)
        }

        urls = append(urls, line)
    }

    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("读取文件错误: %w", err)
    }

    if len(urls) == 0 {
        return nil, fmt.Errorf("文件中没有有效的URL")
    }

    return urls, nil
}
```

---

## 输出格式规范

### 1. 标准输出(Stdout)

#### 实时进度信息

```
[2025-11-15 10:00:00] INFO 开始爬取: https://example.com
[2025-11-15 10:00:01] INFO 模式: all, 深度: 2, 并发: 4
[2025-11-15 10:00:02] INFO 静态爬取中...
进度: ████████████████████ 45/100 (45.0%)
[2025-11-15 10:02:30] INFO 静态爬取完成,获得 89 个文件
[2025-11-15 10:02:31] INFO 动态爬取中...
进度: ████████████████████ 67/100 (67.0%)
[2025-11-15 10:05:00] INFO 动态爬取完成,获得 67 个文件
[2025-11-15 10:05:01] INFO 执行相似度分析...
进度: ████████████████████ 156/156 (100.0%)
[2025-11-15 10:05:15] INFO 相似度分析完成,发现 12 组重复文件
[2025-11-15 10:05:16] INFO 执行反混淆处理...
进度: ████████████████████ 156/156 (100.0%)
[2025-11-15 10:05:30] INFO 反混淆完成,处理 45 个文件
[2025-11-15 10:05:30] INFO 爬取完成!
[2025-11-15 10:05:30] INFO 总文件: 156, 成功: 153, 失败: 3
[2025-11-15 10:05:30] INFO 总大小: 12.5 MB, 耗时: 5分30秒
[2025-11-15 10:05:30] INFO 输出目录: output/example.com
```

#### 批量爬取进度

```
[2025-11-15 10:00:00] INFO 批量爬取开始,共 10 个URL
[2025-11-15 10:00:01] INFO 处理 1/10: https://example1.com
[2025-11-15 10:05:30] INFO ✅ https://example1.com 完成,获得 156 个文件
[2025-11-15 10:05:32] INFO 处理 2/10: https://example2.com
[2025-11-15 10:08:45] INFO ❌ https://example2.com 失败: 连接超时
[2025-11-15 10:08:47] INFO 处理 3/10: https://example3.com
...
[2025-11-15 11:00:00] INFO 批量爬取完成!
[2025-11-15 11:00:00] INFO 成功: 8/10, 失败: 2/10
[2025-11-15 11:00:00] INFO 总文件: 1,234, 总大小: 125.8 MB
```

### 2. 错误输出(Stderr)

```
[2025-11-15 10:00:05] ERROR 下载失败: https://example.com/script.js - 超时
[2025-11-15 10:00:10] WARN 检测到反爬虫机制,切换策略
[2025-11-15 10:00:15] ERROR webcrack执行失败: /path/to/file.js - 未安装
```

### 3. JSON报告文件

#### crawl_report.json

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "target_url": "https://example.com",
  "domain": "example.com",
  "mode": "all",
  "start_time": "2025-11-15T10:00:00Z",
  "end_time": "2025-11-15T10:05:30Z",
  "duration": 330.5,
  "stats": {
    "total_files": 156,
    "static_files": 89,
    "dynamic_files": 67,
    "map_files": 23,
    "failed_files": 3,
    "deobfuscated_files": 45,
    "total_size": 12458960,
    "visited_urls": 234
  },
  "similarity_analysis": {
    "enabled": true,
    "total_files": 156,
    "duplicate_groups": 12,
    "unique_files": 120,
    "duplicate_files": 36,
    "space_saved": 3458960,
    "analysis_duration": 15.2
  },
  "output_dir": "output/example.com",
  "encode_dir": "output/example.com/encode",
  "decode_dir": "output/example.com/decode",
  "config": {
    "depth": 2,
    "wait_time": 3,
    "max_workers": 4,
    "playwright_tabs": 6,
    "headless": true,
    "resume": false,
    "similarity_enabled": true,
    "similarity_threshold": 0.8,
    "similarity_workers": 8
  }
}
```

#### success_files.json

```json
[
  {
    "url": "https://example.com/js/main.js",
    "file_path": "output/example.com/encode/js/main.js",
    "size": 45678,
    "hash": "a1b2c3d4e5f6...",
    "crawl_mode": "static",
    "downloaded_at": "2025-11-15T10:00:15Z"
  },
  {
    "url": "https://example.com/js/app.js",
    "file_path": "output/example.com/encode/js/app.js",
    "size": 123456,
    "hash": "f6e5d4c3b2a1...",
    "crawl_mode": "dynamic",
    "downloaded_at": "2025-11-15T10:02:45Z"
  }
]
```

#### failed_files.json

```json
[
  {
    "url": "https://example.com/js/vendor.js",
    "error_type": "timeout",
    "error_msg": "请求超时: 30秒",
    "retries": 3
  },
  {
    "url": "https://example.com/js/old.js",
    "error_type": "not_found",
    "error_msg": "HTTP 404 Not Found",
    "retries": 0
  }
]
```

---

## 退出码(Exit Codes)

| 退出码 | 含义 | 触发条件 |
|--------|------|---------|
| 0 | 成功 | 所有任务成功完成 |
| 1 | 通用错误 | 未分类的错误 |
| 2 | 参数错误 | 命令行参数无效 |
| 3 | 配置错误 | 配置文件格式错误或缺少必需配置 |
| 4 | 网络错误 | 网络连接失败或超时 |
| 5 | 文件系统错误 | 文件读写权限问题 |
| 6 | 外部依赖错误 | webcrack或Playwright未安装或执行失败 |
| 7 | 部分失败 | 批量模式下部分URL失败(仅当--continue-on-error时) |
| 130 | 用户中断 | 用户按Ctrl+C中断 |

### 退出码使用示例

```go
package main

import (
    "os"
)

const (
    ExitSuccess         = 0
    ExitGeneralError    = 1
    ExitInvalidArgs     = 2
    ExitConfigError     = 3
    ExitNetworkError    = 4
    ExitFileSystemError = 5
    ExitDependencyError = 6
    ExitPartialFailure  = 7
    ExitUserInterrupt   = 130
)

func main() {
    if err := run(); err != nil {
        switch err.(type) {
        case *InvalidArgsError:
            os.Exit(ExitInvalidArgs)
        case *NetworkError:
            os.Exit(ExitNetworkError)
        // ... 其他错误类型
        default:
            os.Exit(ExitGeneralError)
        }
    }
    os.Exit(ExitSuccess)
}
```

---

## 帮助信息格式

### --help输出

```
jsfindcrack - JavaScript文件爬取和反混淆工具

用法:
  jsfindcrack [标志]

示例:
  # 爬取单个网站
  jsfindcrack -u https://example.com

  # 批量爬取网站
  jsfindcrack -f urls.txt

  # 自定义参数爬取
  jsfindcrack -u https://example.com -d 3 -w 5 -t 4

  # 仅静态爬取
  jsfindcrack -u https://example.com --mode static

  # 启用相似度检测
  jsfindcrack -u https://example.com --similarity --similarity-threshold 0.8

标志:
  -u, --url string                     目标网站URL
  -f, --url-file string                URL列表文件路径
  -d, --depth int                      爬取深度 (默认 2)
  -w, --wait int                       页面等待时间(秒) (默认 3)
  -t, --threads int                    静态爬取并行线程数 (默认 2)
      --playwright-tabs int            Playwright标签页数量 (默认 4)
      --headless                       Playwright无头模式 (默认 true)
      --no-headless                    Playwright有头模式
      --mode string                    爬取模式: static/dynamic/all (默认 "all")
  -r, --resume                         从检查点恢复
      --similarity                     启用相似度检测 (默认 true)
      --similarity-threshold float     相似度阈值 (默认 0.8)
      --similarity-workers int         相似度分析并发数 (默认 CPU核心数)
      --batch-delay int                批量模式URL间延迟(秒) (默认 0)
      --continue-on-error              批量模式遇错继续
  -c, --config string                  配置文件路径
  -o, --output string                  输出目录 (默认 "output")
      --log-level string               日志级别: DEBUG/INFO/WARN/ERROR (默认 "INFO")
  -v, --version                        显示版本信息
  -h, --help                           显示帮助信息

注意:
  --url 和 --url-file 参数互斥,必须指定其中一个

更多信息: https://github.com/RecoveryAshes/JsFIndcrack
```

### --version输出

```
jsfindcrack 版本 2.0.0
Go版本: go1.21.0
平台: darwin/arm64
构建时间: 2025-11-15T10:00:00Z
Git提交: a1b2c3d
```

---

## 与Python版本的兼容性清单

| 特性 | Python版本 | Go版本 | 兼容性 |
|------|-----------|--------|--------|
| 所有命令行参数 | ✅ | ✅ | 100% |
| 参数简写 | ✅ | ✅ | 100% |
| 默认值 | ✅ | ✅ | 100% |
| 验证规则 | ✅ | ✅ | 100% |
| 错误消息(中文) | ✅ | ✅ | 100% |
| JSON报告格式 | ✅ | ✅ | 100% |
| 输出目录结构 | ✅ | ✅ | 100% |
| 退出码 | ✅ | ✅ | 100% |
| 进度显示 | ✅ | ✅ | 100% |
| 检查点格式 | ✅ | ✅ | 100% |

---

## 实现建议

### 使用Cobra框架

```go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
    Use:   "jsfindcrack",
    Short: "JavaScript文件爬取和反混淆工具",
    Long:  `一个功能强大的JavaScript文件爬取和反混淆工具...`,
    RunE:  run,
}

func init() {
    // URL参数
    rootCmd.Flags().StringP("url", "u", "", "目标网站URL")
    rootCmd.Flags().StringP("url-file", "f", "", "URL列表文件路径")
    rootCmd.MarkFlagsMutuallyExclusive("url", "url-file")

    // 爬取参数
    rootCmd.Flags().IntP("depth", "d", 2, "爬取深度")
    rootCmd.Flags().IntP("wait", "w", 3, "页面等待时间(秒)")
    rootCmd.Flags().IntP("threads", "t", 2, "静态爬取并行线程数")
    rootCmd.Flags().Int("playwright-tabs", 4, "Playwright标签页数量")
    rootCmd.Flags().Bool("headless", true, "Playwright无头模式")
    rootCmd.Flags().Bool("no-headless", false, "Playwright有头模式")
    rootCmd.Flags().String("mode", "all", "爬取模式: static/dynamic/all")
    rootCmd.Flags().BoolP("resume", "r", false, "从检查点恢复")

    // 相似度参数
    rootCmd.Flags().Bool("similarity", true, "启用相似度检测")
    rootCmd.Flags().Float64("similarity-threshold", 0.8, "相似度阈值")
    rootCmd.Flags().Int("similarity-workers", 0, "相似度分析并发数")

    // 批量参数
    rootCmd.Flags().Int("batch-delay", 0, "批量模式URL间延迟(秒)")
    rootCmd.Flags().Bool("continue-on-error", false, "批量模式遇错继续")

    // 其他参数
    rootCmd.Flags().StringP("config", "c", "", "配置文件路径")
    rootCmd.Flags().StringP("output", "o", "output", "输出目录")
    rootCmd.Flags().String("log-level", "INFO", "日志级别")

    // 绑定到viper
    viper.BindPFlags(rootCmd.Flags())
}

func Execute() error {
    return rootCmd.Execute()
}
```

---

## 总结

本CLI契约规范:

1. ✅ **100%兼容**: 与Python版本所有参数和行为一致
2. ✅ **类型安全**: 利用Go的类型系统进行编译时检查
3. ✅ **验证完整**: 所有参数都有明确的验证规则
4. ✅ **错误清晰**: 提供中文错误消息和退出码
5. ✅ **输出一致**: JSON格式和目录结构与Python版本一致

**下一步**: 生成quickstart.md快速开始指南。
