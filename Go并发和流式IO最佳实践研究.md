# Go语言并发处理和流式IO最佳实践研究

本文档针对JsFindcrack项目的三个核心场景，提供完整的Go语言并发处理和流式IO最佳实践方案。

## 目录
- [场景1: 并发文件下载](#场景1-并发文件下载)
- [场景2: 流式文件处理](#场景2-流式文件处理)
- [场景3: 并发相似度分析](#场景3-并发相似度分析)
- [性能优化建议](#性能优化建议)
- [常见陷阱与解决方案](#常见陷阱与解决方案)

---

## 场景1: 并发文件下载

### 概述
需要同时下载多个JavaScript文件，控制并发数，收集所有结果，并跟踪进度。

### 方案A: 使用 errgroup.Group (推荐)

Go 1.21+ 标准库的 `errgroup.Group` 支持通过 `SetLimit()` 限制并发数，这是最简洁的官方方案。

```go
package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// DownloadTask 表示单个下载任务
type DownloadTask struct {
	URL      string
	Filename string
}

// DownloadResult 表示下载结果
type DownloadResult struct {
	Task     DownloadTask
	Success  bool
	Error    error
	FileSize int64
	Duration time.Duration
}

// ProgressTracker 进度跟踪器
type ProgressTracker struct {
	Total     int32
	Completed atomic.Int32
	Failed    atomic.Int32
	mu        sync.Mutex
	results   []DownloadResult
}

func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		Total:   int32(total),
		results: make([]DownloadResult, 0, total),
	}
}

func (pt *ProgressTracker) Update(result DownloadResult) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.results = append(pt.results, result)
	if result.Success {
		pt.Completed.Add(1)
	} else {
		pt.Failed.Add(1)
	}
}

func (pt *ProgressTracker) GetProgress() (completed, failed, total int32) {
	return pt.Completed.Load(), pt.Failed.Load(), pt.Total
}

func (pt *ProgressTracker) GetResults() []DownloadResult {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return append([]DownloadResult(nil), pt.results...)
}

// ConcurrentDownloader 并发下载器
type ConcurrentDownloader struct {
	MaxConcurrent int
	OutputDir     string
	Timeout       time.Duration
	client        *http.Client
}

func NewConcurrentDownloader(maxConcurrent int, outputDir string, timeout time.Duration) *ConcurrentDownloader {
	return &ConcurrentDownloader{
		MaxConcurrent: maxConcurrent,
		OutputDir:     outputDir,
		Timeout:       timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Download 使用 errgroup 并发下载文件
func (cd *ConcurrentDownloader) Download(ctx context.Context, tasks []DownloadTask) (*ProgressTracker, error) {
	tracker := NewProgressTracker(len(tasks))

	// 创建 errgroup，限制并发数
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(cd.MaxConcurrent)

	// 启动进度打印 goroutine
	done := make(chan struct{})
	go cd.printProgress(tracker, done)

	// 提交所有下载任务
	for _, task := range tasks {
		task := task // 捕获循环变量
		g.Go(func() error {
			result := cd.downloadFile(ctx, task)
			tracker.Update(result)

			// 即使单个文件失败，也继续下载其他文件
			// 如果需要"任一失败即停止"，取消注释下行
			// return result.Error
			return nil
		})
	}

	// 等待所有任务完成
	err := g.Wait()
	close(done)

	return tracker, err
}

// downloadFile 下载单个文件
func (cd *ConcurrentDownloader) downloadFile(ctx context.Context, task DownloadTask) DownloadResult {
	start := time.Now()
	result := DownloadResult{
		Task:    task,
		Success: false,
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", task.URL, nil)
	if err != nil {
		result.Error = fmt.Errorf("创建请求失败: %w", err)
		return result
	}

	// 发送请求
	resp, err := cd.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("下载失败: %w", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("HTTP错误: %d", resp.StatusCode)
		return result
	}

	// 创建输出文件
	outputPath := filepath.Join(cd.OutputDir, task.Filename)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		result.Error = fmt.Errorf("创建目录失败: %w", err)
		return result
	}

	file, err := os.Create(outputPath)
	if err != nil {
		result.Error = fmt.Errorf("创建文件失败: %w", err)
		return result
	}
	defer file.Close()

	// 写入文件内容
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("写入文件失败: %w", err)
		return result
	}

	result.Success = true
	result.FileSize = written
	result.Duration = time.Since(start)
	return result
}

// printProgress 打印下载进度
func (cd *ConcurrentDownloader) printProgress(tracker *ProgressTracker, done <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			completed, failed, total := tracker.GetProgress()
			fmt.Printf("\r下载完成: 成功=%d 失败=%d 总计=%d\n", completed, failed, total)
			return
		case <-ticker.C:
			completed, failed, total := tracker.GetProgress()
			fmt.Printf("\r进度: 成功=%d 失败=%d 总计=%d (%.1f%%)",
				completed, failed, total,
				float64(completed+failed)/float64(total)*100)
		}
	}
}
```

### 方案B: 使用 Worker Pool 模式

当需要更细粒度的控制（如工作队列管理、优先级调度等），可以使用传统的 Worker Pool 模式。

```go
package downloader

import (
	"context"
	"fmt"
	"sync"
)

// WorkerPool 工作池实现
type WorkerPool struct {
	MaxWorkers int
	tasks      chan DownloadTask
	results    chan DownloadResult
	wg         sync.WaitGroup
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		MaxWorkers: maxWorkers,
		tasks:      make(chan DownloadTask, maxWorkers*2), // 缓冲队列
		results:    make(chan DownloadResult, maxWorkers*2),
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start(ctx context.Context, downloader *ConcurrentDownloader) {
	for i := 0; i < wp.MaxWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i, downloader)
	}
}

// worker 工作者 goroutine
func (wp *WorkerPool) worker(ctx context.Context, id int, downloader *ConcurrentDownloader) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-wp.tasks:
			if !ok {
				return
			}

			result := downloader.downloadFile(ctx, task)

			select {
			case wp.results <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task DownloadTask) error {
	select {
	case wp.tasks <- task:
		return nil
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// Close 关闭工作池
func (wp *WorkerPool) Close() {
	close(wp.tasks)
	wp.wg.Wait()
	close(wp.results)
}

// Results 获取结果通道
func (wp *WorkerPool) Results() <-chan DownloadResult {
	return wp.results
}
```

### 使用示例

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	// 准备下载任务
	tasks := []DownloadTask{
		{URL: "https://example.com/file1.js", Filename: "file1.js"},
		{URL: "https://example.com/file2.js", Filename: "file2.js"},
		{URL: "https://example.com/file3.js", Filename: "file3.js"},
		// ... 更多任务
	}

	// 创建下载器
	downloader := NewConcurrentDownloader(
		10,                    // 最多10个并发
		"./downloads",         // 输出目录
		30*time.Second,        // 超时时间
	)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 执行下载
	tracker, err := downloader.Download(ctx, tasks)
	if err != nil {
		log.Fatalf("下载过程出错: %v", err)
	}

	// 打印结果摘要
	results := tracker.GetResults()
	for _, result := range results {
		if result.Success {
			fmt.Printf("✓ %s (%.2f MB, %v)\n",
				result.Task.Filename,
				float64(result.FileSize)/(1024*1024),
				result.Duration)
		} else {
			fmt.Printf("✗ %s: %v\n",
				result.Task.Filename,
				result.Error)
		}
	}

	completed, failed, total := tracker.GetProgress()
	fmt.Printf("\n总计: %d 成功: %d 失败: %d\n", total, completed, failed)
}
```

### Context 管理和取消机制

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 示例：优雅取消下载
func DownloadWithGracefulShutdown() {
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在单独的 goroutine 中处理信号
	go func() {
		<-sigChan
		fmt.Println("\n收到中断信号，正在取消下载...")
		cancel()
	}()

	// 执行下载
	downloader := NewConcurrentDownloader(10, "./downloads", 30*time.Second)
	tasks := []DownloadTask{
		// ... 任务列表
	}

	tracker, err := downloader.Download(ctx, tasks)
	if err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Println("下载被用户取消")
		} else {
			fmt.Printf("下载失败: %v\n", err)
		}
		return
	}

	fmt.Printf("下载完成: %d/%d 成功\n",
		tracker.Completed.Load(), tracker.Total)
}

// 示例：多阶段超时控制
func DownloadWithMultiStageTimeout() {
	// 整体超时5分钟
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 为单个文件设置30秒超时
	downloader := NewConcurrentDownloader(10, "./downloads", 30*time.Second)

	tasks := []DownloadTask{
		// ... 任务列表
	}

	tracker, err := downloader.Download(ctx, tasks)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("下载超时")
		} else {
			fmt.Printf("下载失败: %v\n", err)
		}
		return
	}

	fmt.Printf("下载完成: %d/%d 成功\n",
		tracker.Completed.Load(), tracker.Total)
}
```

### 最佳实践总结

1. **并发控制**
   - 使用 `errgroup.SetLimit()` 限制并发数（推荐）
   - 或使用 buffered channel 作为信号量
   - 默认并发数设为 CPU 核心数的 2-4 倍

2. **错误处理**
   - 使用 `errgroup` 自动传播第一个错误
   - 如需继续处理，在 `g.Go()` 中返回 `nil`
   - 收集所有错误到结果结构中

3. **进度跟踪**
   - 使用 `atomic` 包进行线程安全的计数
   - 用独立 goroutine 定期打印进度
   - 避免在热路径中使用 `sync.Mutex`

4. **资源管理**
   - 使用 `defer` 确保资源释放
   - 复用 `http.Client` 而非每次创建
   - 设置合理的超时时间

5. **Context 使用**
   - 始终传递 context 到长期运行的操作
   - 使用 `WithTimeout` 防止无限等待
   - 监听 `ctx.Done()` 支持取消

---

## 场景2: 流式文件处理

### 概述
需要处理最大50MB的JavaScript文件，避免一次性加载到内存，同时计算哈希值用于去重。

### 方案：流式读取 + 哈希计算

```go
package processor

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

// FileProcessor 文件处理器
type FileProcessor struct {
	BufferSize int // 读取缓冲区大小（字节）
}

func NewFileProcessor(bufferSize int) *FileProcessor {
	if bufferSize <= 0 {
		bufferSize = 64 * 1024 // 默认 64KB
	}
	return &FileProcessor{
		BufferSize: bufferSize,
	}
}

// ProcessFileWithHash 流式处理文件并计算哈希
type ProcessResult struct {
	Hash      string
	LineCount int
	ByteCount int64
	Err       error
}

func (fp *FileProcessor) ProcessFileWithHash(filePath string, lineProcessor func(line []byte) error) ProcessResult {
	result := ProcessResult{}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		result.Err = fmt.Errorf("打开文件失败: %w", err)
		return result
	}
	defer file.Close()

	// 创建 SHA256 哈希
	hasher := sha256.New()

	// 创建 TeeReader：同时读取和计算哈希
	teeReader := io.TeeReader(file, hasher)

	// 创建带缓冲的 Scanner
	scanner := bufio.NewScanner(teeReader)

	// 设置缓冲区大小（适用于长行）
	buf := make([]byte, 0, fp.BufferSize)
	scanner.Buffer(buf, fp.BufferSize)

	// 逐行处理
	for scanner.Scan() {
		line := scanner.Bytes() // 使用 Bytes() 避免内存分配
		result.ByteCount += int64(len(line)) + 1 // +1 for newline
		result.LineCount++

		// 处理该行
		if lineProcessor != nil {
			if err := lineProcessor(line); err != nil {
				result.Err = fmt.Errorf("处理第%d行失败: %w", result.LineCount, err)
				return result
			}
		}
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		result.Err = fmt.Errorf("扫描文件失败: %w", err)
		return result
	}

	// 计算最终哈希
	result.Hash = hex.EncodeToString(hasher.Sum(nil))

	return result
}

// ProcessFileInChunks 按块处理文件（适合二进制文件或不需要按行处理的场景）
func (fp *FileProcessor) ProcessFileInChunks(filePath string, chunkProcessor func(chunk []byte) error) ProcessResult {
	result := ProcessResult{}

	file, err := os.Open(filePath)
	if err != nil {
		result.Err = fmt.Errorf("打开文件失败: %w", err)
		return result
	}
	defer file.Close()

	// 创建哈希
	hasher := sha256.New()

	// 创建带缓冲的读取器
	reader := bufio.NewReaderSize(file, fp.BufferSize)

	// 分配缓冲区
	buffer := make([]byte, fp.BufferSize)

	// 循环读取
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			result.ByteCount += int64(n)

			// 更新哈希
			hasher.Write(chunk)

			// 处理块
			if chunkProcessor != nil {
				if err := chunkProcessor(chunk); err != nil {
					result.Err = fmt.Errorf("处理块失败: %w", err)
					return result
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			result.Err = fmt.Errorf("读取文件失败: %w", err)
			return result
		}
	}

	result.Hash = hex.EncodeToString(hasher.Sum(nil))
	return result
}

// MultiHashProcessor 同时计算多种哈希
type MultiHashResult struct {
	SHA256    string
	SHA1      string
	MD5       string
	LineCount int
	ByteCount int64
	Err       error
}

func (fp *FileProcessor) ProcessWithMultiHash(filePath string) MultiHashResult {
	result := MultiHashResult{}

	file, err := os.Open(filePath)
	if err != nil {
		result.Err = fmt.Errorf("打开文件失败: %w", err)
		return result
	}
	defer file.Close()

	// 创建多个哈希
	import (
		"crypto/md5"
		"crypto/sha1"
	)

	sha256Hash := sha256.New()
	sha1Hash := sha1.New()
	md5Hash := md5.New()

	// 使用 MultiWriter 同时写入多个哈希
	multiWriter := io.MultiWriter(sha256Hash, sha1Hash, md5Hash)
	teeReader := io.TeeReader(file, multiWriter)

	// 创建 Scanner
	scanner := bufio.NewScanner(teeReader)
	buf := make([]byte, 0, fp.BufferSize)
	scanner.Buffer(buf, fp.BufferSize)

	// 扫描文件
	for scanner.Scan() {
		result.ByteCount += int64(len(scanner.Bytes())) + 1
		result.LineCount++
	}

	if err := scanner.Err(); err != nil {
		result.Err = err
		return result
	}

	// 获取所有哈希值
	result.SHA256 = hex.EncodeToString(sha256Hash.Sum(nil))
	result.SHA1 = hex.EncodeToString(sha1Hash.Sum(nil))
	result.MD5 = hex.EncodeToString(md5Hash.Sum(nil))

	return result
}
```

### 内存池优化

对于高频处理场景，使用 `sync.Pool` 减少内存分配：

```go
package processor

import (
	"bufio"
	"sync"
)

// PooledFileProcessor 使用对象池的文件处理器
type PooledFileProcessor struct {
	bufferPool *sync.Pool
	scannerPool *sync.Pool
}

func NewPooledFileProcessor(bufferSize int) *PooledFileProcessor {
	return &PooledFileProcessor{
		bufferPool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, bufferSize)
				return &buf
			},
		},
		scannerPool: &sync.Pool{
			New: func() interface{} {
				return bufio.NewScanner(nil)
			},
		},
	}
}

func (pfp *PooledFileProcessor) ProcessFile(filePath string, processor func(line []byte) error) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 从池中获取 buffer
	bufPtr := pfp.bufferPool.Get().(*[]byte)
	defer pfp.bufferPool.Put(bufPtr)

	// 从池中获取 scanner
	scanner := pfp.scannerPool.Get().(*bufio.Scanner)
	defer pfp.scannerPool.Put(scanner)

	// 重置 scanner
	scanner.Reset(file)
	scanner.Buffer(*bufPtr, cap(*bufPtr))

	// 处理文件
	for scanner.Scan() {
		if err := processor(scanner.Bytes()); err != nil {
			return err
		}
	}

	return scanner.Err()
}
```

### 使用示例

```go
package main

import (
	"fmt"
	"log"
	"strings"
)

func main() {
	// 示例1：逐行处理并计算哈希
	processor := NewFileProcessor(64 * 1024) // 64KB 缓冲区

	var functionCount int
	result := processor.ProcessFileWithHash("large-file.js", func(line []byte) error {
		// 统计函数定义
		if strings.Contains(string(line), "function") {
			functionCount++
		}
		return nil
	})

	if result.Err != nil {
		log.Fatalf("处理失败: %v", result.Err)
	}

	fmt.Printf("文件哈希: %s\n", result.Hash)
	fmt.Printf("总行数: %d\n", result.LineCount)
	fmt.Printf("总字节数: %d\n", result.ByteCount)
	fmt.Printf("函数数量: %d\n", functionCount)

	// 示例2：按块处理（适合二进制文件）
	result2 := processor.ProcessFileInChunks("binary-file.dat", func(chunk []byte) error {
		// 对块进行处理
		// 例如：写入另一个文件、压缩等
		return nil
	})

	if result2.Err != nil {
		log.Fatalf("处理失败: %v", result2.Err)
	}

	fmt.Printf("文件哈希: %s\n", result2.Hash)

	// 示例3：计算多种哈希
	multiResult := processor.ProcessWithMultiHash("file.js")
	if multiResult.Err != nil {
		log.Fatalf("处理失败: %v", multiResult.Err)
	}

	fmt.Printf("SHA256: %s\n", multiResult.SHA256)
	fmt.Printf("SHA1: %s\n", multiResult.SHA1)
	fmt.Printf("MD5: %s\n", multiResult.MD5)
}
```

### 去重系统实现

```go
package deduplication

import (
	"fmt"
	"sync"
)

// FileRegistry 文件去重注册表
type FileRegistry struct {
	mu     sync.RWMutex
	hashes map[string][]string // hash -> file paths
}

func NewFileRegistry() *FileRegistry {
	return &FileRegistry{
		hashes: make(map[string][]string),
	}
}

// Register 注册文件哈希
// 返回 true 如果是新文件，false 如果是重复文件
func (fr *FileRegistry) Register(filePath, hash string) bool {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	paths, exists := fr.hashes[hash]
	if exists {
		// 检查是否同一文件路径
		for _, p := range paths {
			if p == filePath {
				return false
			}
		}
		// 不同路径但相同哈希（重复文件）
		fr.hashes[hash] = append(paths, filePath)
		return false
	}

	// 新文件
	fr.hashes[hash] = []string{filePath}
	return true
}

// GetDuplicates 获取所有重复文件
func (fr *FileRegistry) GetDuplicates() map[string][]string {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	duplicates := make(map[string][]string)
	for hash, paths := range fr.hashes {
		if len(paths) > 1 {
			duplicates[hash] = append([]string(nil), paths...)
		}
	}

	return duplicates
}

// GetStats 获取统计信息
func (fr *FileRegistry) GetStats() (totalFiles, uniqueFiles, duplicateGroups int) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()

	for _, paths := range fr.hashes {
		totalFiles += len(paths)
		uniqueFiles++
		if len(paths) > 1 {
			duplicateGroups++
		}
	}

	return
}
```

### 最佳实践总结

1. **缓冲区大小选择**
   - 默认使用 64KB（平衡内存和性能）
   - 对于大文件或快速磁盘，可增至 256KB 或 512KB
   - 对于 SSD，4MB 缓冲区可进一步提升性能

2. **Scanner vs Reader**
   - 文本文件按行处理：使用 `bufio.Scanner`
   - 二进制文件或大块处理：使用 `bufio.Reader`
   - 超长行文件：使用 `scanner.Buffer()` 增大缓冲区

3. **内存优化**
   - 使用 `scanner.Bytes()` 而非 `scanner.Text()`（减少分配）
   - 高频处理使用 `sync.Pool` 复用对象
   - 避免在循环中创建大对象

4. **哈希计算**
   - 使用 `io.TeeReader` 边读边哈希
   - 需要多种哈希时使用 `io.MultiWriter`
   - SHA256 足够用于去重，MD5 更快但安全性较低

5. **错误处理**
   - 始终检查 `scanner.Err()`
   - 使用 `defer file.Close()` 确保文件关闭
   - 包装错误信息提供上下文

---

## 场景3: 并发相似度分析

### 概述
对数千个文件进行两两相似度比较，充分利用多核CPU，避免内存溢出。

### 架构设计

```go
package similarity

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// FileInfo 文件信息
type FileInfo struct {
	Path    string
	Hash    string
	Content []byte // 可选：预加载内容
	Size    int64
}

// SimilarityPair 相似度对
type SimilarityPair struct {
	File1      string
	File2      string
	Similarity float64
}

// SimilarityAnalyzer 相似度分析器
type SimilarityAnalyzer struct {
	MaxConcurrent int
	Threshold     float64 // 相似度阈值（0-1）

	// 用于限制内存使用
	MaxMemoryMB   int
	currentMemory atomic.Int64
}

func NewSimilarityAnalyzer(maxConcurrent, maxMemoryMB int, threshold float64) *SimilarityAnalyzer {
	if maxConcurrent <= 0 {
		maxConcurrent = runtime.NumCPU()
	}

	return &SimilarityAnalyzer{
		MaxConcurrent: maxConcurrent,
		MaxMemoryMB:   maxMemoryMB,
		Threshold:     threshold,
	}
}

// CompareFunc 相似度比较函数类型
type CompareFunc func(file1, file2 *FileInfo) (float64, error)

// AnalyzeBatch 批量分析相似度（方案A：任务分片）
func (sa *SimilarityAnalyzer) AnalyzeBatch(ctx context.Context, files []*FileInfo, compareFunc CompareFunc) ([]SimilarityPair, error) {
	n := len(files)
	totalPairs := n * (n - 1) / 2

	fmt.Printf("开始分析 %d 个文件，共 %d 对比较\n", n, totalPairs)

	// 结果收集
	resultsChan := make(chan SimilarityPair, 100)
	results := make([]SimilarityPair, 0, totalPairs/10) // 预估10%相似

	// 启动结果收集 goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for pair := range resultsChan {
			if pair.Similarity >= sa.Threshold {
				results = append(results, pair)
			}
		}
	}()

	// 创建 errgroup
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(sa.MaxConcurrent)

	// 进度跟踪
	var completed atomic.Int64

	// 生成并发任务
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			i, j := i, j // 捕获变量

			g.Go(func() error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				// 执行相似度计算
				similarity, err := compareFunc(files[i], files[j])
				if err != nil {
					return fmt.Errorf("比较 %s 和 %s 失败: %w",
						files[i].Path, files[j].Path, err)
				}

				// 发送结果
				if similarity >= sa.Threshold {
					select {
					case resultsChan <- SimilarityPair{
						File1:      files[i].Path,
						File2:      files[j].Path,
						Similarity: similarity,
					}:
					case <-ctx.Done():
						return ctx.Err()
					}
				}

				// 更新进度
				done := completed.Add(1)
				if done%1000 == 0 {
					fmt.Printf("\r进度: %d/%d (%.1f%%)",
						done, totalPairs,
						float64(done)/float64(totalPairs)*100)
				}

				return nil
			})
		}
	}

	// 等待所有任务完成
	err := g.Wait()
	close(resultsChan)
	wg.Wait()

	fmt.Printf("\n分析完成，发现 %d 对相似文件\n", len(results))

	return results, err
}

// AnalyzeWithMemoryControl 带内存控制的分析（方案B：分批加载）
func (sa *SimilarityAnalyzer) AnalyzeWithMemoryControl(
	ctx context.Context,
	files []*FileInfo,
	compareFunc CompareFunc,
	loadFunc func(path string) ([]byte, error),
) ([]SimilarityPair, error) {

	n := len(files)
	totalPairs := n * (n - 1) / 2

	// 使用信号量控制内存
	maxMemoryBytes := int64(sa.MaxMemoryMB) * 1024 * 1024
	memorySem := semaphore.NewWeighted(maxMemoryBytes)

	resultsChan := make(chan SimilarityPair, 100)
	results := make([]SimilarityPair, 0)

	// 结果收集
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for pair := range resultsChan {
			results = append(results, pair)
		}
	}()

	// 创建 errgroup
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(sa.MaxConcurrent)

	var completed atomic.Int64

	// 生成任务
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			i, j := i, j

			g.Go(func() error {
				// 获取内存配额
				memNeeded := files[i].Size + files[j].Size
				if err := memorySem.Acquire(ctx, memNeeded); err != nil {
					return err
				}
				defer memorySem.Release(memNeeded)

				// 动态加载文件内容
				content1, err := loadFunc(files[i].Path)
				if err != nil {
					return err
				}

				content2, err := loadFunc(files[j].Path)
				if err != nil {
					return err
				}

				// 创建临时 FileInfo
				tempFile1 := &FileInfo{
					Path:    files[i].Path,
					Hash:    files[i].Hash,
					Content: content1,
					Size:    int64(len(content1)),
				}

				tempFile2 := &FileInfo{
					Path:    files[j].Path,
					Hash:    files[j].Hash,
					Content: content2,
					Size:    int64(len(content2)),
				}

				// 计算相似度
				similarity, err := compareFunc(tempFile1, tempFile2)
				if err != nil {
					return err
				}

				// 发送结果
				if similarity >= sa.Threshold {
					select {
					case resultsChan <- SimilarityPair{
						File1:      files[i].Path,
						File2:      files[j].Path,
						Similarity: similarity,
					}:
					case <-ctx.Done():
						return ctx.Err()
					}
				}

				// 更新进度
				done := completed.Add(1)
				if done%1000 == 0 {
					fmt.Printf("\r进度: %d/%d (%.1f%%)",
						done, totalPairs,
						float64(done)/float64(totalPairs)*100)
				}

				return nil
			})
		}
	}

	err := g.Wait()
	close(resultsChan)
	wg.Wait()

	return results, err
}

// AnalyzeInBatches 分批分析（方案C：文件分组）
type BatchResult struct {
	BatchID int
	Pairs   []SimilarityPair
}

func (sa *SimilarityAnalyzer) AnalyzeInBatches(
	ctx context.Context,
	files []*FileInfo,
	batchSize int,
	compareFunc CompareFunc,
) ([]SimilarityPair, error) {

	n := len(files)
	numBatches := (n + batchSize - 1) / batchSize

	fmt.Printf("分析 %d 个文件，分为 %d 批\n", n, numBatches)

	// 批次结果通道
	batchResults := make(chan BatchResult, numBatches)

	// 创建 errgroup
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(sa.MaxConcurrent)

	// 分批处理
	for batchID := 0; batchID < numBatches; batchID++ {
		batchID := batchID
		start := batchID * batchSize
		end := min(start+batchSize, n)
		batchFiles := files[start:end]

		g.Go(func() error {
			pairs, err := sa.compareBatch(ctx, batchFiles, compareFunc)
			if err != nil {
				return err
			}

			select {
			case batchResults <- BatchResult{
				BatchID: batchID,
				Pairs:   pairs,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}

			fmt.Printf("批次 %d/%d 完成\n", batchID+1, numBatches)
			return nil
		})
	}

	// 等待完成
	err := g.Wait()
	close(batchResults)

	if err != nil {
		return nil, err
	}

	// 合并结果
	var allPairs []SimilarityPair
	for result := range batchResults {
		allPairs = append(allPairs, result.Pairs...)
	}

	return allPairs, nil
}

// compareBatch 比较单个批次内的文件
func (sa *SimilarityAnalyzer) compareBatch(
	ctx context.Context,
	files []*FileInfo,
	compareFunc CompareFunc,
) ([]SimilarityPair, error) {

	var pairs []SimilarityPair
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(sa.MaxConcurrent)

	n := len(files)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			i, j := i, j

			g.Go(func() error {
				similarity, err := compareFunc(files[i], files[j])
				if err != nil {
					return err
				}

				if similarity >= sa.Threshold {
					mu.Lock()
					pairs = append(pairs, SimilarityPair{
						File1:      files[i].Path,
						File2:      files[j].Path,
						Similarity: similarity,
					})
					mu.Unlock()
				}

				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return pairs, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

### 相似度算法实现示例

```go
package similarity

import (
	"math"
	"strings"
	"unicode"
)

// JaccardSimilarity 计算 Jaccard 相似度（基于词集）
func JaccardSimilarity(file1, file2 *FileInfo) (float64, error) {
	// 提取词集
	words1 := extractWords(string(file1.Content))
	words2 := extractWords(string(file2.Content))

	// 计算交集和并集
	intersection := 0
	union := make(map[string]bool)

	for word := range words1 {
		union[word] = true
		if words2[word] {
			intersection++
		}
	}

	for word := range words2 {
		union[word] = true
	}

	if len(union) == 0 {
		return 0, nil
	}

	return float64(intersection) / float64(len(union)), nil
}

// CosineSimilarity 计算余弦相似度（基于词频向量）
func CosineSimilarity(file1, file2 *FileInfo) (float64, error) {
	// 提取词频
	freq1 := wordFrequency(string(file1.Content))
	freq2 := wordFrequency(string(file2.Content))

	// 计算点积和模长
	var dotProduct, mag1, mag2 float64

	for word, count1 := range freq1 {
		if count2, exists := freq2[word]; exists {
			dotProduct += float64(count1 * count2)
		}
		mag1 += float64(count1 * count1)
	}

	for _, count2 := range freq2 {
		mag2 += float64(count2 * count2)
	}

	if mag1 == 0 || mag2 == 0 {
		return 0, nil
	}

	return dotProduct / (math.Sqrt(mag1) * math.Sqrt(mag2)), nil
}

// LevenshteinSimilarity 基于编辑距离的相似度
func LevenshteinSimilarity(file1, file2 *FileInfo) (float64, error) {
	s1 := string(file1.Content)
	s2 := string(file2.Content)

	// 对于大文件，先进行采样或哈希比较
	maxLen := 10000
	if len(s1) > maxLen {
		s1 = s1[:maxLen]
	}
	if len(s2) > maxLen {
		s2 = s2[:maxLen]
	}

	distance := levenshteinDistance(s1, s2)
	maxLen = max(len(s1), len(s2))

	if maxLen == 0 {
		return 1.0, nil
	}

	return 1.0 - float64(distance)/float64(maxLen), nil
}

// 辅助函数
func extractWords(content string) map[string]bool {
	words := make(map[string]bool)

	// 简单的词提取（可根据需要改进）
	for _, word := range strings.FieldsFunc(content, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		word = strings.ToLower(word)
		if len(word) > 2 { // 过滤短词
			words[word] = true
		}
	}

	return words
}

func wordFrequency(content string) map[string]int {
	freq := make(map[string]int)

	for _, word := range strings.FieldsFunc(content, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		word = strings.ToLower(word)
		if len(word) > 2 {
			freq[word]++
		}
	}

	return freq
}

func levenshteinDistance(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)

	// 创建 DP 矩阵（优化：只使用两行）
	prev := make([]int, len2+1)
	curr := make([]int, len2+1)

	for j := 0; j <= len2; j++ {
		prev[j] = j
	}

	for i := 1; i <= len1; i++ {
		curr[0] = i
		for j := 1; j <= len2; j++ {
			if s1[i-1] == s2[j-1] {
				curr[j] = prev[j-1]
			} else {
				curr[j] = min3(prev[j-1], prev[j], curr[j-1]) + 1
			}
		}
		prev, curr = curr, prev
	}

	return prev[len2]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
```

### 使用示例

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
)

func main() {
	// 准备文件列表
	files := []*FileInfo{
		{Path: "file1.js", Size: 1024},
		{Path: "file2.js", Size: 2048},
		// ... 更多文件
	}

	// 创建分析器
	analyzer := NewSimilarityAnalyzer(
		runtime.NumCPU()*2, // 并发数
		512,                // 最大内存 512MB
		0.8,                // 相似度阈值 80%
	)

	// 创建上下文
	ctx := context.Background()

	// 文件加载函数
	loadFunc := func(path string) ([]byte, error) {
		return os.ReadFile(path)
	}

	// 执行分析
	pairs, err := analyzer.AnalyzeWithMemoryControl(
		ctx,
		files,
		CosineSimilarity, // 使用余弦相似度
		loadFunc,
	)

	if err != nil {
		log.Fatalf("分析失败: %v", err)
	}

	// 打印结果
	fmt.Printf("发现 %d 对相似文件:\n", len(pairs))
	for _, pair := range pairs {
		fmt.Printf("%.2f%% - %s <-> %s\n",
			pair.Similarity*100,
			pair.File1,
			pair.File2)
	}
}
```

### 最佳实践总结

1. **并发策略**
   - 使用 `errgroup.SetLimit()` 控制并发数
   - 设置为 CPU 核心数的 2-4 倍
   - 对于 I/O 密集型任务可增加并发数

2. **内存控制**
   - 使用 `semaphore.Weighted` 限制内存使用
   - 动态加载文件而非全部预加载
   - 大文件场景下使用分批处理

3. **任务分配**
   - 小规模（<1000文件）：直接全量并发
   - 中规模（1000-10000）：使用内存信号量
   - 大规模（>10000）：分批处理

4. **算法选择**
   - Jaccard：快速、适合代码去重
   - Cosine：平衡精度和性能
   - Levenshtein：精确但慢，需要采样

5. **性能优化**
   - 预先过滤：按文件大小、哈希等快速筛选
   - 缓存结果：避免重复计算
   - 使用 `atomic` 减少锁竞争

---

## 性能优化建议

### 1. CPU 优化

```go
// 设置 GOMAXPROCS（通常自动设置，除非特殊需求）
runtime.GOMAXPROCS(runtime.NumCPU())

// 性能分析
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
// 访问 http://localhost:6060/debug/pprof/
```

### 2. 内存优化

```go
// 定期触发 GC（谨慎使用）
import "runtime/debug"

debug.SetGCPercent(50) // 降低 GC 阈值

// 内存限制
debug.SetMemoryLimit(1 * 1024 * 1024 * 1024) // 1GB
```

### 3. I/O 优化

```go
// 使用 io.ReaderFrom 和 io.WriterTo 接口
var buf bytes.Buffer
io.Copy(&buf, reader) // 优化的拷贝路径

// 并行 I/O
for _, file := range files {
    go processFile(file)
}
```

### 4. 监控和调试

```go
// 添加指标收集
type Metrics struct {
    FilesProcessed atomic.Int64
    BytesProcessed atomic.Int64
    Errors         atomic.Int64
}

// 定期报告
ticker := time.NewTicker(5 * time.Second)
go func() {
    for range ticker.C {
        log.Printf("处理: %d 文件, %d 字节, %d 错误",
            metrics.FilesProcessed.Load(),
            metrics.BytesProcessed.Load(),
            metrics.Errors.Load())
    }
}()
```

---

## 常见陷阱与解决方案

### 1. Goroutine 泄漏

```go
// 错误：忘记关闭通道
func badExample() {
    ch := make(chan int)
    go func() {
        for i := range ch { // 永远等待
            fmt.Println(i)
        }
    }()
    // 忘记 close(ch)
}

// 正确：始终关闭通道
func goodExample() {
    ch := make(chan int)
    go func() {
        for i := range ch {
            fmt.Println(i)
        }
    }()

    // 发送数据...

    close(ch) // 关闭通道
}
```

### 2. Context 传播

```go
// 错误：使用 context.Background()
func badRequest(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background() // 忽略请求上下文
    doWork(ctx)
}

// 正确：传播请求上下文
func goodRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // 使用请求上下文
    doWork(ctx)
}
```

### 3. 循环变量捕获

```go
// 错误：直接使用循环变量
for _, item := range items {
    g.Go(func() error {
        return process(item) // 竞态条件
    })
}

// 正确：捕获循环变量
for _, item := range items {
    item := item // 捕获
    g.Go(func() error {
        return process(item)
    })
}
```

### 4. 过度并发

```go
// 错误：无限制并发
for _, file := range files {
    go processFile(file) // 可能创建数万 goroutine
}

// 正确：限制并发数
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(100) // 最多 100 个并发

for _, file := range files {
    file := file
    g.Go(func() error {
        return processFile(file)
    })
}
```

### 5. 忽略错误

```go
// 错误：忽略 Scanner 错误
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    process(scanner.Text())
}
// 没有检查 scanner.Err()

// 正确：检查所有错误
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    process(scanner.Text())
}
if err := scanner.Err(); err != nil {
    return fmt.Errorf("扫描失败: %w", err)
}
```

---

## 总结

本研究文档涵盖了三个核心场景的Go语言最佳实践：

1. **并发文件下载**：使用 `errgroup` + 限流 + 进度跟踪
2. **流式文件处理**：使用 `bufio` + `TeeReader` + 哈希计算 + 对象池
3. **并发相似度分析**：使用内存信号量 + 分批处理 + 多种相似度算法

### 关键要点

- 优先使用 Go 标准库（`sync/errgroup`, `bufio`, `context`）
- 通过 `SetLimit()` 或信号量控制并发数
- 使用 `context` 支持取消和超时
- 流式处理大文件，避免一次性加载
- 使用 `atomic` 和 channel 进行线程安全的数据收集
- 合理使用 `sync.Pool` 减少内存分配
- 始终检查错误并提供上下文信息

### 下一步

建议根据项目实际需求，从这些模式中选择合适的方案，并通过基准测试（`go test -bench`）和性能分析（`pprof`）进行优化。
