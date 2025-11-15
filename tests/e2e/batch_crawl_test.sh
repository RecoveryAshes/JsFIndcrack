#!/bin/bash
# E2E测试: 批量爬取10个URL

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== E2E测试: 批量爬取10个URL ===${NC}"

# 测试配置
TEST_FILE="test_urls.txt"
OUTPUT_DIR="output"
BINARY="./jsfindcrack"

# 检查二进制文件是否存在
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}错误: 找不到二进制文件 $BINARY${NC}"
    echo "请先运行: go build ./cmd/jsfindcrack"
    exit 1
fi

# 检查URL文件是否存在
if [ ! -f "$TEST_FILE" ]; then
    echo -e "${RED}错误: 找不到URL文件 $TEST_FILE${NC}"
    exit 1
fi

# 统计URL数量
URL_COUNT=$(grep -c '^[^#]' "$TEST_FILE" || echo "0")
echo -e "${YELLOW}URL文件包含 $URL_COUNT 个URL${NC}"

if [ "$URL_COUNT" -eq 0 ]; then
    echo -e "${RED}错误: URL文件为空${NC}"
    exit 1
fi

# 执行批量爬取
echo -e "${YELLOW}开始批量爬取...${NC}"
START_TIME=$(date +%s)

$BINARY -f "$TEST_FILE" --mode static --log-level warn --threads 2 --batch-delay 1 --continue-on-error

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo -e "${GREEN}批量爬取完成,耗时: ${DURATION}秒${NC}"

# 验证输出
echo -e "${YELLOW}验证输出结果...${NC}"

# 统计生成的域名目录数
DOMAIN_COUNT=$(find "$OUTPUT_DIR" -maxdepth 1 -type d ! -name "$OUTPUT_DIR" | wc -l | tr -d ' ')
echo -e "${GREEN}✓ 处理了 $DOMAIN_COUNT 个域名${NC}"

# 检查每个域名的报告文件
REPORT_COUNT=0
for domain_dir in "$OUTPUT_DIR"/*/ ; do
    if [ -d "$domain_dir" ]; then
        DOMAIN=$(basename "$domain_dir")
        REPORT="$domain_dir/reports/crawl_report.json"

        if [ -f "$REPORT" ]; then
            REPORT_COUNT=$((REPORT_COUNT + 1))
            echo -e "${GREEN}  ✓ $DOMAIN: 报告存在${NC}"
        else
            echo -e "${YELLOW}  ! $DOMAIN: 报告缺失${NC}"
        fi
    fi
done

echo -e "${GREEN}✓ 生成了 $REPORT_COUNT 个爬取报告${NC}"

# 最终验证
if [ "$REPORT_COUNT" -gt 0 ]; then
    echo -e "${GREEN}=== E2E测试通过 ===${NC}"
    echo -e "${GREEN}成功处理: $REPORT_COUNT/$URL_COUNT 个URL${NC}"
    exit 0
else
    echo -e "${RED}=== E2E测试失败 ===${NC}"
    echo -e "${RED}未生成任何爬取报告${NC}"
    exit 1
fi
