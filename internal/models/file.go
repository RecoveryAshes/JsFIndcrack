package models

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// MaxFileSize 最大文件大小 50MB
	MaxFileSize = 50 * 1024 * 1024
)

// JSFileExtensions 支持的JavaScript文件扩展名
var JSFileExtensions = []string{".js", ".mjs", ".jsx"}

// MapFileExtensions 支持的Source Map文件扩展名
var MapFileExtensions = []string{".map", ".js.map"}

// JSFile JavaScript文件
type JSFile struct {
	// 标识信息
	ID       string `json:"id"`        // 文件唯一ID
	URL      string `json:"url"`       // 文件URL
	FilePath string `json:"file_path"` // 本地存储路径

	// 元数据
	Hash        string `json:"hash"`         // SHA-256哈希值
	Size        int64  `json:"size"`         // 文件大小(字节)
	Extension   string `json:"extension"`    // 文件扩展名(.js, .mjs, .jsx)
	ContentType string `json:"content_type"` // HTTP Content-Type

	// 来源信息
	SourceURL string    `json:"source_url"` // 发现该文件的页面URL
	CrawlMode CrawlMode `json:"crawl_mode"` // 爬取模式(static/dynamic)
	Depth     int       `json:"depth"`      // 爬取深度

	// 状态标记
	IsObfuscated   bool `json:"is_obfuscated"`    // 是否混淆
	IsDeobfuscated bool `json:"is_deobfuscated"`  // 是否已反混淆
	IsDuplicate    bool `json:"is_duplicate"`     // 是否重复

	// 时间戳
	DownloadedAt time.Time  `json:"downloaded_at"`            // 下载时间
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`   // 处理时间

	// 关联信息
	MapFileURL string `json:"map_file_url,omitempty"` // 关联的Source Map URL
	HasMapFile bool   `json:"has_map_file"`           // 是否有Source Map
}

// IsValidExtension 检查扩展名是否有效
func (f *JSFile) IsValidExtension() bool {
	for _, ext := range JSFileExtensions {
		if f.Extension == ext {
			return true
		}
	}
	return false
}

// ValidateSize 验证文件大小
func (f *JSFile) ValidateSize() error {
	if f.Size <= 0 {
		return fmt.Errorf("文件大小必须大于0")
	}
	if f.Size > MaxFileSize {
		return fmt.Errorf("文件大小超过限制: %d > %d", f.Size, MaxFileSize)
	}
	return nil
}

// ToJSON 序列化为JSON
func (f *JSFile) ToJSON() ([]byte, error) {
	return json.MarshalIndent(f, "", "  ")
}

// MapFile Source Map文件
type MapFile struct {
	// 标识信息
	ID       string `json:"id"`
	URL      string `json:"url"`
	FilePath string `json:"file_path"`

	// 元数据
	Hash      string `json:"hash"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"` // .map, .js.map

	// 关联信息
	JSFileURL string `json:"js_file_url"` // 关联的JS文件URL
	JSFileID  string `json:"js_file_id"`  // 关联的JS文件ID

	// 时间戳
	DownloadedAt time.Time `json:"downloaded_at"`
}

// ToJSON 序列化为JSON
func (m *MapFile) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}
