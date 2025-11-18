package crawlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog/log"
)

// PageHealthStatus 标签页健康状态
// T040 [US3]: 跟踪每个标签页的健康状况,用于重试和销毁决策
type PageHealthStatus struct {
	CleanFailureCount int       // 清理失败次数
	LastSuccessTime   time.Time // 最后一次成功使用时间
	IsDirty           bool      // 是否标记为"脏"状态(清理失败2次)
}

// PagePool 标签页池管理器
// 职责: 管理浏览器标签页的生命周期,动态调整数量,协调并发访问
type PagePool struct {
	// 浏览器实例
	browser *rod.Browser

	// 所有活跃的标签页
	pages []*rod.Page

	// 可用标签页channel
	availablePages chan *rod.Page

	// 资源监控器
	resourceMonitor *ResourceMonitor

	// URL队列引用
	urlQueue *URLQueue

	// 保护pages切片的锁
	mu sync.Mutex

	// 控制生命周期的context
	ctx context.Context

	// 是否已关闭
	closed bool

	// T038 [US3]: 标签页健康状态跟踪
	pageHealth map[*rod.Page]*PageHealthStatus
	healthMu   sync.RWMutex // 保护pageHealth的锁
}

// NewPagePool 创建标签页池实例
func NewPagePool(browser *rod.Browser, resourceMonitor *ResourceMonitor, urlQueue *URLQueue, ctx context.Context) *PagePool {
	return &PagePool{
		browser:         browser,
		pages:           make([]*rod.Page, 0),
		availablePages:  make(chan *rod.Page, 32), // buffered channel, 最多缓存32个
		resourceMonitor: resourceMonitor,
		urlQueue:        urlQueue,
		ctx:             ctx,
		closed:          false,
		pageHealth:      make(map[*rod.Page]*PageHealthStatus), // T038 [US3]: 初始化健康状态map
	}
}

// AcquirePage 获取一个可用的标签页
func (pp *PagePool) AcquirePage(ctx context.Context) (*rod.Page, error) {
	// 检查是否已关闭
	pp.mu.Lock()
	if pp.closed {
		pp.mu.Unlock()
		return nil, fmt.Errorf("标签页池已关闭")
	}
	pp.mu.Unlock()

	// 尝试从可用池获取
	select {
	case page := <-pp.availablePages:
		return page, nil
	default:
		// 没有可用标签页,尝试创建新的
	}

	// 检查是否可以创建新标签页
	pp.mu.Lock()
	currentSize := len(pp.pages)
	maxSize := pp.resourceMonitor.CalculateMaxTabs()
	pp.mu.Unlock()

	if currentSize >= maxSize {
		// 已达上限,阻塞等待可用标签页
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case page := <-pp.availablePages:
			return page, nil
		}
	}

	// 检查资源可用性
	canCreate, reason := pp.resourceMonitor.CheckResourceAvailability()
	if !canCreate {
		log.Warn().Msgf("资源不足,无法创建新标签页: %s", reason)
		// 等待可用标签页
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case page := <-pp.availablePages:
			return page, nil
		}
	}

	// 创建新标签页
	page, err := pp.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		// 浏览器可能已崩溃或连接断开
		log.Error().Err(err).Msg("创建标签页失败,浏览器可能已崩溃")
		return nil, fmt.Errorf("创建标签页失败(浏览器可能已崩溃): %w", err)
	}

	// 添加到pages列表
	pp.mu.Lock()
	pp.pages = append(pp.pages, page)
	currentSize = len(pp.pages)
	pp.mu.Unlock()

	// T038 [US3]: 初始化新标签页的健康状态
	pp.healthMu.Lock()
	pp.pageHealth[page] = &PageHealthStatus{
		CleanFailureCount: 0,
		LastSuccessTime:   time.Now(),
		IsDirty:           false,
	}
	pp.healthMu.Unlock()

	log.Debug().Msgf("创建新标签页,当前标签页数: %d, 最大限制: %d", currentSize, maxSize)

	return page, nil
}

