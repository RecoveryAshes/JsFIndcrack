package unit

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RecoveryAshes/JsFIndcrack/internal/config"
	"github.com/RecoveryAshes/JsFIndcrack/internal/core"
	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
)

// TestEdgeCases_EmptyHeaders 测试空头部边缘情况
func TestEdgeCases_EmptyHeaders(t *testing.T) {
	t.Run("空的CLI头部数组", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{})
		_, err := cliHeaders.Parse()
		if err != nil {
			t.Errorf("空数组应该无错误, 得到: %v", err)
		}
	})

	t.Run("nil的CLI头部数组", func(t *testing.T) {
		var cliHeaders models.CliHeaders
		_, err := cliHeaders.Parse()
		if err != nil {
			t.Errorf("nil数组应该无错误, 得到: %v", err)
		}
	})

	t.Run("空配置文件", func(t *testing.T) {
		tmpDir, _ := ioutil.TempDir("", "edge-test-*")
		defer os.RemoveAll(tmpDir)

		configPath := filepath.Join(tmpDir, "empty.yaml")
		_ = ioutil.WriteFile(configPath, []byte(""), 0644)

		loader := config.NewHeaderConfigLoader(configPath)
		cfg, err := loader.LoadConfig()
		if err != nil {
			t.Errorf("空配置文件应该可以加载, 得到错误: %v", err)
		}
		if cfg.Headers == nil {
			t.Error("空配置应该初始化Headers为空map")
		}
	})
}

// TestEdgeCases_WhitespaceHandling 测试空白字符处理
func TestEdgeCases_WhitespaceHandling(t *testing.T) {
	t.Run("头部名称前后空格", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"  User-Agent  : Mozilla/5.0"})
		headers, err := cliHeaders.Parse()
		if err != nil {
			t.Fatalf("应该自动trim空格, 得到错误: %v", err)
		}
		// 检查是否正确trim
		if _, ok := headers["User-Agent"]; !ok {
			t.Error("应该trim头部名称的空格")
		}
	})

	t.Run("头部值前后空格", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"User-Agent:  Mozilla/5.0  "})
		headers, err := cliHeaders.Parse()
		if err != nil {
			t.Fatalf("应该自动trim空格, 得到错误: %v", err)
		}
		if val := headers.Get("User-Agent"); !strings.HasPrefix(val, "Mozilla") {
			t.Errorf("应该trim头部值的前导空格, 得到: '%s'", val)
		}
	})

	t.Run("值中间的空格应该保留", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"X-Custom: value with spaces"})
		headers, err := cliHeaders.Parse()
		if err != nil {
			t.Fatalf("应该允许值中间有空格, 得到错误: %v", err)
		}
		if val := headers.Get("X-Custom"); val != "value with spaces" {
			t.Errorf("应该保留值中间的空格, 得到: '%s'", val)
		}
	})

	t.Run("多个连续空格", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"X-Test:     multiple     spaces     "})
		headers, err := cliHeaders.Parse()
		if err != nil {
			t.Fatalf("应该允许多个空格, 得到错误: %v", err)
		}
		// 前后trim,但中间保留
		if val := headers.Get("X-Test"); !strings.Contains(val, "multiple") {
			t.Errorf("应该保留部分空格, 得到: '%s'", val)
		}
	})
}

// TestEdgeCases_SpecialCharacters 测试特殊字符边缘情况
func TestEdgeCases_SpecialCharacters(t *testing.T) {
	validator := utils.NewHeaderValidator()

	t.Run("值中包含冒号", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"X-URL: https://example.com:8080/path"})
		headers, err := cliHeaders.Parse()
		if err != nil {
			t.Fatalf("应该允许值中包含冒号, 得到错误: %v", err)
		}
		if val := headers.Get("X-URL"); !strings.Contains(val, "https://") {
			t.Errorf("值中的冒号应该保留, 得到: '%s'", val)
		}
	})

	t.Run("值中包含等号", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"X-Equation: 1+1=2"})
		headers, err := cliHeaders.Parse()
		if err != nil {
			t.Fatalf("应该允许值中包含等号, 得到错误: %v", err)
		}
		if val := headers.Get("X-Equation"); val != "1+1=2" {
			t.Errorf("值中的等号应该保留, 得到: '%s'", val)
		}
	})

	t.Run("值包含引号", func(t *testing.T) {
		err := validator.ValidateValue("X-Quote", `value "with" quotes`)
		if err != nil {
			t.Errorf("应该允许值中包含引号, 得到错误: %v", err)
		}
	})

	t.Run("值包含中文字符", func(t *testing.T) {
		err := validator.ValidateValue("X-Chinese", "测试中文")
		// RFC 7230不允许非ASCII字符,应该报错
		if err == nil {
			t.Error("中文字符应该被拒绝")
		}
	})

	t.Run("值包含Unicode表情", func(t *testing.T) {
		err := validator.ValidateValue("X-Emoji", "test 😀 emoji")
		if err == nil {
			t.Error("emoji应该被拒绝")
		}
	})
}

