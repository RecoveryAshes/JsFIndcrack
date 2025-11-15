package crawlers

import (
	"context"
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
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/google/uuid"
)

// DynamicCrawler 动态爬取器(使用Rod)
type DynamicCrawler struct {
	browser   *rod.Browser
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
	globalFileHashes map[string]string // hash -> URL (shared with static crawler)
	globalMu         *sync.RWMutex     // 保护globalFileHashes的互斥锁

	// 统计
	visitedURLs []string
	stats       models.TaskStats

	// 页面池用于并发
	pagePool chan *rod.Page
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewDynamicCrawler 创建动态爬取器
func NewDynamicCrawler(config models.CrawlConfig, outputDir string, domain string, globalFileHashes map[string]string, globalMu *sync.RWMutex, headerProvider models.HeaderProvider) *DynamicCrawler {
	ctx, cancel := context.WithCancel(context.Background())

	// 动态计算最优标签页数
	// 策略: 基于CPU核心数和内存,避免过度消耗
	optimalTabs := calculateOptimalTabs(config.PlaywrightTabs)

	utils.Debugf("动态爬取器标签页池优化: 配置=%d, CPU核心=%d, 最优标签页=%d",
		config.PlaywrightTabs, runtime.NumCPU(), optimalTabs)

	dc := &DynamicCrawler{
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
		pagePool:         make(chan *rod.Page, optimalTabs), // 使用优化后的标签页数
		ctx:              ctx,
		cancel:           cancel,
	}

	// 更新config中的PlaywrightTabs为优化后的值
	dc.config.PlaywrightTabs = optimalTabs

	return dc
}

// Crawl 开始动态爬取
func (dc *DynamicCrawler) Crawl(targetURL string) error {
	startTime := time.Now()

	utils.Infof("🌐 动态爬取模式启动")
	utils.Infof("目标URL: %s", targetURL)
	utils.Infof("等待时间: %d秒", dc.config.WaitTime)
	utils.Infof("标签页数: %d", dc.config.PlaywrightTabs)

	// 启动浏览器
	if err := dc.launchBrowser(); err != nil {
		return fmt.Errorf("启动浏览器失败: %w", err)
	}
	defer dc.closeBrowser()

	// 初始化页面池
	if err := dc.initPagePool(); err != nil {
		return fmt.Errorf("初始化页面池失败: %w", err)
	}

	// 爬取目标URL
	if err := dc.crawlPage(targetURL, 0); err != nil {
		utils.Errorf("爬取失败: %v", err)
		return err
	}

	duration := time.Since(startTime)
	dc.stats.Duration = duration.Seconds()

	utils.Infof("✅ 动态爬取完成")
	utils.Infof("访问URL数: %d", dc.stats.VisitedURLs)
	utils.Infof("下载文件数: %d", dc.stats.DynamicFiles)
	utils.Infof("失败文件数: %d", dc.stats.FailedFiles)
	utils.Infof("总耗时: %.2f秒", dc.stats.Duration)

	return nil
}

// launchBrowser 启动浏览器
func (dc *DynamicCrawler) launchBrowser() error {
	// 配置launcher
	l := launcher.New()

	if dc.config.Headless {
		l = l.Headless(true)
	} else {
		l = l.Headless(false)
	}

	// 启动浏览器
	controlURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("启动浏览器失败: %w", err)
	}

	// 连接到浏览器
	dc.browser = rod.New().ControlURL(controlURL)
	if err := dc.browser.Connect(); err != nil {
		return fmt.Errorf("连接浏览器失败: %w", err)
	}

	utils.Debugf("浏览器已启动: %s", controlURL)
	return nil
}

// closeBrowser 关闭浏览器
func (dc *DynamicCrawler) closeBrowser() {
	if dc.browser != nil {
		dc.cancel()
		close(dc.pagePool)
		dc.browser.MustClose()
		utils.Debugf("浏览器已关闭")
	}
}

// initPagePool 初始化页面池
func (dc *DynamicCrawler) initPagePool() error {
	for i := 0; i < dc.config.PlaywrightTabs; i++ {
		page, err := dc.browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			return fmt.Errorf("创建页面失败: %w", err)
		}

		// 设置网络拦截
		if err := dc.setupNetworkIntercept(page); err != nil {
			return fmt.Errorf("设置网络拦截失败: %w", err)
		}

		dc.pagePool <- page
		utils.Debugf("创建页面池标签页 %d/%d", i+1, dc.config.PlaywrightTabs)
	}

	return nil
}

