package models

// URLItem 表示队列中的一个URL项
// 用途:
//   - 在channel中传递URL和深度信息
//   - 支持深度优先或广度优先策略(通过不同channel实现)
type URLItem struct {
	// URL 完整的URL字符串
	URL string

	// Depth URL的深度层级
	//   - 0: 入口URL
	//   - 1: 从入口页面发现的链接
	//   - 2: 从深度1页面发现的链接
	//   - 以此类推...
	Depth int

	// SourceURL 发现此URL的源页面(可选,用于调试)
	SourceURL string
}
