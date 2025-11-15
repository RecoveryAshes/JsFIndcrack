package crawlers

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
	"github.com/gocolly/colly/v2"
	"github.com/google/uuid"
)

// StaticCrawler 静态爬取器(使用Colly)
type StaticCrawler struct {
	collector *colly.Collector
	config    models.CrawlConfig
	outputDir string
	domain    string

	// HTTP头部提供者
	headerProvider models.HeaderProvider

	// 文件存储
	jsFiles  map[string]*models.JSFile  // URL -> JSFile
	mapFiles map[string]*models.MapFile // URL -> MapFile
	mu       sync.RWMutex               // 保护maps

	// 全局文件哈希表(用于跨爬取器去重)
	globalFileHashes map[string]string // hash -> URL (shared with dynamic crawler)
	globalMu         *sync.RWMutex     // 保护globalFileHashes的互斥锁

	// 统计
	visitedURLs []string
	stats       models.TaskStats
}

// NewStaticCrawler 创建静态爬取器
func NewStaticCrawler(config models.CrawlConfig, outputDir string, domain string, globalFileHashes map[string]string, globalMu *sync.RWMutex, headerProvider models.HeaderProvider) *StaticCrawler {
	// 创建Colly collector
	c := colly.NewCollector(
		colly.MaxDepth(config.Depth),
		colly.Async(true),
		colly.AllowedDomains(domain),
	)

	// 动态计算最优并发数
	// 策略: MaxWorkers * min(CPU核心数, 4)
	// 避免在低核心机器上过度并发,同时充分利用多核
	optimalWorkers := calculateOptimalWorkers(config.MaxWorkers)

	// 配置并发限制
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: optimalWorkers,
		Delay:       1 * time.Second,
	})

	utils.Debugf("静态爬取器并发优化: 配置=%d, CPU核心=%d, 最优并发=%d",
		config.MaxWorkers, runtime.NumCPU(), optimalWorkers)

	// 设置超时
	c.SetRequestTimeout(30 * time.Second)

	sc := &StaticCrawler{
		collector:        c,
		config:           config,
		outputDir:        outputDir,
		domain:           domain,
		headerProvider:   headerProvider,
		jsFiles:          make(map[string]*models.JSFile),
		mapFiles:         make(map[string]*models.MapFile),
		globalFileHashes: globalFileHashes,
		globalMu:         globalMu,
		visitedURLs:      make([]string, 0),
		stats:            models.TaskStats{},
	}

	// 设置回调
	sc.setupCallbacks()

	return sc
}

// setupCallbacks 设置Colly回调
func (sc *StaticCrawler) setupCallbacks() {
	// 提取script标签中的JavaScript文件
	sc.collector.OnHTML("script[src]", func(e *colly.HTMLElement) {
		jsURL := e.Request.AbsoluteURL(e.Attr("src"))
		if sc.isJavaScriptURL(jsURL) {
			utils.Debugf("发现JS文件: %s", jsURL)

			// 访问JS文件URL以下载
			e.Request.Visit(jsURL)
		}
	})

	// 提取内联script标签
	sc.collector.OnHTML("script:not([src])", func(e *colly.HTMLElement) {
		// 保存内联脚本
		content := e.Text
		if len(content) > 100 { // 只保存有实质内容的脚本
			utils.Debugf("发现内联脚本,长度: %d", len(content))
			// TODO: 保存内联脚本
		}
	})

	// 处理响应
	sc.collector.OnResponse(func(r *colly.Response) {
		requestURL := r.Request.URL.String()

		// 如果是JavaScript文件,下载并保存
		if sc.isJavaScriptURL(requestURL) {
			sc.downloadJSFile(requestURL, r.Body, r.Headers.Get("Content-Type"))
		}
	})

	// 错误处理
	sc.collector.OnError(func(r *colly.Response, err error) {
		utils.Errorf("爬取错误 [%s]: %v", r.Request.URL, err)
		sc.stats.FailedFiles++
	})

	// 访问前
	sc.collector.OnRequest(func(r *colly.Request) {
		// 应用自定义HTTP头部
		if sc.headerProvider != nil {
			headers, err := sc.headerProvider.GetHeaders()
			if err != nil {
				utils.Warnf("获取HTTP头部失败: %v", err)
			} else {
				for name, values := range headers {
					if len(values) > 0 {
						r.Headers.Set(name, values[0])
					}
				}
			}
		}

		utils.Debugf("访问: %s", r.URL.String())
		sc.mu.Lock()
		sc.visitedURLs = append(sc.visitedURLs, r.URL.String())
		sc.stats.VisitedURLs++
		sc.mu.Unlock()
	})
}

