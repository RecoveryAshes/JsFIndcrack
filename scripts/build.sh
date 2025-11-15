#!/bin/bash

# 交叉编译脚本
# 为Linux, macOS, Windows平台构建二进制文件

set -e

BINARY_NAME="jsfindcrack"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

echo "==================================="
echo "  JsFIndcrack 交叉编译脚本"
echo "==================================="
echo "版本: ${VERSION}"
echo "构建时间: ${BUILD_TIME}"
echo ""

# 创建输出目录
mkdir -p dist

# Linux AMD64
echo "📦 构建 Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "dist/${BINARY_NAME}-linux-amd64" ./cmd/jsfindcrack
echo "✅ dist/${BINARY_NAME}-linux-amd64"

# macOS AMD64
echo "📦 构建 macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "dist/${BINARY_NAME}-darwin-amd64" ./cmd/jsfindcrack
echo "✅ dist/${BINARY_NAME}-darwin-amd64"

# macOS ARM64 (Apple Silicon)
echo "📦 构建 macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "dist/${BINARY_NAME}-darwin-arm64" ./cmd/jsfindcrack
echo "✅ dist/${BINARY_NAME}-darwin-arm64"

# Windows AMD64
echo "📦 构建 Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "dist/${BINARY_NAME}-windows-amd64.exe" ./cmd/jsfindcrack
echo "✅ dist/${BINARY_NAME}-windows-amd64.exe"

echo ""
echo "==================================="
echo "✅ 交叉编译完成!"
echo "==================================="
echo "构建产物位于 dist/ 目录:"
ls -lh dist/

echo ""
echo "打包发布文件..."
cd dist
tar -czf "${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz" "${BINARY_NAME}-linux-amd64"
tar -czf "${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz" "${BINARY_NAME}-darwin-amd64"
tar -czf "${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz" "${BINARY_NAME}-darwin-arm64"
zip -q "${BINARY_NAME}-${VERSION}-windows-amd64.zip" "${BINARY_NAME}-windows-amd64.exe"
cd ..

echo "✅ 打包完成!"
echo ""
echo "发布文件:"
ls -lh dist/*.tar.gz dist/*.zip 2>/dev/null || true
