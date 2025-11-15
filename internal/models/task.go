package models

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待执行
	TaskStatusRunning   TaskStatus = "running"   // 执行中
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// CrawlMode 爬取模式
type CrawlMode string

const (
	ModeAll     CrawlMode = "all"     // 静态+动态
	ModeStatic  CrawlMode = "static"  // 仅静态
	ModeDynamic CrawlMode = "dynamic" // 仅动态
)

// TaskStats 任务统计
type TaskStats struct {
	TotalFiles        int     `json:"total_files"`        // 总文件数
	StaticFiles       int     `json:"static_files"`       // 静态爬取文件数
	DynamicFiles      int     `json:"dynamic_files"`      // 动态爬取文件数
	MapFiles          int     `json:"map_files"`          // Source Map文件数
	FailedFiles       int     `json:"failed_files"`       // 失败文件数
	DeobfuscatedFiles int     `json:"deobfuscated_files"` // 反混淆文件数
	TotalSize         int64   `json:"total_size"`         // 总大小(字节)
	Duration          float64 `json:"duration"`           // 总耗时(秒)
	VisitedURLs       int     `json:"visited_urls"`       // 已访问URL数
}

// CrawlConfig 爬取配置
type CrawlConfig struct {
	Depth               int     `json:"depth"`                // 爬取深度 (默认:2)
	WaitTime            int     `json:"wait_time"`            // 页面等待时间(秒) (默认:3)
	MaxWorkers          int     `json:"max_workers"`          // 静态爬取并发线程数 (默认:2)
	PlaywrightTabs      int     `json:"playwright_tabs"`      // Playwright标签页数量 (默认:4)
	Headless            bool    `json:"headless"`             // 无头模式 (默认:true)
	Resume              bool    `json:"resume"`               // 是否从检查点恢复
	SimilarityEnabled   bool    `json:"similarity_enabled"`   // 启用相似度检测 (默认:true)
	SimilarityThreshold float64 `json:"similarity_threshold"` // 相似度阈值 (默认:0.8)
	SimilarityWorkers   int     `json:"similarity_workers"`   // 相似度分析并发数
}

// Validate 验证配置
func (c *CrawlConfig) Validate() error {
	if c.Depth < 1 || c.Depth > 10 {
		return fmt.Errorf("深度必须在1-10之间")
	}
	if c.WaitTime < 0 || c.WaitTime > 60 {
		return fmt.Errorf("等待时间必须在0-60秒之间")
	}
	if c.MaxWorkers < 1 || c.MaxWorkers > 100 {
		return fmt.Errorf("并发数必须在1-100之间")
	}
	if c.PlaywrightTabs < 1 || c.PlaywrightTabs > 20 {
		return fmt.Errorf("标签页数必须在1-20之间")
	}
	if c.SimilarityThreshold < 0.0 || c.SimilarityThreshold > 1.0 {
		return fmt.Errorf("相似度阈值必须在0.0-1.0之间")
	}
	return nil
}

// CrawlTask 爬取任务
type CrawlTask struct {
	// 基本信息
	ID          string     `json:"id"`                     // 任务唯一ID (UUID)
	TargetURL   string     `json:"target_url"`             // 目标URL
	Domain      string     `json:"domain"`                 // 解析的域名
	CreatedAt   time.Time  `json:"created_at"`             // 创建时间
	StartedAt   *time.Time `json:"started_at,omitempty"`   // 开始时间
	CompletedAt *time.Time `json:"completed_at,omitempty"` // 完成时间

	// 配置参数
	Config CrawlConfig `json:"config"` // 爬取配置

	// 执行状态
	Status       TaskStatus `json:"status"`        // 任务状态
	Mode         CrawlMode  `json:"mode"`          // 爬取模式
	CurrentDepth int        `json:"current_depth"` // 当前深度

	// 统计信息
	Stats TaskStats `json:"stats"` // 任务统计

	// 错误信息
	ErrorMessage string `json:"error_message,omitempty"` // 错误消息
}

// NewCrawlTask 创建新任务
func NewCrawlTask(targetURL string, config CrawlConfig) (*CrawlTask, error) {
	if err := ValidateURL(targetURL); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	parsed, _ := url.Parse(targetURL)
	domain := parsed.Host

	return &CrawlTask{
		ID:           generateID(),
		TargetURL:    targetURL,
		Domain:       domain,
		CreatedAt:    time.Now(),
		Config:       config,
		Status:       TaskStatusPending,
		Mode:         ModeAll,
		CurrentDepth: 0,
		Stats:        TaskStats{},
	}, nil
}

// ToJSON 序列化为JSON
func (t *CrawlTask) ToJSON() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}

// FromJSON 从JSON反序列化
func (t *CrawlTask) FromJSON(data []byte) error {
	return json.Unmarshal(data, t)
}

// BatchCrawlTask 批量爬取任务
type BatchCrawlTask struct {
	// 基本信息
	ID          string     `json:"id"`
	URLsFile    string     `json:"urls_file"`              // URL列表文件路径
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// 配置
	Config          CrawlConfig `json:"config"`             // 爬取配置
	BatchDelay      int         `json:"batch_delay"`        // URL之间延迟(秒)
	ContinueOnError bool        `json:"continue_on_error"`  // 遇到错误继续

	// 状态
	Status TaskStatus `json:"status"`

	// 统计
	TotalURLs      int   `json:"total_urls"`
	SuccessfulURLs int   `json:"successful_urls"`
	FailedURLs     int   `json:"failed_urls"`
	TotalFiles     int   `json:"total_files"`
	TotalSize      int64 `json:"total_size"`

	// 子任务
	SubTasks []string `json:"sub_tasks"` // 子任务ID列表
}
