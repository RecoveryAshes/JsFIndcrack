package utils

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ReadURLsFromFile 从文件中读取URL列表
func ReadURLsFromFile(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("打开URL文件失败: %w", err)
	}
	defer file.Close()

	urls := make([]string, 0)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 验证URL格式
		if err := ValidateURL(line); err != nil {
			Warnf("跳过无效URL (行 %d): %s - %v", lineNum, line, err)
			continue
		}

		urls = append(urls, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取URL文件失败: %w", err)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("URL文件中没有有效的URL")
	}

	Infof("从文件加载了 %d 个URL", len(urls))
	return urls, nil
}

// ValidateURL 验证URL格式
func ValidateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL格式无效: %w", err)
	}

	if parsed.Scheme == "" {
		return fmt.Errorf("URL缺少协议(http/https)")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL协议必须是http或https")
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL缺少主机名")
	}

	return nil
}
