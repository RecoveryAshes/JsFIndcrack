package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/spf13/viper"
)

// Config 应用程序配置
type Config struct {
	Crawl      models.CrawlConfig `mapstructure:"crawl"`
	Logging    LoggingConfig      `mapstructure:"logging"`
	Output     OutputConfig       `mapstructure:"output"`
	Similarity SimilarityConfig   `mapstructure:"similarity"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level   string          `mapstructure:"level"`
	LogDir  string          `mapstructure:"log_dir"`
	Rotation RotationConfig  `mapstructure:"rotation"`
}

// RotationConfig 日志轮转配置
type RotationConfig struct {
	MaxSize    int  `mapstructure:"max_size"`
	MaxBackups int  `mapstructure:"max_backups"`
	MaxAge     int  `mapstructure:"max_age"`
	Compress   bool `mapstructure:"compress"`
}

// OutputConfig 输出配置
type OutputConfig struct {
	BaseDir          string `mapstructure:"base_dir"`
	DomainSeparation bool   `mapstructure:"domain_separation"`
}

// SimilarityConfig 相似度配置
type SimilarityConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	Threshold float64 `mapstructure:"threshold"`
	Workers   int     `mapstructure:"workers"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件
	if configPath != "" {
		// 使用指定的配置文件
		v.SetConfigFile(configPath)
	} else {
		// 搜索默认位置
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// 添加配置搜索路径
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")

		// 用户主目录
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".jsfindcrack"))
		}
	}

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 如果配置文件不存在,使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在,使用默认值
	}

	// 解析配置
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// 爬取配置默认值
	v.SetDefault("crawl.depth", 2)
	v.SetDefault("crawl.wait_time", 3)
	v.SetDefault("crawl.max_workers", 2)
	v.SetDefault("crawl.playwright_tabs", 4)
	v.SetDefault("crawl.headless", true)
	v.SetDefault("crawl.resume", false)
	v.SetDefault("crawl.similarity_enabled", true)
	v.SetDefault("crawl.similarity_threshold", 0.8)
	v.SetDefault("crawl.similarity_workers", 8)

	// 日志配置默认值
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.log_dir", "logs")
	v.SetDefault("logging.rotation.max_size", 10)
	v.SetDefault("logging.rotation.max_backups", 3)
	v.SetDefault("logging.rotation.max_age", 28)
	v.SetDefault("logging.rotation.compress", true)

	// 输出配置默认值
	v.SetDefault("output.base_dir", "output")
	v.SetDefault("output.domain_separation", true)

	// 相似度配置默认值
	v.SetDefault("similarity.enabled", true)
	v.SetDefault("similarity.threshold", 0.8)
	v.SetDefault("similarity.workers", 8)
}

// GetCrawlConfig 从配置中提取爬取配置
func (c *Config) GetCrawlConfig() models.CrawlConfig {
	return c.Crawl
}

// MergeCLIFlags 合并命令行参数到配置
func (c *Config) MergeCLIFlags(
	depth int,
	waitTime int,
	maxWorkers int,
	playwrightTabs int,
	headless bool,
	resume bool,
	similarityThreshold float64,
	mode string,
) {
	// 命令行参数优先于配置文件
	if depth > 0 {
		c.Crawl.Depth = depth
	}
	if waitTime >= 0 {
		c.Crawl.WaitTime = waitTime
	}
	if maxWorkers > 0 {
		c.Crawl.MaxWorkers = maxWorkers
	}
	if playwrightTabs > 0 {
		c.Crawl.PlaywrightTabs = playwrightTabs
	}
	c.Crawl.Headless = headless
	c.Crawl.Resume = resume
	if similarityThreshold > 0 {
		c.Crawl.SimilarityThreshold = similarityThreshold
	}
}
