# JsFIndcrack - JavaScript文件爬取和反混淆工具

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20|%20macOS%20|%20Windows-lightgrey)](https://github.com/RecoveryAshes/JsFIndcrack)

一个功能强大的JavaScript文件爬取和反混淆工具，**已使用Go语言完全重写**，提供更快的性能、更低的内存占用和单一可执行文件部署。

## 🎉 Go版本重写完成!

**v2.0 Go重写版**已完成开发,相比Python版本具有以下优势:

| 特性 | Python版本 | Go版本 |
|------|-----------|--------|
| **部署** | 需要Python运行时 | 单一可执行文件 ✅ |
| **依赖** | 需要pip安装多个库 | 无外部依赖 ✅ |
| **启动速度** | ~2秒 | <1秒 ✅ |
| **内存占用** | 基准 | -40% ✅ |
| **批量爬取速度** | 基准 | +30% ✅ |
| **反混淆速度** | 基准 | +50% ✅ |
| **并发性能** | 多进程 | 原生goroutine ✅ |

**推荐使用Go版本以获得最佳体验!** 👉 [快速开始](#快速开始-go版本)

---

## 主要特性

### 核心功能
- ✅ **单一可执行文件**: 无需Python运行时,无需依赖库
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

## 快速开始 (Go版本)

### 选项1: 下载预编译二进制文件 (推荐)

从[Releases页面](https://github.com/RecoveryAshes/JsFIndcrack/releases)下载对应平台的二进制文件:

```bash
# Linux
wget https://github.com/RecoveryAshes/JsFIndcrack/releases/download/v2.0/jsfindcrack-linux-amd64.tar.gz
tar -xzf jsfindcrack-linux-amd64.tar.gz
chmod +x jsfindcrack
./jsfindcrack --help

# macOS (Intel)
curl -LO https://github.com/RecoveryAshes/JsFIndcrack/releases/download/v2.0/jsfindcrack-darwin-amd64.tar.gz
tar -xzf jsfindcrack-darwin-amd64.tar.gz
chmod +x jsfindcrack
./jsfindcrack --help

# macOS (Apple Silicon)
curl -LO https://github.com/RecoveryAshes/JsFIndcrack/releases/download/v2.0/jsfindcrack-darwin-arm64.tar.gz
tar -xzf jsfindcrack-darwin-arm64.tar.gz
chmod +x jsfindcrack
./jsfindcrack --help

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/RecoveryAshes/JsFIndcrack/releases/download/v2.0/jsfindcrack-windows-amd64.zip" -OutFile jsfindcrack.zip
Expand-Archive jsfindcrack.zip
.\jsfindcrack\jsfindcrack.exe --help
```

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

---

## 从Python版本迁移

### 参数映射

| Python | Go |
|--------|-----|
| `python main.py -u URL` | `./jsfindcrack -u URL` |
| `--depth N` | `-d N` |
| `--threads N` | `--threads N` |
| `--mode static` | `--mode static` |
| `--headless` | `--headless` |
| `--url-file FILE` | `-f FILE` |
| `--batch-delay N` | `--batch-delay N` |

### 输出目录结构 (100%兼容)

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

Go版本使用`configs/headers.yaml`用于HTTP头配置:

```yaml
headers:
  User-Agent: "JsFIndcrack/2.0"
  Accept-Language: "zh-CN,zh;q=0.9"
```

---

## 快速开始 (Python版本 - 已废弃)

**注意**: Python版本已停止维护,强烈建议使用Go版本。

如需使用Python版本(仅供参考):

```bash
# 查看Python版本分支
git checkout python-legacy

# 手动安装
pip install -r requirements.txt
npm install -g webcrack
playwright install
```

### 基本使用

#### 单个网站爬取

```bash
# 爬取单个网站（默认模式：静态+动态）
python main.py -u https://example.com

# 仅静态爬取
python main.py -u https://example.com --mode static

# 仅动态爬取
python main.py -u https://example.com --mode dynamic

# 自定义参数
python main.py -u https://example.com -d 3 -w 5 -t 4 --playwright-tabs 6

# 启用相似度检测和去重
python main.py -u https://example.com --similarity --similarity-threshold 0.8
```

#### 批量网站爬取

```bash
# 从文件批量爬取网站
python main.py -f urls.txt

# 批量爬取，遇到错误继续处理下一个URL
python main.py -f urls.txt --continue-on-error

# 批量爬取，设置URL之间的延迟时间
python main.py -f urls.txt --batch-delay 2 --continue-on-error

# 批量爬取，自定义参数
python main.py -f urls.txt -d 2 -t 4 --batch-delay 1 --continue-on-error --mode static

# 批量爬取，启用相似度检测
python main.py -f urls.txt --continue-on-error --similarity --similarity-threshold 0.8
```

#### URL文件格式

创建一个文本文件（如 `urls.txt`），每行一个URL：

```
# 这是注释行，会被忽略
https://example1.com
https://example2.com
https://example3.com

# 空行也会被忽略
https://example4.com
```

## 项目结构 (Go版本)

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
│   │   └── dynamic.go        # 动态爬取(Rod)
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
├── scripts/
│   ├── build.sh              # 交叉编译脚本
│   └── release.sh            # 发布打包脚本
├── go.mod                    # Go模块定义
├── go.sum                    # 依赖锁定
├── Makefile                  # 构建任务
└── README.md                 # 本文件
```

---

## 构建和开发 (Go版本)

### 本地开发

```bash
# 安装依赖
go mod download

# 运行测试
make test
# 或
go test ./...

# 运行E2E测试
./tests/e2e/single_url_test.sh
./tests/e2e/batch_crawl_test.sh

# 格式化代码
gofmt -w .

# 代码检查
go vet ./...
```

### 交叉编译

```bash
# 构建所有平台
./scripts/build.sh

# 输出位于 dist/ 目录:
# - jsfindcrack-linux-amd64
# - jsfindcrack-darwin-amd64
# - jsfindcrack-darwin-arm64
# - jsfindcrack-windows-amd64.exe
```

### 创建发布包

```bash
# 生成包含文档的完整发布包
./scripts/release.sh

# 输出:
# - dist/jsfindcrack-VERSION-linux-amd64-full.tar.gz
# - dist/jsfindcrack-VERSION-darwin-amd64-full.tar.gz
# - dist/jsfindcrack-VERSION-darwin-arm64-full.tar.gz
# - dist/jsfindcrack-VERSION-windows-amd64-full.zip
```

---

## 使用示例 (Go版本)

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

## 性能基准 (Go版本)

**测试环境**: macOS M1, 16GB RAM

| 场景 | 文件数 | 模式 | 并发 | 耗时 | 内存 |
|------|--------|------|------|------|------|
| 小型网站 | ~20 | static | 2 | 15秒 | 60MB |
| 中型SPA | ~50 | dynamic | 4 | 45秒 | 250MB |
| 大型网站 | ~200 | all | 8 | 3分钟 | 500MB |
| 批量10个URL | ~150 | static | 4 | 2分钟 | 200MB |
| 批量100个URL | ~800 | static | 8 | 18分钟 | 600MB |

**性能提升** (对比Python版本):
- 批量爬取: 30% 更快
- 内存占用: 40% 更低
- 反混淆: 50% 更快
- 启动时间: <1秒 (Python版本~2秒)

---

## 命令行参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--url` | `-u` | 必需* | 目标网站URL（与--url-file互斥） |
| `--url-file` | `-f` | 必需* | URL列表文件路径（与--url互斥） |
| `--batch-delay` | - | 0 | 批量模式下URL之间的延迟时间(秒) |
| `--continue-on-error` | - | False | 批量模式下遇到错误时继续处理下一个URL |
| `--depth` | `-d` | 2 | 爬取深度 |
| `--wait` | `-w` | 3 | 页面等待时间(秒) |
| `--threads` | `-t` | 2 | 静态爬取并行线程数 |
| `--playwright-tabs` | - | 4 | Playwright同时打开的标签页数量 |
| `--headless` | - | True | Playwright无头模式运行 |
| `--no-headless` | - | - | Playwright有头模式运行 |
| `--mode` | - | all | 爬取模式: static/dynamic/all |
| `--resume` | `-r` | - | 从检查点恢复 |
| `--similarity` | - | True | 启用智能相似度检测和去重 |
| `--similarity-threshold` | - | 0.8 | 相似度阈值(0.0-1.0) |
| `--similarity-workers` | - | CPU核心数 | 相似度分析并行工作线程数 |

*注：`--url` 和 `--url-file` 参数互斥，必须指定其中一个

## 接口

### 基本用法

```python
from src.core.js_crawler import JSCrawler

# 创建爬虫实例
crawler = JSCrawler("https://example.com")

# 执行爬取
result = crawler.crawl()

# 查看结果
print(f"静态JS文件: {result['static']['downloaded']} 个")
print(f"动态JS文件: {result['dynamic']['downloaded']} 个")
print(f"Source Map文件: {result['static']['map_files']} 个")
print(f"反混淆文件: {result['deobfuscation']['processed_files']} 个")
```

### 高级配置

```python
# 自定义参数爬取
result = crawler.crawl(
    depth=3,                    # 爬取深度
    wait_time=5,               # 页面等待时间
    max_workers=4,             # 并行线程数
    playwright_tabs=6,         # Playwright标签页数
    headless=True,             # 无头模式
    mode='all',                # 爬取模式
    resume=False,              # 是否恢复
    similarity_enabled=True,   # 启用相似度检测
    similarity_threshold=0.8   # 相似度阈值
)
```

### 批量处理

#### 使用BatchJSCrawler类

```python
from src.core.js_crawler import BatchJSCrawler

# 从文件加载URL列表
urls_file = "urls.txt"
batch_crawler = BatchJSCrawler()

# 执行批量爬取
result = batch_crawler.crawl_batch(
    urls_file=urls_file,
    depth=2,
    wait_time=3,
    max_workers=2,
    playwright_tabs=4,
    headless=True,
    mode='all',
    batch_delay=1,           # URL之间延迟1秒
    continue_on_error=True   # 遇到错误继续处理
)

# 查看批量处理结果
print(f"总URL数量: {result['total_urls']}")
print(f"成功处理: {result['successful_urls']}")
print(f"失败数量: {result['failed_urls']}")
print(f"总文件数: {result['total_files']}")
```

#### 手动批量处理

```python
from src.core.js_crawler import JSCrawler

urls = [
    "https://site1.com",
    "https://site2.com", 
    "https://site3.com"
]

results = []
for i, url in enumerate(urls, 1):
    print(f"处理第 {i}/{len(urls)} 个URL: {url}")
    
    try:
        crawler = JSCrawler(url)
        result = crawler.crawl()
        results.append({
            'url': url,
            'success': True,
            'total_files': result['total_files']
        })
        print(f"✅ {url}: 完成，共处理 {result['total_files']} 个文件")
    except Exception as e:
        results.append({
            'url': url,
            'success': False,
            'error': str(e)
        })
        print(f"❌ {url}: 失败 - {e}")
    
    # 添加延迟
    import time
    time.sleep(1)

# 统计结果
successful = sum(1 for r in results if r['success'])
total_files = sum(r.get('total_files', 0) for r in results if r['success'])
print(f"\n批量处理完成: {successful}/{len(urls)} 成功，共获得 {total_files} 个文件")
```

## 支持的文件类型

### JavaScript文件
- `.js` - 标准JavaScript文件
- `.mjs` - ES6模块文件
- `.jsx` - React JSX文件

### Source Map文件
- `.map` - 标准Source Map文件
- `.js.map` - JavaScript Source Map文件

## 工作流程

1. **静态爬取阶段**
   - 解析HTML页面，提取script标签
   - 发现JavaScript和Source Map文件链接
   - 多线程并行下载文件
   - 自动去重和文件验证

2. **动态爬取阶段**
   - 启动浏览器（Selenium/Playwright）
   - 监控网络请求，捕获动态加载的JS文件
   - 执行页面交互，触发更多资源加载
   - 跨模式去重，避免重复下载

3. **相似度检测与去重**
   - 基于文件内容计算相似度
   - 智能识别重复和相似文件
   - 生成去重报告和统计信息
   - 支持并行处理提升效率

4. **反混淆处理**
   - 自动检测混淆的JavaScript文件
   - 使用webcrack工具进行反混淆
   - 保持原始文件结构和命名

5. **报告生成**
   - 生成详细的爬取统计报告
   - 记录成功/失败的文件信息
   - 提供文件大小和类型统计
   - 分类日志管理（错误日志、调试日志）

## 输出目录结构

```
output/
└── example.com/
    ├── encode/               # 原始下载文件
    │   ├── js/              # JavaScript文件
    │   └── maps/            # Source Map文件
    ├── decode/              # 反混淆后的文件
    ├── checkpoints/         # 检查点文件
    │   └── crawler_checkpoint.json
    ├── similarity_analysis_[timestamp]/ # 相似度分析结果
    │   ├── duplicate_groups.json
    │   ├── similarity_matrix.json
    │   └── deduplication_report.json
    ├── crawl_report.json    # 爬取报告
    ├── crawl_summary.json   # 爬取摘要
    ├── success_files.json   # 成功文件列表
    └── failed_files.json    # 失败文件列表

logs/                        # 全局日志目录
├── js_crawler.log          # 主日志文件
└── js_crawler_error.log    # 错误日志文件
```

## 配置说明

主要配置项位于 `src/core/config.py`：

```python
# 网络配置
REQUEST_TIMEOUT = 30          # 请求超时时间
MAX_RETRIES = 3              # 最大重试次数
MAX_FILE_SIZE = 50 * 1024 * 1024  # 最大文件大小(50MB)

# 浏览器配置
BROWSER_ENGINE = "playwright"  # 浏览器引擎
USE_EMBEDDED_BROWSER = True   # 使用内嵌浏览器

# 文件类型配置
SUPPORTED_JS_EXTENSIONS = ['.js', '.mjs', '.jsx']
SUPPORTED_MAP_EXTENSIONS = ['.map', '.js.map']

# 相似度检测配置
SIMILARITY_THRESHOLD = 0.8    # 默认相似度阈值
MIN_FILE_SIZE_FOR_SIMILARITY = 1024  # 最小文件大小
```

### 调试模式

```bash
# 启用详细日志
export LOG_LEVEL=DEBUG
python main.py -u https://example.com

# 查看日志文件
tail -f logs/js_crawler.log
tail -f logs/js_crawler_error.log
```

## 性能优化

- **并行处理**: 支持多线程下载和相似度分析
- **智能去重**: 避免重复下载相同文件
- **断点续爬**: 支持中断恢复，节省时间
- **内存优化**: 大文件流式处理，降低内存占用
- **缓存机制**: 文件哈希缓存，提升去重效率


---

## Python版本参考

原Python版本代码仍在仓库中保留(用于对比和迁移参考)。如需使用Python版本,请参考:

```bash
# 查看Python版本分支
git checkout python-legacy

# 安装Python版本
pip install -r requirements.txt
playwright install

# 运行Python版本
python main.py -u https://example.com
```

**注意**: 新功能和优化仅在Go版本中提供,推荐使用Go版本。

---

## 故障排查

### Go版本常见问题

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

## 技术栈 (Go版本)

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

### v2.1 (计划中)
- [ ] Web UI界面
- [ ] 实时进度WebSocket推送
- [ ] 插件系统
- [ ] 更多反混淆引擎支持

### v2.2 (计划中)
- [ ] 分布式爬取
- [ ] Redis缓存支持
- [ ] Docker镜像
- [ ] Kubernetes部署

---

## 许可证

本项目采用MIT许可证 - 查看[LICENSE](LICENSE)文件了解详情。

---

## 致谢

### Go版本
- [Colly](https://github.com/gocolly/colly) - 静态爬取引擎
- [Rod](https://github.com/go-rod/rod) - 浏览器自动化
- [Cobra](https://github.com/spf13/cobra) - CLI框架
- [Zerolog](https://github.com/rs/zerolog) - 日志库
- [webcrack](https://github.com/j4k0xb/webcrack) - JavaScript反混淆

### Python版本 (Legacy)
- [Playwright](https://playwright.dev/) - 浏览器自动化
- [BeautifulSoup](https://www.crummy.com/software/BeautifulSoup/) - HTML解析

---

## 联系方式

- **项目主页**: https://github.com/RecoveryAshes/JsFIndcrack
- **问题反馈**: [Issues](https://github.com/RecoveryAshes/JsFIndcrack/issues)
- **功能请求**: [Discussions](https://github.com/RecoveryAshes/JsFIndcrack/discussions)

---

**如果这个项目对你有帮助,请给它一个星标!** ⭐

**Go重写版v2.0 - 更快、更轻、更强大!** 🚀