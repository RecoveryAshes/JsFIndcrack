# 数据模型设计

**项目**: JsFIndcrack - Python到Go迁移
**分支**: 001-py-to-go-migration
**日期**: 2025-11-15
**版本**: 1.0

---

## 概述

本文档定义JsFIndcrack Go版本的核心数据模型,包括实体定义、字段说明、关系映射、验证规则和状态转换。所有模型基于现有Python版本的数据结构,保持向后兼容的同时利用Go的类型安全特性。

---

## 核心实体

### 1. CrawlTask (爬取任务)

**描述**: 表示一个完整的爬取操作,包含目标URL、配置参数和执行状态。

**Go结构体定义**:

```go
package models

import (
    "time"
)

// CrawlTask 爬取任务
type CrawlTask struct {
    // 基本信息
    ID          string    `json:"id"`           // 任务唯一ID (UUID)
    TargetURL   string    `json:"target_url"`   // 目标URL
    Domain      string    `json:"domain"`       // 解析的域名
    CreatedAt   time.Time `json:"created_at"`   // 创建时间
    StartedAt   *time.Time `json:"started_at,omitempty"`  // 开始时间
    CompletedAt *time.Time `json:"completed_at,omitempty"` // 完成时间

    // 配置参数
    Config CrawlConfig `json:"config"` // 爬取配置

    // 执行状态
    Status      TaskStatus `json:"status"`      // 任务状态
    Mode        CrawlMode  `json:"mode"`        // 爬取模式
    CurrentDepth int       `json:"current_depth"` // 当前深度

    // 统计信息
    Stats TaskStats `json:"stats"` // 任务统计

    // 错误信息
    ErrorMessage string `json:"error_message,omitempty"` // 错误消息
}

// CrawlConfig 爬取配置
type CrawlConfig struct {
    Depth            int    `json:"depth"`              // 爬取深度 (默认:2)
    WaitTime         int    `json:"wait_time"`          // 页面等待时间(秒) (默认:3)
    MaxWorkers       int    `json:"max_workers"`        // 静态爬取并发线程数 (默认:2)
    PlaywrightTabs   int    `json:"playwright_tabs"`    // Playwright标签页数量 (默认:4)
    Headless         bool   `json:"headless"`           // 无头模式 (默认:true)
    Resume           bool   `json:"resume"`             // 是否从检查点恢复
    SimilarityEnabled bool  `json:"similarity_enabled"` // 启用相似度检测 (默认:true)
    SimilarityThreshold float64 `json:"similarity_threshold"` // 相似度阈值 (默认:0.8)
    SimilarityWorkers int   `json:"similarity_workers"` // 相似度分析并发数
}

// TaskStatus 任务状态
type TaskStatus string

const (
    TaskStatusPending    TaskStatus = "pending"     // 待执行
    TaskStatusRunning    TaskStatus = "running"     // 执行中
    TaskStatusCompleted  TaskStatus = "completed"   // 已完成
    TaskStatusFailed     TaskStatus = "failed"      // 失败
    TaskStatusCancelled  TaskStatus = "cancelled"   // 已取消
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
    TotalFiles      int     `json:"total_files"`       // 总文件数
    StaticFiles     int     `json:"static_files"`      // 静态爬取文件数
    DynamicFiles    int     `json:"dynamic_files"`     // 动态爬取文件数
    MapFiles        int     `json:"map_files"`         // Source Map文件数
    FailedFiles     int     `json:"failed_files"`      // 失败文件数
    DeobfuscatedFiles int   `json:"deobfuscated_files"` // 反混淆文件数
    TotalSize       int64   `json:"total_size"`        // 总大小(字节)
    Duration        float64 `json:"duration"`          // 总耗时(秒)
    VisitedURLs     int     `json:"visited_urls"`      // 已访问URL数
}
```

**字段验证规则**:

- `TargetURL`: 必填,必须是有效的HTTP/HTTPS URL
- `Depth`: 范围 [1, 10]
- `WaitTime`: 范围 [0, 60]
- `MaxWorkers`: 范围 [1, 100]
- `PlaywrightTabs`: 范围 [1, 20]
- `SimilarityThreshold`: 范围 [0.0, 1.0]

