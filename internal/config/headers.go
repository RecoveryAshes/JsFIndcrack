package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
	"github.com/spf13/viper"
)

const (
	// DefaultConfigFile 默认配置文件路径
	DefaultConfigFile = "configs/headers.yaml"

	// MaxConfigFileSize 配置文件最大大小 (1MB)
	MaxConfigFileSize = 1 * 1024 * 1024
)

//go:embed headers_template.yaml
var defaultHeaderTemplate string

// HeaderConfigLoader 配置文件加载器
// 负责加载、验证和解析HTTP头部配置文件
type HeaderConfigLoader struct {
	configPath string
}

// NewHeaderConfigLoader 创建配置文件加载器
func NewHeaderConfigLoader(configPath string) *HeaderConfigLoader {
	if configPath == "" {
		configPath = DefaultConfigFile
	}
	return &HeaderConfigLoader{
		configPath: configPath,
	}
}

// EnsureConfigExists 确保配置文件存在,如不存在则自动生成模板
// 返回: 错误信息 (如果无法创建)
func (hcl *HeaderConfigLoader) EnsureConfigExists() error {
	// 检查文件是否存在
	if _, err := os.Stat(hcl.configPath); os.IsNotExist(err) {
		// 创建目录
		dir := filepath.Dir(hcl.configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("无法创建配置目录 [%s]: %w", dir, err)
		}

		// 写入模板
		if err := os.WriteFile(hcl.configPath, []byte(defaultHeaderTemplate), 0644); err != nil {
			return fmt.Errorf("无法生成配置文件 [%s]: %w", hcl.configPath, err)
		}
	}
	return nil
}

// ValidateFileSize 验证配置文件大小是否在限制内
// 返回: 如果文件过大,返回错误
func (hcl *HeaderConfigLoader) ValidateFileSize() error {
	info, err := os.Stat(hcl.configPath)
	if err != nil {
		return fmt.Errorf("无法读取配置文件信息 [%s]: %w", hcl.configPath, err)
	}

	if info.Size() > MaxConfigFileSize {
		return &models.ConfigError{
			FilePath: hcl.configPath,
			Cause: fmt.Errorf("配置文件过大: %d 字节 (最大 %d 字节)",
				info.Size(), MaxConfigFileSize),
		}
	}

	return nil
}

// LoadConfig 加载配置文件并解析为HeaderConfig
// 执行流程:
//  1. 确保配置文件存在 (不存在则自动创建)
//  2. 验证文件大小是否在限制内
//  3. 使用Viper解析YAML
//  4. 绑定到HeaderConfig结构体
//  5. 处理空配置情况
//
// 返回: HeaderConfig和错误信息
func (hcl *HeaderConfigLoader) LoadConfig() (*models.HeaderConfig, error) {
	// 1. 确保配置文件存在
	if err := hcl.EnsureConfigExists(); err != nil {
		return nil, err
	}

	// 2. 验证文件大小
	if err := hcl.ValidateFileSize(); err != nil {
		return nil, err
	}

	// 3. 使用viper解析YAML
	v := viper.New()
	v.SetConfigFile(hcl.configPath)
	v.SetConfigType("yaml")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 检查是否为文件锁定错误
		// 当配置文件被其他进程占用时,优雅降级使用默认配置
		if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) {
			utils.Warnf("配置文件被锁定 [%s], 使用默认配置", hcl.configPath)
			// 返回空配置,系统将使用默认头部
			return &models.HeaderConfig{
				Headers: make(map[string]string),
			}, nil
		}

		return nil, &models.ConfigError{
			FilePath: hcl.configPath,
			Cause:    err,
		}
	}

	// 4. 绑定到结构体
	var config models.HeaderConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, &models.ConfigError{
			FilePath: hcl.configPath,
			Cause:    fmt.Errorf("配置绑定失败: %w", err),
		}
	}

	// 5. 处理空配置 (配置文件存在但headers为空)
	// 初始化空map避免nil指针异常
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}

	return &config, nil
}
