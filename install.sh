#!/bin/bash

# JavaScript爬取和反混淆工具安装脚本
# JsFIndcrack Installation Script

set -e  # 遇到错误时退出

echo "🚀 开始安装 JsFIndcrack..."

# 检查Python版本
echo "检查Python版本..."
python_version=$(python3 --version 2>&1 | cut -d' ' -f2 | cut -d'.' -f1,2)
required_version="3.7"

if [ "$(printf '%s\n' "$required_version" "$python_version" | sort -V | head -n1)" != "$required_version" ]; then
    echo "错误: 需要Python 3.7或更高版本，当前版本: $python_version"
    exit 1
fi
echo "✓ Python版本检查通过: $python_version"

# 检查pip
echo "检查pip..."
if ! command -v pip3 &> /dev/null; then
    echo "错误: pip3未找到，请先安装pip"
    exit 1
fi
echo "✓ pip3可用"

# 安装Python依赖
echo "安装Python依赖..."
if [ -f "requirements.txt" ]; then
    pip3 install -r requirements.txt
    echo "✓ Python依赖安装完成"
else
    echo "警告: requirements.txt文件未找到"
fi

# 检查Node.js和npm
echo "检查Node.js和npm..."
if ! command -v node &> /dev/null; then
    echo "警告: Node.js未找到"
    echo "请访问 https://nodejs.org/ 安装Node.js"
    echo "或使用包管理器安装:"
    echo "  macOS: brew install node"
    echo "  Ubuntu: sudo apt install nodejs npm"
    echo "  CentOS: sudo yum install nodejs npm"
else
    node_version=$(node --version)
    echo "✓ Node.js版本: $node_version"
fi

if ! command -v npm &> /dev/null; then
    echo "警告: npm未找到"
else
    npm_version=$(npm --version)
    echo "✓ npm版本: $npm_version"
    
    # 安装webcrack
    echo "安装webcrack工具..."
    if npm list -g webcrack &> /dev/null; then
        echo "✓ webcrack已安装"
    else
        echo "正在安装webcrack..."
        npm install -g webcrack
        echo "✓ webcrack安装完成"
    fi
fi

# 检查Chrome浏览器
echo "检查Chrome浏览器..."
chrome_found=false

# macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    if [ -d "/Applications/Google Chrome.app" ]; then
        chrome_found=true
    fi
# Linux
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    if command -v google-chrome &> /dev/null || command -v chromium-browser &> /dev/null; then
        chrome_found=true
    fi
fi

if [ "$chrome_found" = true ]; then
    echo "✓ Chrome浏览器已安装"
else
    echo "警告: Chrome浏览器未找到"
    echo "请安装Chrome浏览器:"
    echo "  macOS: 从 https://www.google.com/chrome/ 下载"
    echo "  Ubuntu: sudo apt install google-chrome-stable"
    echo "  或者: sudo apt install chromium-browser"
fi

# 创建必要目录
echo "创建项目目录..."
mkdir -p output/{original/{static,dynamic},decrypted/{static,dynamic},logs,checkpoints}
echo "✓ 目录结构创建完成"

# 设置执行权限
echo "设置执行权限..."
chmod +x js_crawler.py
chmod +x test_crawler.py
chmod +x example_usage.py
echo "✓ 执行权限设置完成"

# 运行测试
echo "运行基本测试..."
if python3 test_crawler.py &> /dev/null; then
    echo "✓ 基本测试通过"
else
    echo "警告: 基本测试未完全通过，请检查依赖"
fi

echo "=================================="
echo "安装完成!"
echo "=================================="
echo ""
echo "使用方法:"
echo "  基本使用: python3 js_crawler.py https://example.com"
echo "  运行测试: python3 test_crawler.py"
echo "  查看示例: python3 example_usage.py"
echo "  查看帮助: python3 js_crawler.py --help"
echo ""
echo "注意事项:"
echo "  1. 确保网络连接正常"
echo "  2. 遵守目标网站的robots.txt"
echo "  3. 合理设置爬取参数避免过载"
echo ""