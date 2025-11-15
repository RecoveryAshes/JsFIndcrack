package models

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Checkpoint 检查点
type Checkpoint struct {
	// 任务信息
	TaskID    string `json:"task_id"`    // 关联的任务ID
	TargetURL string `json:"target_url"` // 目标URL

	// 进度信息
	VisitedURLs     []string `json:"visited_urls"`      // 已访问URL列表
	DownloadedFiles []string `json:"downloaded_files"`  // 已下载文件URL列表
	FailedURLs      []string `json:"failed_urls"`       // 失败URL列表
	PendingURLs     []string `json:"pending_urls"`      // 待处理URL列表
	CurrentDepth    int      `json:"current_depth"`     // 当前深度

	// 统计信息
	Stats TaskStats `json:"stats"` // 当前统计

	// 时间戳
	CreatedAt time.Time `json:"created_at"` // 检查点创建时间
	UpdatedAt time.Time `json:"updated_at"` // 最后更新时间

	// 配置快照
	Config CrawlConfig `json:"config"` // 配置快照
}

// CheckpointFilename 生成检查点文件名
func CheckpointFilename(domain string) string {
	return fmt.Sprintf("checkpoint_%s.json", domain)
}

// ToJSON 序列化为JSON
func (c *Checkpoint) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// FromJSON 从JSON反序列化
func (c *Checkpoint) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

// SaveToFile 保存到文件
func (c *Checkpoint) SaveToFile(filepath string) error {
	data, err := c.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, data, 0644)
}

// LoadFromFile 从文件加载
func LoadCheckpointFromFile(filepath string) (*Checkpoint, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var cp Checkpoint
	if err := cp.FromJSON(data); err != nil {
		return nil, err
	}

	return &cp, nil
}
