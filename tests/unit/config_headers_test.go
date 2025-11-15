package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RecoveryAshes/JsFIndcrack/internal/config"
)

func TestHeaderConfigLoader_LoadConfig(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "headers.yaml")

	t.Run("首次运行自动生成配置文件", func(t *testing.T) {
		loader := config.NewHeaderConfigLoader(configPath)

		// 确保文件不存在
		if _, err := os.Stat(configPath); !os.IsNotExist(err) {
			t.Fatal("配置文件不应该存在")
		}

		// 加载配置 (应该自动生成)
		cfg, err := loader.LoadConfig()
		if err != nil {
			t.Fatalf("加载配置失败: %v", err)
		}

		// 验证文件已生成
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("配置文件应该被自动生成")
		}

		// 验证配置非空
		if cfg == nil {
			t.Fatal("配置不应该为nil")
		}

		// 验证Headers map已初始化
		if cfg.Headers == nil {
			t.Fatal("Headers map应该被初始化")
		}
	})

	t.Run("加载已存在的配置文件", func(t *testing.T) {
		// 为这个测试创建新的临时目录
		tmpDir2 := t.TempDir()
		configPath2 := filepath.Join(tmpDir2, "headers.yaml")

		// 创建测试配置文件
		testConfig := `headers:
  User-Agent: "Test Bot/1.0"
  X-Custom: "test value"
`
		if err := os.WriteFile(configPath2, []byte(testConfig), 0644); err != nil {
			t.Fatalf("写入测试配置失败: %v", err)
		}

		loader := config.NewHeaderConfigLoader(configPath2)
		cfg, err := loader.LoadConfig()
		if err != nil {
			t.Fatalf("加载配置失败: %v", err)
		}

		// 验证配置内容 (viper会将键名转换为小写)
		if cfg.Headers["user-agent"] != "Test Bot/1.0" {
			t.Errorf("期望 user-agent='Test Bot/1.0', 实际='%s'", cfg.Headers["user-agent"])
		}

		if cfg.Headers["x-custom"] != "test value" {
			t.Errorf("期望 x-custom='test value', 实际='%s'", cfg.Headers["x-custom"])
		}
	})

	t.Run("YAML格式错误返回错误", func(t *testing.T) {
		// 创建格式错误的配置文件
		badConfig := `headers:
  User-Agent: "Test Bot
  X-Custom: missing quote
`
		if err := os.WriteFile(configPath, []byte(badConfig), 0644); err != nil {
			t.Fatalf("写入错误配置失败: %v", err)
		}

		loader := config.NewHeaderConfigLoader(configPath)
		_, err := loader.LoadConfig()
		if err == nil {
			t.Fatal("期望返回错误,但成功了")
		}
	})

	t.Run("空配置文件处理", func(t *testing.T) {
		// 创建空配置文件
		emptyConfig := `headers:`
		if err := os.WriteFile(configPath, []byte(emptyConfig), 0644); err != nil {
			t.Fatalf("写入空配置失败: %v", err)
		}

		loader := config.NewHeaderConfigLoader(configPath)
		cfg, err := loader.LoadConfig()
		if err != nil {
			t.Fatalf("加载空配置失败: %v", err)
		}

		// 验证Headers map已初始化 (即使为空)
		if cfg.Headers == nil {
			t.Fatal("Headers map应该被初始化为空map")
		}
	})

	t.Run("配置文件大小验证", func(t *testing.T) {
		// 创建超大配置文件 (>1MB)
		largeConfig := make([]byte, config.MaxConfigFileSize+1)
		if err := os.WriteFile(configPath, largeConfig, 0644); err != nil {
			t.Fatalf("写入大配置失败: %v", err)
		}

		loader := config.NewHeaderConfigLoader(configPath)
		_, err := loader.LoadConfig()
		if err == nil {
			t.Fatal("期望超大配置文件被拒绝,但成功了")
		}
	})
}
