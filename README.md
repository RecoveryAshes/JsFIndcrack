# JsFIndcrack - JavaScript文件爬取和反混淆工具

一个功能强大的JavaScript文件爬取和反混淆工具，支持静态和动态爬取，具备断点续爬、并发处理、智能去重和自动反混淆等高级功能。

## 主要特性

- **批量扫描**: 支持从文件读取URL列表进行批量爬取，提高工作效率
- **多模式爬取**: 支持静态HTML解析和动态JavaScript执行两种爬取模式
- **Source Map支持**: 自动识别和下载JavaScript Source Map文件(.map, .js.map)
- **断点续爬**: 支持中断后从检查点恢复，避免重复工作
- **并发处理**: 多线程并行下载，显著提升爬取效率
- **智能反混淆**: 集成webcrack工具，自动识别和反混淆JavaScript代码
- **智能去重**: 基于文件内容的相似度检测和去重功能
- **错误容错**: 批量模式下支持遇到错误继续处理下一个URL
- **详细统计**: 实时进度显示和完整的爬取报告
- **反爬虫检测**: 智能识别反爬虫机制并自动切换策略
- **多浏览器支持**: 支持Selenium和Playwright两种浏览器引擎
- **日志系统**: 完整的日志记录和分类管理

## 快速开始

### 安装依赖

```bash
# 克隆项目
git clone https://github.com/RecoveryAshes/JsFIndcrack.git
cd JsFIndcrack

# 运行安装脚本（推荐）
chmod +x install.sh
./install.sh

# 或手动安装
pip install -r requirements.txt

# 安装webcrack（用于反混淆）
npm install -g webcrack

# 安装Playwright浏览器（必需，用于动态爬取）
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

## 项目结构

```
JsFIndcrack/
├── main.py                    # 程序入口
├── requirements.txt           # Python依赖
├── install.sh                # 安装脚本
├── README.md                 # 项目说明
├── LICENSE                   # 许可证文件
├── src/                      # 源代码目录
│   ├── core/                 # 核心模块
│   │   ├── config.py         # 配置文件
│   │   ├── js_crawler.py     # 主爬取器
│   │   └── deobfuscator.py   # 反混淆器
│   ├── crawlers/             # 爬取器模块
│   │   ├── static_crawler.py # 静态爬取器
│   │   ├── dynamic_crawler.py# 动态爬取器
│   │   └── playwright_crawler.py # Playwright爬取器
│   └── utils/                # 工具模块
│       ├── logger.py         # 日志系统
│       ├── utils.py          # 通用工具
│       ├── report_generator.py # 报告生成器
│       ├── similarity_analyzer.py # 相似度分析器
│       └── parallel_similarity_analyzer.py # 并行相似度处理器
├── logs/                     # 全局日志目录
│   ├── js_crawler.log       # 主日志文件
│   └── js_crawler_error.log # 错误日志文件
└── output/                   # 输出目录
    └── [domain]/            # 按域名分类的输出
        ├── encode/          # 原始文件
        ├── decode/          # 反混淆文件
        ├── checkpoints/     # 检查点文件
        └── similarity_analysis_[timestamp]/ # 相似度分析结果
```

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

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 致谢

- [webcrack](https://github.com/j4k0xb/webcrack) - JavaScript反混淆工具
- [Playwright](https://playwright.dev/) - 现代浏览器自动化
- [Selenium](https://selenium.dev/) - 浏览器自动化框架
- [BeautifulSoup](https://www.crummy.com/software/BeautifulSoup/) - HTML解析库

## 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 [Issue](https://github.com/RecoveryAshes/JsFIndcrack/issues)

---

如果这个项目对你有帮助，请给它一个星标！