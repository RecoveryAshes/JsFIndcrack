package core

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/RecoveryAshes/JsFIndcrack/internal/crawlers"
	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
)

// Crawler 主爬取器协调器
type Crawler struct {
	config    models.CrawlConfig
	targetURL string
	domain    string
	outputDir string
	mode      string

	// HTTP头部提供者
	headerProvider models.HeaderProvider

	// 爬取器实例
	staticCrawler  *crawlers.StaticCrawler
	dynamicCrawler *crawlers.DynamicCrawler

	// 反混淆器
	deobfuscator *Deobfuscator

	// 全局文件去重(跨模式)
	fileHashes map[string]string // hash -> URL
	mu         sync.RWMutex

	// 统计信息
	stats models.TaskStats
}

// NewCrawler 创建主爬取器
func NewCrawler(targetURL string, config models.CrawlConfig, outputDir string, mode string, headerProvider models.HeaderProvider) (*Crawler, error) {
	// 解析URL获取域名
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("解析URL失败: %w", err)
	}

	domain := parsedURL.Host
	if domain == "" {
		return nil, fmt.Errorf("无法从URL中提取域名: %s", targetURL)
	}

	return &Crawler{
		config:         config,
		targetURL:      targetURL,
		domain:         domain,
		outputDir:      outputDir,
		mode:           mode,
		headerProvider: headerProvider,
		deobfuscator:   NewDeobfuscator(),
		fileHashes:     make(map[string]string),
		stats:          models.TaskStats{},
	}, nil
}

// Crawl 执行爬取任务
// 执行流程:
//  1. 创建输出目录结构
//  2. 根据模式执行爬取 (static/dynamic/all)
//  3. 合并统计信息
//  4. 执行反混淆处理
//  5. 生成爬取报告
//
// 返回: 错误信息 (如果失败)
func (c *Crawler) Crawl() error {
	startTime := time.Now()

	utils.Infof("🚀 开始爬取任务")
	utils.Infof("目标URL: %s", c.targetURL)
	utils.Infof("域名: %s", c.domain)
	utils.Infof("爬取模式: %s", c.mode)
	utils.Infof("输出目录: %s", c.outputDir)

	// 创建输出目录结构
	if err := c.setupOutputDirectories(); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 根据模式执行爬取
	switch c.mode {
	case "static":
		if err := c.runStaticCrawl(); err != nil {
			return err
		}
	case "dynamic":
		if err := c.runDynamicCrawl(); err != nil {
			return err
		}
	case "all":
		// 先静态后动态,静态失败不影响动态爬取
		if err := c.runStaticCrawl(); err != nil {
			utils.Warnf("静态爬取失败,继续动态爬取: %v", err)
		}
		if err := c.runDynamicCrawl(); err != nil {
			utils.Warnf("动态爬取失败: %v", err)
		}
	default:
		return fmt.Errorf("无效的爬取模式: %s", c.mode)
	}

	// 合并统计信息
	c.mergeStats()

	// 执行反混淆
	allFiles := c.GetAllFiles()
	if len(allFiles) > 0 {
		utils.Infof("🔧 开始反混淆处理...")
		successCount, failCount, err := c.deobfuscator.DeobfuscateAll(allFiles, c.outputDir)
		if err != nil {
			utils.Warnf("反混淆过程出现错误: %v", err)
		}
		utils.Infof("✅ 反混淆完成: 成功 %d, 失败 %d", successCount, failCount)
	}

	duration := time.Since(startTime)
	c.stats.Duration = duration.Seconds()

	// 生成爬取报告
	reporter := utils.NewReporter(c.outputDir, c.domain)
	if err := reporter.GenerateReport(c.targetURL, c.stats, allFiles, []string{}, c.config); err != nil {
		utils.Warnf("生成报告失败: %v", err)
	}

	utils.Infof("✅ 爬取任务完成")
	utils.Infof("总文件数: %d", c.stats.TotalFiles)
	utils.Infof("总耗时: %.2f秒", c.stats.Duration)

	return nil
}

// setupOutputDirectories 创建输出目录结构
func (c *Crawler) setupOutputDirectories() error {
	// 主输出目录: output/domain/
	basePath := filepath.Join(c.outputDir, c.domain)

	// 创建子目录结构
	dirs := []string{
		filepath.Join(basePath, "encode", "js"),  // 混淆JS文件
		filepath.Join(basePath, "encode", "map"), // Source Map文件
		filepath.Join(basePath, "decode", "js"),  // 反混淆JS文件
		filepath.Join(basePath, "similarity"),    // 相似度分析结果
		filepath.Join(basePath, "reports"),       // 报告文件
		filepath.Join(basePath, "checkpoints"),   // 检查点文件
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败 [%s]: %w", dir, err)
		}
		utils.Debugf("创建目录: %s", dir)
	}

	utils.Infof("✅ 输出目录结构创建完成: %s", basePath)
	return nil
}

// runStaticCrawl 执行静态爬取
func (c *Crawler) runStaticCrawl() error {
	utils.Infof("🔍 静态爬取模式启动")

	c.staticCrawler = crawlers.NewStaticCrawler(c.config, c.outputDir, c.domain, c.fileHashes, &c.mu, c.headerProvider)

	if err := c.staticCrawler.Crawl(c.targetURL); err != nil {
		return fmt.Errorf("静态爬取失败: %w", err)
	}

	// 注意: 文件哈希已在爬取过程中添加到全局哈希表

	utils.Infof("✅ 静态爬取完成")
	return nil
}

