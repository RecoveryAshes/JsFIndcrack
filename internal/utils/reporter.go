package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/schollz/progressbar/v3"
)

// Reporter 报告生成器
type Reporter struct {
	outputDir string
	domain    string
}

// NewReporter 创建报告生成器
func NewReporter(outputDir string, domain string) *Reporter {
	return &Reporter{
		outputDir: outputDir,
		domain:    domain,
	}
}

// GenerateReport 生成爬取报告
func (r *Reporter) GenerateReport(
	targetURL string,
	stats models.TaskStats,
	successFiles []*models.JSFile,
	failedFiles []string,
	config models.CrawlConfig,
) error {
	reportsDir := filepath.Join(r.outputDir, r.domain, "reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("创建报告目录失败: %w", err)
	}

	// 转换成功文件列表
	fileInfos := make([]models.FileInfo, 0, len(successFiles))
	for _, file := range successFiles {
		if !file.IsDuplicate {
			fileInfos = append(fileInfos, models.FileInfo{
				URL:          file.URL,
				FilePath:     file.FilePath,
				Size:         file.Size,
				Hash:         file.Hash,
				CrawlMode:    file.CrawlMode,
				DownloadedAt: file.DownloadedAt,
			})
		}
	}

	// 转换失败文件列表
	failedFileInfos := make([]models.FailedFileInfo, 0, len(failedFiles))
	for _, url := range failedFiles {
		failedFileInfos = append(failedFileInfos, models.FailedFileInfo{
			URL:       url,
			ErrorType: "download_failed",
			ErrorMsg:  "下载失败",
			Retries:   0,
		})
	}

	// 创建爬取报告
	crawlReport := models.CrawlReport{
		TaskID:       "",
		TargetURL:    targetURL,
		Domain:       r.domain,
		Mode:         "", // 将在后面设置
		StartTime:    time.Now().Add(-time.Duration(stats.Duration) * time.Second),
		EndTime:      time.Now(),
		Duration:     stats.Duration,
		Stats:        stats,
		SuccessFiles: fileInfos,
		FailedFiles:  failedFileInfos,
		OutputDir:    filepath.Join(r.outputDir, r.domain),
		EncodeDir:    filepath.Join(r.outputDir, r.domain, "encode"),
		DecodeDir:    filepath.Join(r.outputDir, r.domain, "decode"),
		Config:       config,
	}

	// 保存主报告
	if err := r.saveJSONReport(reportsDir, "crawl_report.json", crawlReport); err != nil {
		return err
	}

	// 保存成功文件列表
	if err := r.saveJSONReport(reportsDir, "success_files.json", fileInfos); err != nil {
		return err
	}

	// 保存失败文件列表
	if err := r.saveJSONReport(reportsDir, "failed_files.json", failedFileInfos); err != nil {
		return err
	}

	Infof("✅ 报告已生成: %s", reportsDir)
	return nil
}

// saveJSONReport 保存JSON报告
func (r *Reporter) saveJSONReport(dir string, filename string, data interface{}) error {
	filepath := filepath.Join(dir, filename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化JSON失败: %w", err)
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("写入报告文件失败: %w", err)
	}

	Debugf("保存报告: %s", filepath)
	return nil
}

// NewProgressBar 创建进度条
func NewProgressBar(max int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}
