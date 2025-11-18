package crawlers

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/google/uuid"
)

// 错误类型定义 (Feature 010-fix-domain-crawl-bugs)
var (
	ErrBrowserCrashed    = errors.New("浏览器崩溃")
	ErrMaxRetriesReached = errors.New("已达最大重试次数")
	ErrInvalidContent    = errors.New("无效内容,非JS文件")
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

	// 新增: 自适应标签页池
	pagePool        *PagePool
	resourceMonitor *ResourceMonitor
	urlQueue        *URLQueue

	// 标签页ID映射 (用于日志显示)
	pageIDs   map[*rod.Page]int
	pageIDsMu sync.RWMutex
	nextPageID int

	// 浏览器会话管理 (Feature 010-fix-domain-crawl-bugs)
	browserRetryCount int // 当前浏览器重启次数
	maxBrowserRetries int // 最大浏览器重启次数(默认3)

	// Worker活跃计数器(用于检测所有worker空闲)
	activeWorkers int32 // 使用atomic操作
	workersMu     sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewDynamicCrawler 创建动态爬取器
func NewDynamicCrawler(config models.CrawlConfig, outputDir string, domain string, globalFileHashes map[string]string, globalMu *sync.RWMutex, headerProvider models.HeaderProvider) *DynamicCrawler {
	ctx, cancel := context.WithCancel(context.Background())

	dc := &DynamicCrawler{
		config:            config,
		outputDir:         outputDir,
		domain:            domain,
		headerProvider:    headerProvider,
		jsFiles:           make(map[string]*models.JSFile),
		mapFiles:          make(map[string]*models.MapFile),
		globalFileHashes:  globalFileHashes,
		globalMu:          globalMu,
		visitedURLs:       make([]string, 0),
		stats:             models.TaskStats{},
		pageIDs:           make(map[*rod.Page]int),
		nextPageID:        1,
		browserRetryCount: 0, // 初始化重试计数
		maxBrowserRetries: 3, // 默认最多重启3次
		ctx:               ctx,
		cancel:            cancel,
	}

	return dc
}

// Crawl 开始动态爬取 (Feature 010-fix-domain-crawl-bugs: T029-T032)
// 支持浏览器崩溃自动重启,最多重试3次
func (dc *DynamicCrawler) Crawl(targetURL string) error {
	startTime := time.Now()

	// 验证入口URL
	if targetURL == "" {
		return fmt.Errorf("入口URL为空,无法开始爬取")
	}

	// 验证URL格式
	parsedTestURL, err := url.Parse(targetURL)
	if err != nil || parsedTestURL.Scheme == "" || parsedTestURL.Host == "" {
		return fmt.Errorf("入口URL格式无效: %s", targetURL)
	}

	utils.Infof("🌐 动态爬取模式启动(自适应标签页池)")
	utils.Infof("目标URL: %s", targetURL)
	utils.Infof("等待时间: %d秒", dc.config.WaitTime)
	utils.Infof("最大深度: %d", dc.config.Depth)

	// 解析目标URL获取域名
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("解析目标URL失败: %w", err)
	}
	targetDomain := parsedURL.Host

	// 初始化ResourceMonitor (在重试循环外,避免重复创建)
	resourceConfig := ResourceMonitorConfig{
		SafetyReserveMemory: 1024 * 1024 * 1024, // 1GB
		SafetyThreshold:     500 * 1024 * 1024,  // 500MB
		CPULoadThreshold:    80,                 // 80%
		MaxTabsLimit:        16,                 // 16个标签页
		TabMemoryUsage:      100 * 1024 * 1024,  // 100MB per tab
	}
	dc.resourceMonitor = NewResourceMonitor(resourceConfig)
	dc.resourceMonitor.StartMonitoring(1 * time.Second)
	defer dc.resourceMonitor.StopMonitoring()

	// 初始化URLQueue (在重试循环外,保持visitedURLs状态 - T033)
	dc.urlQueue = NewURLQueue(targetDomain, dc.config.AllowCrossDomain, dc.config.Depth)
	defer dc.urlQueue.Close()

	// 将入口URL添加到队列
	err = dc.urlQueue.Push(targetURL, 0)
	if err != nil {
		return fmt.Errorf("添加入口URL失败: %w", err)
	}

	// T030: 浏览器崩溃重试循环 (最多3次)
	for dc.browserRetryCount = 0; dc.browserRetryCount <= dc.maxBrowserRetries; dc.browserRetryCount++ {
		// 启动浏览器
		if err := dc.launchBrowser(); err != nil {
			utils.Errorf("浏览器启动失败(重试%d/%d): %v", dc.browserRetryCount, dc.maxBrowserRetries, err)
			if dc.browserRetryCount == dc.maxBrowserRetries {
				return fmt.Errorf("浏览器启动失败,已达最大重试次数: %w", err)
			}
			// T032: 浏览器重启Warn日志
			utils.Warnf("浏览器启动失败,准备重启(重试%d/%d)", dc.browserRetryCount+1, dc.maxBrowserRetries)
			time.Sleep(2 * time.Second) // 等待2秒后重试
			continue
		}

		// 记录证书跳过信息 (WARN级别日志)
		utils.Warnf("浏览器已配置为跳过HTTPS证书验证,适用于内网/开发环境的自签名证书")

		// T029: 调用crawlWithBrowser执行爬取逻辑
		err = dc.crawlWithBrowser(targetURL, targetDomain)

		// 关闭浏览器
		dc.closeBrowser()

		// T030: 检测浏览器崩溃
		if errors.Is(err, ErrBrowserCrashed) {
			dc.stats.BrowserRestarts++ // 记录重启次数
			utils.Warnf("浏览器崩溃,准备重启(重试%d/%d)", dc.browserRetryCount+1, dc.maxBrowserRetries)

			// 如果达到最大重试次数,返回错误
			if dc.browserRetryCount == dc.maxBrowserRetries {
				return fmt.Errorf("浏览器崩溃,已达最大重试次数: %w", ErrMaxRetriesReached)
			}

			time.Sleep(2 * time.Second) // 等待2秒后重启
			continue                    // 继续重试循环
		}

		// 其他错误或成功完成,退出重试循环
		if err != nil {
			return err
		}
		break // 成功完成,退出重试循环
	}

	// 输出统计信息
	duration := time.Since(startTime)
	dc.stats.Duration = duration.Seconds()

	utils.Infof("✅ 动态爬取完成")
	utils.Infof("访问URL数: %d", dc.stats.VisitedURLs)
	utils.Infof("下载文件数: %d", dc.stats.DynamicFiles)
	utils.Infof("失败文件数: %d", dc.stats.FailedFiles)
	if dc.stats.BrowserRestarts > 0 {
		utils.Infof("浏览器重启次数: %d", dc.stats.BrowserRestarts)
	}
	utils.Infof("总耗时: %.2f秒", dc.stats.Duration)

	return nil
}

