package crawlers

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
	"github.com/andybalholm/brotli"
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

	// URL队列管理(替代visitedURLs)
	urlQueue *URLQueue

	// 资源监控器
	resourceMonitor *ResourceMonitor

	// 统计
	stats models.TaskStats
}

// NewStaticCrawler 创建静态爬取器
func NewStaticCrawler(config models.CrawlConfig, outputDir string, domain string, globalFileHashes map[string]string, globalMu *sync.RWMutex, headerProvider models.HeaderProvider) *StaticCrawler {
	// Bug #1修复 (T012, T013): 创建自定义HTTP客户端,禁用TLS证书验证
	// 参考: research.md - 静态爬取证书修复方案
	// HTTP超时时间直接使用配置文件的 wait_time 值(秒)
	httpTimeout := time.Duration(config.WaitTime) * time.Second

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 跳过证书验证,允许访问自签名、过期或主机名不匹配的HTTPS站点
			},
		},
		Timeout: httpTimeout,
	}
	utils.Debugf("静态爬取器: HTTP超时设置为 %d 秒 (wait_time=%d)", int(httpTimeout.Seconds()), config.WaitTime)

	// 创建Colly collector
	// T053: 不使用colly.MaxDepth,改为应用层手动管理深度
	// 注意: 必须设置 colly.AllowURLRevisit(false) 来禁用Colly的内部域名检查
	c := colly.NewCollector(
		// colly.MaxDepth(config.Depth), // 移除自动深度限制
		colly.Async(true),
		// 不设置AllowedDomains,完全由应用层控制域名访问
	)

	// 设置自定义HTTP客户端 (包含TLS配置和超时设置)
	c.SetClient(httpClient)
	utils.Debugf("静态爬取器: TLS证书验证已禁用,适用于内网/开发环境的自签名证书")

	// 根据AllowCrossDomain配置决定是否限制域名
	// 注意: 不使用Colly的AllowedDomains,因为它会导致"Forbidden"错误
	// 原因: Colly的AllowedDomains是精确匹配,会阻止子域名访问(如 www.baidu.com -> xueshu.baidu.com)
	// 改为在OnRequest回调中手动检查域名,提供更灵活的控制
	if !config.AllowCrossDomain {
		utils.Debugf("静态爬取器: 将在应用层检查域名限制 (目标域名: %s)", domain)
	} else {
		utils.Debugf("静态爬取器: 允许跨域爬取 (无域名限制)")
	}

	// 初始化URL队列
	urlQueue := NewURLQueue(domain, config.AllowCrossDomain, config.Depth)

	// 初始化资源监控器
	resourceMonitor := NewResourceMonitor(ResourceMonitorConfig{
		SafetyReserveMemory: int64(config.SafetyReserveMemory) * 1024 * 1024, // MB转字节
		SafetyThreshold:     int64(config.SafetyThreshold) * 1024 * 1024,
		CPULoadThreshold:    config.CPULoadThreshold,
		MaxTabsLimit:        config.MaxTabsLimit,
		TabMemoryUsage:      100 * 1024 * 1024, // 100MB per worker
	})

	// 启动资源监控(1秒采样间隔)
	resourceMonitor.StartMonitoring(1 * time.Second)

	// 初始并发数设为配置的MaxWorkers
	initialWorkers := config.MaxWorkers
	if initialWorkers < 1 {
		initialWorkers = 1
	}

	// 配置并发限制
	// 无延迟,最大化爬取速度
	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: initialWorkers,
		Delay:       0, // 无延迟
	}); err != nil {
		utils.Warnf("设置并发限制失败: %v", err)
	}

	utils.Debugf("静态爬取器并发优化: 初始并发=%d, 最大限制=%d, 无延迟",
		initialWorkers, config.MaxTabsLimit)

	// 设置超时
	c.SetRequestTimeout(30 * time.Second)

	// T059-T061: 应用自定义HTTP客户端到Colly,以启用TLS跳过验证
	c.WithTransport(httpClient.Transport)

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
		urlQueue:         urlQueue,
		resourceMonitor:  resourceMonitor,
		stats:            models.TaskStats{},
	}

	// 设置回调
	sc.setupCallbacks()

	return sc
}

