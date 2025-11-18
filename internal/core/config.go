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
	Resource   ResourceConfig     `mapstructure:"resource"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level    string         `mapstructure:"level"`
	LogDir   string         `mapstructure:"log_dir"`
	Rotation RotationConfig `mapstructure:"rotation"`
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

// ResourceConfig 资源优化配置
type ResourceConfig struct {
	SafetyReserveMemory int `mapstructure:"safety_reserve_memory"` // 安全保留内存(MB)
	SafetyThreshold     int `mapstructure:"safety_threshold"`      // 安全阈值(MB)
	CPULoadThreshold    int `mapstructure:"cpu_load_threshold"`    // CPU负载阈值(%)
	MaxTabsLimit        int `mapstructure:"max_tabs_limit"`        // 绝对最大标签页数
}

// Validate 验证资源配置
// 确保配置值在合理范围内
func (r *ResourceConfig) Validate() error {
	if r.SafetyReserveMemory < 512 {
		return fmt.Errorf("安全保留内存必须 >= 512MB, 当前值: %dMB", r.SafetyReserveMemory)
	}
	if r.SafetyThreshold < 200 {
		return fmt.Errorf("安全阈值必须 >= 200MB, 当前值: %dMB", r.SafetyThreshold)
	}
	if r.CPULoadThreshold < 50 || r.CPULoadThreshold > 999 {
		return fmt.Errorf("CPU负载阈值必须在50-999之间, 当前值: %d%%", r.CPULoadThreshold)
	}
	if r.MaxTabsLimit < 1 || r.MaxTabsLimit > 32 {
		return fmt.Errorf("最大标签页数必须在1-32之间, 当前值: %d", r.MaxTabsLimit)
	}
	return nil
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

	// 验证资源配置
	if err := config.Resource.Validate(); err != nil {
		return nil, fmt.Errorf("资源配置验证失败: %w", err)
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
	v.SetDefault("crawl.allow_cross_domain", true)

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

	// 资源优化配置默认值
	v.SetDefault("resource.safety_reserve_memory", 1024) // 1GB
	v.SetDefault("resource.safety_threshold", 500)       // 500MB
	v.SetDefault("resource.cpu_load_threshold", 80)      // 80%
	v.SetDefault("resource.max_tabs_limit", 16)          // 16个标签页
}

// GetCrawlConfig 从配置中提取爬取配置(合并Resource配置)
func (c *Config) GetCrawlConfig() models.CrawlConfig {
	crawlConfig := c.Crawl
	// 合并Resource配置到CrawlConfig
	crawlConfig.SafetyReserveMemory = c.Resource.SafetyReserveMemory
	crawlConfig.SafetyThreshold = c.Resource.SafetyThreshold
	crawlConfig.CPULoadThreshold = c.Resource.CPULoadThreshold
	crawlConfig.MaxTabsLimit = c.Resource.MaxTabsLimit
	return crawlConfig
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
