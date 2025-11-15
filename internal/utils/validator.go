package utils

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
)

const (
	// MaxHeaderValueLength HTTP头部值最大长度 (8KB)
	MaxHeaderValueLength = 8192
)

var (
	// ForbiddenHeaders 禁止用户配置的头部 (由HTTP客户端管理)
	ForbiddenHeaders = []string{
		"Host",
		"Content-Length",
		"Transfer-Encoding",
		"Connection",
	}
)

// HeaderValidator 验证HTTP头部是否符合RFC 7230规范
type HeaderValidator struct {
	// nameRegex 验证头部名称 (字母数字连字符)
	nameRegex *regexp.Regexp

	// valueRegex 验证头部值 (可打印ASCII)
	valueRegex *regexp.Regexp

	// maxValueLength 头部值最大长度 (字节)
	maxValueLength int

	// forbiddenHeaders 禁止用户配置的头部 (如Host, Content-Length)
	forbiddenHeaders map[string]bool
}

// NewHeaderValidator 创建验证器
func NewHeaderValidator() *HeaderValidator {
	// 构建禁止头部的map (不区分大小写)
	forbidden := make(map[string]bool)
	for _, h := range ForbiddenHeaders {
		forbidden[strings.ToLower(h)] = true
	}

	return &HeaderValidator{
		// HTTP头部名称验证 (RFC 7230): 允许字母、数字和连字符
		nameRegex: regexp.MustCompile(`^[A-Za-z0-9-]+$`),

		// HTTP头部值验证: 可打印ASCII + 空格/制表符
		valueRegex: regexp.MustCompile(`^[\x20-\x7E\t]*$`),

		maxValueLength:   MaxHeaderValueLength,
		forbiddenHeaders: forbidden,
	}
}

// ValidateName 验证头部名称
// 返回: 如果名称非法,返回ValidationError
func (hv *HeaderValidator) ValidateName(name string) error {
	if name == "" {
		return &models.ValidationError{
			Field:      "name",
			HeaderName: name,
			Reason:     "头部名称不能为空",
		}
	}

	if !hv.nameRegex.MatchString(name) {
		return &models.ValidationError{
			Field:      "name",
			HeaderName: name,
			Reason:     "头部名称包含非法字符 (仅允许字母、数字和连字符)",
			Suggestion: "使用字母、数字和连字符 (如 'User-Agent', 'X-Custom-Header')",
		}
	}

	return nil
}

// ValidateValue 验证头部值
// 返回: 如果值非法,返回ValidationError
func (hv *HeaderValidator) ValidateValue(name, value string) error {
	// 检查长度
	if len(value) > hv.maxValueLength {
		return &models.ValidationError{
			Field:      "value",
			HeaderName: name,
			Reason:     fmt.Sprintf("头部值过长: %d 字节 (最大 %d)", len(value), hv.maxValueLength),
			Suggestion: fmt.Sprintf("将值缩短至 %d 字节以内", hv.maxValueLength),
		}
	}

	// 检查字符合法性
	if !hv.valueRegex.MatchString(value) {
		return &models.ValidationError{
			Field:      "value",
			HeaderName: name,
			Reason:     "头部值包含非法字符 (仅允许可打印ASCII字符)",
			Suggestion: "移除控制字符和非ASCII字符",
		}
	}

	return nil
}

// ValidateHeader 验证头部名称+值
// 返回: 如果头部非法,返回ValidationError
func (hv *HeaderValidator) ValidateHeader(name, value string) error {
	// 1. 检查是否为禁止头部
	if hv.IsForbidden(name) {
		return &models.ValidationError{
			Field:      "name",
			HeaderName: name,
			Reason:     "此头部由HTTP客户端自动管理,不允许自定义",
			Suggestion: fmt.Sprintf("移除 '%s' 头部配置", name),
		}
	}

	// 2. 验证名称
	if err := hv.ValidateName(name); err != nil {
		return err
	}

	// 3. 验证值
	if err := hv.ValidateValue(name, value); err != nil {
		return err
	}

	return nil
}

// IsForbidden 检查头部是否被禁止
func (hv *HeaderValidator) IsForbidden(name string) bool {
	return hv.forbiddenHeaders[strings.ToLower(name)]
}

// Validate 验证http.Header中的所有头部
// 返回: 如果有任何头部非法,返回第一个ValidationError
func (hv *HeaderValidator) Validate(headers http.Header) error {
	for name, values := range headers {
		for _, value := range values {
			if err := hv.ValidateHeader(name, value); err != nil {
				return err
			}
		}
	}
	return nil
}