// setupNetworkIntercept 设置网络请求拦截
func (dc *DynamicCrawler) setupNetworkIntercept(page *rod.Page) error {
	// 启用网络域
	router := page.HijackRequests()

	router.MustAdd("*", func(ctx *rod.Hijack) {
		// 应用自定义HTTP头部
		if dc.headerProvider != nil {
			headers, err := dc.headerProvider.GetHeaders()
			if err != nil {
				utils.Warnf("获取HTTP头部失败: %v", err)
			} else {
				for name, values := range headers {
					if len(values) > 0 {
						ctx.Request.Req().Header.Set(name, values[0])
					}
				}
			}
		}

		// 获取请求URL
		requestURL := ctx.Request.URL().String()

		// 检查是否为JavaScript文件
		if dc.isJavaScriptURL(requestURL) {
			utils.Debugf("拦截JS请求: %s", requestURL)
		}

		// 继续请求
		ctx.MustLoadResponse()

		// 如果是JavaScript文件,保存响应
		if dc.isJavaScriptURL(requestURL) {
			if ctx.Response != nil {
				body := ctx.Response.Body()
				// 获取Content-Type
				contentType := "application/javascript"
				dc.downloadJSFile(requestURL, []byte(body), contentType)
			}
		}
	})

	go router.Run()

	return nil
}

// crawlPage 爬取单个页面
func (dc *DynamicCrawler) crawlPage(pageURL string, depth int) error {
	// 检查深度限制
	if depth > dc.config.Depth {
		return nil
	}

	// 记录访问
	dc.mu.Lock()
	dc.visitedURLs = append(dc.visitedURLs, pageURL)
	dc.stats.VisitedURLs++
	dc.mu.Unlock()

	utils.Debugf("访问页面: %s (深度: %d)", pageURL, depth)

	// 从页面池获取页面
	page := <-dc.pagePool
	defer func() {
		// 页面复用前清理状态
		// 清理缓存、Cookie、存储,避免状态污染
		cleanupPage(page)
		dc.pagePool <- page // 归还页面到池
	}()

	// 导航到目标URL
	if err := page.Navigate(pageURL); err != nil {
		utils.Errorf("导航失败 [%s]: %v", pageURL, err)
		dc.stats.FailedFiles++
		return err
	}

	// 等待页面加载
	if err := page.WaitLoad(); err != nil {
		utils.Errorf("等待页面加载失败 [%s]: %v", pageURL, err)
		return err
	}

	// 额外等待时间(等待动态JS加载)
	time.Sleep(time.Duration(dc.config.WaitTime) * time.Second)

	utils.Debugf("页面加载完成: %s", pageURL)

	return nil
}

// downloadJSFile 下载并保存JavaScript文件
func (dc *DynamicCrawler) downloadJSFile(fileURL string, content []byte, contentType string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// 检查是否已下载
	if _, exists := dc.jsFiles[fileURL]; exists {
		utils.Debugf("文件已存在,跳过: %s", fileURL)
		return nil
	}

	// 计算文件哈希
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	// 先检查全局哈希表(跨爬取器去重)
	if dc.globalFileHashes != nil && dc.globalMu != nil {
		dc.globalMu.RLock()
		if existingURL, exists := dc.globalFileHashes[hash]; exists {
			dc.globalMu.RUnlock()
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
				CrawlMode:    models.ModeDynamic,
				Depth:        0,
				IsObfuscated: false,
				IsDuplicate:  true,
				DownloadedAt: time.Now(),
				HasMapFile:   false,
			}
			dc.jsFiles[fileURL] = jsFile
			return nil
		}
		dc.globalMu.RUnlock()
	}

	// 检查本地哈希去重
	for _, existingFile := range dc.jsFiles {
		if existingFile.Hash == hash {
			utils.Debugf("发现重复文件(哈希相同): %s", fileURL)
			dc.jsFiles[fileURL] = existingFile
			existingFile.IsDuplicate = true
			return nil
		}
	}

	// 生成文件路径
	filePath, err := dc.generateFilePath(fileURL, "encode/js")
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
		CrawlMode:    models.ModeDynamic,
		Depth:        0, // TODO: 跟踪实际深度
		IsObfuscated: false,
		DownloadedAt: time.Now(),
		HasMapFile:   false,
	}

	dc.jsFiles[fileURL] = jsFile
	dc.stats.DynamicFiles++
	dc.stats.TotalFiles++
	dc.stats.TotalSize += int64(len(content))

	// 添加到全局哈希表
	if dc.globalFileHashes != nil && dc.globalMu != nil {
		dc.globalMu.Lock()
		dc.globalFileHashes[hash] = fileURL
		dc.globalMu.Unlock()
	}

	utils.Infof("📥 下载成功: %s (%d bytes)", filepath.Base(filePath), len(content))

	// 检查是否有Source Map
	dc.checkAndDownloadSourceMap(fileURL, content)

	return nil
}