// Crawl 开始爬取
func (sc *StaticCrawler) Crawl(targetURL string) error {
	startTime := time.Now()

	utils.Infof("🔍 静态爬取模式启动")
	utils.Infof("目标URL: %s", targetURL)
	utils.Infof("最大深度: %d", sc.config.Depth)
	utils.Infof("并发数: %d", sc.config.MaxWorkers)

	// 访问目标URL
	if err := sc.collector.Visit(targetURL); err != nil {
		return fmt.Errorf("访问目标URL失败: %w", err)
	}

	// 等待所有异步请求完成
	sc.collector.Wait()

	duration := time.Since(startTime)
	sc.stats.Duration = duration.Seconds()

	utils.Infof("✅ 静态爬取完成")
	utils.Infof("访问URL数: %d", sc.stats.VisitedURLs)
	utils.Infof("下载文件数: %d", sc.stats.StaticFiles)
	utils.Infof("失败文件数: %d", sc.stats.FailedFiles)
	utils.Infof("总耗时: %.2f秒", sc.stats.Duration)

	return nil
}

// downloadJSFile 下载并保存JavaScript文件
// 处理流程:
//  1. 检查URL是否已下载 (避免重复下载)
//  2. 计算文件哈希
//  3. 检查全局哈希表 (跨爬取器去重)
//  4. 检查本地哈希表 (爬取器内去重)
//  5. 生成文件路径并保存到磁盘
//  6. 创建JSFile元数据对象
//  7. 添加到全局哈希表
//  8. 检查并下载Source Map文件
//
// 参数:
//   - fileURL: JavaScript文件的完整URL
//   - content: 文件内容 (字节数组)
//   - contentType: HTTP Content-Type头部
//
// 返回: 错误信息 (如果失败)
func (sc *StaticCrawler) downloadJSFile(fileURL string, content []byte, contentType string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 检查是否已下载
	if _, exists := sc.jsFiles[fileURL]; exists {
		utils.Debugf("文件已存在,跳过: %s", fileURL)
		return nil
	}

	// 计算文件哈希
	hash := calculateHash(content)

	// 先检查全局哈希表(跨爬取器去重)
	if sc.globalFileHashes != nil && sc.globalMu != nil {
		sc.globalMu.RLock()
		if existingURL, exists := sc.globalFileHashes[hash]; exists {
			sc.globalMu.RUnlock()
			utils.Debugf("发现全局重复文件(哈希相同): %s (与 %s 相同)", fileURL, existingURL)

			// 创建一个标记为重复的JSFile对象,但不保存到磁盘
			jsFile := &models.JSFile{
				ID:           uuid.New().String(),
				URL:          fileURL,
				FilePath:     "", // 不保存文件
				Hash:         hash,
				Size:         int64(len(content)),
				Extension:    filepath.Ext(fileURL),
				ContentType:  contentType,
				SourceURL:    fileURL,
				CrawlMode:    models.ModeStatic,
				Depth:        0,
				IsObfuscated: false,
				IsDuplicate:  true,
				DownloadedAt: time.Now(),
				HasMapFile:   false,
			}
			sc.jsFiles[fileURL] = jsFile
			return nil
		}
		sc.globalMu.RUnlock()
	}

	// 检查本地哈希去重
	for _, existingFile := range sc.jsFiles {
		if existingFile.Hash == hash {
			utils.Debugf("发现重复文件(哈希相同): %s", fileURL)
			sc.jsFiles[fileURL] = existingFile
			existingFile.IsDuplicate = true
			return nil
		}
	}

	// 生成文件路径
	filePath, err := sc.generateFilePath(fileURL, "encode/js")
	if err != nil {
		return fmt.Errorf("生成文件路径失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	// 创建JSFile对象
	jsFile := &models.JSFile{
		ID:           uuid.New().String(),
		URL:          fileURL,
		FilePath:     filePath,
		Hash:         hash,
		Size:         int64(len(content)),
		Extension:    filepath.Ext(fileURL),
		ContentType:  contentType,
		SourceURL:    fileURL,
		CrawlMode:    models.ModeStatic,
		Depth:        0, // TODO: 跟踪实际深度
		IsObfuscated: false,
		DownloadedAt: time.Now(),
		HasMapFile:   false,
	}

	sc.jsFiles[fileURL] = jsFile
	sc.stats.StaticFiles++
	sc.stats.TotalFiles++
	sc.stats.TotalSize += int64(len(content))

	// 添加到全局哈希表
	if sc.globalFileHashes != nil && sc.globalMu != nil {
		sc.globalMu.Lock()
		sc.globalFileHashes[hash] = fileURL
		sc.globalMu.Unlock()
	}

	utils.Infof("📥 下载成功: %s (%d bytes)", filepath.Base(filePath), len(content))

	// 检查是否有Source Map
	sc.checkAndDownloadSourceMap(fileURL, content)

	return nil
}

// checkAndDownloadSourceMap 检查并下载Source Map文件
func (sc *StaticCrawler) checkAndDownloadSourceMap(jsURL string, jsContent []byte) {
	// 在文件内容中查找sourceMappingURL注释
	content := string(jsContent)

	// 查找 //# sourceMappingURL=xxx.map
	if idx := strings.Index(content, "sourceMappingURL="); idx != -1 {
		start := idx + len("sourceMappingURL=")
		end := strings.IndexAny(content[start:], "\n\r ")
		if end == -1 {
			end = len(content) - start
		}

		mapURL := strings.TrimSpace(content[start : start+end])

		// 构造完整URL
		baseURL, _ := url.Parse(jsURL)
		fullMapURL, err := baseURL.Parse(mapURL)
		if err == nil {
			utils.Infof("🗺️  发现Source Map: %s", fullMapURL.String())
			// TODO: 下载Source Map文件
			sc.stats.MapFiles++
		}
	}
}

// isJavaScriptURL 判断是否为JavaScript文件URL
func (sc *StaticCrawler) isJavaScriptURL(urlStr string) bool {
	urlStr = strings.ToLower(urlStr)

	// 检查扩展名
	for _, ext := range models.JSFileExtensions {
		if strings.HasSuffix(urlStr, ext) {
			return true
		}
	}

	// 检查常见JS模式
	if strings.Contains(urlStr, ".js?") ||
		strings.Contains(urlStr, ".mjs?") ||
		strings.Contains(urlStr, ".jsx?") {
		return true
	}

	return false
}

// generateFilePath 生成本地文件路径
func (sc *StaticCrawler) generateFilePath(fileURL string, subdir string) (string, error) {
	parsed, err := url.Parse(fileURL)
	if err != nil {
		return "", err
	}

	// 使用URL路径作为文件名
	filename := filepath.Base(parsed.Path)
	if filename == "" || filename == "." {
		filename = "index.js"
	}

	// 构造完整路径: output/domain/encode/js/filename
	fullPath := filepath.Join(sc.outputDir, sc.domain, subdir, filename)

	// 如果文件已存在,添加编号
	if _, err := os.Stat(fullPath); err == nil {
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		for i := 1; ; i++ {
			newPath := filepath.Join(sc.outputDir, sc.domain, subdir, fmt.Sprintf("%s_%d%s", base, i, ext))
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				fullPath = newPath
				break
			}
		}
	}

	return fullPath, nil
}

// calculateHash 计算SHA-256哈希
func calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// GetStats 获取统计信息
func (sc *StaticCrawler) GetStats() models.TaskStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.stats
}

