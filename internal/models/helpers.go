package models

import (
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

// ValidateURL 验证URL
func ValidateURL(urlStr string) error {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("无效的URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL必须是HTTP或HTTPS协议")
	}
	if parsed.Host == "" {
		return fmt.Errorf("URL必须包含主机名")
	}
	return nil
}

// generateID 生成唯一ID
func generateID() string {
	return uuid.New().String()
}