// checkAndDownloadSourceMap 检查并下载Source Map文件
func (dc *DynamicCrawler) checkAndDownloadSourceMap(jsURL string, jsContent []byte) {
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
			dc.stats.MapFiles++
		}
	}
}

// isJavaScriptURL 判断是否为JavaScript文件URL
func (dc *DynamicCrawler) isJavaScriptURL(urlStr string) bool {
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

	// 检查Content-Type (如果可用)
	return false
}

// generateFilePath 生成本地文件路径
func (dc *DynamicCrawler) generateFilePath(fileURL string, subdir string) (string, error) {
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
	fullPath := filepath.Join(dc.outputDir, dc.domain, subdir, filename)

	// 如果文件已存在,添加编号
	if _, err := os.Stat(fullPath); err == nil {
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		for i := 1; ; i++ {
			newPath := filepath.Join(dc.outputDir, dc.domain, subdir, fmt.Sprintf("%s_%d%s", base, i, ext))
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				fullPath = newPath
				break
			}
		}
	}

	return fullPath, nil
}

// GetStats 获取统计信息
func (dc *DynamicCrawler) GetStats() models.TaskStats {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.stats
}

// GetJSFiles 获取所有下载的JS文件
func (dc *DynamicCrawler) GetJSFiles() []*models.JSFile {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	files := make([]*models.JSFile, 0, len(dc.jsFiles))
	for _, f := range dc.jsFiles {
		files = append(files, f)
	}
	return files
}

// calculateOptimalTabs 动态计算最优标签页数
// 根据CPU核心数和内存智能调整标签页数
// 浏览器标签页比普通线程更消耗资源,需要保守估计
func calculateOptimalTabs(configTabs int) int {
	numCPU := runtime.NumCPU()

	// 基础值
	baseTabs := configTabs
	if baseTabs < 1 {
		baseTabs = 4 // 默认4个标签页
	}

	// 浏览器标签页消耗大,最多不超过 min(CPU核心数, 配置值*2)
	maxTabs := numCPU
	if baseTabs*2 < maxTabs {
		maxTabs = baseTabs * 2
	}

	// 保守策略,避免浏览器卡顿
	switch {
	case numCPU <= 2:
		// 低核心: 最多2个标签页
		if maxTabs > 2 {
			return 2
		}
		return maxTabs
	case numCPU <= 4:
		// 中等: 最多4个标签页
		if maxTabs > 4 {
			return 4
		}
		return maxTabs
	case numCPU <= 8:
		// 多核: 最多6个标签页
		if maxTabs > 6 {
			return 6
		}
		return maxTabs
	default:
		// 高核心: 最多8个标签页 (避免内存溢出)
		if maxTabs > 8 {
			return 8
		}
		return maxTabs
	}
}

// cleanupPage 清理页面状态以供复用
// 清除缓存、Cookie、LocalStorage等,避免页面间状态污染
func cleanupPage(page *rod.Page) {
	// 忽略错误,因为清理失败不应影响后续爬取
	_, _ = page.Eval(`() => {
		// 清理LocalStorage
		try { localStorage.clear(); } catch(e) {}
		// 清理SessionStorage
		try { sessionStorage.clear(); } catch(e) {}
		// 清理IndexedDB (异步,尽力而为)
		try {
			if (window.indexedDB && window.indexedDB.databases) {
				window.indexedDB.databases().then(dbs => {
					dbs.forEach(db => {
						if (db.name) {
							window.indexedDB.deleteDatabase(db.name);
						}
					});
				});
			}
		} catch(e) {}
	}`)
}
