# 快速开始指南

**项目**: JsFIndcrack Go版本
**分支**: 001-py-to-go-migration
**日期**: 2025-11-15
**版本**: 1.0

---

## 概述

本指南帮助开发者快速上手JsFIndcrack Go版本的开发、构建、测试和部署。适用于参与迁移项目的开发者和贡献者。

---

## 环境要求

### 必需软件

| 软件 | 最低版本 | 推荐版本 | 用途 |
|------|---------|---------|------|
| **Go** | 1.21 | 1.21+ | Go语言运行时 |
| **Node.js** | 14.0 | 18.0+ | webcrack依赖 |
| **npm** | 6.0 | 9.0+ | webcrack安装 |
| **Playwright** | 1.30 | 最新 | 浏览器自动化 |
| **Git** | 2.20 | 最新 | 版本控制 |

### 可选软件

| 软件 | 用途 |
|------|------|
| **Make** | 简化构建命令 |
| **Docker** | 容器化部署 |
| **golangci-lint** | 代码质量检查 |

---

## 安装步骤

### 1. 克隆项目

```bash
# 克隆仓库
git clone https://github.com/RecoveryAshes/JsFIndcrack.git
cd JsFIndcrack

# 切换到Go迁移分支
git checkout 001-py-to-go-migration
```

### 2. 安装Go依赖

```bash
# 初始化Go模块(如果尚未初始化)
go mod init github.com/RecoveryAshes/JsFIndcrack

# 下载依赖
go mod download

# 整理依赖
go mod tidy
```

**核心依赖列表**:
```
# 网页爬取
github.com/gocolly/colly/v2 v2.1.0
github.com/PuerkitoBio/goquery v1.8.1
github.com/go-rod/rod v0.114.0

# 日志系统
github.com/rs/zerolog v1.32.0
gopkg.in/natefinch/lumberjack.v2 v2.2.1

# 并发控制
golang.org/x/sync v0.6.0

# 命令行
github.com/spf13/cobra v1.8.0
github.com/spf13/viper v1.18.2

# 进度条
github.com/schollz/progressbar/v3 v3.14.1

# 工具库
github.com/google/uuid v1.6.0
```

### 3. 安装外部依赖

#### 安装webcrack

```bash
# 全局安装webcrack
npm install -g webcrack

# 验证安装
webcrack --version
```

#### 安装Playwright

```bash
# 安装Playwright
npm install -g playwright

# 安装浏览器(Chromium)
playwright install chromium

# 验证安装
playwright --version
```

### 4. 验证安装

```bash
# 运行验证脚本
go run scripts/verify_setup.go
```

**验证脚本示例** (`scripts/verify_setup.go`):
```go
package main

import (
    "fmt"
    "os/exec"
    "runtime"
)

func main() {
    fmt.Println("JsFIndcrack环境验证")
    fmt.Println("==================")

    // 检查Go版本
    goVersion := runtime.Version()
    fmt.Printf("✅ Go版本: %s\n", goVersion)

    // 检查webcrack
    if _, err := exec.LookPath("webcrack"); err != nil {
        fmt.Println("❌ webcrack未安装")
    } else {
        fmt.Println("✅ webcrack已安装")
    }

    // 检查playwright
    if _, err := exec.LookPath("playwright"); err != nil {
        fmt.Println("❌ Playwright未安装")
    } else {
        fmt.Println("✅ Playwright已安装")
    }

    fmt.Println("==================")
    fmt.Println("环境验证完成!")
}
```

---

## 项目结构

```
JsFIndcrack/
├── cmd/                      # 命令行入口
│   └── jsfindcrack/
│       └── main.go           # 主程序入口
├── internal/                 # 内部包(不对外暴露)
│   ├── core/                 # 核心逻辑
│   │   ├── crawler.go        # 主爬取器
│   │   ├── deobfuscator.go   # 反混淆器
│   │   └── config.go         # 配置管理
│   ├── crawlers/             # 爬取器实现
│   │   ├── static.go         # 静态爬取器(Colly)
│   │   ├── dynamic.go        # 动态爬取器(Rod)
│   │   └── common.go         # 共用逻辑
│   ├── utils/                # 工具模块
│   │   ├── logger.go         # 日志系统
│   │   ├── reporter.go       # 报告生成器
│   │   ├── similarity.go     # 相似度分析器
│   │   ├── checkpoint.go     # 检查点管理
│   │   └── helpers.go        # 辅助函数
│   └── models/               # 数据模型
│       ├── task.go           # 爬取任务
│       ├── file.go           # JavaScript文件
│       ├── report.go         # 报告结构
│       └── checkpoint.go     # 检查点结构
├── pkg/                      # 公共包(可导出)
├── tests/                    # 测试
│   ├── unit/                 # 单元测试
│   ├── integration/          # 集成测试
│   └── e2e/                  # 端到端测试
├── configs/                  # 配置文件
│   └── config.yaml           # 默认配置
├── scripts/                  # 脚本
│   ├── build.sh              # 构建脚本
│   └── verify_setup.go       # 环境验证脚本
├── specs/                    # 设计文档
│   └── 001-py-to-go-migration/
│       ├── spec.md           # 功能规格
│       ├── plan.md           # 实施计划
│       ├── research.md       # 技术研究
│       ├── data-model.md     # 数据模型
│       ├── quickstart.md     # 本文档
│       └── contracts/        # 接口契约
├── output/                   # 输出目录
├── logs/                     # 日志目录
├── go.mod                    # Go模块定义
├── go.sum                    # 依赖锁定
├── Makefile                  # 构建配置
└── README.md                 # 项目说明
```