// setupCallbacks 设置Colly回调
func (sc *StaticCrawler) setupCallbacks() {
	// 提取页面链接(用于深度爬取导航)
	// T053: 手动管理深度检查,仅对页面链接应用深度限制
	sc.collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))

		// 检查URL有效性
		if link == "" || !strings.HasPrefix(link, "http") {
			return
		}

		// 检查是否已访问
		if sc.urlQueue.IsVisited(link) {
			return
		}

		// 手动深度检查: 只对页面链接检查深度
		currentDepth := e.Request.Depth
		if currentDepth >= sc.config.Depth {
			utils.Debugf("页面深度达到限制: %s (深度=%d, 限制=%d)", link, currentDepth, sc.config.Depth)
			return
		}

		// 手动域名检查(如果AllowCrossDomain=false)
		if !sc.config.AllowCrossDomain {
			parsedURL, err := url.Parse(link)
			if err == nil && parsedURL.Host != sc.domain {
				utils.Debugf("跳过跨域链接: %s (目标域名: %s)", link, sc.domain)
				return
			}
		}

		// 标记已访问
		sc.urlQueue.MarkVisited(link)

		// 访问链接(用于页面导航)
		if err := e.Request.Visit(link); err != nil {
			// 只在非Forbidden错误时记录日志
			if !strings.Contains(err.Error(), "Forbidden") {
				utils.Debugf("访问链接失败 [%s]: %v", link, err)
			}
		}
	})

	// 提取script标签中的JavaScript文件(JS资源)
	// T056: JS文件不检查深度,无条件访问(深度豁免)
	sc.collector.OnHTML("script[src]", func(e *colly.HTMLElement) {
		jsURL := e.Request.AbsoluteURL(e.Attr("src"))
		if sc.isJavaScriptURL(jsURL) {
			utils.Debugf("发现JS文件: %s", jsURL)

			// 无条件访问JS文件,不检查深度(深度豁免)
			if err := e.Request.Visit(jsURL); err != nil {
				utils.Warnf("访问JS文件失败 [%s]: %v", jsURL, err)
			}
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
	// T013: 集成isValidJavaScript内容检测,绕过假404响应
	sc.collector.OnResponse(func(r *colly.Response) {
		requestURL := r.Request.URL.String()

		// 如果是JavaScript文件,进行内容检测后下载
		if sc.isJavaScriptURL(requestURL) {
			contentType := r.Headers.Get("Content-Type")
			contentEncoding := r.Headers.Get("Content-Encoding")

			// 解压响应体(如果有压缩)
			body := r.Body
			if contentEncoding != "" {
				decompressed, err := decompressResponse(contentEncoding, r.Body)
				if err != nil {
					utils.Warnf("解压响应失败 [%s] (编码=%s): %v", requestURL, contentEncoding, err)
					// 解压失败,仍然尝试使用原始body
				} else {
					body = decompressed
					utils.Debugf("成功解压响应 [%s]: 原始=%d bytes, 解压后=%d bytes", requestURL, len(r.Body), len(body))
				}
			}

			// 智能内容检测: 无论HTTP状态码如何,都检查内容是否为有效JS
			if isValidJavaScript(contentType, body) {
				// 内容检测通过,下载文件
				if err := sc.downloadJSFile(requestURL, body, contentType); err != nil {
					utils.Warnf("下载JS文件失败 [%s]: %v", requestURL, err)
					sc.stats.FailedFiles++
				} else {
					// HTTP错误但内容有效的情况,记录FakeHTTPErrors
					if r.StatusCode >= 400 {
						sc.stats.FakeHTTPErrors++
						utils.Debugf("检测到假HTTP错误 [%s]: 状态码%d但内容有效", requestURL, r.StatusCode)
					}
				}
			} else {
				// T014: 内容检测失败,记录为FailedFiles
				utils.Infof("访问JS文件失败 [%s]: 内容检测失败,非有效JavaScript文件", requestURL)
				sc.stats.FailedFiles++
			}
		}
	})

	// 错误处理
	sc.collector.OnError(func(r *colly.Response, err error) {
		// 如果是Forbidden错误且配置允许跨域,则忽略(这是Colly的内部域名检查导致的误报)
		if strings.Contains(err.Error(), "Forbidden") && sc.config.AllowCrossDomain {
			utils.Debugf("忽略Colly的域名检查错误 [%s]: %v (配置允许跨域)", r.Request.URL, err)
			return
		}

		utils.Errorf("爬取错误 [%s]: %v", r.Request.URL, err)
		sc.stats.FailedFiles++
	})

	// 访问前
	sc.collector.OnRequest(func(r *colly.Request) {
		// 手动域名检查(如果AllowCrossDomain=false)
		if !sc.config.AllowCrossDomain {
			parsedURL, err := url.Parse(r.URL.String())
			if err == nil {
				// 检查是否为同一域名
				if parsedURL.Host != sc.domain {
					utils.Debugf("拒绝跨域请求: %s (目标域名: %s)", r.URL.String(), sc.domain)
					r.Abort()
					return
				}
			}
		}

		// T054: 判断是否为JavaScript资源
		if IsJavaScriptResource(r.URL.String()) {
			// T055: 对JS资源请求设置context标记,跳过深度检查
			r.Ctx.Put("is_js_resource", "true")
			// T057: 添加DEBUG级别日志记录JS文件豁免行为
			utils.Debugf("检测到JS资源,豁免深度限制: %s", r.URL.String())
		}

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
		sc.stats.VisitedURLs++
		sc.mu.Unlock()

		// 动态调整并发数(基于队列长度和资源限制)
		sc.adjustConcurrency()
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

	// 添加进度监控goroutine
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				sc.mu.RLock()
				visitedCount := sc.stats.VisitedURLs
				downloadedCount := sc.stats.StaticFiles
				failedCount := sc.stats.FailedFiles
				sc.mu.RUnlock()

				utils.Infof("进度: 已访问 %d 个URL, 已下载 %d 个文件, 失败 %d 个",
					visitedCount, downloadedCount, failedCount)
			}
		}
	}()

	// 使用带超时的Wait,避免无限等待
	// 创建一个goroutine来执行Wait
	waitDone := make(chan struct{})
	go func() {
		sc.collector.Wait()
		close(waitDone)
	}()

	// 设置全局超时: 最多等待5分钟
	globalTimeout := 5 * time.Minute
	select {
	case <-waitDone:
		// 正常完成
		utils.Debugf("静态爬取正常完成")
	case <-time.After(globalTimeout):
		// 超时
		utils.Warnf("静态爬取超时(等待%v),强制结束", globalTimeout)
	}

	close(done) // 通知监控goroutine退出

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

	utils.Infof("📥 下载成功: %s (%d bytes) - %s", filepath.Base(filePath), len(content), fileURL)

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

			// 下载Source Map文件
			sc.downloadSourceMapFile(fullMapURL.String())
		}
	}
}