// crawlWithBrowser 在浏览器实例中执行爬取逻辑 (T029, T031)
// 返回ErrBrowserCrashed表示浏览器崩溃,需要重启
func (dc *DynamicCrawler) crawlWithBrowser(targetURL string, targetDomain string) (err error) {
	// T031: 使用defer捕获panic,转换为ErrBrowserCrashed
	defer func() {
		if r := recover(); r != nil {
			utils.Errorf("浏览器操作panic: %v", r)
			err = ErrBrowserCrashed
		}
	}()

	// 初始化PagePool (每次浏览器重启都需要重新创建)
	dc.pagePool = NewPagePool(dc.browser, dc.resourceMonitor, dc.urlQueue, dc.ctx)
	defer dc.pagePool.Close()

	// T039 [EC2]: 计算初始worker数量为min(16, resourceMonitor.CalculateMaxTabs())
	maxWorkerLimit := 16
	initialMaxTabs := dc.resourceMonitor.CalculateMaxTabs()
	maxWorkers := maxWorkerLimit
	if initialMaxTabs < maxWorkerLimit {
		maxWorkers = initialMaxTabs
	}
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	// T041 [EC2]: worker启动Debug日志
	utils.Debugf("动态爬取启动: 初始worker数量=%d, 可用标签页数=%d, 最大限制=%d",
		maxWorkers, initialMaxTabs, maxWorkerLimit)

	utils.Infof("开始爬取,初始标签页数: 1")

	// T040 [EC2]: 添加goroutine,每5秒检查资源并调用pagePool.AdjustSize
	adjustCtx, adjustCancel := context.WithCancel(dc.ctx)
	defer adjustCancel()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-adjustCtx.Done():
				return
			case <-ticker.C:
				// 获取队列中待处理URL数量
				pendingCount := dc.urlQueue.PendingCount()
				// 调用PagePool的动态调整方法
				dc.pagePool.AdjustSize(pendingCount)
			}
		}
	}()

	// 添加监控goroutine,检测所有worker空闲且队列为空时关闭队列
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-adjustCtx.Done():
				return
			case <-ticker.C:
				// 检查是否所有worker都空闲且队列为空
				activeCount := atomic.LoadInt32(&dc.activeWorkers)
				pendingCount := dc.urlQueue.PendingCount()

				if activeCount == 0 && pendingCount == 0 {
					// 所有worker空闲且队列为空,关闭队列
					utils.Debugf("检测到所有worker空闲且队列为空,关闭队列")
					dc.urlQueue.Close()
					return
				}
			}
		}
	}()

	// Worker pool模式处理URL队列
	var wg sync.WaitGroup

	// 初始化活跃worker数量
	atomic.StoreInt32(&dc.activeWorkers, int32(maxWorkers))

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			dc.worker(workerID)
		}(i)
	}

	wg.Wait()

	return nil
}

