package utils

import (
	"net/http"
	"strings"
)

var (
	// SensitiveKeywords 敏感头部名称关键字 (用于脱敏)
	SensitiveKeywords = []string{
		"authorization",
		"token",
		"key",
		"secret",
		"password",
		"credential",
		"api-key",
	}
)

// HeaderRedactor 头部脱敏器
// 负责识别并脱敏敏感HTTP头部
type HeaderRedactor struct {
	sensitiveKeywords []string
}

// NewHeaderRedactor 创建头部脱敏器
func NewHeaderRedactor() *HeaderRedactor {
	return &HeaderRedactor{
		sensitiveKeywords: SensitiveKeywords,
	}
}

// IsSensitiveHeader 检查头部是否为敏感头部
// 根据头部名称关键字判断
func (hr *HeaderRedactor) IsSensitiveHeader(name string) bool {
	nameLower := strings.ToLower(name)
	for _, keyword := range hr.sensitiveKeywords {
		if strings.Contains(nameLower, keyword) {
			return true
		}
	}
	return false
}

// RedactHeaderValue 脱敏单个头部值
// 根据值的格式选择不同的脱敏策略
func (hr *HeaderRedactor) RedactHeaderValue(name, value string) string {
	if !hr.IsSensitiveHeader(name) {
		return value
	}

	// 策略1: Bearer Token - 仅显示前缀
	if strings.HasPrefix(value, "Bearer ") {
		return "Bearer ***"
	}

	// 策略2: API Key - 显示前4位+后4位 (如果足够长)
	if len(value) > 8 {
		return value[:4] + "***" + value[len(value)-4:]
	}

	// 策略3: 短密钥 - 完全隐藏
	return "***"
}

// Redact 脱敏整个http.Header,返回安全的字符串map (用于日志)
// 返回的map中,敏感头部的值已被脱敏
func (hr *HeaderRedactor) Redact(headers http.Header) map[string]string {
	result := make(map[string]string)
	for name, values := range headers {
		if len(values) == 0 {
			continue
		}

		// 只取第一个值 (大多数头部只有一个值)
		value := values[0]
		if hr.IsSensitiveHeader(name) {
			result[name] = hr.RedactHeaderValue(name, value)
		} else {
			result[name] = value
		}
	}
	return result
}

// RedactToString 脱敏http.Header并返回格式化字符串 (用于日志输出)
// 格式: "Header1: value1, Header2: value2, ..."
func (hr *HeaderRedactor) RedactToString(headers http.Header) string {
	redacted := hr.Redact(headers)
	var parts []string
	for name, value := range redacted {
		parts = append(parts, name+": "+value)
	}
	return strings.Join(parts, ", ")
}
