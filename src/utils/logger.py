"""
日志记录工具模块
"""
import logging
import sys
from pathlib import Path
from colorama import Fore, Style, init

# 初始化colorama
init(autoreset=True)

# 日志配置常量（避免循环导入）
def get_base_dir():
    """获取基础目录"""
    # 从 src/utils 向上两级到项目根目录
    return Path(__file__).parent.parent.parent

BASE_DIR = get_base_dir()
LOGS_DIR = BASE_DIR / "logs"
LOG_LEVEL = "INFO"
LOG_FORMAT = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
LOG_FILE = LOGS_DIR / "js_crawler.log"
ERROR_LOG_FILE = LOGS_DIR / "js_crawler_error.log"

class ColoredFormatter(logging.Formatter):
    """彩色日志格式化器"""
    
    COLORS = {
        'DEBUG': Fore.CYAN,
        'INFO': Fore.GREEN,
        'WARNING': Fore.YELLOW,
        'ERROR': Fore.RED,
        'CRITICAL': Fore.MAGENTA + Style.BRIGHT,
    }

    def format(self, record):
        log_color = self.COLORS.get(record.levelname, '')
        record.levelname = f"{log_color}{record.levelname}{Style.RESET_ALL}"
        return super().format(record)

def setup_logger(name="js_crawler"):
    """设置日志记录器"""
    # 确保日志目录存在
    LOGS_DIR.mkdir(parents=True, exist_ok=True)
    
    logger = logging.getLogger(name)
    logger.setLevel(getattr(logging, LOG_LEVEL))
    
    # 避免重复添加处理器
    if logger.handlers:
        return logger
    
    # 控制台处理器
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setLevel(getattr(logging, LOG_LEVEL))  # 使用配置的日志级别
    console_formatter = ColoredFormatter(LOG_FORMAT)
    console_handler.setFormatter(console_formatter)
    
    # 文件处理器（所有日志）
    file_handler = logging.FileHandler(LOG_FILE, encoding='utf-8')
    file_handler.setLevel(logging.DEBUG)
    file_formatter = logging.Formatter(LOG_FORMAT)
    file_handler.setFormatter(file_formatter)
    
    # 错误日志处理器（只记录ERROR和CRITICAL）
    error_handler = logging.FileHandler(ERROR_LOG_FILE, encoding='utf-8')
    error_handler.setLevel(logging.ERROR)
    error_formatter = logging.Formatter(LOG_FORMAT)
    error_handler.setFormatter(error_formatter)
    
    logger.addHandler(console_handler)
    logger.addHandler(file_handler)
    logger.addHandler(error_handler)
    
    return logger

def get_logger(name=None):
    """获取日志记录器"""
    if name:
        return logging.getLogger(f"js_crawler.{name}")
    return logging.getLogger("js_crawler")