// worker Worker goroutine,从队列拉取URL并爬取
func (dc *DynamicCrawler) worker(workerID int) {
	for {
		// Worker进入空闲状态(等待URL)
		atomic.AddInt32(&dc.activeWorkers, -1)

		// 从队列获取URL
		urlStr, depth, ok := dc.urlQueue.Pop(dc.ctx)
		if !ok {
			// 队列已关闭或context取消
			return
		}

		// Worker进入工作状态
		atomic.AddInt32(&dc.activeWorkers, 1)

		// 检查队列长度,动态调整标签页池大小
		pendingCount := dc.urlQueue.PendingCount()
		dc.pagePool.AdjustSize(pendingCount)

		// 爬取页面
		err := dc.crawlPage(urlStr, depth)
		if err != nil {
			utils.Warnf("Worker %d 爬取失败 [%s]: %v", workerID, urlStr, err)
		}

		// 不在这里检查退出条件,让Pop阻塞等待新URL
		// 当队列关闭时,Pop会返回ok=false,worker自然退出
	}
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

	// Bug #1修复: 添加证书忽略参数,允许访问自签名、过期或主机名不匹配的HTTPS站点
	// 参考: research.md - TLS证书验证修复方案
	l = l.Set("ignore-certificate-errors")
	utils.Debugf("浏览器启动参数: --ignore-certificate-errors (跳过TLS证书验证)")

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
		dc.browser.MustClose()
		utils.Debugf("浏览器已关闭")
	}
}

// setupNetworkIntercept 设置网络请求拦截
func (dc *DynamicCrawler) setupNetworkIntercept(page *rod.Page) error {
	// 分配并注册页面ID
	dc.pageIDsMu.Lock()
	pageID := dc.nextPageID
	dc.pageIDs[page] = pageID
	dc.nextPageID++
	dc.pageIDsMu.Unlock()

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

		// 让浏览器继续处理请求(不拦截,只监听响应)
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	// 监听响应完成事件来捕获JS文件
	go page.EachEvent(func(e *proto.NetworkResponseReceived) {
		// 检查是否为JavaScript文件
		resp := e.Response
		if resp.MIMEType == "application/javascript" || resp.MIMEType == "text/javascript" ||
			strings.HasSuffix(resp.URL, ".js") {
			utils.Debugf("检测到JS响应: %s", resp.URL)

			// 获取响应体
			body, err := proto.NetworkGetResponseBody{RequestID: e.RequestID}.Call(page)
			if err != nil {
				utils.Warnf("获取响应体失败 [%s]: %v", resp.URL, err)
				return
			}

			var content []byte
			if body.Base64Encoded {
				content, err = base64.StdEncoding.DecodeString(body.Body)
				if err != nil {
					utils.Warnf("解码Base64失败 [%s]: %v", resp.URL, err)
					return
				}
			} else {
				content = []byte(body.Body)
			}

			// 下载JS文件,传入页面ID
			contentType := resp.MIMEType
			if contentType == "" {
				contentType = "application/javascript"
			}
			if err := dc.downloadJSFileWithPageID(resp.URL, content, contentType, pageID); err != nil {
				utils.Warnf("下载JS文件失败 [%s]: %v", resp.URL, err)
			}
		}
	})()

	go router.Run()

	return nil
}