// runDynamicCrawl 执行动态爬取
func (c *Crawler) runDynamicCrawl() error {
	utils.Infof("🌐 动态爬取模式启动")

	c.dynamicCrawler = crawlers.NewDynamicCrawler(c.config, c.outputDir, c.domain, c.fileHashes, &c.mu, c.headerProvider)

	if err := c.dynamicCrawler.Crawl(c.targetURL); err != nil {
		return fmt.Errorf("动态爬取失败: %w", err)
	}

	// 注意: 文件哈希已在爬取过程中添加到全局哈希表
	// 跨模式去重已在动态爬取器的downloadJSFile中完成,不需要额外处理

	utils.Infof("✅ 动态爬取完成")
	return nil
}

// updateFileHashes 更新全局文件哈希表
func (c *Crawler) updateFileHashes(files []*models.JSFile) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, file := range files {
		if file.IsDuplicate {
			continue // 跳过已标记为重复的文件
		}

		// 检查是否已存在相同哈希
		if existingURL, exists := c.fileHashes[file.Hash]; exists {
			utils.Debugf("发现重复文件: %s (与 %s 相同)", file.URL, existingURL)
			file.IsDuplicate = true
		} else {
			c.fileHashes[file.Hash] = file.URL
		}
	}
}

// performCrossModeDedupe 执行跨模式去重
func (c *Crawler) performCrossModeDedupe() {
	c.mu.Lock()
	defer c.mu.Unlock()

	staticFiles := c.staticCrawler.GetJSFiles()
	dynamicFiles := c.dynamicCrawler.GetJSFiles()

	duplicateCount := 0

	// 检查动态爬取的文件是否与静态爬取重复
	for _, dynFile := range dynamicFiles {
		for _, staticFile := range staticFiles {
			if dynFile.Hash == staticFile.Hash && !dynFile.IsDuplicate {
				utils.Debugf("跨模式重复: %s (动态) == %s (静态)", dynFile.URL, staticFile.URL)
				dynFile.IsDuplicate = true
				duplicateCount++

				// 删除重复的动态爬取文件
				if err := os.Remove(dynFile.FilePath); err != nil {
					utils.Warnf("删除重复文件失败 [%s]: %v", dynFile.FilePath, err)
				} else {
					utils.Debugf("已删除重复文件: %s", dynFile.FilePath)
				}
				break
			}
		}
	}

	if duplicateCount > 0 {
		utils.Infof("🔄 跨模式去重: 删除了 %d 个重复文件", duplicateCount)
	}
}

// mergeStats 合并统计信息
func (c *Crawler) mergeStats() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.staticCrawler != nil {
		staticStats := c.staticCrawler.GetStats()
		c.stats.VisitedURLs += staticStats.VisitedURLs
		c.stats.StaticFiles = staticStats.StaticFiles
		c.stats.TotalFiles += staticStats.StaticFiles
		c.stats.TotalSize += staticStats.TotalSize
		c.stats.FailedFiles += staticStats.FailedFiles
		c.stats.MapFiles += staticStats.MapFiles
	}

	if c.dynamicCrawler != nil {
		dynamicStats := c.dynamicCrawler.GetStats()
		c.stats.VisitedURLs += dynamicStats.VisitedURLs
		c.stats.DynamicFiles = dynamicStats.DynamicFiles
		c.stats.TotalFiles += dynamicStats.DynamicFiles
		c.stats.TotalSize += dynamicStats.TotalSize
		c.stats.FailedFiles += dynamicStats.FailedFiles
		c.stats.MapFiles += dynamicStats.MapFiles
	}

	// 去除重复URL计数
	uniqueURLs := make(map[string]bool)
	if c.staticCrawler != nil {
		for _, file := range c.staticCrawler.GetJSFiles() {
			if !file.IsDuplicate {
				uniqueURLs[file.URL] = true
			}
		}
	}
	if c.dynamicCrawler != nil {
		for _, file := range c.dynamicCrawler.GetJSFiles() {
			if !file.IsDuplicate {
				uniqueURLs[file.URL] = true
			}
		}
	}

	c.stats.TotalFiles = len(uniqueURLs)
}

// GetStats 获取统计信息
func (c *Crawler) GetStats() models.TaskStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// GetAllFiles 获取所有下载的文件(去重后)
func (c *Crawler) GetAllFiles() []*models.JSFile {
	c.mu.RLock()
	defer c.mu.RUnlock()

	allFiles := make([]*models.JSFile, 0)

	if c.staticCrawler != nil {
		for _, file := range c.staticCrawler.GetJSFiles() {
			if !file.IsDuplicate {
				allFiles = append(allFiles, file)
			}
		}
	}

	if c.dynamicCrawler != nil {
		for _, file := range c.dynamicCrawler.GetJSFiles() {
			if !file.IsDuplicate {
				allFiles = append(allFiles, file)
			}
		}
	}

	return allFiles
}

// GetOutputDir 获取输出目录路径
func (c *Crawler) GetOutputDir() string {
	return filepath.Join(c.outputDir, c.domain)
}