**状态转换**:

```
pending → running → completed
              ↓
            failed
              ↓
          cancelled
```

---

### 2. JSFile (JavaScript文件)

**描述**: 表示从网站下载的JavaScript源文件,包含元数据和内容信息。

**Go结构体定义**:

```go
package models

import (
    "time"
)

// JSFile JavaScript文件
type JSFile struct {
    // 标识信息
    ID       string    `json:"id"`        // 文件唯一ID
    URL      string    `json:"url"`       // 文件URL
    FilePath string    `json:"file_path"` // 本地存储路径

    // 元数据
    Hash       string    `json:"hash"`        // SHA-256哈希值
    Size       int64     `json:"size"`        // 文件大小(字节)
    Extension  string    `json:"extension"`   // 文件扩展名(.js, .mjs, .jsx)
    ContentType string   `json:"content_type"` // HTTP Content-Type

    // 来源信息
    SourceURL  string    `json:"source_url"`  // 发现该文件的页面URL
    CrawlMode  CrawlMode `json:"crawl_mode"`  // 爬取模式(static/dynamic)
    Depth      int       `json:"depth"`       // 爬取深度

    // 状态标记
    IsObfuscated bool      `json:"is_obfuscated"` // 是否混淆
    IsDeobfuscated bool    `json:"is_deobfuscated"` // 是否已反混淆
    IsDuplicate  bool      `json:"is_duplicate"`  // 是否重复

    // 时间戳
    DownloadedAt time.Time `json:"downloaded_at"` // 下载时间
    ProcessedAt  *time.Time `json:"processed_at,omitempty"` // 处理时间

    // 关联信息
    MapFileURL   string    `json:"map_file_url,omitempty"` // 关联的Source Map URL
    HasMapFile   bool      `json:"has_map_file"`           // 是否有Source Map
}

// JSFileExtension 支持的JavaScript文件扩展名
var JSFileExtensions = []string{".js", ".mjs", ".jsx"}

// IsValidExtension 检查扩展名是否有效
func (f *JSFile) IsValidExtension() bool {
    for _, ext := range JSFileExtensions {
        if f.Extension == ext {
            return true
        }
    }
    return false
}
```

**字段验证规则**:

- `URL`: 必填,必须是有效URL
- `Hash`: 必填,64字符十六进制字符串(SHA-256)
- `Size`: 必须 > 0 且 <= 50MB (52,428,800字节)
- `Extension`: 必须是 `.js`, `.mjs`, 或 `.jsx`

---

### 3. MapFile (Source Map文件)

**描述**: 表示JavaScript对应的Source Map文件。

**Go结构体定义**:

```go
package models

import (
    "time"
)

// MapFile Source Map文件
type MapFile struct {
    // 标识信息
    ID       string    `json:"id"`
    URL      string    `json:"url"`
    FilePath string    `json:"file_path"`

    // 元数据
    Hash      string    `json:"hash"`
    Size      int64     `json:"size"`
    Extension string    `json:"extension"` // .map, .js.map

    // 关联信息
    JSFileURL string    `json:"js_file_url"` // 关联的JS文件URL
    JSFileID  string    `json:"js_file_id"`  // 关联的JS文件ID

    // 时间戳
    DownloadedAt time.Time `json:"downloaded_at"`
}

// MapFileExtensions 支持的Source Map文件扩展名
var MapFileExtensions = []string{".map", ".js.map"}
```

---

### 4. Checkpoint (检查点)

**描述**: 表示爬取任务的中间状态,用于断点续爬。

**Go结构体定义**:

```go
package models

import (
    "time"
)

// Checkpoint 检查点
type Checkpoint struct {
    // 任务信息
    TaskID    string    `json:"task_id"`     // 关联的任务ID
    TargetURL string    `json:"target_url"`  // 目标URL

    // 进度信息
    VisitedURLs    []string `json:"visited_urls"`     // 已访问URL列表
    DownloadedFiles []string `json:"downloaded_files"` // 已下载文件URL列表
    FailedURLs     []string `json:"failed_urls"`      // 失败URL列表
    PendingURLs    []string `json:"pending_urls"`     // 待处理URL列表
    CurrentDepth   int      `json:"current_depth"`    // 当前深度

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
```

