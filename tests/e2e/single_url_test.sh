#!/bin/bash
# E2E测试: 单URL爬取完整流程

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== E2E测试: 单URL爬取完整流程 ===${NC}"

# 测试配置
TEST_URL="https://example.com"
TEST_DEPTH=1
OUTPUT_DIR="output"
BINARY="./jsfindcrack"

# 检查二进制文件是否存在
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}错误: 找不到二进制文件 $BINARY${NC}"
    echo "请先运行: go build ./cmd/jsfindcrack"
    exit 1
fi

# 清理旧的输出目录
echo -e "${YELLOW}清理旧的测试输出...${NC}"
if [ -d "$OUTPUT_DIR/example.com" ]; then
    rm -rf "$OUTPUT_DIR/example.com"
fi

# 执行爬取
echo -e "${YELLOW}开始爬取: $TEST_URL (深度: $TEST_DEPTH)${NC}"
$BINARY -u "$TEST_URL" -d "$TEST_DEPTH" --mode static --log-level info

# 验证输出结构
echo -e "${YELLOW}验证输出目录结构...${NC}"

EXPECTED_DIRS=(
    "$OUTPUT_DIR/example.com"
    "$OUTPUT_DIR/example.com/encode"
    "$OUTPUT_DIR/example.com/encode/js"
)

for dir in "${EXPECTED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓ 目录存在: $dir${NC}"
    else
        echo -e "${RED}✗ 目录缺失: $dir${NC}"
        exit 1
    fi
done

# 检查JSON报告
REPORT_FILE="$OUTPUT_DIR/example.com/reports/crawl_report.json"
if [ -f "$REPORT_FILE" ]; then
    echo -e "${GREEN}✓ 爬取报告存在: $REPORT_FILE${NC}"

    # 验证JSON格式
    if command -v jq &> /dev/null; then
        if jq empty "$REPORT_FILE" 2>/dev/null; then
            echo -e "${GREEN}✓ JSON格式有效${NC}"

            # 显示统计信息
            TOTAL_FILES=$(jq '.total_files' "$REPORT_FILE")
            DURATION=$(jq '.duration' "$REPORT_FILE")
            echo -e "${GREEN}  总文件数: $TOTAL_FILES${NC}"
            echo -e "${GREEN}  耗时: ${DURATION}秒${NC}"
        else
            echo -e "${RED}✗ JSON格式无效${NC}"
            exit 1
        fi
    else
        echo -e "${YELLOW}! 未安装jq,跳过JSON验证${NC}"
    fi
else
    echo -e "${RED}✗ 爬取报告缺失: $REPORT_FILE${NC}"
    exit 1
fi

# 检查是否下载了JS文件
JS_FILE_COUNT=$(find "$OUTPUT_DIR/example.com/encode/js" -name "*.js" 2>/dev/null | wc -l | tr -d ' ')
if [ "$JS_FILE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ 下载了 $JS_FILE_COUNT 个JavaScript文件${NC}"
else
    echo -e "${YELLOW}! 警告: 未下载任何JavaScript文件 (可能是测试URL无JS资源)${NC}"
fi

echo -e "${GREEN}=== E2E测试通过 ===${NC}"
exit 0
