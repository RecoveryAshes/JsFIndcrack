#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
JavaScript爬取和反混淆工具 - 主入口文件
"""

import sys
import os
import multiprocessing as mp
from pathlib import Path

# PyInstaller多进程保护
def is_frozen():
    """检查是否在PyInstaller打包环境中运行"""
    return getattr(sys, 'frozen', False) and hasattr(sys, '_MEIPASS')

# 添加src目录到Python路径
sys.path.insert(0, str(Path(__file__).parent / "src"))

from src.core.js_crawler import main

if __name__ == "__main__":
    # 多进程保护 - 在PyInstaller环境中完全避免设置start_method
    if not is_frozen():
        try:
            # 只在开发环境中设置start_method
            mp.set_start_method('spawn', force=True)
        except RuntimeError:
            # 如果已经设置过start_method，忽略错误
            pass
    # 在打包环境中不设置start_method，避免子进程问题
    
    main()