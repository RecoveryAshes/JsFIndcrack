package core

import (
	"fmt"
	"time"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
)

// BatchCrawler 批量爬取器
type BatchCrawler struct {
	config         models.CrawlConfig
	outputDir      string
	mode           string
	batchDelay     time.Duration
	continueOnErr  bool
	headerProvider models.HeaderProvider
}

// BatchResult 批量爬取结果
type BatchResult struct {
	URL         string
	Success     bool
	Error       error
	Stats       models.TaskStats
	ProcessedAt time.Time
	Duration    float64
}

// BatchSummary 批量爬取摘要
type BatchSummary struct {
	TotalURLs     int
	SuccessCount  int
	FailCount     int
	TotalFiles    int
	TotalSize     int64
	TotalDuration float64
	Results       []BatchResult
}

// NewBatchCrawler 创建批量爬取器
func NewBatchCrawler(config models.CrawlConfig, outputDir string, mode string, batchDelay int, continueOnErr bool, headerProvider models.HeaderProvider) *BatchCrawler {
	return &BatchCrawler{
		config:         config,
		outputDir:      outputDir,
		mode:           mode,
		batchDelay:     time.Duration(batchDelay) * time.Second,
		continueOnErr:  continueOnErr,
		headerProvider: headerProvider,
	}
}

// CrawlBatch 批量爬取URL列表
func (bc *BatchCrawler) CrawlBatch(urls []string) (*BatchSummary, error) {
	utils.Infof("🚀 开始批量爬取: %d个URL", len(urls))

	summary := &BatchSummary{
		TotalURLs: len(urls),
		Results:   make([]BatchResult, 0, len(urls)),
	}

	startTime := time.Now()

	for i, targetURL := range urls {
		utils.Infof("\n==================== [%d/%d] ====================", i+1, len(urls))
		utils.Infof("🎯 目标URL: %s", targetURL)

		// 执行单个URL爬取
		result := bc.crawlSingleURL(targetURL, i+1)
		summary.Results = append(summary.Results, result)

		// 更新统计
		if result.Success {
			summary.SuccessCount++
			summary.TotalFiles += result.Stats.TotalFiles
			summary.TotalSize += result.Stats.TotalSize

			// 目标完成后的隔离日志
			utils.Infof("✅ 目标 %d/%d 完成,独立统计:", i+1, len(urls))
			utils.Infof("   - 访问URL数: %d", result.Stats.VisitedURLs)
			utils.Infof("   - 下载文件数: %d", result.Stats.TotalFiles)
			utils.Infof("   - 文件大小: %.2f MB", float64(result.Stats.TotalSize)/(1024*1024))
			utils.Infof("   - 耗时: %.2f秒", result.Duration)
			utils.Debugf("目标 %d 队列已清空,标签页池已重置,准备处理下一个目标", i+1)
		} else {
			summary.FailCount++
			utils.Errorf("❌ 目标 %d/%d 爬取失败: %v", i+1, len(urls), result.Error)

			// 如果不继续处理错误,则停止
			if !bc.continueOnErr {
				utils.Warn("批量爬取中止 (--continue-on-error=false)")
				break
			}
		}

		// 批量延迟(最后一个URL不需要延迟)
		if i < len(urls)-1 && bc.batchDelay > 0 {
			utils.Debugf("等待 %.0f 秒后处理下一个URL...", bc.batchDelay.Seconds())
			time.Sleep(bc.batchDelay)
		}
	}

	summary.TotalDuration = time.Since(startTime).Seconds()

	// 显示批量爬取摘要
	bc.printSummary(summary)

	return summary, nil
}

// crawlSingleURL 爬取单个URL
// 参数:
//   - targetURL: 目标URL
//   - targetIndex: 目标索引(用于日志显示)
func (bc *BatchCrawler) crawlSingleURL(targetURL string, targetIndex int) BatchResult {
	result := BatchResult{
		URL:         targetURL,
		ProcessedAt: time.Now(),
	}

	startTime := time.Now()

	utils.Debugf("开始爬取目标 %d: %s", targetIndex, targetURL)

	// 创建爬取器
	crawler, err := NewCrawler(targetURL, bc.config, bc.outputDir, bc.mode, bc.headerProvider)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("创建爬取器失败: %w", err)
		result.Duration = time.Since(startTime).Seconds()
		return result
	}

	// 执行爬取
	if err := crawler.Crawl(); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("爬取失败: %w", err)
		result.Duration = time.Since(startTime).Seconds()
		return result
	}

	// 成功
	result.Success = true
	result.Stats = crawler.GetStats()
	result.Duration = time.Since(startTime).Seconds()

	return result
}

// printSummary 打印批量爬取摘要
func (bc *BatchCrawler) printSummary(summary *BatchSummary) {
	utils.Info("\n==================================================")
	utils.Info("📊 批量爬取摘要")
	utils.Info("==================================================")
	utils.Infof("总URL数: %d", summary.TotalURLs)
	utils.Infof("✅ 成功: %d", summary.SuccessCount)
	utils.Infof("❌ 失败: %d", summary.FailCount)
	utils.Infof("📦 总文件数: %d", summary.TotalFiles)
	utils.Infof("📦 总大小: %.2f MB", float64(summary.TotalSize)/(1024*1024))
	utils.Infof("⏱️  总耗时: %.2f秒", summary.TotalDuration)
	utils.Info("==================================================")

	// 显示失败的URL
	if summary.FailCount > 0 {
		utils.Warn("\n失败的URL:")
		for _, result := range summary.Results {
			if !result.Success {
				utils.Warnf("  - %s: %v", result.URL, result.Error)
			}
		}
	}
}
