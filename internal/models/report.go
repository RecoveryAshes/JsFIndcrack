package models

import (
	"encoding/json"
	"time"
)

// CrawlReport 爬取报告
type CrawlReport struct {
	// 任务信息
	TaskID    string    `json:"task_id"`
	TargetURL string    `json:"target_url"`
	Domain    string    `json:"domain"`
	Mode      CrawlMode `json:"mode"`

	// 时间信息
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  float64   `json:"duration"` // 秒

	// 统计信息
	Stats TaskStats `json:"stats"`

	// 文件列表
	SuccessFiles []FileInfo       `json:"success_files"` // 成功下载的文件
	FailedFiles  []FailedFileInfo `json:"failed_files"`  // 失败文件

	// 分析结果
	SimilarityAnalysis *SimilarityAnalysisResult `json:"similarity_analysis,omitempty"`

	// 输出路径
	OutputDir string `json:"output_dir"` // 输出目录
	EncodeDir string `json:"encode_dir"` // 原始文件目录
	DecodeDir string `json:"decode_dir"` // 反混淆文件目录

	// 配置快照
	Config CrawlConfig `json:"config"`
}

// FileInfo 文件信息
type FileInfo struct {
	URL          string    `json:"url"`
	FilePath     string    `json:"file_path"`
	Size         int64     `json:"size"`
	Hash         string    `json:"hash"`
	CrawlMode    CrawlMode `json:"crawl_mode"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

// FailedFileInfo 失败文件信息
type FailedFileInfo struct {
	URL       string `json:"url"`
	ErrorType string `json:"error_type"` // timeout, network_error, invalid_content等
	ErrorMsg  string `json:"error_msg"`
	Retries   int    `json:"retries"`
}

// SimilarityAnalysisResult 相似度分析结果
type SimilarityAnalysisResult struct {
	Enabled          bool              `json:"enabled"`
	TotalFiles       int               `json:"total_files"`
	DuplicateGroups  []SimilarityGroup `json:"duplicate_groups"`
	UniqueFiles      int               `json:"unique_files"`
	DuplicateFiles   int               `json:"duplicate_files"`
	SpaceSaved       int64             `json:"space_saved"`       // 字节
	AnalysisDuration float64           `json:"analysis_duration"` // 秒
}

// ToJSON 序列化为JSON
func (r *CrawlReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// FromJSON 从JSON反序列化
func (r *CrawlReport) FromJSON(data []byte) error {
	return json.Unmarshal(data, r)
}
