package crawlers

import (
	"testing"
)

// TestIsValidJavaScript 测试JavaScript内容检测函数
// 契约参考: contracts/module-contracts.md - 内容检测契约
func TestIsValidJavaScript(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		expected    bool
		reason      string
	}{
		{
			name:        "Content-Type为application/javascript",
			contentType: "application/javascript; charset=utf-8",
			body:        []byte("var x = 1;"),
			expected:    true,
			reason:      "Content-Type明确指示为JavaScript",
		},
		{
			name:        "Content-Type为text/javascript",
			contentType: "text/javascript",
			body:        []byte("function test() {}"),
			expected:    true,
			reason:      "Content-Type明确指示为JavaScript",
		},
		{
			name:        "Content-Type为text/html但包含足够的JS关键字",
			contentType: "text/html",
			body:        []byte("function foo() { var x = 1; const y = 2; }"),
			expected:    true,
			reason:      "包含3个JS关键字(function, var, const)",
		},
		{
			name:        "Content-Type为text/html且仅包含1个JS关键字",
			contentType: "text/html",
			body:        []byte("<html><body>function is a keyword</body></html>"),
			expected:    false,
			reason:      "只包含1个关键字,不足以判断为JS",
		},
		{
			name:        "HTTP 404但Content-Type为javascript",
			contentType: "application/javascript",
			body:        []byte("// Not Found\nfunction realCode() { return true; }"),
			expected:    true,
			reason:      "假404,Content-Type正确",
		},
		{
			name:        "HTTP 404但包含大量JS代码",
			contentType: "text/plain",
			body:        []byte("const app = () => { let x = 1; class Foo {} export default app; }"),
			expected:    true,
			reason:      "包含5个JS关键字(const, let, class, export, =>)",
		},
		{
			name:        "纯HTML内容无JS特征",
			contentType: "text/html",
			body:        []byte("<html><head><title>Test</title></head><body>Hello World</body></html>"),
			expected:    false,
			reason:      "无JS关键字",
		},
		{
			name:        "空内容",
			contentType: "text/plain",
			body:        []byte(""),
			expected:    false,
			reason:      "空内容无法判断",
		},
		{
			name:        "大文件检测(仅检查前1KB)",
			contentType: "text/plain",
			body:        append([]byte("function test() { var x = 1; } "), make([]byte, 2000)...),
			expected:    true,
			reason:      "前1KB包含足够关键字",
		},
		{
			name:        "箭头函数语法",
			contentType: "text/plain",
			body:        []byte("const fn = () => { return x; }"),
			expected:    true,
			reason:      "包含const和=>",
		},
		{
			name:        "ES6模块语法",
			contentType: "text/plain",
			body:        []byte("import React from 'react'; export default App;"),
			expected:    true,
			reason:      "包含import和export",
		},
		{
			name:        "类定义语法",
			contentType: "text/plain",
			body:        []byte("class MyClass { constructor() {} }"),
			expected:    true,
			reason:      "包含class和function(构造函数)",
		},
		{
			name:        "JSON数据(不是JS代码)",
			contentType: "application/json",
			body:        []byte(`{"key": "value", "number": 123}`),
			expected:    false,
			reason:      "JSON不包含JS关键字",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidJavaScript(tt.contentType, tt.body)
			if result != tt.expected {
				t.Errorf("isValidJavaScript() = %v, 期望 %v\n原因: %s\nContent-Type: %s\nBody前100字节: %s",
					result, tt.expected, tt.reason, tt.contentType, string(tt.body[:min(len(tt.body), 100)]))
			}
		})
	}
}

// TestIsValidJavaScriptEdgeCases 测试边界情况
func TestIsValidJavaScriptEdgeCases(t *testing.T) {
	t.Run("Content-Type大小写不敏感", func(t *testing.T) {
		body := []byte("var x = 1;")

		testCases := []string{
			"APPLICATION/JAVASCRIPT",
			"Application/JavaScript",
			"text/JAVASCRIPT",
		}

		for _, ct := range testCases {
			if !isValidJavaScript(ct, body) {
				t.Errorf("Content-Type %s 应该被识别为JavaScript", ct)
			}
		}
	})

	t.Run("关键字阈值验证", func(t *testing.T) {
		// 1个关键字 - 应该返回false
		body1 := []byte("function only one keyword here")
		if isValidJavaScript("text/plain", body1) {
			t.Error("1个关键字不应被识别为JavaScript")
		}

		// 2个关键字 - 应该返回true
		body2 := []byte("function test() { var x = 1; }")
		if !isValidJavaScript("text/plain", body2) {
			t.Error("2个关键字应该被识别为JavaScript")
		}

		// 3个关键字 - 应该返回true
		body3 := []byte("const fn = () => { let x = 1; }")
		if !isValidJavaScript("text/plain", body3) {
			t.Error("3个关键字应该被识别为JavaScript")
		}
	})

	t.Run("性能测试-大文件", func(t *testing.T) {
		// 创建10MB的测试数据
		largeBody := make([]byte, 10*1024*1024)
		copy(largeBody, []byte("function test() { var x = 1; }"))

		// 应该只检查前1KB,性能良好
		result := isValidJavaScript("text/plain", largeBody)
		if !result {
			t.Error("大文件前部包含JS关键字应被正确识别")
		}
	})
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