// GetJSFiles 获取所有下载的JS文件
func (sc *StaticCrawler) GetJSFiles() []*models.JSFile {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	files := make([]*models.JSFile, 0, len(sc.jsFiles))
	for _, f := range sc.jsFiles {
		files = append(files, f)
	}
	return files
}

// calculateOptimalWorkers 动态计算最优并发数
// 根据CPU核心数和配置值智能调整并发数
// 策略:
//   - 单核/双核: 使用配置值 (避免过度并发)
//   - 4核及以上: 配置值 * min(CPU核心数/2, 4)
//   - 最大不超过 配置值 * 4
func calculateOptimalWorkers(configWorkers int) int {
	numCPU := runtime.NumCPU()

	// 基础值
	baseWorkers := configWorkers
	if baseWorkers < 1 {
		baseWorkers = 2 // 默认最小并发
	}

	// 根据CPU核心数调整
	switch {
	case numCPU <= 2:
		// 低核心机器: 保持配置值
		return baseWorkers
	case numCPU <= 4:
		// 中等核心: 配置值 * 2
		return baseWorkers * 2
	case numCPU <= 8:
		// 多核: 配置值 * 3
		return baseWorkers * 3
	default:
		// 高核心: 配置值 * 4 (避免过度并发)
		return baseWorkers * 4
	}
}