**持久化**:
- 存储格式: JSON
- 存储位置: `output/[domain]/checkpoints/crawler_checkpoint.json`
- 更新频率: 每处理10个文件或每分钟(取较快者)

---

### 5. SimilarityGroup (相似度组)

**描述**: 表示一组相似或重复的JavaScript文件。

**Go结构体定义**:

```go
package models

// SimilarityGroup 相似度组
type SimilarityGroup struct {
    // 组标识
    GroupID      string    `json:"group_id"`      // 组唯一ID
    RepresentFile string   `json:"represent_file"` // 代表文件URL

    // 成员信息
    Members      []SimilarityMember `json:"members"`       // 组成员
    MemberCount  int                `json:"member_count"`  // 成员数量

    // 相似度信息
    AvgSimilarity float64 `json:"avg_similarity"` // 平均相似度
    MinSimilarity float64 `json:"min_similarity"` // 最小相似度
    MaxSimilarity float64 `json:"max_similarity"` // 最大相似度

    // 去重建议
    DuplicateFiles []string `json:"duplicate_files"` // 建议删除的重复文件
    TotalSavedSize int64    `json:"total_saved_size"` // 去重后节省空间(字节)
}

// SimilarityMember 相似度组成员
type SimilarityMember struct {
    FileURL    string  `json:"file_url"`    // 文件URL
    FilePath   string  `json:"file_path"`   // 本地路径
    FileSize   int64   `json:"file_size"`   // 文件大小
    Similarity float64 `json:"similarity"`  // 与代表文件的相似度
}

// SimilarityMatrix 相似度矩阵
type SimilarityMatrix struct {
    Files   []string      `json:"files"`   // 文件URL列表
    Matrix  [][]float64   `json:"matrix"`  // NxN相似度矩阵
}
```

---

### 6. CrawlReport (爬取报告)

**描述**: 爬取任务完成后的详细报告。

**Go结构体定义**:

```go
package models

import (
    "time"
)

// CrawlReport 爬取报告
type CrawlReport struct {
    // 任务信息
    TaskID      string    `json:"task_id"`
    TargetURL   string    `json:"target_url"`
    Domain      string    `json:"domain"`
    Mode        CrawlMode `json:"mode"`

    // 时间信息
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    Duration    float64   `json:"duration"` // 秒

    // 统计信息
    Stats       TaskStats `json:"stats"`

    // 文件列表
    SuccessFiles []FileInfo `json:"success_files"` // 成功下载的文件
    FailedFiles  []FailedFileInfo `json:"failed_files"` // 失败文件

    // 分析结果
    SimilarityAnalysis *SimilarityAnalysisResult `json:"similarity_analysis,omitempty"`

    // 输出路径
    OutputDir   string   `json:"output_dir"`   // 输出目录
    EncodeDir   string   `json:"encode_dir"`   // 原始文件目录
    DecodeDir   string   `json:"decode_dir"`   // 反混淆文件目录

    // 配置快照
    Config      CrawlConfig `json:"config"`
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
    Enabled         bool              `json:"enabled"`
    TotalFiles      int               `json:"total_files"`
    DuplicateGroups []SimilarityGroup `json:"duplicate_groups"`
    UniqueFiles     int               `json:"unique_files"`
    DuplicateFiles  int               `json:"duplicate_files"`
    SpaceSaved      int64             `json:"space_saved"` // 字节
    AnalysisDuration float64          `json:"analysis_duration"` // 秒
}
```

**输出格式**: JSON文件
**存储位置**:
- 完整报告: `output/[domain]/crawl_report.json`
- 成功文件列表: `output/[domain]/success_files.json`
- 失败文件列表: `output/[domain]/failed_files.json`
- 摘要报告: `output/[domain]/crawl_summary.json`

