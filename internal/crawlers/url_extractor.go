package crawlers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-rod/rod"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

// URLExtractor URL提取器
// 职责: 从页面中提取链接,根据配置过滤,并添加到队列
type URLExtractor struct {
	// URL队列引用
	queue *URLQueue

	// 目标主机名(用于跨域检查)
	targetHost string

	// 是否允许跨域
	allowCrossDomain bool

	// 最大深度
	maxDepth int
}

// NewURLExtractor 创建URL提取器实例
func NewURLExtractor(queue *URLQueue, targetHost string, allowCrossDomain bool, maxDepth int) *URLExtractor {
	return &URLExtractor{
		queue:            queue,
		targetHost:       targetHost,
		allowCrossDomain: allowCrossDomain,
		maxDepth:         maxDepth,
	}
}

// ExtractFromPage 从go-rod页面提取链接(动态爬取)
func (e *URLExtractor) ExtractFromPage(page *rod.Page, currentURL string, currentDepth int) (int, error) {
	// 执行JavaScript提取所有链接
	// T021-T023 [US2]: 重写JavaScript代码,添加错误处理和完整的URL提取逻辑
	// 使用page.Evaluate代替page.Eval,支持多语句JavaScript
	result, err := page.Evaluate(&rod.EvalOptions{
		JS: `() => {
			var linkElements = document.querySelectorAll('a[href]');
			var links = [];
			for (var i = 0; i < linkElements.length; i++) {
				var href = linkElements[i].href;
				if (href && (href.indexOf('http://') === 0 || href.indexOf('https://') === 0)) {
					links.push(href);
				}
			}

			var scriptElements = document.querySelectorAll('script[src]');
			var scripts = [];
			for (var j = 0; j < scriptElements.length; j++) {
				var src = scriptElements[j].src;
				if (src && (src.indexOf('http://') === 0 || src.indexOf('https://') === 0)) {
					scripts.push(src);
				}
			}

			var allLinks = links.concat(scripts);
			var uniqueLinks = [];
			var seen = {};
			for (var k = 0; k < allLinks.length; k++) {
				if (!seen[allLinks[k]]) {
					seen[allLinks[k]] = true;
					uniqueLinks.push(allLinks[k]);
				}
			}

			return uniqueLinks;
		}`,
	})
	if err != nil {
		// JavaScript执行失败时记录ERROR日志并返回空结果
		log.Error().Err(err).Str("url", currentURL).Msg("JavaScript执行失败")
		return 0, fmt.Errorf("执行JavaScript提取链接失败: %w", err)
	}

	// 将结果转换为字符串数组
	links := []string{}
	if result.Value.Arr() != nil {
		for _, item := range result.Value.Arr() {
			if item.Str() != "" {
				links = append(links, item.Str())
			}
		}
	}

	// 提取并过滤链接
	extractedCount := 0
	for _, linkStr := range links {
		// 检查链接是否应该被跟随
		shouldFollow, _ := e.ShouldFollowLink(linkStr, currentDepth)
		if !shouldFollow {
			continue
		}

		// 添加到队列
		err := e.queue.Push(linkStr, currentDepth+1)
		if err == nil {
			extractedCount++
		}
	}

	return extractedCount, nil
}

// ExtractFromHTML 从HTML字符串提取链接(静态爬取)
func (e *URLExtractor) ExtractFromHTML(htmlContent string, baseURL string, currentDepth int) ([]string, error) {
	// 解析HTML
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}

	// 解析baseURL
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("解析baseURL失败: %w", err)
	}

	// 提取链接
	var links []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			// 查找href属性
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					// 解析链接
					linkURL, err := url.Parse(attr.Val)
					if err != nil {
						continue
					}

					// 转换为绝对URL
					absoluteURL := base.ResolveReference(linkURL).String()

					// 检查是否应该跟随
					shouldFollow, _ := e.ShouldFollowLink(absoluteURL, currentDepth)
					if shouldFollow {
						links = append(links, absoluteURL)
					}
					break
				}
			}
		}

		// 递归处理子节点
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	return links, nil
}

// ShouldFollowLink 判断链接是否应该被跟随
func (e *URLExtractor) ShouldFollowLink(linkURL string, currentDepth int) (bool, string) {
	// 解析URL
	parsedURL, err := url.Parse(linkURL)
	if err != nil {
		return false, "URL格式无效"
	}

	// 检查协议
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false, "不支持的协议"
	}

	// 检查是否已访问
	if e.queue.IsVisited(linkURL) {
		return false, "URL已访问"
	}

	// 检查深度限制
	if currentDepth+1 > e.maxDepth {
		return false, "深度超过限制"
	}

	// 检查跨域
	if !e.allowCrossDomain && parsedURL.Host != e.targetHost {
		// 添加Debug日志记录跨域过滤
		log.Debug().Msgf("跨域链接已过滤: %s (目标域: %s)", linkURL, e.targetHost)
		return false, "跨域链接已过滤"
	}

	return true, ""
}
