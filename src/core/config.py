"""
JavaScript文件爬取工具配置文件
"""
import os
from pathlib import Path

# 基础配置
# 获取项目根目录（从 src/core 向上两级到项目根目录）
BASE_DIR = Path(__file__).parent.parent.parent
BASE_CONFIG = {
    'base_output_dir': 'output',  # 基础输出目录
    'original_dir': 'encode',      # 原始文件目录
    'decrypted_dir': 'decode',     # 反混淆后文件目录
    'logs_dir': 'logs',
    'checkpoints_dir': 'checkpoints',
}

OUTPUT_DIR = BASE_DIR / BASE_CONFIG['base_output_dir']
ORIGINAL_DIR = OUTPUT_DIR / BASE_CONFIG['original_dir']
DECRYPTED_DIR = OUTPUT_DIR / BASE_CONFIG['decrypted_dir']
LOGS_DIR = BASE_DIR / BASE_CONFIG['logs_dir']

# 网络配置
REQUEST_TIMEOUT = 30
MAX_RETRIES = 3
DELAY_BETWEEN_REQUESTS = 1
USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

# SSL配置
VERIFY_SSL = False  # 是否验证SSL证书，设为False可忽略证书错误
SSL_WARNINGS = False  # 是否显示SSL警告

# Selenium配置
SELENIUM_TIMEOUT = 30
PAGE_LOAD_TIMEOUT = 60
IMPLICIT_WAIT = 10
HEADLESS_MODE = True

# 浏览器引擎配置
BROWSER_ENGINE = "playwright"  # "selenium" 或 "playwright"
PLAYWRIGHT_BROWSER = "chromium"  # "chromium", "firefox", "webkit"
USE_EMBEDDED_BROWSER = True  # 是否使用内置浏览器

# 文件处理配置
SUPPORTED_JS_EXTENSIONS = ['.js', '.mjs', '.jsx']
SUPPORTED_MAP_EXTENSIONS = ['.map', '.js.map']  # 支持的source map文件扩展名
SUPPORTED_FILE_EXTENSIONS = SUPPORTED_JS_EXTENSIONS + SUPPORTED_MAP_EXTENSIONS  # 所有支持的文件扩展名
MAX_FILE_SIZE = 50 * 1024 * 1024  # 50MB
ENCODING = 'utf-8'

# 日志配置
LOG_LEVEL = "INFO"
LOG_FORMAT = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
LOG_FILE = LOGS_DIR / "js_crawler.log"

# webcrack配置
WEBCRACK_COMMAND = "webcrack"
WEBCRACK_TIMEOUT = 300  # 5分钟

# 检查点配置
CHECKPOINT_FILE = BASE_DIR / "checkpoint.json"
SAVE_CHECKPOINT_INTERVAL = 10  # 每处理10个文件保存一次检查点

# 结果报告配置
REPORT_CONFIG = {
    'crawl_report': 'crawl_report.json',      # 爬取结果报告文件名
    'success_report': 'success_files.json',   # 成功文件列表
    'failed_report': 'failed_files.json',     # 失败文件列表
    'summary_report': 'crawl_summary.json',   # 爬取摘要报告
    'detailed_log': 'detailed_log.txt',       # 详细日志文件
    'error_log': 'error.log',                 # 错误日志文件
    'debug_log': 'debug.log',                 # 调试日志文件
}

def get_target_output_dir(url):
    """根据目标URL生成输出目录路径"""
    from urllib.parse import urlparse
    import re
    
    # 解析URL获取域名
    parsed_url = urlparse(url)
    domain = parsed_url.netloc or parsed_url.path
    
    # 清理域名，移除不合法的文件名字符
    clean_domain = re.sub(r'[<>:"/\\|?*]', '_', domain)
    clean_domain = re.sub(r'^www\.', '', clean_domain)  # 移除www前缀
    clean_domain = clean_domain.strip('.')
    
    # 如果域名为空，使用默认名称
    if not clean_domain:
        clean_domain = 'unknown_target'
    
    return BASE_DIR / BASE_CONFIG['base_output_dir'] / clean_domain

def get_directory_structure(target_url):
    """获取完整的目录结构"""
    target_output_dir = get_target_output_dir(target_url)
    
    return {
        'base_dir': BASE_DIR,
        'target_output_dir': target_output_dir,
        'original_dir': target_output_dir / BASE_CONFIG['original_dir'],
        'decrypted_dir': target_output_dir / BASE_CONFIG['decrypted_dir'],
        # 为了向后兼容，保留这些别名
        'static_original_dir': target_output_dir / BASE_CONFIG['original_dir'],
        'dynamic_original_dir': target_output_dir / BASE_CONFIG['original_dir'],
        'static_decrypted_dir': target_output_dir / BASE_CONFIG['decrypted_dir'],
        'dynamic_decrypted_dir': target_output_dir / BASE_CONFIG['decrypted_dir'],
        'logs_dir': target_output_dir / BASE_CONFIG['logs_dir'],
        'checkpoints_dir': target_output_dir / BASE_CONFIG['checkpoints_dir'],
    }

def ensure_directories(target_url=None):
    """确保所有必要的目录存在"""
    if target_url:
        # 为特定目标URL创建目录
        dirs = get_directory_structure(target_url)
        directories = [
            dirs['target_output_dir'],
            dirs['original_dir'],
            dirs['decrypted_dir'],
            dirs['logs_dir'],
            dirs['checkpoints_dir'],
        ]
    else:
        # 创建默认目录（向后兼容）
        directories = [
            OUTPUT_DIR,
            ORIGINAL_DIR,
            DECRYPTED_DIR,
            LOGS_DIR,
            BASE_DIR / "checkpoints"
        ]
    
    for directory in directories:
        directory.mkdir(parents=True, exist_ok=True)