// TestEdgeCases_MalformedInput 测试格式错误的输入
func TestEdgeCases_MalformedInput(t *testing.T) {
	t.Run("缺少冒号分隔符", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"User-Agent Mozilla/5.0"})
		_, err := cliHeaders.Parse()
		if err == nil {
			t.Error("缺少冒号应该报错")
		}
	})

	t.Run("只有冒号没有值", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"User-Agent:"})
		headers, err := cliHeaders.Parse()
		if err != nil {
			t.Fatalf("空值应该被允许, 得到错误: %v", err)
		}
		if val := headers.Get("User-Agent"); val != "" {
			t.Errorf("空值应该为空字符串, 得到: '%s'", val)
		}
	})

	t.Run("只有冒号没有名称", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{":value"})
		_, err := cliHeaders.Parse()
		if err == nil {
			t.Error("缺少头部名称应该报错")
		}
	})

	t.Run("多个冒号", func(t *testing.T) {
		cliHeaders := models.CliHeaders([]string{"Authorization: Bearer: token"})
		headers, err := cliHeaders.Parse()
		if err != nil {
			t.Fatalf("多个冒号应该按第一个冒号分割, 得到错误: %v", err)
		}
		// 第一个冒号后的所有内容都是值
		if val := headers.Get("Authorization"); !strings.Contains(val, "Bearer:") {
			t.Errorf("后续冒号应该保留在值中, 得到: '%s'", val)
		}
	})
}

// TestEdgeCases_BoundaryValues 测试边界值
func TestEdgeCases_BoundaryValues(t *testing.T) {
	validator := utils.NewHeaderValidator()

	t.Run("最大长度头部值", func(t *testing.T) {
		// 创建最大长度的值
		maxValue := strings.Repeat("a", utils.MaxHeaderValueLength)
		err := validator.ValidateValue("X-Max", maxValue)
		if err != nil {
			t.Errorf("最大长度值应该被接受, 得到错误: %v", err)
		}
	})

	t.Run("超过最大长度", func(t *testing.T) {
		// 超过最大长度
		tooLongValue := strings.Repeat("a", utils.MaxHeaderValueLength+1)
		err := validator.ValidateValue("X-TooLong", tooLongValue)
		if err == nil {
			t.Error("超长值应该被拒绝")
		}
	})

	t.Run("最小长度头部名称", func(t *testing.T) {
		err := validator.ValidateName("X")
		if err != nil {
			t.Errorf("单字符名称应该被接受, 得到错误: %v", err)
		}
	})

	t.Run("零长度头部值", func(t *testing.T) {
		err := validator.ValidateValue("X-Empty", "")
		if err != nil {
			t.Errorf("空值应该被接受, 得到错误: %v", err)
		}
	})
}

// TestEdgeCases_CaseSensitivity 测试大小写敏感性
func TestEdgeCases_CaseSensitivity(t *testing.T) {
	validator := utils.NewHeaderValidator()

	t.Run("禁止头部不区分大小写", func(t *testing.T) {
		tests := []string{"Host", "host", "HOST", "HoSt"}
		for _, name := range tests {
			if !validator.IsForbidden(name) {
				t.Errorf("禁止头部应该不区分大小写: %s", name)
			}
		}
	})

	t.Run("头部名称规范化", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("user-agent", "test1")
		headers.Set("User-Agent", "test2")
		// http.Header会自动规范化为User-Agent
		if headers.Get("User-Agent") != "test2" {
			t.Error("http.Header应该规范化头部名称")
		}
	})
}