// crawlPage 爬取单个页面
func (dc *DynamicCrawler) crawlPage(pageURL string, depth int) (err error) {
	// T030-T031 [US2]: 添加defer+recover机制捕获panic,记录结构化错误日志
	defer func() {
		if r := recover(); r != nil {
			// 捕获panic并转换为error
			err = fmt.Errorf("页面爬取panic: %v", r)

			// T031: 记录结构化错误日志(包含URL、错误类型、堆栈跟踪)
			utils.Errorf("捕获panic: URL=%s, 深度=%d, 错误=%v, 类型=panic恢复", pageURL, depth, r)

			// 增加失败计数
			dc.mu.Lock()
			dc.stats.FailedFiles++
			dc.mu.Unlock()
		}
	}()

	// 标记为已访问
	dc.urlQueue.MarkVisited(pageURL)

	// 记录访问
	dc.mu.Lock()
	dc.visitedURLs = append(dc.visitedURLs, pageURL)
	dc.stats.VisitedURLs++
	dc.mu.Unlock()

	utils.Debugf("访问页面: %s (深度: %d)", pageURL, depth)

	// 从PagePool获取标签页
	page, pageErr := dc.pagePool.AcquirePage(dc.ctx)
	if pageErr != nil {
		utils.Errorf("获取标签页失败 [%s]: %v", pageURL, pageErr)
		dc.stats.FailedFiles++
		return pageErr
	}
	defer dc.pagePool.ReleasePage(page)

	// 设置网络拦截,捕获动态加载的JS文件
	// 使用LoadResponse让浏览器处理请求(浏览器已配置--ignore-certificate-errors)
	if interceptErr := dc.setupNetworkIntercept(page); interceptErr != nil {
		utils.Warnf("设置网络拦截失败 [%s]: %v", pageURL, interceptErr)
	}

	// 导航到目标URL
	if navErr := page.Navigate(pageURL); navErr != nil {
		utils.Errorf("导航失败 [%s]: %v", pageURL, navErr)
		dc.stats.FailedFiles++
		return navErr
	}

	// 等待页面加载
	if loadErr := page.WaitLoad(); loadErr != nil {
		utils.Errorf("等待页面加载失败 [%s]: %v", pageURL, loadErr)
		return loadErr
	}

	// 额外等待时间(等待动态JS加载)
	time.Sleep(time.Duration(dc.config.WaitTime) * time.Second)

	utils.Debugf("页面加载完成: %s", pageURL)

	// 提取页面链接(如果未达到最大深度)
	if depth < dc.config.Depth {
		// 创建URLExtractor
		parsedURL, _ := url.Parse(pageURL)
		extractor := NewURLExtractor(dc.urlQueue, parsedURL.Host, dc.config.AllowCrossDomain, dc.config.Depth)

		// 从页面提取链接
		extractedCount, extractErr := extractor.ExtractFromPage(page, pageURL, depth)
		if extractErr != nil {
			utils.Warnf("提取链接失败 [%s]: %v", pageURL, extractErr)
		} else if extractedCount > 0 {
			utils.Infof("从页面提取了 %d 个链接: %s", extractedCount, pageURL)

			// 记录当前状态
			currentTabs := dc.pagePool.CurrentSize()
			pendingURLs := dc.urlQueue.PendingCount()
			maxTabs := dc.pagePool.MaxSize()
			utils.Infof("当前标签页: %d, 待爬URL数: %d, 最大限制: %d", currentTabs, pendingURLs, maxTabs)
		}
	}

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

	utils.Infof("📥 下载成功: %s (%d bytes) - %s", filepath.Base(filePath), len(content), fileURL)

	// 检查是否有Source Map
	dc.checkAndDownloadSourceMap(fileURL, content)

	return nil
}

// downloadJSFileWithPageID 下载JS文件并保存(带页面ID显示)
func (dc *DynamicCrawler) downloadJSFileWithPageID(fileURL string, content []byte, contentType string, pageID int) error {
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

	// 带标签页ID的日志
	utils.Infof("📥 下载成功 [标签页#%d]: %s (%d bytes) - %s", pageID, filepath.Base(filePath), len(content), fileURL)

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

			// 下载Source Map文件
			dc.downloadSourceMapFile(fullMapURL.String())
		}
	}
}

