#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
JavaScript爬取和反混淆工具 - 主入口文件
"""

import sys
import multiprocessing as mp
from pathlib import Path

# 添加src目录到Python路径
sys.path.insert(0, str(Path(__file__).parent / "src"))

from src.core.js_crawler import main

if __name__ == "__main__":
    try:
        # 设置多进程启动方法
        mp.set_start_method('spawn', force=True)
    except RuntimeError:
        # 如果已经设置过start_method，忽略错误
        pass
    
    main()