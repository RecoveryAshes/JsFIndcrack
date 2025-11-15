package unit

import (
	"net/http"
	"testing"

	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
)

func TestHeaderValidator_ValidateName(t *testing.T) {
	validator := utils.NewHeaderValidator()

	tests := []struct {
		name        string
		headerName  string
		expectError bool
	}{
		{"合法名称-字母", "User-Agent", false},
		{"合法名称-数字", "X-Request-ID-123", false},
		{"合法名称-连字符", "Accept-Language", false},
		{"非法名称-空格", "User Agent", true},
		{"非法名称-下划线", "User_Agent", true},
		{"非法名称-特殊字符", "User@Agent", true},
		{"非法名称-空字符串", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateName(tt.headerName)
			if (err != nil) != tt.expectError {
				t.Errorf("期望错误=%v, 实际错误=%v", tt.expectError, err)
			}
		})
	}
}

func TestHeaderValidator_ValidateValue(t *testing.T) {
	validator := utils.NewHeaderValidator()

	// 创建一个有效的长字符串 (重复空格)
	longValidString := ""
	for i := 0; i < 8000; i++ {
		longValidString += " "
	}

	tests := []struct {
		name        string
		headerName  string
		headerValue string
		expectError bool
	}{
		{"合法值-ASCII", "User-Agent", "Mozilla/5.0", false},
		{"合法值-空字符串", "X-Empty", "", false},
		{"合法值-长字符串", "X-Long", longValidString, false},
		{"非法值-超长", "X-TooLong", string(make([]byte, utils.MaxHeaderValueLength+1)), true},
		{"非法值-控制字符", "X-Bad", "value\x00with\x01null", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateValue(tt.headerName, tt.headerValue)
			if (err != nil) != tt.expectError {
				t.Errorf("期望错误=%v, 实际错误=%v", tt.expectError, err)
			}
		})
	}
}

func TestHeaderValidator_ValidateHeader(t *testing.T) {
	validator := utils.NewHeaderValidator()

	tests := []struct {
		name        string
		headerName  string
		headerValue string
		expectError bool
	}{
		{"合法头部", "User-Agent", "Mozilla/5.0", false},
		{"禁止头部-Host", "Host", "example.com", true},
		{"禁止头部-Content-Length", "Content-Length", "123", true},
		{"禁止头部-不区分大小写", "host", "example.com", true},
		{"非法名称", "User Agent", "value", true},
		{"非法值", "User-Agent", "value\x00bad", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateHeader(tt.headerName, tt.headerValue)
			if (err != nil) != tt.expectError {
				t.Errorf("期望错误=%v, 实际错误=%v", tt.expectError, err)
			}
		})
	}
}

func TestHeaderValidator_IsForbidden(t *testing.T) {
	validator := utils.NewHeaderValidator()

	tests := []struct {
		name       string
		headerName string
		expected   bool
	}{
		{"Host-禁止", "Host", true},
		{"host-禁止-不区分大小写", "host", true},
		{"Content-Length-禁止", "Content-Length", true},
		{"User-Agent-允许", "User-Agent", false},
		{"X-Custom-允许", "X-Custom", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsForbidden(tt.headerName)
			if result != tt.expected {
				t.Errorf("期望=%v, 实际=%v", tt.expected, result)
			}
		})
	}
}

func TestHeaderValidator_Validate(t *testing.T) {
	validator := utils.NewHeaderValidator()

	t.Run("验证合法的http.Header", func(t *testing.T) {
		headers := http.Header{
			"User-Agent":     []string{"Mozilla/5.0"},
			"Accept":         []string{"*/*"},
			"X-Custom":       []string{"value"},
		}

		err := validator.Validate(headers)
		if err != nil {
			t.Errorf("期望无错误, 实际错误=%v", err)
		}
	})

	t.Run("验证包含非法头部的http.Header", func(t *testing.T) {
		headers := http.Header{
			"User-Agent": []string{"Mozilla/5.0"},
			"Host":       []string{"example.com"}, // 禁止头部
		}

		err := validator.Validate(headers)
		if err == nil {
			t.Error("期望返回错误, 但无错误")
		}
	})

	t.Run("验证包含非法值的http.Header", func(t *testing.T) {
		headers := http.Header{
			"User-Agent": []string{"value\x00bad"}, // 控制字符
		}

		err := validator.Validate(headers)
		if err == nil {
			t.Error("期望返回错误, 但无错误")
		}
	})
}