// downloadSourceMapFile 下载Source Map文件
// 注意: 调用此函数前调用者必须已持有 sc.mu 锁
func (sc *StaticCrawler) downloadSourceMapFile(mapURL string) {
	// 检查是否已下载 (不需要额外加锁,调用者已持有锁)
	if _, exists := sc.mapFiles[mapURL]; exists {
		utils.Debugf("Source Map文件已存在,跳过: %s", mapURL)
		return
	}

	// 临时释放锁以执行HTTP请求(避免阻塞其他操作)
	sc.mu.Unlock()
	defer sc.mu.Lock()

	// 发起HTTP请求下载
	client := &http.Client{
		Timeout: 30 * time.Second,
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
	filePath, err := sc.generateFilePath(mapURL, "encode/map")
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

	// 注意: 此时锁已经被重新获取(defer sc.mu.Lock())
	// 创建MapFile对象
	mapFile := &models.MapFile{
		ID:           uuid.New().String(),
		URL:          mapURL,
		FilePath:     filePath,
		Size:         int64(len(content)),
		DownloadedAt: time.Now(),
	}

	sc.mapFiles[mapURL] = mapFile
	sc.stats.MapFiles++

	utils.Infof("📥 下载Source Map成功: %s (%d bytes)", filepath.Base(filePath), len(content))
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

// IsJavaScriptResource 判断URL是否为JavaScript资源文件
// 用于深度豁免逻辑: JS资源不受深度限制影响
//
// 判断规则:
//   - 方法1: 检查文件扩展名 (.js, .mjs)
//   - 方法2: 检查URL路径特征 (/js/, /javascript/, /scripts/, .min.js)
//
// 返回: true表示URL是JS资源,应豁免深度限制
func IsJavaScriptResource(urlStr string) bool {
	// 转换为小写以支持大小写不敏感匹配
	lowerURL := strings.ToLower(urlStr)

	// 方法1: 检查文件扩展名 (.js, .mjs)
	// 需要处理查询参数,如 app.js?v=1.0
	if strings.HasSuffix(lowerURL, ".js") || strings.HasSuffix(lowerURL, ".mjs") {
		return true
	}
	// 检查是否包含.js?或.mjs?模式
	if strings.Contains(lowerURL, ".js?") || strings.Contains(lowerURL, ".mjs?") {
		return true
	}

	// 方法2: 检查URL路径特征 (如 /static/js/, /assets/scripts/)
	jsPathPatterns := []string{"/js/", "/javascript/", "/scripts/", ".min.js"}
	for _, pattern := range jsPathPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}

	return false
}

// generateFilePath 生成本地文件路径
// 路径格式: output/{target_domain}/encode/js/{source_domain}/filename.js
// 例如: output/www.baidu.com/encode/js/map.baidu.com/app.js
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

	// 获取JS文件的来源域名
	sourceDomain := parsed.Host
	if sourceDomain == "" {
		sourceDomain = "unknown"
	}

	// 构造完整路径: output/{target_domain}/encode/js/{source_domain}/filename
	// 在js目录下按来源域名分类
	fullPath := filepath.Join(sc.outputDir, sc.domain, subdir, sourceDomain, filename)

	// 如果文件已存在,添加编号
	if _, err := os.Stat(fullPath); err == nil {
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		for i := 1; ; i++ {
			newPath := filepath.Join(sc.outputDir, sc.domain, subdir, sourceDomain, fmt.Sprintf("%s_%d%s", base, i, ext))
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

// adjustConcurrency 动态调整并发数(基于队列长度和资源限制)
// 策略:
//   - 基于ResourceMonitor计算的maxTabs作为并发上限
//   - 根据待爬URL数量(PendingCount)按需调整
//   - 并发数 = min(PendingCount, maxTabs)
func (sc *StaticCrawler) adjustConcurrency() {
	// 获取资源限制的最大并发数
	maxTabs := sc.resourceMonitor.CalculateMaxTabs()

	// 获取待爬URL数量
	pendingCount := sc.urlQueue.PendingCount()

	// 计算最优并发数: min(待爬URL数, 资源限制)
	// 至少保持1个并发
	optimalWorkers := 1
	if pendingCount > 0 {
		if pendingCount < maxTabs {
			optimalWorkers = pendingCount
		} else {
			optimalWorkers = maxTabs
		}
	}

	// 更新Colly的并发限制
	if err := sc.collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: optimalWorkers,
		Delay:       0, // 无延迟
	}); err != nil {
		utils.Warnf("更新并发限制失败: %v", err)
		return
	}

	utils.Debugf("静态爬取器并发调整: 待爬URL=%d, 当前并发=%d, 最大限制=%d",
		pendingCount, optimalWorkers, maxTabs)
}

// Reset 重置爬取器状态,用于批量爬取场景
//
// 职责:
//   - 清空URL队列(调用URLQueue.Reset)
//   - 清空内部状态(jsFiles, mapFiles, stats)
//   - 重新创建Colly collector实例
//
// 使用场景:
//   - 批量爬取(-f参数)中,每个目标完成后调用
//   - 确保目标间的完全隔离,无URL或文件污染
//
// 注意:
//   - 不重置全局文件哈希表(globalFileHashes),因为需要跨目标去重
//   - 不重置ResourceMonitor,因为是全局资源监控
//   - 需要重新创建collector,因为Colly的访问历史无法清空
func (sc *StaticCrawler) Reset() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 重置URL队列
	if sc.urlQueue != nil {
		sc.urlQueue.Reset()
	}

	// 清空内部状态
	sc.jsFiles = make(map[string]*models.JSFile)
	sc.mapFiles = make(map[string]*models.MapFile)
	sc.stats = models.TaskStats{}

	// 重新创建collector实例
	sc.collector = colly.NewCollector(
		// colly.MaxDepth(sc.config.Depth), // 移除自动深度限制,使用应用层手动管理
		colly.Async(true),
	)

	// 不使用AllowedDomains,改为在OnRequest中手动检查
	// 参见setupCallbacks()中的域名检查逻辑

	// 设置超时
	sc.collector.SetRequestTimeout(30 * time.Second)

	// 配置初始并发限制
	initialWorkers := sc.config.MaxWorkers
	if initialWorkers < 1 {
		initialWorkers = 1
	}
	if err := sc.collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: initialWorkers,
		Delay:       0, // 无延迟
	}); err != nil {
		utils.Warnf("设置并发限制失败: %v", err)
	}

	// 重新设置回调
	sc.setupCallbacks()

	utils.Debugf("静态爬取器状态已重置")
	return nil
}

