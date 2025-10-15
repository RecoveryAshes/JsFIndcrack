"""
核心模块
包含主要的爬取器、配置和反混淆功能
"""

# 延迟导入，避免循环导入
def get_js_crawler():
    from .js_crawler import JSCrawler
    return JSCrawler

def get_js_crawler_manager():
    from .js_crawler import JSCrawlerManager
    return JSCrawlerManager

def get_js_deobfuscator():
    from .deobfuscator import JSDeobfuscator
    return JSDeobfuscator

# 导出配置
from .config import *

__all__ = ['get_js_crawler', 'get_js_crawler_manager', 'get_js_deobfuscator']