// ReleasePage 归还标签页到池中
// T039 [US3]: 实现清理失败重试策略
func (pp *PagePool) ReleasePage(page *rod.Page) {
	if page == nil {
		return
	}

	// 获取当前页面的健康状态
	pp.healthMu.RLock()
	health, exists := pp.pageHealth[page]
	pp.healthMu.RUnlock()

	if !exists {
		// 页面不存在健康记录(可能是旧页面),直接销毁
		log.Warn().Msg("标签页没有健康记录,直接销毁")
		pp.destroyPage(page)
		return
	}

	// 清理标签页状态
	err := pp.cleanPage(page)
	if err != nil {
		// T039 [US3]: 清理失败,执行重试策略
		pp.healthMu.Lock()
		health.CleanFailureCount++
		failureCount := health.CleanFailureCount
		pp.healthMu.Unlock()

		log.Warn().Err(err).Msgf("清理标签页状态失败 (第%d次失败)", failureCount)

		if failureCount == 1 {
			// 第一次失败: 重试一次
			log.Info().Msg("第一次清理失败,尝试重试")
			err = pp.cleanPage(page)
			if err == nil {
				// 重试成功,重置失败计数
				pp.healthMu.Lock()
				health.CleanFailureCount = 0
				health.LastSuccessTime = time.Now()
				health.IsDirty = false
				pp.healthMu.Unlock()
				log.Info().Msg("重试清理成功,标签页恢复正常")
			} else {
				// 重试仍然失败,增加失败计数
				pp.healthMu.Lock()
				health.CleanFailureCount++
				pp.healthMu.Unlock()
				log.Warn().Err(err).Msg("重试清理失败")
			}
		} else if failureCount == 2 {
			// 第二次失败: 标记为"脏"状态,但仍然保留
			pp.healthMu.Lock()
			health.IsDirty = true
			pp.healthMu.Unlock()
			log.Warn().Msg("标签页标记为'脏'状态(清理失败2次),下次失败将销毁")
		} else {
			// 第三次失败: 销毁该标签页
			log.Warn().Msg("清理失败超过3次,销毁该标签页")
			pp.destroyPage(page)
			return
		}
	} else {
		// 清理成功,重置健康状态
		pp.healthMu.Lock()
		health.CleanFailureCount = 0
		health.LastSuccessTime = time.Now()
		health.IsDirty = false
		pp.healthMu.Unlock()
	}

	// 检查是否应该销毁
	pp.mu.Lock()
	currentSize := len(pp.pages)
	pendingCount := pp.urlQueue.PendingCount()
	pp.mu.Unlock()

	// 如果队列为空且当前标签页数>1,销毁该标签页
	if pendingCount == 0 && currentSize > 1 {
		pp.destroyPage(page)
		return
	}

	// 归还到可用池
	select {
	case pp.availablePages <- page:
		// 成功归还
	default:
		// channel已满,销毁该标签页
		pp.destroyPage(page)
	}
}

// cleanPage 清理标签页状态
func (pp *PagePool) cleanPage(page *rod.Page) error {
	// T025-T028 [US2]: 修改JavaScript代码,添加防御性检查
	// 使用page.Evaluate代替page.Eval,支持多语句JavaScript
	_, err := page.Evaluate(&rod.EvalOptions{
		JS: `() => {
			// 清理localStorage
			if (typeof localStorage !== 'undefined' && localStorage !== null) {
				try {
					localStorage.clear();
				} catch (e) {
					// ignore
				}
			}

			// 清理sessionStorage
			if (typeof sessionStorage !== 'undefined' && sessionStorage !== null) {
				try {
					sessionStorage.clear();
				} catch (e) {
					// ignore
				}
			}

			// 清理cookies
			if (typeof document !== 'undefined' && document !== null && document.cookie) {
				try {
					var cookies = document.cookie.split(";");
					for (var i = 0; i < cookies.length; i++) {
						var c = cookies[i];
						var eqPos = c.indexOf("=");
						var name = eqPos > -1 ? c.substr(0, eqPos) : c;
						document.cookie = name.replace(/^ +/, "") + "=;expires=Thu, 01 Jan 1970 00:00:00 UTC;path=/";
					}
				} catch (e) {
					// ignore
				}
			}

			return true;
		}`,
	})
	if err != nil {
		return fmt.Errorf("清理标签页状态失败: %w", err)
	}

	return nil
}

// destroyPage 销毁标签页
func (pp *PagePool) destroyPage(page *rod.Page) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	// 从pages列表中移除
	for i, p := range pp.pages {
		if p == page {
			pp.pages = append(pp.pages[:i], pp.pages[i+1:]...)
			break
		}
	}

	// T038 [US3]: 清理健康状态记录
	pp.healthMu.Lock()
	delete(pp.pageHealth, page)
	pp.healthMu.Unlock()

	// 关闭标签页
	err := page.Close()
	if err != nil {
		log.Warn().Err(err).Msg("关闭标签页失败")
	}

	log.Debug().Msgf("销毁标签页,当前标签页数: %d", len(pp.pages))
}