// isValidJavaScript 检测HTTP响应内容是否为有效的JavaScript文件
// 用于绕过反爬虫的假404响应(返回404但body包含真实JS代码)
// 参数:
//
//	contentType: HTTP响应头Content-Type
//	body: 响应体内容
//
// 返回: 是否为有效JavaScript文件
// 契约参考: contracts/module-contracts.md - 内容检测契约
func isValidJavaScript(contentType string, body []byte) bool {
	// 1. Content-Type检测: 最可靠的指标
	if strings.Contains(strings.ToLower(contentType), "javascript") {
		return true
	}

	// 2. 内容特征检测(检查前1KB,避免性能问题)
	sample := body
	if len(body) > 1024 {
		sample = body[:1024]
	}

	// JavaScript关键字列表
	jsKeywords := []string{"function", "var", "const", "let", "class", "import", "export", "=>"}
	matchCount := 0
	for _, keyword := range jsKeywords {
		if strings.Contains(string(sample), keyword) {
			matchCount++
		}
	}

	// 至少匹配2个关键字才认为是JS(避免误判,如HTML中偶尔出现"function"字样)
	return matchCount >= 2
}

// decompressResponse 根据Content-Encoding头部解压响应体
// 支持 gzip, deflate, br (Brotli) 三种压缩格式
func decompressResponse(contentEncoding string, body []byte) ([]byte, error) {
	encoding := strings.ToLower(strings.TrimSpace(contentEncoding))

	switch encoding {
	case "gzip":
		// GZIP解压
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("gzip解压失败: %w", err)
		}
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("gzip读取失败: %w", err)
		}
		return decompressed, nil

	case "deflate":
		// Deflate解压
		reader := flate.NewReader(bytes.NewReader(body))
		defer reader.Close()

		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("deflate读取失败: %w", err)
		}
		return decompressed, nil

	case "br":
		// Brotli解压
		reader := brotli.NewReader(bytes.NewReader(body))
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("brotli读取失败: %w", err)
		}
		return decompressed, nil

	case "":
		// 没有压缩,直接返回原始内容
		return body, nil

	default:
		// 未知编码,返回警告但仍然返回原始内容
		utils.Warnf("未知的Content-Encoding: %s", contentEncoding)
		return body, nil
	}
}
