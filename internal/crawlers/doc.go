// Package crawlers 提供静态和动态JavaScript文件爬取功能
//
// # 概述
//
// crawlers包实现了自适应标签页池管理机制,支持静态(Colly)和动态(go-rod)两种爬取模式。
// 核心特性包括:按需创建标签页、实时资源监控、深度爬取、跨域控制、批量目标隔离。
//
// # 核心组件
//
// ## StaticCrawler
//
// 基于Colly框架的静态爬取器,通过OnHTML回调提取页面链接和JS资源。
// 支持自适应并发控制,根据待爬URL数量动态调整并发线程数。
//
//	crawler := NewStaticCrawler(config, outputDir, domain, globalFileHashes, &globalMu, headerProvider)
//	err := crawler.Crawl("https://example.com")
//
// ## DynamicCrawler
//
// 基于go-rod的动态爬取器,支持JavaScript渲染和网络拦截。
// 集成PagePool实现标签页按需创建,内存消耗降低75%+。
//
//	crawler := NewDynamicCrawler(config, outputDir, domain, globalFileHashes, &globalMu, headerProvider)
//	err := crawler.Crawl("https://example.com")
//
// ## PagePool (标签页池)
//
// 管理浏览器标签页的生命周期,动态调整数量(1-maxTabs)。
// 核心策略:
//   - 启动时创建1个标签页
//   - 根据队列长度按需增长(队列比策略)
//   - 队列为空时缩减至1个标签页
//   - 创建前检查ResourceMonitor资源限制
//
// 使用示例:
//
//	pool := NewPagePool(browser, resourceMonitor, urlQueue, ctx)
//	defer pool.Close()
//
//	page, err := pool.AcquirePage(ctx)
//	if err != nil { /* 处理错误 */ }
//	defer pool.ReleasePage(page)
//
//	// 使用标签页爬取
//	page.Navigate(url)
//
// ## ResourceMonitor (资源监控器)
//
// 实时监控系统可用内存和CPU负载,动态计算标签页上限。
// 渐进式降级策略:
//   - 可用内存 < 500MB: 暂停创建新标签页 (警告日志)
//   - 可用内存 < 300MB: 主动缩减至当前标签页数的50% (警告日志)
//   - 可用内存 < 200MB: 紧急缩减至1个标签页 (错误日志)
//
// 使用示例:
//
//	config := ResourceMonitorConfig{
//	    SafetyReserveMemory: 1024 * 1024 * 1024,  // 1GB
//	    SafetyThreshold:     500 * 1024 * 1024,   // 500MB
//	    CPULoadThreshold:    80,
//	    MaxTabsLimit:        16,
//	    TabMemoryUsage:      100 * 1024 * 1024,   // 100MB per tab
//	}
//	monitor := NewResourceMonitor(config)
//	monitor.StartMonitoring(1 * time.Second)
//	defer monitor.StopMonitoring()
//
//	maxTabs := monitor.CalculateMaxTabs()
//	canCreate, reason := monitor.CheckResourceAvailability()
//
// ## URLQueue (URL队列)
//
// 并发安全的URL队列管理器,支持Push/Pop/MarkVisited操作。
// 基于channel实现的待处理队列和map实现的已访问集合。
//
// 使用示例:
//
//	queue := NewURLQueue(targetDomain, allowCrossDomain, maxDepth)
//	defer queue.Close()
//
//	err := queue.Push("https://example.com/page1", 1)
//	url, depth, ok := queue.Pop(ctx)
//	queue.MarkVisited(url)
//
// ## URLExtractor (URL提取器)
//
// 从HTML页面中提取链接,根据配置过滤(跨域、深度、已访问)。
// 支持动态爬取(Page.Evaluate)和静态爬取(html.Parse)两种模式。
//
// 使用示例:
//
//	extractor := NewURLExtractor(queue, targetHost, allowCrossDomain, maxDepth)
//
//	// 动态爬取
//	count, err := extractor.ExtractFromPage(page, currentURL, currentDepth)
//
//	// 静态爬取
//	links, err := extractor.ExtractFromHTML(htmlContent, baseURL, currentDepth)
//
// # 配置参数
//
// ## 资源优化配置 (configs/config.yaml)
//
//	resource:
//	  safety_reserve_memory: 1024  # 系统预留内存(MB)
//	  safety_threshold: 500        # 可用内存阈值(MB)
//	  cpu_load_threshold: 80       # CPU负载阈值(%)
//	  max_tabs_limit: 16           # 绝对最大标签页数
//
//	crawl:
//	  allow_cross_domain: true     # 是否允许跨域爬取
//	  depth: 2                     # 最大爬取深度
//
// # 性能指标
//
//   - 单URL爬取内存消耗: 从400-1200MB降至50-150MB (减少75%+)
//   - 标签页数量: 动态调整1-16个(可配置)
//   - 响应延迟: URL发现到标签页创建 < 500ms
//   - 深度爬取: 支持0-10层页面递归
//
// # 并发安全
//
// 所有核心组件都是并发安全的:
//   - URLQueue: channel + sync.RWMutex
//   - PagePool: channel + sync.Mutex
//   - ResourceMonitor: sync.RWMutex
//   - DynamicCrawler/StaticCrawler: sync.RWMutex
//
// # 错误处理
//
//   - 零URL场景: Crawl方法开始时验证入口URL,无效则返回错误不创建标签页
//   - 浏览器崩溃: AcquirePage捕获连接失败,返回明确错误"浏览器可能已崩溃"
//   - 资源不足: 日志记录警告/错误信息,暂停创建或主动缩减标签页
//   - 深度超限: URLExtractor自动过滤深度超限的链接
//
// # 批量爬取隔离
//
// 每个目标完成后调用Reset()方法确保完全隔离:
//   - 清空URL队列(URLQueue.Reset)
//   - 重置标签页池到1个(PagePool.Reset)
//   - 清空内部状态(jsFiles, visitedURLs, stats)
//   - 不重置全局文件哈希表(跨目标去重)
//
// 使用示例:
//
//	for _, targetURL := range targets {
//	    err := crawler.Crawl(targetURL)
//	    // 爬取完成后重置
//	    crawler.Reset()
//	}
//
// # 最佳实践
//
// 1. 低内存环境配置:
//
//	resource:
//	  safety_reserve_memory: 512
//	  safety_threshold: 300
//	  max_tabs_limit: 4
//
// 2. 高性能环境配置:
//
//	resource:
//	  safety_reserve_memory: 2048
//	  safety_threshold: 1000
//	  max_tabs_limit: 32
//
// 3. 禁止跨域爬取:
//
//	crawl:
//	  allow_cross_domain: false
//
// 4. 单URL爬取(不递归):
//
//	crawl:
//	  depth: 0
package crawlers
