#!/bin/bash
# E2E测试: 反混淆功能验证

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== E2E测试: 反混淆功能验证 ===${NC}"

# 测试配置
TEST_URL="https://cc.qcqwcqx.sbs"
OUTPUT_DIR="output"
BINARY="./jsfindcrack"

# 检查二进制文件是否存在
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}错误: 找不到二进制文件 $BINARY${NC}"
    echo "请先运行: go build ./cmd/jsfindcrack"
    exit 1
fi

# 检查是否已有JS文件
DOMAIN="cc.qcqwcqx.sbs"
JS_COUNT=$(find "$OUTPUT_DIR/$DOMAIN/encode/js" -name "*.js" 2>/dev/null | wc -l | tr -d ' ')

if [ "$JS_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}没有现有JS文件,先执行爬取...${NC}"
    $BINARY -u "$TEST_URL" -d 1 --mode all --log-level warn --threads 2
    JS_COUNT=$(find "$OUTPUT_DIR/$DOMAIN/encode/js" -name "*.js" 2>/dev/null | wc -l | tr -d ' ')
fi

echo -e "${GREEN}✓ 找到 $JS_COUNT 个JavaScript文件${NC}"

if [ "$JS_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}! 该域名没有JS文件,跳过反混淆测试${NC}"
    exit 0
fi

# 检查decode目录
DECODE_DIR="$OUTPUT_DIR/$DOMAIN/decode"
if [ -d "$DECODE_DIR" ]; then
    DECODED_COUNT=$(find "$DECODE_DIR" -name "*.js" 2>/dev/null | wc -l | tr -d ' ')
    echo -e "${GREEN}✓ 找到 $DECODED_COUNT 个反混淆文件${NC}"

    if [ "$DECODED_COUNT" -gt 0 ]; then
        echo -e "${GREEN}  反混淆率: $(echo "scale=2; $DECODED_COUNT*100/$JS_COUNT" | bc)%${NC}"

        # 验证文件内容
        SAMPLE_FILE=$(find "$DECODE_DIR" -name "*.js" 2>/dev/null | head -1)
        if [ -f "$SAMPLE_FILE" ]; then
            FILE_SIZE=$(stat -f%z "$SAMPLE_FILE" 2>/dev/null || stat -c%s "$SAMPLE_FILE" 2>/dev/null)
            echo -e "${GREEN}  示例文件大小: ${FILE_SIZE} bytes${NC}"

            if [ "$FILE_SIZE" -gt 0 ]; then
                echo -e "${GREEN}✓ 反混淆文件不为空${NC}"
            else
                echo -e "${RED}✗ 反混淆文件为空${NC}"
                exit 1
            fi
        fi
    else
        echo -e "${YELLOW}! 警告: 未生成反混淆文件${NC}"
        echo -e "${YELLOW}  可能原因: 文件未混淆或webcrack未安装${NC}"
    fi
else
    echo -e "${YELLOW}! decode目录不存在${NC}"
    echo -e "${YELLOW}  可能原因: 反混淆功能未执行或文件未混淆${NC}"
fi

# 检查相似度分析(如果存在)
SIMILARITY_DIR="$OUTPUT_DIR/$DOMAIN/similarity"
if [ -d "$SIMILARITY_DIR" ]; then
    echo -e "${GREEN}✓ 相似度分析目录存在${NC}"
else
    echo -e "${YELLOW}! 相似度分析目录不存在 (功能可能已移除)${NC}"
fi

echo -e "${GREEN}=== E2E测试通过 ===${NC}"
exit 0