// downloadSourceMapFile 下载Source Map文件
// 注意: 调用此函数前调用者必须已持有 dc.mu 锁
func (dc *DynamicCrawler) downloadSourceMapFile(mapURL string) {
	// 检查是否已下载 (不需要额外加锁,调用者已持有锁)
	if _, exists := dc.mapFiles[mapURL]; exists {
		utils.Debugf("Source Map文件已存在,跳过: %s", mapURL)
		return
	}

	// 临时释放锁以执行HTTP请求(避免阻塞其他操作)
	dc.mu.Unlock()
	defer dc.mu.Lock()

	// HTTP超时时间直接使用配置文件的 wait_time 值(秒)
	httpTimeout := time.Duration(dc.config.WaitTime) * time.Second

	// 发起HTTP请求下载
	client := &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	resp, err := client.Get(mapURL)
	if err != nil {
		utils.Warnf("下载Source Map失败 [%s]: %v", mapURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.Warnf("下载Source Map失败 [%s]: HTTP %d", mapURL, resp.StatusCode)
		return
	}

	// 读取内容
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Warnf("读取Source Map内容失败 [%s]: %v", mapURL, err)
		return
	}

	// 生成文件路径 (保存到 encode/map/{domain}/ 目录)
	filePath, err := dc.generateFilePath(mapURL, "encode/map")
	if err != nil {
		utils.Warnf("生成Source Map文件路径失败 [%s]: %v", mapURL, err)
		return
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		utils.Warnf("创建Source Map目录失败: %v", err)
		return
	}

	// 写入文件
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		utils.Warnf("写入Source Map文件失败: %v", err)
		return
	}

	// 注意: 此时锁已经被重新获取(defer dc.mu.Lock())
	// 创建MapFile对象
	mapFile := &models.MapFile{
		ID:           uuid.New().String(),
		URL:          mapURL,
		FilePath:     filePath,
		Size:         int64(len(content)),
		DownloadedAt: time.Now(),
	}

	dc.mapFiles[mapURL] = mapFile
	dc.stats.MapFiles++

	utils.Infof("📥 下载Source Map成功: %s (%d bytes)", filepath.Base(filePath), len(content))
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
// 路径格式: output/{target_domain}/encode/js/{source_domain}/filename.js
// 例如: output/www.baidu.com/encode/js/map.baidu.com/app.js
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

	// 获取JS文件的来源域名
	sourceDomain := parsed.Host
	if sourceDomain == "" {
		sourceDomain = "unknown"
	}

	// 构造完整路径: output/{target_domain}/encode/js/{source_domain}/filename
	// 在js目录下按来源域名分类
	fullPath := filepath.Join(dc.outputDir, dc.domain, subdir, sourceDomain, filename)

	// 如果文件已存在,添加编号
	if _, err := os.Stat(fullPath); err == nil {
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		for i := 1; ; i++ {
			newPath := filepath.Join(dc.outputDir, dc.domain, subdir, sourceDomain, fmt.Sprintf("%s_%d%s", base, i, ext))
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

// Reset 重置爬取器状态,用于批量爬取场景
//
// 职责:
//   - 清空URL队列(调用URLQueue.Reset)
//   - 重置标签页池到1个标签页(调用PagePool.Reset)
//   - 清空内部状态(jsFiles, mapFiles, visitedURLs, stats)
//
// 使用场景:
//   - 批量爬取(-f参数)中,每个目标完成后调用
//   - 确保目标间的完全隔离,无URL或文件污染
//
// 注意:
//   - 不重置全局文件哈希表(globalFileHashes),因为需要跨目标去重
//   - 不关闭浏览器,复用同一浏览器实例
func (dc *DynamicCrawler) Reset() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// 重置URL队列
	if dc.urlQueue != nil {
		dc.urlQueue.Reset()
	}

	// 重置标签页池到1个标签页
	if dc.pagePool != nil {
		if err := dc.pagePool.Reset(); err != nil {
			return fmt.Errorf("重置标签页池失败: %w", err)
		}
	}

	// 清空内部状态
	dc.jsFiles = make(map[string]*models.JSFile)
	dc.mapFiles = make(map[string]*models.MapFile)
	dc.visitedURLs = make([]string, 0)
	dc.stats = models.TaskStats{}

	utils.Debugf("动态爬取器状态已重置")
	return nil
}