// AdjustSize 根据待爬URL数量和资源限制调整标签页池大小
func (pp *PagePool) AdjustSize(pendingURLCount int) {
	pp.mu.Lock()
	currentSize := len(pp.pages)
	pp.mu.Unlock()

	maxSize := pp.resourceMonitor.CalculateMaxTabs()

	// 如果待爬URL数量大于当前标签页数,且未达上限,创建新标签页
	if pendingURLCount > currentSize && currentSize < maxSize {
		targetSize := pendingURLCount
		if targetSize > maxSize {
			targetSize = maxSize
		}

		toCreate := targetSize - currentSize
		for i := 0; i < toCreate; i++ {
			// 检查资源可用性
			canCreate, reason := pp.resourceMonitor.CheckResourceAvailability()
			if !canCreate {
				log.Warn().Msgf("资源不足,无法创建更多标签页: %s", reason)
				break
			}

			// 创建新标签页
			page, err := pp.browser.Page(proto.TargetCreateTarget{})
			if err != nil {
				log.Error().Err(err).Msg("创建标签页失败,浏览器可能已崩溃")
				break
			}

			pp.mu.Lock()
			pp.pages = append(pp.pages, page)
			currentSize = len(pp.pages)
			pp.mu.Unlock()

			// T038 [US3]: 初始化新标签页的健康状态
			pp.healthMu.Lock()
			pp.pageHealth[page] = &PageHealthStatus{
				CleanFailureCount: 0,
				LastSuccessTime:   time.Now(),
				IsDirty:           false,
			}
			pp.healthMu.Unlock()

			// 添加到可用池
			pp.availablePages <- page

			log.Info().Msgf("当前标签页: %d, 待爬URL数: %d, 最大限制: %d", currentSize, pendingURLCount, maxSize)
		}
	}

	// 如果待爬URL为0且当前标签页数>1,缩减到1个
	if pendingURLCount == 0 && currentSize > 1 {
		pp.mu.Lock()
		toDestroy := pp.pages[1:] // 保留第一个
		pp.pages = pp.pages[:1]
		pp.mu.Unlock()

		for _, page := range toDestroy {
			err := page.Close()
			if err != nil {
				log.Warn().Err(err).Msg("关闭标签页失败")
			}
		}

		log.Info().Msgf("爬取完成,缩减标签页至1个")
	}
}

// CurrentSize 返回当前标签页池的大小
func (pp *PagePool) CurrentSize() int {
	pp.mu.Lock()
	defer pp.mu.Unlock()
	return len(pp.pages)
}

// MaxSize 返回当前允许的最大标签页数
func (pp *PagePool) MaxSize() int {
	return pp.resourceMonitor.CalculateMaxTabs()
}

// Reset 重置标签页池到初始状态(1个标签页)
func (pp *PagePool) Reset() error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	// 清空可用池channel
	for len(pp.availablePages) > 0 {
		<-pp.availablePages
	}

	// 销毁所有标签页(除了第一个)
	if len(pp.pages) > 1 {
		for _, page := range pp.pages[1:] {
			err := page.Close()
			if err != nil {
				log.Warn().Err(err).Msg("关闭标签页失败")
			}
		}
		pp.pages = pp.pages[:1]
	}

	// 如果没有标签页,创建一个
	if len(pp.pages) == 0 {
		page, err := pp.browser.Page(proto.TargetCreateTarget{})
		if err != nil {
			return fmt.Errorf("创建标签页失败: %w", err)
		}
		pp.pages = append(pp.pages, page)
		pp.availablePages <- page
	} else {
		// 将第一个标签页放回可用池
		pp.availablePages <- pp.pages[0]
	}

	log.Info().Msg("标签页池已重置为1个标签页")
	return nil
}

// Close 关闭标签页池,释放所有资源
func (pp *PagePool) Close() error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if pp.closed {
		return nil
	}

	// 关闭所有标签页
	for _, page := range pp.pages {
		err := page.Close()
		if err != nil {
			log.Warn().Err(err).Msg("关闭标签页失败")
		}
	}

	pp.pages = nil
	close(pp.availablePages)
	pp.closed = true

	log.Info().Msg("标签页池已关闭")
	return nil
}
