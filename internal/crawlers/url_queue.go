package crawlers

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
)

// URLQueue URL队列管理器
// 职责: 管理待爬取和已访问的URL,支持并发安全的Push/Pop操作
type URLQueue struct {
	// 待处理URL队列
	pendingURLs chan models.URLItem

	// 已访问URL标记集合
	visitedURLs map[string]bool

	// 保护visitedURLs的读写锁
	mu sync.RWMutex

	// 目标域名(用于跨域过滤)
	targetDomain string

	// 是否允许跨域爬取
	allowCrossDomain bool

	// 最大爬取深度
	maxDepth int

	// 队列是否已关闭
	closed bool
}

// NewURLQueue 创建URL队列实例
func NewURLQueue(targetDomain string, allowCrossDomain bool, maxDepth int) *URLQueue {
	return &URLQueue{
		pendingURLs:      make(chan models.URLItem, 1000), // buffered channel,容量1000
		visitedURLs:      make(map[string]bool),
		targetDomain:     targetDomain,
		allowCrossDomain: allowCrossDomain,
		maxDepth:         maxDepth,
		closed:           false,
	}
}

// Push 添加URL到待爬队列
// 检查URL有效性、深度限制、跨域过滤、已访问检查
func (q *URLQueue) Push(urlStr string, depth int) error {
	// 检查队列是否已关闭
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return fmt.Errorf("队列已关闭")
	}
	q.mu.RUnlock()

	// 检查URL有效性
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("URL格式无效: %w", err)
	}

	// 检查协议
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("不支持的协议: %s", parsedURL.Scheme)
	}

	// 检查深度限制
	if depth > q.maxDepth {
		return fmt.Errorf("深度超过限制: %d > %d", depth, q.maxDepth)
	}

	// 检查跨域
	if !q.allowCrossDomain && parsedURL.Host != q.targetDomain {
		return fmt.Errorf("跨域链接已过滤: %s (目标域名: %s)", parsedURL.Host, q.targetDomain)
	}

	// 检查是否已访问
	q.mu.RLock()
	if q.visitedURLs[urlStr] {
		q.mu.RUnlock()
		return fmt.Errorf("URL已访问: %s", urlStr)
	}
	q.mu.RUnlock()

	// 添加到队列
	q.pendingURLs <- models.URLItem{
		URL:   urlStr,
		Depth: depth,
	}

	return nil
}

// Pop 从队列中取出下一个待爬URL
// 从channel读取URL,支持context取消,阻塞等待
func (q *URLQueue) Pop(ctx context.Context) (string, int, bool) {
	select {
	case <-ctx.Done():
		// Context取消
		return "", 0, false
	case item, ok := <-q.pendingURLs:
		if !ok {
			// Channel已关闭
			return "", 0, false
		}
		return item.URL, item.Depth, true
	}
}

// MarkVisited 标记URL为已访问
// 读写锁保护visited map
func (q *URLQueue) MarkVisited(urlStr string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.visitedURLs[urlStr] = true
}

// IsVisited 检查URL是否已访问
func (q *URLQueue) IsVisited(urlStr string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.visitedURLs[urlStr]
}

// PendingCount 返回当前待处理URL数量
// 返回len(channel),O(1)时间复杂度
func (q *URLQueue) PendingCount() int {
	return len(q.pendingURLs)
}

// Reset 清空队列,重置所有状态
// 为下一个爬取目标准备全新状态
func (q *URLQueue) Reset() {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 清空pending队列 (drain channel)
	for len(q.pendingURLs) > 0 {
		<-q.pendingURLs
	}

	// 清空visited集合
	q.visitedURLs = make(map[string]bool)
}

// Close 关闭队列,释放资源
// 关闭channel,后续Push调用应该返回错误
func (q *URLQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		close(q.pendingURLs)
		q.closed = true
	}
}
