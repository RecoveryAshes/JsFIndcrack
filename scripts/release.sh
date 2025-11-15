#!/bin/bash
# 创建完整的发布包
# 包含二进制文件和所有文档

set -e

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
RELEASE_DIR="dist/release-${VERSION}"
DOCS=(INSTALL.md USAGE.md RELEASE_NOTES.md)

echo "========================================"
echo "  创建发布包: ${VERSION}"
echo "========================================"

# 创建临时发布目录
mkdir -p "${RELEASE_DIR}"

# 复制文档到发布目录
echo "📄 复制文档文件..."
for doc in "${DOCS[@]}"; do
    if [ -f "dist/${doc}" ]; then
        cp "dist/${doc}" "${RELEASE_DIR}/"
        echo "  ✓ ${doc}"
    fi
done

# 复制主README(如果存在)
if [ -f "README.md" ]; then
    cp README.md "${RELEASE_DIR}/"
    echo "  ✓ README.md"
fi

# 创建各平台完整发布包
echo ""
echo "📦 创建平台发布包..."

# Linux
LINUX_RELEASE="${RELEASE_DIR}/jsfindcrack-${VERSION}-linux-amd64"
mkdir -p "${LINUX_RELEASE}"
cp dist/jsfindcrack-linux-amd64 "${LINUX_RELEASE}/jsfindcrack"
for doc in "${DOCS[@]}"; do
    [ -f "dist/${doc}" ] && cp "dist/${doc}" "${LINUX_RELEASE}/"
done
[ -f "README.md" ] && cp README.md "${LINUX_RELEASE}/"
cd "${RELEASE_DIR}" && tar -czf "../jsfindcrack-${VERSION}-linux-amd64-full.tar.gz" "$(basename ${LINUX_RELEASE})" && cd ../..
echo "  ✓ jsfindcrack-${VERSION}-linux-amd64-full.tar.gz"

# macOS AMD64
MACOS_AMD64_RELEASE="${RELEASE_DIR}/jsfindcrack-${VERSION}-darwin-amd64"
mkdir -p "${MACOS_AMD64_RELEASE}"
cp dist/jsfindcrack-darwin-amd64 "${MACOS_AMD64_RELEASE}/jsfindcrack"
for doc in "${DOCS[@]}"; do
    [ -f "dist/${doc}" ] && cp "dist/${doc}" "${MACOS_AMD64_RELEASE}/"
done
[ -f "README.md" ] && cp README.md "${MACOS_AMD64_RELEASE}/"
cd "${RELEASE_DIR}" && tar -czf "../jsfindcrack-${VERSION}-darwin-amd64-full.tar.gz" "$(basename ${MACOS_AMD64_RELEASE})" && cd ../..
echo "  ✓ jsfindcrack-${VERSION}-darwin-amd64-full.tar.gz"

# macOS ARM64
MACOS_ARM64_RELEASE="${RELEASE_DIR}/jsfindcrack-${VERSION}-darwin-arm64"
mkdir -p "${MACOS_ARM64_RELEASE}"
cp dist/jsfindcrack-darwin-arm64 "${MACOS_ARM64_RELEASE}/jsfindcrack"
for doc in "${DOCS[@]}"; do
    [ -f "dist/${doc}" ] && cp "dist/${doc}" "${MACOS_ARM64_RELEASE}/"
done
[ -f "README.md" ] && cp README.md "${MACOS_ARM64_RELEASE}/"
cd "${RELEASE_DIR}" && tar -czf "../jsfindcrack-${VERSION}-darwin-arm64-full.tar.gz" "$(basename ${MACOS_ARM64_RELEASE})" && cd ../..
echo "  ✓ jsfindcrack-${VERSION}-darwin-arm64-full.tar.gz"

# Windows
WINDOWS_RELEASE="${RELEASE_DIR}/jsfindcrack-${VERSION}-windows-amd64"
mkdir -p "${WINDOWS_RELEASE}"
cp dist/jsfindcrack-windows-amd64.exe "${WINDOWS_RELEASE}/jsfindcrack.exe"
for doc in "${DOCS[@]}"; do
    [ -f "dist/${doc}" ] && cp "dist/${doc}" "${WINDOWS_RELEASE}/"
done
[ -f "README.md" ] && cp README.md "${WINDOWS_RELEASE}/"
cd "${RELEASE_DIR}" && zip -qr "../jsfindcrack-${VERSION}-windows-amd64-full.zip" "$(basename ${WINDOWS_RELEASE})" && cd ../..
echo "  ✓ jsfindcrack-${VERSION}-windows-amd64-full.zip"

echo ""
echo "✅ 发布包创建完成!"
echo ""
echo "========================================"
echo "  发布文件清单"
echo "========================================"
echo ""
echo "完整发布包(包含文档):"
ls -lh dist/*-full.tar.gz dist/*-full.zip 2>/dev/null | awk '{print "  " $5 "\t" $9}'
echo ""
echo "单独二进制文件包(仅可执行文件):"
ls -lh dist/jsfindcrack-${VERSION}-*.tar.gz dist/jsfindcrack-${VERSION}-*.zip 2>/dev/null | grep -v "full" | awk '{print "  " $5 "\t" $9}' || echo "  (无)"
echo ""
echo "临时发布目录: ${RELEASE_DIR}"
echo ""
echo "🎉 发布准备完毕! 可以分发以下文件:"
echo "  - dist/jsfindcrack-${VERSION}-linux-amd64-full.tar.gz"
echo "  - dist/jsfindcrack-${VERSION}-darwin-amd64-full.tar.gz"
echo "  - dist/jsfindcrack-${VERSION}-darwin-arm64-full.tar.gz"
echo "  - dist/jsfindcrack-${VERSION}-windows-amd64-full.zip"
echo ""
