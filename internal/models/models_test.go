package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"有效的HTTP URL", "http://example.com", false},
		{"有效的HTTPS URL", "https://example.com", false},
		{"带路径的URL", "https://example.com/path/to/resource", false},
		{"无效的协议", "ftp://example.com", true},
		{"无效的URL", "not a url", true},
		{"空URL", "", true},
		{"无协议", "example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCrawlConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CrawlConfig
		wantErr bool
	}{
		{
			name: "有效配置",
			config: CrawlConfig{
				Depth:               2,
				WaitTime:            3,
				MaxWorkers:          4,
				PlaywrightTabs:      6,
				SimilarityThreshold: 0.8,
			},
			wantErr: false,
		},
		{
			name: "深度过小",
			config: CrawlConfig{
				Depth:      0,
				WaitTime:   3,
				MaxWorkers: 4,
			},
			wantErr: true,
		},
		{
			name: "深度过大",
			config: CrawlConfig{
				Depth:      11,
				WaitTime:   3,
				MaxWorkers: 4,
			},
			wantErr: true,
		},
		{
			name: "相似度阈值无效",
			config: CrawlConfig{
				Depth:               2,
				WaitTime:            3,
				MaxWorkers:          4,
				PlaywrightTabs:      6,
				SimilarityThreshold: 1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewCrawlTask(t *testing.T) {
	config := CrawlConfig{
		Depth:               2,
		WaitTime:            3,
		MaxWorkers:          4,
		PlaywrightTabs:      6,
		SimilarityThreshold: 0.8,
	}

	task, err := NewCrawlTask("https://example.com", config)
	if err != nil {
		t.Fatalf("NewCrawlTask() error = %v", err)
	}

	if task.ID == "" {
		t.Error("任务ID不应为空")
	}

	if task.TargetURL != "https://example.com" {
		t.Errorf("TargetURL = %v, want %v", task.TargetURL, "https://example.com")
	}

	if task.Domain != "example.com" {
		t.Errorf("Domain = %v, want %v", task.Domain, "example.com")
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Status = %v, want %v", task.Status, TaskStatusPending)
	}

	if task.Mode != ModeAll {
		t.Errorf("Mode = %v, want %v", task.Mode, ModeAll)
	}
}

func TestCrawlTask_JSON(t *testing.T) {
	config := CrawlConfig{
		Depth:               2,
		WaitTime:            3,
		MaxWorkers:          4,
		PlaywrightTabs:      6,
		SimilarityThreshold: 0.8,
	}

	task, err := NewCrawlTask("https://example.com", config)
	if err != nil {
		t.Fatalf("NewCrawlTask() error = %v", err)
	}

	// 测试ToJSON
	jsonData, err := task.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON数据不应为空")
	}

	// 测试FromJSON
	var decoded CrawlTask
	err = decoded.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if decoded.ID != task.ID {
		t.Errorf("解码后的ID不匹配: got %v, want %v", decoded.ID, task.ID)
	}

	if decoded.TargetURL != task.TargetURL {
		t.Errorf("解码后的TargetURL不匹配: got %v, want %v", decoded.TargetURL, task.TargetURL)
	}
}

func TestJSFile_ValidateSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"正常大小", 1024 * 1024, false}, // 1MB
		{"最大大小", MaxFileSize, false}, // 50MB
		{"零大小", 0, true},
		{"负数大小", -1, true},
		{"超过最大", MaxFileSize + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &JSFile{Size: tt.size}
			err := file.ValidateSize()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSFile_IsValidExtension(t *testing.T) {
	tests := []struct {
		name      string
		extension string
		want      bool
	}{
		{".js扩展名", ".js", true},
		{".mjs扩展名", ".mjs", true},
		{".jsx扩展名", ".jsx", true},
		{"无效扩展名", ".txt", false},
		{"空扩展名", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &JSFile{Extension: tt.extension}
			if got := file.IsValidExtension(); got != tt.want {
				t.Errorf("IsValidExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckpoint_SaveAndLoad(t *testing.T) {
	// 创建临时文件
	tempDir := t.TempDir()
	filepath := tempDir + "/test_checkpoint.json"

	// 创建检查点
	checkpoint := &Checkpoint{
		TaskID:    "test-task-123",
		TargetURL: "https://example.com",
		VisitedURLs: []string{
			"https://example.com",
			"https://example.com/page1",
		},
		DownloadedFiles: []string{
			"https://example.com/app.js",
		},
		CurrentDepth: 2,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Config: CrawlConfig{
			Depth:      2,
			WaitTime:   3,
			MaxWorkers: 4,
		},
	}

	// 测试保存
	err := checkpoint.SaveToFile(filepath)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// 测试加载
	loaded, err := LoadCheckpointFromFile(filepath)
	if err != nil {
		t.Fatalf("LoadCheckpointFromFile() error = %v", err)
	}

	// 验证数据
	if loaded.TaskID != checkpoint.TaskID {
		t.Errorf("TaskID不匹配: got %v, want %v", loaded.TaskID, checkpoint.TaskID)
	}

	if loaded.TargetURL != checkpoint.TargetURL {
		t.Errorf("TargetURL不匹配: got %v, want %v", loaded.TargetURL, checkpoint.TargetURL)
	}

	if len(loaded.VisitedURLs) != len(checkpoint.VisitedURLs) {
		t.Errorf("VisitedURLs长度不匹配: got %v, want %v", len(loaded.VisitedURLs), len(checkpoint.VisitedURLs))
	}
}

func TestSimilarityGroup_JSON(t *testing.T) {
	group := &SimilarityGroup{
		GroupID:       "group-1",
		RepresentFile: "https://example.com/app.js",
		Members: []SimilarityMember{
			{
				FileURL:    "https://example.com/app.js",
				FilePath:   "/output/example.com/app.js",
				FileSize:   1024,
				Similarity: 1.0,
			},
			{
				FileURL:    "https://example.com/app.min.js",
				FilePath:   "/output/example.com/app.min.js",
				FileSize:   512,
				Similarity: 0.95,
			},
		},
		MemberCount:   2,
		AvgSimilarity: 0.975,
	}

	// 测试JSON序列化
	jsonData, err := group.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// 验证可以解码
	var decoded SimilarityGroup
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.GroupID != group.GroupID {
		t.Errorf("GroupID不匹配: got %v, want %v", decoded.GroupID, group.GroupID)
	}

	if len(decoded.Members) != len(group.Members) {
		t.Errorf("Members长度不匹配: got %v, want %v", len(decoded.Members), len(group.Members))
	}
}

func TestCrawlReport_JSON(t *testing.T) {
	report := &CrawlReport{
		TaskID:    "task-123",
		TargetURL: "https://example.com",
		Domain:    "example.com",
		Mode:      ModeAll,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Minute),
		Duration:  300.5,
		Stats: TaskStats{
			TotalFiles:   156,
			StaticFiles:  89,
			DynamicFiles: 67,
		},
		OutputDir: "/output/example.com",
		EncodeDir: "/output/example.com/encode",
		DecodeDir: "/output/example.com/decode",
		Config: CrawlConfig{
			Depth:      2,
			WaitTime:   3,
			MaxWorkers: 4,
		},
	}

	// 测试JSON序列化
	jsonData, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// 测试JSON反序列化
	var decoded CrawlReport
	err = decoded.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if decoded.TaskID != report.TaskID {
		t.Errorf("TaskID不匹配: got %v, want %v", decoded.TaskID, report.TaskID)
	}

	if decoded.Stats.TotalFiles != report.Stats.TotalFiles {
		t.Errorf("TotalFiles不匹配: got %v, want %v", decoded.Stats.TotalFiles, report.Stats.TotalFiles)
	}
}