---

### 7. BatchCrawlTask (批量爬取任务)

**描述**: 批量处理多个URL的任务。

**Go结构体定义**:

```go
package models

import (
    "time"
)

// BatchCrawlTask 批量爬取任务
type BatchCrawlTask struct {
    // 基本信息
    ID          string    `json:"id"`
    URLsFile    string    `json:"urls_file"`    // URL列表文件路径
    CreatedAt   time.Time `json:"created_at"`
    StartedAt   *time.Time `json:"started_at,omitempty"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`

    // 配置
    Config      CrawlConfig `json:"config"`       // 爬取配置
    BatchDelay  int         `json:"batch_delay"`  // URL之间延迟(秒)
    ContinueOnError bool    `json:"continue_on_error"` // 遇到错误继续

    // 状态
    Status      TaskStatus  `json:"status"`

    // 统计
    TotalURLs      int `json:"total_urls"`
    SuccessfulURLs int `json:"successful_urls"`
    FailedURLs     int `json:"failed_urls"`
    TotalFiles     int `json:"total_files"`
    TotalSize      int64 `json:"total_size"`

    // 子任务
    SubTasks    []string `json:"sub_tasks"` // 子任务ID列表
}
```

---

## 实体关系图

```
┌─────────────────┐
│  BatchCrawlTask │
└────────┬────────┘
         │ 1:N
         ▼
┌─────────────────┐       ┌──────────────┐
│   CrawlTask     │◄──────┤  Checkpoint  │
└────────┬────────┘ 1:1   └──────────────┘
         │ 1:N
         ▼
┌─────────────────┐       ┌──────────────┐
│     JSFile      │◄─────►│   MapFile    │
└────────┬────────┘ 1:0..1 └──────────────┘
         │ N:M
         ▼
┌─────────────────┐       ┌──────────────┐
│SimilarityGroup  │◄──────┤ CrawlReport  │
└─────────────────┘ 1:N   └──────────────┘
```

---

## 数据流转

### 1. 单URL爬取流程

```
CrawlTask创建
    ↓
加载Checkpoint(如resume=true)
    ↓
静态爬取 → 生成JSFile/MapFile
    ↓
动态爬取 → 生成JSFile/MapFile
    ↓
去重检查(基于Hash)
    ↓
相似度分析 → 生成SimilarityGroup
    ↓
反混淆处理
    ↓
生成CrawlReport
    ↓
保存Checkpoint
```

### 2. 批量爬取流程

```
BatchCrawlTask创建
    ↓
解析URL列表
    ↓
For each URL:
    创建CrawlTask
    ↓
    执行单URL流程
    ↓
    延迟batch_delay秒
    ↓
汇总统计
    ↓
生成批量报告
```

---

## JSON序列化示例

### CrawlTask示例

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "target_url": "https://example.com",
  "domain": "example.com",
  "created_at": "2025-11-15T10:00:00Z",
  "started_at": "2025-11-15T10:00:05Z",
  "completed_at": "2025-11-15T10:05:30Z",
  "config": {
    "depth": 2,
    "wait_time": 3,
    "max_workers": 4,
    "playwright_tabs": 6,
    "headless": true,
    "resume": false,
    "similarity_enabled": true,
    "similarity_threshold": 0.8,
    "similarity_workers": 8
  },
  "status": "completed",
  "mode": "all",
  "current_depth": 2,
  "stats": {
    "total_files": 156,
    "static_files": 89,
    "dynamic_files": 67,
    "map_files": 23,
    "failed_files": 3,
    "deobfuscated_files": 45,
    "total_size": 12458960,
    "duration": 325.5,
    "visited_urls": 234
  }
}
```

### CrawlReport示例

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "target_url": "https://example.com",
  "domain": "example.com",
  "mode": "all",
  "start_time": "2025-11-15T10:00:05Z",
  "end_time": "2025-11-15T10:05:30Z",
  "duration": 325.5,
  "stats": {
    "total_files": 156,
    "static_files": 89,
    "dynamic_files": 67,
    "map_files": 23,
    "failed_files": 3,
    "deobfuscated_files": 45,
    "total_size": 12458960,
    "duration": 325.5,
    "visited_urls": 234
  },
  "similarity_analysis": {
    "enabled": true,
    "total_files": 156,
    "duplicate_groups": 12,
    "unique_files": 120,
    "duplicate_files": 36,
    "space_saved": 3458960,
    "analysis_duration": 15.2
  },
  "output_dir": "output/example.com",
  "encode_dir": "output/example.com/encode",
  "decode_dir": "output/example.com/decode"
}
```

