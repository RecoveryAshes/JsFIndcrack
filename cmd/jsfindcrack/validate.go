package main

import (
	"fmt"
	"net/url"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
)

// ValidateURL 验证URL格式
func ValidateURL(urlStr string) error {
	return models.ValidateURL(urlStr)
}

// ValidateFlags 验证命令行标志
func ValidateFlags(
	targetURL string,
	depth int,
	waitTime int,
	maxWorkers int,
	playwrightTabs int,
	similarityThreshold float64,
	mode string,
) error {
	// 验证URL
	if targetURL != "" {
		if err := ValidateURL(targetURL); err != nil {
			return fmt.Errorf("无效的目标URL: %w", err)
		}
	}

	// 验证深度
	if depth < 1 || depth > 10 {
		return fmt.Errorf("爬取深度必须在1-10之间,当前值: %d", depth)
	}

	// 验证等待时间
	if waitTime < 0 || waitTime > 60 {
		return fmt.Errorf("等待时间必须在0-60秒之间,当前值: %d", waitTime)
	}

	// 验证并发数
	if maxWorkers < 1 || maxWorkers > 100 {
		return fmt.Errorf("并发数必须在1-100之间,当前值: %d", maxWorkers)
	}

	// 验证标签页数
	if playwrightTabs < 1 || playwrightTabs > 20 {
		return fmt.Errorf("Playwright标签页数必须在1-20之间,当前值: %d", playwrightTabs)
	}

	// 验证相似度阈值
	if similarityThreshold < 0.0 || similarityThreshold > 1.0 {
		return fmt.Errorf("相似度阈值必须在0.0-1.0之间,当前值: %.2f", similarityThreshold)
	}

	// 验证模式
	validModes := map[string]bool{
		"all":     true,
		"static":  true,
		"dynamic": true,
	}
	if !validModes[mode] {
		return fmt.Errorf("无效的爬取模式: %s (有效值: all, static, dynamic)", mode)
	}

	return nil
}

// ValidateURLFile 验证URL文件路径
func ValidateURLFile(filepath string) error {
	if filepath == "" {
		return fmt.Errorf("URL文件路径不能为空")
	}
	// 文件存在性检查将在运行时进行
	return nil
}

// NormalizeURL 规范化URL
func NormalizeURL(urlStr string) (string, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// 如果没有协议,默认使用https
	if parsed.Scheme == "" {
		urlStr = "https://" + urlStr
		parsed, err = url.Parse(urlStr)
		if err != nil {
			return "", err
		}
	}

	return parsed.String(), nil
}
