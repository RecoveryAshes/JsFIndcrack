package unit

import (
	"testing"

	"github.com/RecoveryAshes/JsFIndcrack/internal/core"
)

func TestHeaderManager_GetMergedHeaders(t *testing.T) {
	t.Run("默认头部存在", func(t *testing.T) {
		hm, err := core.NewHeaderManager("", nil)
		if err != nil {
			t.Fatalf("创建HeaderManager失败: %v", err)
		}

		headers := hm.GetMergedHeaders()

		// 验证默认User-Agent存在
		ua := headers.Get("User-Agent")
		if ua == "" {
			t.Error("期望默认User-Agent存在")
		}
	})

	t.Run("命令行头部覆盖默认", func(t *testing.T) {
		cliHeaders := []string{
			"User-Agent: CustomBot/1.0",
		}

		hm, err := core.NewHeaderManager("", cliHeaders)
		if err != nil {
			t.Fatalf("创建HeaderManager失败: %v", err)
		}

		headers := hm.GetMergedHeaders()
		ua := headers.Get("User-Agent")

		if ua != "CustomBot/1.0" {
			t.Errorf("期望User-Agent='CustomBot/1.0', 实际='%s'", ua)
		}
	})

	t.Run("多个命令行头部", func(t *testing.T) {
		cliHeaders := []string{
			"User-Agent: CustomBot/1.0",
			"X-Custom: value1",
			"Authorization: Bearer token123",
		}

		hm, err := core.NewHeaderManager("", cliHeaders)
		if err != nil {
			t.Fatalf("创建HeaderManager失败: %v", err)
		}

		headers := hm.GetMergedHeaders()

		if headers.Get("User-Agent") != "CustomBot/1.0" {
			t.Error("User-Agent未正确设置")
		}

		if headers.Get("X-Custom") != "value1" {
			t.Error("X-Custom未正确设置")
		}

		if headers.Get("Authorization") != "Bearer token123" {
			t.Error("Authorization未正确设置")
		}
	})
}

func TestHeaderManager_GetSafeHeaders(t *testing.T) {
	t.Run("敏感头部脱敏", func(t *testing.T) {
		cliHeaders := []string{
			"User-Agent: CustomBot/1.0",
			"Authorization: Bearer secret-token-12345",
			"X-API-Key: api-key-67890",
		}

		hm, err := core.NewHeaderManager("", cliHeaders)
		if err != nil {
			t.Fatalf("创建HeaderManager失败: %v", err)
		}

		safeHeaders := hm.GetSafeHeaders()

		// 验证普通头部未脱敏
		if safeHeaders["User-Agent"] != "CustomBot/1.0" {
			t.Error("普通头部不应该被脱敏")
		}

		// 验证Authorization被脱敏
		if safeHeaders["Authorization"] == "Bearer secret-token-12345" {
			t.Error("Authorization应该被脱敏")
		}

		if safeHeaders["Authorization"] != "Bearer ***" {
			t.Errorf("期望Authorization='Bearer ***', 实际='%s'", safeHeaders["Authorization"])
		}

		// 验证API Key被脱敏
		if safeHeaders["X-API-Key"] == "api-key-67890" {
			t.Error("X-API-Key应该被脱敏")
		}
	})
}

func TestHeaderManager_GetHeaders(t *testing.T) {
	t.Run("非法命令行参数返回错误", func(t *testing.T) {
		cliHeaders := []string{
			"InvalidFormat",  // 缺少冒号
		}

		_, err := core.NewHeaderManager("", cliHeaders)
		if err == nil {
			t.Error("期望返回错误, 但成功了")
		}
	})

	t.Run("禁止头部返回验证错误", func(t *testing.T) {
		cliHeaders := []string{
			"Host: example.com",  // 禁止头部
		}

		hm, err := core.NewHeaderManager("", cliHeaders)
		if err != nil {
			t.Fatalf("创建HeaderManager失败: %v", err)
		}

		_, err = hm.GetHeaders()
		if err == nil {
			t.Error("期望返回验证错误, 但成功了")
		}
	})

	t.Run("成功场景", func(t *testing.T) {
		cliHeaders := []string{
			"User-Agent: TestBot/1.0",
			"X-Custom: test-value",
		}

		hm, err := core.NewHeaderManager("", cliHeaders)
		if err != nil {
			t.Fatalf("创建HeaderManager失败: %v", err)
		}

		headers, err := hm.GetHeaders()
		if err != nil {
			t.Fatalf("GetHeaders失败: %v", err)
		}

		if headers.Get("User-Agent") != "TestBot/1.0" {
			t.Error("User-Agent未正确设置")
		}

		if headers.Get("X-Custom") != "test-value" {
			t.Error("X-Custom未正确设置")
		}
	})
}