---

## 开发工作流

### 1. 创建新模块

以创建静态爬取器为例:

```bash
# 创建文件
touch internal/crawlers/static.go

# 编写代码
cat > internal/crawlers/static.go << 'EOF'
package crawlers

import (
    "github.com/gocolly/colly/v2"
    "github.com/RecoveryAshes/JsFIndcrack/internal/models"
)

type StaticCrawler struct {
    collector *colly.Collector
    config    *models.CrawlConfig
}

func NewStaticCrawler(config *models.CrawlConfig) *StaticCrawler {
    c := colly.NewCollector(
        colly.MaxDepth(config.Depth),
        colly.Async(true),
    )

    c.Limit(&colly.LimitRule{
        Parallelism: config.MaxWorkers,
    })

    return &StaticCrawler{
        collector: c,
        config:    config,
    }
}

func (sc *StaticCrawler) Crawl(targetURL string) error {
    // 实现爬取逻辑
    return sc.collector.Visit(targetURL)
}
EOF

# 创建测试文件
touch internal/crawlers/static_test.go
```

### 2. 编写单元测试

```go
// internal/crawlers/static_test.go
package crawlers

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/RecoveryAshes/JsFIndcrack/internal/models"
)

func TestNewStaticCrawler(t *testing.T) {
    config := &models.CrawlConfig{
        Depth:      2,
        MaxWorkers: 4,
    }

    crawler := NewStaticCrawler(config)

    assert.NotNil(t, crawler)
    assert.Equal(t, config, crawler.config)
}

func TestStaticCrawler_Crawl(t *testing.T) {
    // TODO: 实现测试逻辑
}
```

### 3. 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/crawlers

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 4. 代码格式化和检查

```bash
# 格式化代码
go fmt ./...

# 运行go vet
go vet ./...

# 运行golangci-lint(推荐)
golangci-lint run
```

### 5. 构建程序

```bash
# 构建当前平台的可执行文件
go build -o jsfindcrack cmd/jsfindcrack/main.go

# 或使用Make
make build

# 运行程序
./jsfindcrack --help
```

### 6. 交叉编译

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o jsfindcrack-linux-amd64 cmd/jsfindcrack/main.go

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o jsfindcrack-darwin-amd64 cmd/jsfindcrack/main.go

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o jsfindcrack-darwin-arm64 cmd/jsfindcrack/main.go

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o jsfindcrack-windows-amd64.exe cmd/jsfindcrack/main.go

# 或使用Make构建所有平台
make build-all
```

---

## Makefile示例

创建 `Makefile` 简化常用命令:

```makefile
.PHONY: build test clean install run fmt vet lint build-all

# 变量
BINARY_NAME=jsfindcrack
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# 默认目标
all: fmt vet test build

# 构建当前平台
build:
	@echo "构建 ${BINARY_NAME}..."
	go build ${LDFLAGS} -o ${BINARY_NAME} cmd/jsfindcrack/main.go

# 交叉编译所有平台
build-all:
	@echo "交叉编译所有平台..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 cmd/jsfindcrack/main.go
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 cmd/jsfindcrack/main.go
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 cmd/jsfindcrack/main.go
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe cmd/jsfindcrack/main.go

# 运行测试
test:
	@echo "运行测试..."
	go test -v -cover ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "生成覆盖率报告..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 运行go vet
vet:
	@echo "运行 go vet..."
	go vet ./...

# 运行linter
lint:
	@echo "运行 golangci-lint..."
	golangci-lint run

# 安装依赖
install:
	@echo "安装依赖..."
	go mod download
	go mod tidy

# 运行程序
run:
	go run cmd/jsfindcrack/main.go

# 清理
clean:
	@echo "清理..."
	rm -f ${BINARY_NAME} ${BINARY_NAME}-*
	rm -f coverage.out coverage.html
	rm -rf output/ logs/

