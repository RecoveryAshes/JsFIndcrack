#!/bin/bash
# 性能基准测试脚本
# 对比Go版本和Python版本的性能差异

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}====================================${NC}"
echo -e "${BLUE}    JsFIndcrack 性能基准测试${NC}"
echo -e "${BLUE}====================================${NC}"
echo

# 配置
GO_BINARY="./jsfindcrack"
PYTHON_SCRIPT="../python/main.py"  # 假设Python版本位置
TEST_URLS_FILE="test_urls.txt"
BENCHMARK_DIR="tests/benchmark"
RESULTS_DIR="$BENCHMARK_DIR/results"

# 创建结果目录
mkdir -p "$RESULTS_DIR"

# 检查Go二进制
if [ ! -f "$GO_BINARY" ]; then
    echo -e "${RED}错误: 找不到Go二进制文件 $GO_BINARY${NC}"
    echo "请先运行: go build ./cmd/jsfindcrack"
    exit 1
fi

# 检查URL测试文件
if [ ! -f "$TEST_URLS_FILE" ]; then
    echo -e "${YELLOW}创建测试URL文件...${NC}"
    cat > "$TEST_URLS_FILE" <<EOF
# 性能测试URL列表
https://example.com
https://example.org
https://httpbin.org
EOF
fi

URL_COUNT=$(grep -c '^[^#]' "$TEST_URLS_FILE" || echo "0")
echo -e "${YELLOW}测试URL数量: $URL_COUNT${NC}"
echo

# ==========================================
# 测试1: 单URL爬取性能
# ==========================================
echo -e "${BLUE}[测试1] 单URL爬取性能${NC}"
TEST_URL="https://example.com"

# Go版本
echo -e "${YELLOW}运行Go版本...${NC}"
GO_START=$(date +%s.%N)
$GO_BINARY -u "$TEST_URL" -d 2 --mode static --log-level error > /dev/null 2>&1 || true
GO_END=$(date +%s.%N)
GO_TIME=$(echo "$GO_END - $GO_START" | bc)
echo -e "${GREEN}Go版本耗时: ${GO_TIME}秒${NC}"

# 记录结果
echo "single_url_go,$GO_TIME" >> "$RESULTS_DIR/benchmark_$(date +%Y%m%d).csv"

echo

# ==========================================
# 测试2: 批量爬取性能
# ==========================================
echo -e "${BLUE}[测试2] 批量爬取性能 ($URL_COUNT URLs)${NC}"

# Go版本
echo -e "${YELLOW}运行Go版本批量爬取...${NC}"
GO_BATCH_START=$(date +%s.%N)
$GO_BINARY -f "$TEST_URLS_FILE" --mode static --log-level error --threads 4 > /dev/null 2>&1 || true
GO_BATCH_END=$(date +%s.%N)
GO_BATCH_TIME=$(echo "$GO_BATCH_END - $GO_BATCH_START" | bc)
echo -e "${GREEN}Go版本批量爬取耗时: ${GO_BATCH_TIME}秒${NC}"

# 记录结果
echo "batch_crawl_go,$GO_BATCH_TIME" >> "$RESULTS_DIR/benchmark_$(date +%Y%m%d).csv"

echo

# ==========================================
# 测试3: 内存占用 (通过time命令)
# ==========================================
echo -e "${BLUE}[测试3] 内存占用测试${NC}"

echo -e "${YELLOW}运行Go版本(监控内存)...${NC}"
/usr/bin/time -l $GO_BINARY -f "$TEST_URLS_FILE" --mode static --log-level error --threads 2 2>&1 | grep "maximum resident set size" || true

echo

# ==========================================
# 测试4: 并发性能测试
# ==========================================
echo -e "${BLUE}[测试4] 并发性能对比${NC}"

for THREADS in 2 4 8; do
    echo -e "${YELLOW}测试并发数: $THREADS${NC}"

    START=$(date +%s.%N)
    $GO_BINARY -f "$TEST_URLS_FILE" --mode static --log-level error --threads $THREADS > /dev/null 2>&1 || true
    END=$(date +%s.%N)
    TIME=$(echo "$END - $START" | bc)

    echo -e "${GREEN}  并发$THREADS 耗时: ${TIME}秒${NC}"
    echo "concurrency_${THREADS}_go,$TIME" >> "$RESULTS_DIR/benchmark_$(date +%Y%m%d).csv"
done

echo

# ==========================================
# 总结
# ==========================================
echo -e "${BLUE}====================================${NC}"
echo -e "${BLUE}         基准测试完成${NC}"
echo -e "${BLUE}====================================${NC}"
echo
echo -e "${GREEN}结果已保存到: $RESULTS_DIR/benchmark_$(date +%Y%m%d).csv${NC}"
echo
echo -e "${YELLOW}性能提升建议:${NC}"
echo "  - 批量爬取建议使用 --threads 4 或更高"
echo "  - 大规模爬取可启用动态模式获取更多JS文件"
echo "  - 内存受限环境建议降低并发数"
echo

exit 0
