# -*- mode: python ; coding: utf-8 -*-
import os
import sys
from pathlib import Path

# 获取Playwright浏览器路径
def get_playwright_browser_path():
    """获取Playwright浏览器的安装路径"""
    try:
        from playwright.sync_api import sync_playwright
        with sync_playwright() as p:
            browser_path = p.chromium.executable_path
            # 返回浏览器目录的根目录
            browser_root = Path(browser_path).parent.parent.parent  # 回到chrome-mac目录
            return str(browser_root)
    except Exception as e:
        print(f"无法获取浏览器路径: {e}")
        return None

# 获取浏览器路径
browser_path = get_playwright_browser_path()
print(f"浏览器路径: {browser_path}")

# 数据文件列表
datas = [
    ('src', 'src'),  # 包含源代码
]

# 如果找到浏览器路径，添加到数据文件中
if browser_path and os.path.exists(browser_path):
    datas.append((browser_path, 'playwright_browsers/chromium-1187/chrome-mac'))
    print(f"添加浏览器文件: {browser_path}")
else:
    print("警告: 未找到Playwright浏览器，将不包含在打包中")

# 隐藏导入
hiddenimports = [
    'selenium',
    'playwright',
    'playwright.async_api',
    'playwright.sync_api',
    'requests',
    'beautifulsoup4',
    'lxml',
    'numpy',
    'scikit-learn',
    'concurrent.futures',
    'asyncio',
    'pathlib',
    'urllib.parse',
    'hashlib',
    'json',
    'time',
    'logging',
    'tqdm',
    'tqdm.asyncio',
]

block_cipher = None

a = Analysis(
    ['main.py'],
    pathex=[],
    binaries=[],
    datas=datas,
    hiddenimports=hiddenimports,
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name='jsfindcrack-macos-arm64',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
)