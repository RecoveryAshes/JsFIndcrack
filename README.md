# JavaScript爬取和反混淆工具 (JsFIndcrack)

[![Python Version](https://img.shields.io/badge/python-3.7+-blue.svg)](https://python.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

一个功能强大的JavaScript文件爬取和反混淆工具，支持静态和动态爬取，具备断点续爬、并发处理和智能反混淆等高级功能。

## ✨ 主要特性

- 🕷️ **多模式爬取**: 支持静态HTML解析和动态JavaScript执行两种爬取模式
- 🔄 **断点续爬**: 支持中断后从检查点恢复，避免重复工作
- ⚡ **并发处理**: 多线程并行下载，显著提升爬取效率
- 🔓 **智能反混淆**: 集成webcrack工具，自动识别和反混淆JavaScript代码
- 🎯 **智能过滤**: 自动去重、文件类型检测和大小限制
- 📊 **详细统计**: 实时进度显示和完整的爬取报告
- 🛡️ **反爬虫检测**: 智能识别反爬虫机制并自动切换策略
- 🌐 **多浏览器支持**: 支持Selenium和Playwright两种浏览器引擎

## 📁 项目结构

```
JsFIndcrack/
├── src/                    # 源代码目录
│   ├── core/              # 核心模块
│   │   ├── js_crawler.py  # 主爬虫类
│   │   ├── config.py      # 配置文件
│   │   └── deobfuscator.py # 反混淆模块
│   ├── crawlers/          # 爬虫实现
│   │   ├── static_crawler.py    # 静态爬虫
│   │   ├── dynamic_crawler.py   # 动态爬虫(Selenium)
│   │   └── playwright_crawler.py # Playwright爬虫
│   └── utils/             # 工具模块
│       ├── logger.py      # 日志系统
│       └── utils.py       # 工具函数
├── examples/              # 使用示例
├── docs/                  # 文档
├── tests/                 # 测试文件
├── main.py               # 主入口文件
├── requirements.txt      # 依赖列表
└── install.sh           # 安装脚本
```

## 🚀 快速开始

### 安装依赖

```bash
# 克隆项目
git clone <repository-url>
cd JsFIndcrack

# 创建虚拟环境
python -m venv .venv
source .venv/bin/activate  # Linux/Mac
# 或
.venv\Scripts\activate     # Windows

# 安装依赖
pip install -r requirements.txt

# 或使用安装脚本
chmod +x install.sh
./install.sh
```

### 基本使用

```bash
# 基本爬取
python main.py https://example.com

# 指定深度和并发数
python main.py https://example.com --depth 3 --workers 4

# 启用动态爬取
python main.py https://example.com --dynamic --wait-time 5

# 从检查点恢复
python main.py https://example.com --resume
```

### 高级用法

```bash
# 完整参数示例
python main.py https://example.com \
    --depth 2 \
    --workers 4 \
    --dynamic \
    --wait-time 3 \
    --output-dir ./custom_output \
    --resume \
    --force-dynamic
```

## 📖 详细说明

### 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `url` | 目标网站URL | 必需 |
| `--depth` | 爬取深度 | 1 |
| `--workers` | 并发工作线程数 | 2 |
| `--dynamic` | 启用动态爬取 | False |
| `--wait-time` | 动态爬取等待时间(秒) | 3 |
| `--output-dir` | 自定义输出目录 | auto |
| `--resume` | 从检查点恢复 | False |
| `--force-dynamic` | 强制执行动态爬取 | False |

### 工作流程

1. **初始化**: 创建输出目录，设置日志系统
2. **静态爬取**: 分析HTML页面，提取JavaScript文件链接
3. **反爬虫检测**: 检查是否遇到反爬虫机制
4. **动态爬取**: 如需要，使用浏览器引擎获取动态加载的JS文件
5. **文件下载**: 并发下载所有发现的JavaScript文件
6. **反混淆处理**: 使用webcrack工具处理混淆的代码
7. **生成报告**: 输出详细的爬取统计信息

### 输出目录结构

```
output/
└── example.com/
    ├── encode/           # 原始JS文件
    │   ├── script1.js
    │   └── script2.min.js
    ├── decode/           # 反混淆后的文件
    │   ├── script1.js
    │   └── script2.js
    └── checkpoint.json   # 检查点文件
```

## ⚙️ 配置说明

主要配置项在 `src/core/config.py` 中：

```python
# 网络配置
REQUEST_TIMEOUT = 30        # 请求超时时间
MAX_RETRIES = 3            # 最大重试次数
MAX_FILE_SIZE = 50 * 1024 * 1024  # 最大文件大小(50MB)

# 浏览器配置
BROWSER_ENGINE = "selenium"  # 浏览器引擎: selenium/playwright
HEADLESS_MODE = True        # 无头模式
PAGE_LOAD_TIMEOUT = 30      # 页面加载超时

# 反混淆配置
WEBCRACK_COMMAND = "webcrack"  # webcrack命令
DEOBFUSCATION_TIMEOUT = 300    # 反混淆超时时间
```

## 🔧 依赖要求

### Python包依赖

- `requests` - HTTP请求库
- `beautifulsoup4` - HTML解析
- `selenium` - 浏览器自动化
- `playwright` - 现代浏览器自动化(可选)
- `tqdm` - 进度条显示
- `colorama` - 彩色输出
- `webdriver-manager` - 浏览器驱动管理

### 外部工具

- **webcrack**: JavaScript反混淆工具
  ```bash
  npm install -g webcrack
  ```

- **浏览器驱动**: Chrome/Firefox驱动(自动管理)

## 📊 使用示例

### Python API使用

```python
from src.core.js_crawler import JSCrawler

# 创建爬虫实例
crawler = JSCrawler("https://example.com")

# 执行爬取
result = crawler.crawl()

# 查看结果
print(f"静态JS文件: {result['static']['downloaded']} 个")
print(f"动态JS文件: {result['dynamic']['downloaded']} 个")
print(f"反混淆文件: {result['deobfuscation']['processed_files']} 个")
```

### 批量处理

```python
urls = [
    "https://site1.com",
    "https://site2.com", 
    "https://site3.com"
]

for url in urls:
    crawler = JSCrawler(url)
    result = crawler.crawl()
    print(f"{url}: 完成，共处理 {result['total_files']} 个文件")
```

## 🐛 故障排除

### 常见问题

1. **webcrack未找到**
   ```bash
   npm install -g webcrack
   # 或设置WEBCRACK_COMMAND环境变量
   ```

2. **浏览器驱动问题**
   - 工具会自动下载和管理驱动
   - 如有问题，请检查网络连接

3. **权限错误**
   ```bash
   chmod +x install.sh
   # 确保有写入输出目录的权限
   ```

4. **内存不足**
   - 减少并发工作线程数
   - 设置更小的MAX_FILE_SIZE

### 调试模式

```bash
# 启用详细日志
export LOG_LEVEL=DEBUG
python main.py https://example.com
```

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [webcrack](https://github.com/j4k0xb/webcrack) - JavaScript反混淆工具
- [Selenium](https://selenium.dev/) - 浏览器自动化框架
- [Playwright](https://playwright.dev/) - 现代浏览器自动化

## 📞 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 Issue
- 发送邮件至: [your-email@example.com]
- 项目主页: [project-homepage]

---

⭐ 如果这个项目对你有帮助，请给它一个星标！