# 显示帮助
help:
	@echo "可用命令:"
	@echo "  make build         - 构建当前平台的二进制文件"
	@echo "  make build-all     - 交叉编译所有平台"
	@echo "  make test          - 运行测试"
	@echo "  make test-coverage - 生成覆盖率报告"
	@echo "  make fmt           - 格式化代码"
	@echo "  make vet           - 运行 go vet"
	@echo "  make lint          - 运行 golangci-lint"
	@echo "  make install       - 安装依赖"
	@echo "  make run           - 运行程序"
	@echo "  make clean         - 清理构建产物"
```

---

## 常用开发命令

### 日常开发

```bash
# 格式化、检查、测试、构建一条龙
make all

# 快速测试
make test

# 运行程序
make run

# 或直接运行
go run cmd/jsfindcrack/main.go -u https://example.com
```

### 调试

```bash
# 使用Delve调试器
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug cmd/jsfindcrack/main.go -- -u https://example.com

# 或使用VSCode调试配置
# .vscode/launch.json:
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Package",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/jsfindcrack",
      "args": ["-u", "https://example.com"]
    }
  ]
}
```

### 性能分析

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# 内存profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# 运行时profiling
go run cmd/jsfindcrack/main.go -cpuprofile=cpu.prof -u https://example.com
go tool pprof cpu.prof
```

---

## 端到端测试

### 对比Python版本输出

```bash
# 运行Python版本
python main.py -u https://example.com -d 2
mv output/example.com output/python_example.com

# 运行Go版本
./jsfindcrack -u https://example.com -d 2
mv output/example.com output/go_example.com

# 对比JSON报告
diff output/python_example.com/crawl_report.json output/go_example.com/crawl_report.json

# 对比文件数量
ls -l output/python_example.com/encode/js | wc -l
ls -l output/go_example.com/encode/js | wc -l

# 对比文件哈希(验证内容一致性)
cd output/python_example.com/encode/js && find . -type f -exec md5 {} \; | sort > /tmp/python_hashes.txt
cd output/go_example.com/encode/js && find . -type f -exec md5 {} \; | sort > /tmp/go_hashes.txt
diff /tmp/python_hashes.txt /tmp/go_hashes.txt
```

---

## 持续集成(CI)

### GitHub Actions配置示例

创建 `.github/workflows/go.yml`:

```yaml
name: Go CI

on:
  push:
    branches: [ 001-py-to-go-migration ]
  pull_request:
    branches: [ 001-py-to-go-migration ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Install dependencies
      run: |
        go mod download
        npm install -g webcrack playwright
        playwright install chromium

    - name: Run tests
      run: make test

    - name: Run linter
      run: make lint

    - name: Build
      run: make build

  build-cross-platform:
    runs-on: ubuntu-latest
    needs: test
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Cross-compile
      run: make build-all

    - name: Upload artifacts
      uses: actions/upload-artifact@v3
      with:
        name: binaries
        path: jsfindcrack-*
```

---

## 故障排查

### 常见问题

**Q: `go: cannot find main module`**
```bash
# 确保在项目根目录
cd /path/to/JsFIndcrack

# 初始化模块(如果需要)
go mod init github.com/RecoveryAshes/JsFIndcrack
```

**Q: `webcrack: command not found`**
```bash
# 全局安装webcrack
npm install -g webcrack

# 或指定路径
export PATH=$PATH:$(npm get prefix)/bin
```

**Q: `cannot find package "github.com/gocolly/colly/v2"`**
```bash
# 下载依赖
go mod download

# 或使用tidy
go mod tidy
```

**Q: 测试失败 - `Playwright浏览器未安装`**
```bash
# 安装Playwright浏览器
playwright install chromium
```

---

## 下一步

完成环境搭建后,建议按以下顺序进行开发:

1. **阅读设计文档**:
   - [spec.md](spec.md) - 功能规格说明
   - [research.md](research.md) - 技术研究报告
   - [data-model.md](data-model.md) - 数据模型设计

2. **实现核心模块**:
   - 日志系统 (`internal/utils/logger.go`)
   - 数据模型 (`internal/models/`)
   - 静态爬取器 (`internal/crawlers/static.go`)
   - 动态爬取器 (`internal/crawlers/dynamic.go`)

3. **集成测试**:
   - 端到端测试
   - 与Python版本输出对比
   - 性能基准测试

4. **文档更新**:
   - README.md更新Go安装说明
   - API文档生成

---

## 参考资料

### 官方文档
- [Go语言文档](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go项目标准布局](https://github.com/golang-standards/project-layout)

### 依赖库文档
- [Colly](https://go-colly.org/)
- [Rod](https://go-rod.github.io/)
- [Zerolog](https://github.com/rs/zerolog)
- [Cobra](https://cobra.dev/)

### 工具
- [Delve调试器](https://github.com/go-delve/delve)
- [golangci-lint](https://golangci-lint.run/)
- [pprof性能分析](https://pkg.go.dev/net/http/pprof)

---

**文档更新**: 2025-11-15
**维护者**: JsFIndcrack开发团队