---

## 验证与约束

### 1. URL验证

```go
func ValidateURL(url string) error {
    parsed, err := url.Parse(url)
    if err != nil {
        return fmt.Errorf("无效的URL: %w", err)
    }
    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return fmt.Errorf("URL必须是HTTP或HTTPS协议")
    }
    return nil
}
```

### 2. 配置验证

```go
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
```

### 3. 文件大小约束

```go
const (
    MaxFileSize = 50 * 1024 * 1024 // 50MB
)

func (f *JSFile) ValidateSize() error {
    if f.Size <= 0 {
        return fmt.Errorf("文件大小必须大于0")
    }
    if f.Size > MaxFileSize {
        return fmt.Errorf("文件大小超过限制: %d > %d", f.Size, MaxFileSize)
    }
    return nil
}
```

---

## 与Python版本的兼容性

### JSON格式兼容

所有JSON输出格式保持与Python版本一致,确保:

1. **字段名称**: 使用snake_case命名(通过json tag实现)
2. **字段类型**: 保持一致(字符串、数字、布尔值)
3. **时间格式**: 使用ISO 8601格式(RFC3339)
4. **文件路径**: 使用相对路径,保持跨平台兼容

### 检查点互操作性

Go版本可以读取Python版本生成的检查点文件,反之亦然:

```go
// 从Python检查点恢复
func LoadCheckpointFromPython(filepath string) (*Checkpoint, error) {
    data, err := os.ReadFile(filepath)
    if err != nil {
        return nil, err
    }

    var cp Checkpoint
    if err := json.Unmarshal(data, &cp); err != nil {
        return nil, err
    }

    return &cp, nil
}
```

---

## 实现建议

### 1. 包结构

```go
internal/models/
├── task.go           // CrawlTask, BatchCrawlTask
├── file.go           // JSFile, MapFile
├── checkpoint.go     // Checkpoint
├── similarity.go     // SimilarityGroup, SimilarityMatrix
├── report.go         // CrawlReport
├── validation.go     // 验证函数
└── constants.go      // 常量定义
```

### 2. 序列化辅助

```go
// ToJSON 序列化为JSON
func (t *CrawlTask) ToJSON() ([]byte, error) {
    return json.MarshalIndent(t, "", "  ")
}

// FromJSON 从JSON反序列化
func (t *CrawlTask) FromJSON(data []byte) error {
    return json.Unmarshal(data, t)
}

// SaveToFile 保存到文件
func (t *CrawlTask) SaveToFile(filepath string) error {
    data, err := t.ToJSON()
    if err != nil {
        return err
    }
    return os.WriteFile(filepath, data, 0644)
}
```

### 3. 构造函数

```go
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
        ID:          uuid.New().String(),
        TargetURL:   targetURL,
        Domain:      domain,
        CreatedAt:   time.Now(),
        Config:      config,
        Status:      TaskStatusPending,
        Mode:        ModeAll,
        CurrentDepth: 0,
        Stats:       TaskStats{},
    }, nil
}
```

---

## 总结

本数据模型设计:

1. ✅ **类型安全**: 利用Go的强类型系统,避免运行时错误
2. ✅ **向后兼容**: JSON格式与Python版本完全兼容
3. ✅ **可扩展**: 支持新增字段而不破坏现有功能
4. ✅ **验证完整**: 所有关键字段都有验证规则
5. ✅ **文档完善**: 每个实体都有清晰的说明和示例

**下一步**: 基于这些数据模型,设计命令行接口契约(contracts/)。
