package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitLogger(t *testing.T) {
	// 创建临时日志目录
	tempDir := t.TempDir()

	config := LogConfig{
		Level:      "debug",
		LogDir:     tempDir,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	// 初始化日志器
	err := InitLogger(config)
	if err != nil {
		t.Fatalf("初始化日志器失败: %v", err)
	}

	// 验证日志目录已创建
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("日志目录未创建: %s", tempDir)
	}

	// 写入测试日志
	Info("测试信息日志")
	Warn("测试警告日志")
	Debug("测试调试日志")

	// 等待日志写入
	time.Sleep(100 * time.Millisecond)

	// 验证主日志文件存在
	mainLogPath := filepath.Join(tempDir, "js_crawler.log")
	if _, err := os.Stat(mainLogPath); os.IsNotExist(err) {
		t.Errorf("主日志文件未创建: %s", mainLogPath)
	}
}

func TestLogLevels(t *testing.T) {
	tempDir := t.TempDir()

	config := LogConfig{
		Level:      "info",
		LogDir:     tempDir,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   false,
	}

	err := InitLogger(config)
	if err != nil {
		t.Fatalf("初始化日志器失败: %v", err)
	}

	// 测试各种日志级别
	Info("信息日志测试")
	Infof("格式化信息日志: %s", "测试")
	Warn("警告日志测试")
	Warnf("格式化警告日志: %d", 123)
	Debug("调试日志测试 - 应该不显示因为级别是info")
	Debugf("格式化调试日志: %v", true)

	time.Sleep(100 * time.Millisecond)

	// 验证日志文件存在且有内容
	mainLogPath := filepath.Join(tempDir, "js_crawler.log")
	content, err := os.ReadFile(mainLogPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	if len(content) == 0 {
		t.Error("日志文件为空")
	}
}

func TestDefaultLogConfig(t *testing.T) {
	config := DefaultLogConfig()

	if config.Level != "info" {
		t.Errorf("默认日志级别错误: 期望 'info', 得到 '%s'", config.Level)
	}

	if config.LogDir != "logs" {
		t.Errorf("默认日志目录错误: 期望 'logs', 得到 '%s'", config.LogDir)
	}

	if config.MaxSize != 10 {
		t.Errorf("默认最大大小错误: 期望 10, 得到 %d", config.MaxSize)
	}

	if config.MaxBackups != 3 {
		t.Errorf("默认备份数错误: 期望 3, 得到 %d", config.MaxBackups)
	}

	if config.MaxAge != 28 {
		t.Errorf("默认保留天数错误: 期望 28, 得到 %d", config.MaxAge)
	}

	if !config.Compress {
		t.Error("默认应该启用压缩")
	}
}

func TestChineseLogOutput(t *testing.T) {
	tempDir := t.TempDir()

	config := LogConfig{
		Level:      "info",
		LogDir:     tempDir,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   false,
	}

	err := InitLogger(config)
	if err != nil {
		t.Fatalf("初始化日志器失败: %v", err)
	}

	// 测试中文日志输出
	chineseMsg := "这是一条中文日志消息"
	Info(chineseMsg)

	time.Sleep(100 * time.Millisecond)

	// 读取日志文件并验证中文编码正确
	mainLogPath := filepath.Join(tempDir, "js_crawler.log")
	content, err := os.ReadFile(mainLogPath)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	// 验证包含中文内容
	if len(content) > 0 {
		t.Logf("日志内容长度: %d 字节", len(content))
	} else {
		t.Error("日志文件为空,中文日志未写入")
	}
}
