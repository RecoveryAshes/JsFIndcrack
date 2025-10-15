# -*- mode: python ; coding: utf-8 -*-
import os
import sys
from pathlib import Path

# 获取Playwright浏览器路径
def get_playwright_browser_paths():
    """获取Playwright浏览器的安装路径"""
    browser_paths = []
    try:
        # 获取Playwright缓存目录
        playwright_cache = Path.home() / "Library" / "Caches" / "ms-playwright"
        
        # 查找chromium浏览器
        chromium_path = playwright_cache / "chromium-1187" / "chrome-mac"
        if chromium_path.exists():
            browser_paths.append((str(chromium_path), 'playwright_browsers/chromium-1187/chrome-mac'))
            print(f"找到Chromium浏览器: {chromium_path}")
        
        # 查找chromium_headless_shell浏览器
        headless_path = playwright_cache / "chromium_headless_shell-1187" / "chrome-mac"
        if headless_path.exists():
            browser_paths.append((str(headless_path), 'playwright_browsers/chromium_headless_shell-1187/chrome-mac'))
            print(f"找到Chromium Headless Shell: {headless_path}")
            
    except Exception as e:
        print(f"无法获取浏览器路径: {e}")
    
    return browser_paths

# 获取浏览器路径
browser_paths = get_playwright_browser_paths()

# 数据文件列表
datas = [
    ('src', 'src'),  # 包含源代码
]

# 添加浏览器文件到数据文件中（作为数据文件处理，避免代码签名问题）
for browser_path, target_path in browser_paths:
    if os.path.exists(browser_path):
        # 将整个浏览器目录作为数据文件添加，保持文件权限
        datas.append((browser_path, target_path))
        print(f"添加浏览器文件: {browser_path} -> {target_path}")

if not browser_paths:
    print("警告: 未找到Playwright浏览器，将不包含在打包中")

# 自定义二进制文件过滤函数，排除浏览器可执行文件的签名检查
def filter_binaries(binaries):
    """过滤二进制文件，排除浏览器相关文件"""
    filtered = []
    for binary in binaries:
        # 跳过浏览器相关的可执行文件
        if 'playwright_browsers' not in binary[1] and 'Chromium' not in binary[0]:
            filtered.append(binary)
        else:
            print(f"跳过浏览器二进制文件: {binary[0]}")
    return filtered

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

# 过滤二进制文件，避免浏览器文件的代码签名问题
a.binaries = filter_binaries(a.binaries)

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
    upx=False,  # 禁用UPX压缩，避免浏览器文件压缩问题
    upx_exclude=[],
    runtime_tmpdir=None,
    console=True,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=None,
    codesign_identity=None,  # 禁用代码签名
    entitlements_file=None,
)