// TestEdgeCases_HeaderManager 测试HeaderManager边缘情况
func TestEdgeCases_HeaderManager(t *testing.T) {
	t.Run("配置文件不存在时自动生成", func(t *testing.T) {
		tmpDir, _ := ioutil.TempDir("", "hm-test-*")
		defer os.RemoveAll(tmpDir)

		nonExistPath := filepath.Join(tmpDir, "nonexist", "headers.yaml")
		loader := config.NewHeaderConfigLoader(nonExistPath)

		err := loader.EnsureConfigExists()
		if err != nil {
			t.Fatalf("应该自动创建配置文件, 得到错误: %v", err)
		}

		// 验证文件已创建
		if _, err := os.Stat(nonExistPath); os.IsNotExist(err) {
			t.Error("配置文件未创建")
		}
	})

	t.Run("配置文件过大", func(t *testing.T) {
		tmpDir, _ := ioutil.TempDir("", "hm-test-*")
		defer os.RemoveAll(tmpDir)

		configPath := filepath.Join(tmpDir, "huge.yaml")
		// 创建超大文件 (>1MB)
		hugeContent := strings.Repeat("headers:\n  X-Test: value\n", 50000)
		_ = ioutil.WriteFile(configPath, []byte(hugeContent), 0644)

		loader := config.NewHeaderConfigLoader(configPath)
		err := loader.ValidateFileSize()
		if err == nil {
			t.Error("超大配置文件应该被拒绝")
		}
	})

	t.Run("同时提供CLI和配置文件头部", func(t *testing.T) {
		tmpDir, _ := ioutil.TempDir("", "hm-test-*")
		defer os.RemoveAll(tmpDir)

		configPath := filepath.Join(tmpDir, "headers.yaml")
		configContent := `headers:
  X-Config: from-config
  User-Agent: config-agent`
		_ = ioutil.WriteFile(configPath, []byte(configContent), 0644)

		// CLI头部优先级更高
		cliHeaders := []string{
			"X-CLI: from-cli",
			"User-Agent: cli-agent",
		}

		hm, err := core.NewHeaderManager(configPath, cliHeaders)
		if err != nil {
			t.Fatalf("创建HeaderManager失败: %v", err)
		}

		if err := hm.LoadConfig(); err != nil {
			t.Fatalf("加载配置失败: %v", err)
		}

		merged := hm.GetMergedHeaders()

		// CLI应该覆盖配置文件
		if val := merged.Get("User-Agent"); val != "cli-agent" {
			t.Errorf("CLI头部应该覆盖配置文件, 得到: %s", val)
		}

		// 应该同时包含两者
		if merged.Get("X-Config") == "" {
			t.Error("应该包含配置文件中的头部")
		}
		if merged.Get("X-CLI") == "" {
			t.Error("应该包含CLI中的头部")
		}
	})
}

// TestEdgeCases_Redaction 测试脱敏边缘情况
func TestEdgeCases_Redaction(t *testing.T) {
	redactor := utils.NewHeaderRedactor()

	t.Run("部分匹配敏感模式", func(t *testing.T) {
		tests := []struct {
			name  string
			value string
		}{
			{"Authorization", "Bearer token123"},
			{"X-Token", "longtoken123456789"},  // 修改为更明确的token关键字
			{"X-Api-Key", "key12345678"},      // 修改为key关键字
			{"X-Secret", "password123456"},    // 修改为secret关键字
		}

		for _, tt := range tests {
			headers := http.Header{}
			headers.Set(tt.name, tt.value)
			redacted := redactor.Redact(headers)

			// 检查是否被识别为敏感头部
			if !redactor.IsSensitiveHeader(tt.name) {
				t.Errorf("应该被识别为敏感头部: %s", tt.name)
				continue
			}

			redactedValue, exists := redacted[tt.name]
			if !exists {
				t.Errorf("头部应该存在于脱敏结果中: %s", tt.name)
				continue
			}

			if redactedValue == tt.value {
				t.Errorf("敏感头部应该被脱敏: %s (原值: %s, 脱敏后: %s)", tt.name, tt.value, redactedValue)
			}
			if !strings.Contains(redactedValue, "*") {
				t.Errorf("脱敏后应该包含星号: %s -> %s", tt.value, redactedValue)
			}
		}
	})

	t.Run("非敏感头部不应脱敏", func(t *testing.T) {
		tests := []struct {
			name  string
			value string
		}{
			{"User-Agent", "Mozilla/5.0"},
			{"Accept", "*/*"},
			{"X-Custom", "value"},
		}

		for _, tt := range tests {
			headers := http.Header{}
			headers.Set(tt.name, tt.value)
			redacted := redactor.Redact(headers)
			redactedValue := redacted[tt.name]

			if redactedValue != tt.value {
				t.Errorf("非敏感头部不应被脱敏: %s", tt.name)
			}
		}
	})

	t.Run("空值脱敏", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "")
		redacted := redactor.Redact(headers)
		redactedValue := redacted["Authorization"]

		if redactedValue != "***" {
			t.Errorf("空敏感头部应该显示为***, 得到: %s", redactedValue)
		}
	})
}
