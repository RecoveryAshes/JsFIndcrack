# JsFIndcrack - JavaScript文件爬取和反混淆工具

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20|%20macOS%20|%20Windows-lightgrey)](https://github.com/RecoveryAshes/JsFIndcrack)
[![GitHub Release](https://img.shields.io/github/v/release/RecoveryAshes/JsFIndcrack)](https://github.com/RecoveryAshes/JsFIndcrack/releases)

一个功能强大的JavaScript文件爬取和反混淆工具，使用Go语言开发，提供高性能、低内存占用和单一可执行文件部署。

## 主要特性

### 核心功能
- ✅ **单一可执行文件**: 无需运行时依赖，直接运行
- ✅ **跨平台支持**: Linux/macOS/Windows三平台原生支持
- ✅ **批量扫描**: 支持从文件读取URL列表进行批量爬取
- ✅ **多模式爬取**: 静态HTML解析(Colly)和动态浏览器执行(Rod)
- ✅ **Source Map支持**: 自动识别和下载JavaScript Source Map文件
- ✅ **智能去重**: SHA-256哈希去重,跨模式/跨域名
- ✅ **智能反混淆**: 集成webcrack工具,自动检测并反混淆
- ✅ **错误容错**: 批量模式下支持遇到错误继续处理
- ✅ **详细报告**: JSON格式爬取报告,成功/失败文件统计
- ✅ **自定义HTTP头**: 命令行或配置文件设置请求头
- ✅ **中文日志**: 彩色日志输出,多级别支持

### 性能优化
- ✅ **CPU自适应并发**: 根据CPU核心数动态调整并发数
- ✅ **页面池复用**: 动态爬取标签页智能复用,降低内存
- ✅ **流式文件处理**: 大文件流式读写,低内存占用
- ✅ **多线程支持**: 可配置并发线程数

---

## 快速开始

### 选项1: 下载预编译二进制文件 (推荐)

访问[Releases页面](https://github.com/RecoveryAshes/JsFIndcrack/releases)下载最新版本的预编译二进制文件。

根据您的操作系统选择对应的文件:
- Linux (amd64): `jsfindcrack-{VERSION}-linux-amd64.tar.gz`
- macOS (Intel): `jsfindcrack-{VERSION}-darwin-amd64.tar.gz`
- macOS (Apple Silicon): `jsfindcrack-{VERSION}-darwin-arm64.tar.gz`
- Windows (amd64): `jsfindcrack-{VERSION}-windows-amd64.zip`

下载并解压后即可使用:

```bash
# Linux
wget https://github.com/RecoveryAshes/JsFIndcrack/releases/download/{VERSION}/jsfindcrack-{VERSION}-linux-amd64.tar.gz
tar -xzf jsfindcrack-{VERSION}-linux-amd64.tar.gz
chmod +x jsfindcrack
./jsfindcrack --help

# macOS (Intel)
curl -LO https://github.com/RecoveryAshes/JsFIndcrack/releases/download/{VERSION}/jsfindcrack-{VERSION}-darwin-amd64.tar.gz
tar -xzf jsfindcrack-{VERSION}-darwin-amd64.tar.gz
chmod +x jsfindcrack
./jsfindcrack --help

# macOS (Apple Silicon)
curl -LO https://github.com/RecoveryAshes/JsFIndcrack/releases/download/{VERSION}/jsfindcrack-{VERSION}-darwin-arm64.tar.gz
tar -xzf jsfindcrack-{VERSION}-darwin-arm64.tar.gz
chmod +x jsfindcrack
./jsfindcrack --help

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/RecoveryAshes/JsFIndcrack/releases/download/{VERSION}/jsfindcrack-{VERSION}-windows-amd64.zip" -OutFile jsfindcrack.zip
Expand-Archive jsfindcrack.zip
.\jsfindcrack\jsfindcrack.exe --help
```

**注意**: 将`{VERSION}`替换为实际版本号，如`v2.0.0`

### 选项2: 从源码构建

```bash
# 1. 克隆项目
git clone https://github.com/RecoveryAshes/JsFIndcrack.git
cd JsFIndcrack

# 2. 确保安装Go 1.21+
go version

# 3. 安装依赖
go mod download

# 4. 构建
make build
# 或直接使用go build
go build -o jsfindcrack ./cmd/jsfindcrack

# 5. 运行
./jsfindcrack --help
```

### 外部依赖(可选)

**webcrack** (用于JavaScript反混淆):
```bash
# 安装Node.js和npm
# Linux: sudo apt install nodejs npm
# macOS: brew install node
# Windows: https://nodejs.org/

# 安装webcrack
npm install -g webcrack
```

**浏览器** (动态爬取模式):
- Rod会自动下载Chromium,首次运行时自动安装
- 或使用系统已安装的Chrome: `export ROD_BROWSER_PATH=/path/to/chrome`

---

## 使用说明

### 基本使用

```bash
# 爬取单个网站
./jsfindcrack -u https://example.com -d 2

# 动态爬取(SPA应用)
./jsfindcrack -u https://app.example.com --mode dynamic --headless

# 批量爬取
./jsfindcrack -f urls.txt --threads 4 --continue-on-error

# 自定义HTTP头
./jsfindcrack -u https://api.example.com -H "Authorization: Bearer TOKEN"

# 混合模式(静态+动态)
./jsfindcrack -u https://example.com --mode all --threads 4
```

### 完整参数列表

```
用法:
  jsfindcrack [flags]
  jsfindcrack [command]

可用命令:
  help        帮助信息
  version     显示版本信息

基本参数:
  -u, --url string              目标URL
  -f, --file string             URL列表文件
  -d, --depth int               爬取深度 (1-10) (默认: 2)
  -m, --mode string             爬取模式: static/dynamic/all (默认: static)
  -o, --output string           输出目录 (默认: output)
  -l, --log-level string        日志级别: debug/info/warn/error (默认: info)

性能参数:
      --threads int             并发线程数 (默认: 2)
      --wait-time int           页面等待时间(秒) (默认: 3)
      --playwright-tabs int     动态爬取标签页数 (默认: 4)
      --batch-delay int         批量爬取延迟(秒) (默认: 0)

HTTP头部参数:
  -H, --headers stringArray     自定义HTTP请求头 (可多次使用)
  -c, --config string           配置文件路径
      --validate-config         验证配置文件

高级参数:
      --headless                无头浏览器模式 (默认: true)
      --continue-on-error       批量爬取出错时继续
      --similarity-enabled      启用相似度分析
      --similarity-threshold float  相似度阈值 (0.0-1.0) (默认: 0.9)
```

### URL列表文件格式

创建一个文本文件（如 `urls.txt`），每行一个URL：

```
# 这是注释行，会被忽略
https://example1.com
https://example2.com
https://example3.com

# 空行也会被忽略
https://example4.com
```

---

## 输出目录结构

```
output/
└── example.com/
    ├── encode/
    │   └── js/          # 原始JS文件
    ├── decode/
    │   └── js/          # 反混淆文件
    └── reports/         # 爬取报告
        ├── crawl_report.json
        ├── success_files.json
        └── failed_files.json
```

### 配置文件

使用`configs/headers.yaml`配置HTTP头:

```yaml
headers:
  User-Agent: "JsFIndcrack/2.0"
  Accept-Language: "zh-CN,zh;q=0.9"
```

---

## 项目结构

```
JsFIndcrack/
├── cmd/
│   └── jsfindcrack/          # 命令行入口
│       ├── main.go           # 主程序
│       └── validate.go       # 参数验证
├── internal/
│   ├── core/                 # 核心模块
│   │   ├── crawler.go        # 主爬取协调器
│   │   ├── deobfuscator.go   # 反混淆器
│   │   ├── batch.go          # 批量处理
│   │   └── header_manager.go # HTTP头管理
│   ├── crawlers/             # 爬取器实现
│   │   ├── static.go         # 静态爬取(Colly)
│   │   ├── dynamic.go        # 动态爬取(Rod)
│   │   ├── page_pool.go      # 标签页池管理
│   │   ├── resource_monitor.go # 资源监控
│   │   ├── url_extractor.go  # URL提取器
│   │   └── url_queue.go      # URL队列
│   ├── models/               # 数据模型
│   │   ├── task.go           # 任务模型
│   │   ├── file.go           # 文件模型
│   │   └── report.go         # 报告模型
│   ├── config/               # 配置管理
│   │   └── headers.go        # 头部配置
│   └── utils/                # 工具函数
│       ├── logger.go         # 日志系统
│       ├── reporter.go       # 报告生成
│       └── validator.go      # 验证工具
├── tests/
│   ├── unit/                 # 单元测试
│   ├── e2e/                  # 端到端测试
│   └── benchmark/            # 性能测试
├── configs/
│   └── config.yaml           # 配置文件
├── go.mod                    # Go模块定义
├── go.sum                    # 依赖锁定
├── Makefile                  # 构建任务
└── README.md                 # 本文件
```

---

## 使用示例

### 场景1: 安全研究

```bash
# 爬取目标网站的所有JS文件
./jsfindcrack -u https://target.com -d 3 --mode all --log-level warn

# 搜索敏感信息
grep -r "api_key\|password\|secret" output/target.com/
```

### 场景2: 前端资产分析

```bash
# 分析React应用的bundle
./jsfindcrack -u https://react-app.com -d 1 --mode dynamic

# 查看文件结构
tree output/react-app.com/encode/js/
```

### 场景3: 批量竞品分析

```bash
# 创建竞品列表
cat > competitors.txt <<EOF
https://competitor1.com
https://competitor2.com
https://competitor3.com
EOF

# 批量爬取
./jsfindcrack -f competitors.txt --mode all --threads 3 --continue-on-error
```

### 场景4: CI/CD集成

```bash
#!/bin/bash
# 持续监控目标网站JS变化

TARGET_URL="https://monitored-site.com"
OUTPUT_DIR="output/$(date +%Y%m%d)"

./jsfindcrack -u "$TARGET_URL" -d 2 --mode static --output "$OUTPUT_DIR"

# 与上一次结果对比
diff -r "$OUTPUT_DIR" "output/previous/" > changes.txt
if [ -s changes.txt ]; then
    echo "检测到JS文件变化!"
    cat changes.txt
fi
```

---

## 构建和开发

### 本地开发

```bash
# 安装依赖
go mod download

# 运行测试
make test
# 或
go test ./...

# 运行E2E测试
make test-e2e

# 格式化代码
go fmt ./...

# 代码检查
go vet ./...
golangci-lint run
```

### 交叉编译

```bash
# 构建所有平台
make build-all

# 输出位于当前目录:
# - jsfindcrack-linux-amd64
# - jsfindcrack-darwin-amd64
# - jsfindcrack-darwin-arm64
# - jsfindcrack-windows-amd64.exe
```

---

## 性能基准

**测试环境**: macOS M1, 16GB RAM

| 场景 | 文件数 | 模式 | 并发 | 耗时 | 内存 |
|------|--------|------|------|------|------|
| 小型网站 | ~20 | static | 2 | 15秒 | 60MB |
| 中型SPA | ~50 | dynamic | 4 | 45秒 | 250MB |
| 大型网站 | ~200 | all | 8 | 3分钟 | 500MB |
| 批量10个URL | ~150 | static | 4 | 2分钟 | 200MB |
| 批量100个URL | ~800 | static | 8 | 18分钟 | 600MB |

---

## 故障排查

### 常见问题

**Q1: macOS提示"无法验证开发者"**
```bash
xattr -d com.apple.quarantine jsfindcrack
```

**Q2: 反混淆失败**
```bash
# 检查webcrack是否安装
which webcrack
npm install -g webcrack
```

**Q3: 动态爬取失败**
```bash
# 启用调试日志
./jsfindcrack -u URL --mode dynamic --log-level debug

# 指定浏览器路径
export ROD_BROWSER_PATH=/path/to/chrome
```

**Q4: 编译错误**
```bash
# 清理缓存重新构建
go clean -cache
go mod download
go build ./cmd/jsfindcrack
```

---

## 技术栈

- **语言**: Go 1.21+
- **静态爬取**: [Colly](https://github.com/gocolly/colly) - 高效HTTP爬取
- **动态爬取**: [Rod](https://github.com/go-rod/rod) - 浏览器自动化
- **日志**: [Zerolog](https://github.com/rs/zerolog) - 高性能结构化日志
- **CLI**: [Cobra](https://github.com/spf13/cobra) - 现代CLI框架
- **配置**: [Viper](https://github.com/spf13/viper) - 配置管理
- **并发**: errgroup, sync - Go标准库并发原语
- **反混淆**: [webcrack](https://github.com/j4k0xb/webcrack) (外部依赖)

---

## 贡献指南

欢迎贡献代码!请遵循以下步骤:

1. Fork本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启Pull Request

**代码规范**:
- 使用`gofmt`格式化代码
- 通过`go vet`检查
- 添加单元测试
- 更新文档

---

## 开发路线图

### v2.1 (已完成) ✅
- [x] 自适应标签页池管理
- [x] 系统资源监控
- [x] URL优先级队列
- [x] GitHub Actions自动发布

### v2.2 - MCP化 (下一版本) 🎯
- [ ] **MCP服务器实现** - 将工具转换为Model Context Protocol服务
- [ ] **工具函数接口** - 提供爬取、分析、反混淆等标准化API
- [ ] **与AI助手集成** - 支持Claude等AI直接调用
- [ ] **异步任务管理** - 长时间运行任务的状态跟踪
- [ ] **结果流式输出** - 实时返回爬取进度和结果

### v2.3 (计划中)
- [ ] Web UI界面
- [ ] 实时进度WebSocket推送
- [ ] 插件系统
- [ ] 更多反混淆引擎支持

### v2.4 (计划中)
- [ ] 分布式爬取
- [ ] Redis缓存支持
- [ ] Docker镜像
- [ ] Kubernetes部署

---

## 许可证

本项目采用MIT许可证 - 查看[LICENSE](LICENSE)文件了解详情。

---

## 致谢

- [Colly](https://github.com/gocolly/colly) - 静态爬取引擎
- [Rod](https://github.com/go-rod/rod) - 浏览器自动化
- [Cobra](https://github.com/spf13/cobra) - CLI框架
- [Zerolog](https://github.com/rs/zerolog) - 日志库
- [webcrack](https://github.com/j4k0xb/webcrack) - JavaScript反混淆

---

## 联系方式

- **项目主页**: https://github.com/RecoveryAshes/JsFIndcrack
- **问题反馈**: [Issues](https://github.com/RecoveryAshes/JsFIndcrack/issues)
- **功能请求**: [Discussions](https://github.com/RecoveryAshes/JsFIndcrack/discussions)

---

**如果这个项目对你有帮助,请给它一个星标!** ⭐

**更快、更轻、更强大!** 🚀