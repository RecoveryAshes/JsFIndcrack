package models

import (
	"fmt"
	"net/http"
	"strings"
)

// HeaderConfig 表示headers.yaml配置文件的结构
// 从YAML文件加载的HTTP头部配置
type HeaderConfig struct {
	// Headers 存储所有自定义HTTP头部 (键值对)
	// 键: 头部名称 (如 "User-Agent")
	// 值: 头部值 (如 "Mozilla/5.0...")
	Headers map[string]string `mapstructure:"headers" yaml:"headers"`
}

// CliHeaders 表示命令行传递的头部列表
// 每个字符串格式为 "Name: Value"
type CliHeaders []string

// Parse 将字符串列表解析为 http.Header
// 返回解析后的头部和错误信息
func (ch CliHeaders) Parse() (http.Header, error) {
	result := make(http.Header)
	for i, s := range ch {
		name, value, err := parseHeaderString(s)
		if err != nil {
			return nil, fmt.Errorf("参数 --header 第%d项格式错误: %w", i+1, err)
		}
		result.Set(name, value)
	}
	return result, nil
}

// parseHeaderString 解析单个头部字符串 "Name: Value"
// 返回头部名称、值和错误信息
func parseHeaderString(s string) (name, value string, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("格式错误: 缺少冒号分隔符,应为 'Name: Value'")
	}

	name = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])

	if name == "" {
		return "", "", fmt.Errorf("头部名称不能为空")
	}

	return name, value, nil
}

// HeaderProvider 定义HTTP头部提供者接口
// 实现此接口的类型负责管理和提供HTTP请求头部
type HeaderProvider interface {
	// GetHeaders 返回当前有效的HTTP请求头部
	// 返回的http.Header已按优先级合并(默认 < 配置 < 命令行)
	//
	// 返回值:
	//   - http.Header: 可直接应用于http.Request的头部集合
	//   - error: 如果头部加载或验证失败,返回错误
	//
	// 错误情况:
	//   - 配置文件解析失败
	//   - 头部验证失败
	//   - 配置文件不可读
	GetHeaders() (http.Header, error)
}

// ValidationError 头部验证错误
// 表示头部验证失败的详细信息
type ValidationError struct {
	// Field 出错的字段 ("name" 或 "value")
	Field string

	// HeaderName 头部名称
	HeaderName string

	// Reason 错误原因
	Reason string

	// Suggestion 修复建议 (可选)
	Suggestion string
}

// Error 实现error接口
func (e *ValidationError) Error() string {
	msg := fmt.Sprintf("头部验证失败 [%s]: %s", e.HeaderName, e.Reason)
	if e.Suggestion != "" {
		msg += fmt.Sprintf(" (建议: %s)", e.Suggestion)
	}
	return msg
}

// ConfigError 配置文件错误
// 表示配置文件解析失败
type ConfigError struct {
	// FilePath 配置文件路径
	FilePath string

	// Cause 底层错误 (如viper.ConfigParseError)
	Cause error
}

// Error 实现error接口
func (e *ConfigError) Error() string {
	return fmt.Sprintf("配置文件错误 [%s]: %v", e.FilePath, e.Cause)
}

// Unwrap 支持errors.Unwrap
func (e *ConfigError) Unwrap() error {
	return e.Cause
}
