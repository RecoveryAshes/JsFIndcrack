package crawlers

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// ResourceMonitor 系统资源监控器
// 职责: 实时监控内存和CPU,计算标签页上限,实施渐进式降级策略
type ResourceMonitor struct {
	// 配置参数
	config ResourceMonitorConfig

	// 缓存的内存统计数据
	lastMemStats runtime.MemStats

	// 系统总内存(字节)
	totalMemory uint64

	// T038 [EC2]: 缓存的CalculateMaxTabs结果
	cachedMaxTabs int
	lastCacheTime time.Time
	cacheMu       sync.RWMutex // 保护缓存的读写锁

	// CPU使用率监控
	lastCPUTime     time.Time
	lastCPUUsage    float64
	cpuUsageMu      sync.RWMutex // 保护CPU使用率的读写锁

	// 保护lastMemStats的读写锁
	mu sync.RWMutex

	// 监控控制
	cancelFunc context.CancelFunc
	isRunning  bool
}

// ResourceMonitorConfig 资源监控器配置
type ResourceMonitorConfig struct {
	SafetyReserveMemory int64 // 安全保留内存(字节)
	SafetyThreshold     int64 // 安全阈值(字节)
	CPULoadThreshold    int   // CPU负载阈值(%)
	MaxTabsLimit        int   // 绝对最大标签页数
	TabMemoryUsage      int64 // 单个标签页平均内存消耗(字节)
}

// MemoryStatus 内存状态信息
type MemoryStatus struct {
	TotalMemory     uint64 // 系统总内存(字节)
	AllocatedMemory uint64 // 当前程序已分配内存(字节)
	AvailableMemory int64  // 可用内存(字节)
	SafetyReserve   int64  // 安全保留内存(字节)
	SafetyThreshold int64  // 安全阈值(字节)
	MemoryPressure  string // 内存压力等级
}

// NewResourceMonitor 创建资源监控器实例
func NewResourceMonitor(config ResourceMonitorConfig) *ResourceMonitor {
	// 初始化默认值
	if config.TabMemoryUsage == 0 {
		config.TabMemoryUsage = 100 * 1024 * 1024 // 100MB
	}

	// 获取系统总内存(使用gopsutil获取真实系统内存)
	vmStat, err := mem.VirtualMemory()
	var totalMem uint64
	if err != nil {
		log.Warn().Err(err).Msg("获取系统内存失败,使用默认值")
		totalMem = 4 * 1024 * 1024 * 1024 // 默认4GB
		log.Info().Msgf("系统总内存: 4.00 GB (默认值)")
	} else {
		totalMem = vmStat.Total
		log.Info().Msgf("系统总内存: %.2f GB", float64(totalMem)/(1024*1024*1024))
	}

	// 读取初始内存统计
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &ResourceMonitor{
		config:        config,
		totalMemory:   totalMem,
		lastMemStats:  memStats,
		isRunning:     false,
		lastCPUTime:   time.Now(),
		lastCPUUsage:  0.0,
	}
}

// StartMonitoring 启动资源监控
// 启动后台goroutine周期性采样runtime.MemStats
func (rm *ResourceMonitor) StartMonitoring(interval time.Duration) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 如果已经在运行,直接返回(幂等)
	if rm.isRunning {
		return
	}

	// 创建可取消的context
	ctx, cancel := context.WithCancel(context.Background())
	rm.cancelFunc = cancel
	rm.isRunning = true

	// 启动后台采样goroutine
	go rm.monitoringLoop(ctx, interval)
}

// monitoringLoop 后台监控循环
func (rm *ResourceMonitor) monitoringLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 采样内存统计
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			// 更新内存缓存
			rm.mu.Lock()
			rm.lastMemStats = memStats
			rm.mu.Unlock()

			// 更新CPU使用率
			cpuUsage := rm.getCPUUsage()
			rm.cpuUsageMu.Lock()
			rm.lastCPUUsage = cpuUsage
			rm.lastCPUTime = time.Now()
			rm.cpuUsageMu.Unlock()
		}
	}
}

// getCPUUsage 获取当前进程的CPU使用率(百分比)
// 使用gopsutil/v3/cpu获取真实的系统CPU使用率
func (rm *ResourceMonitor) getCPUUsage() float64 {
	// 获取CPU使用率 (100毫秒采样间隔,避免阻塞过久)
	// perCPU=false 返回所有CPU的平均使用率
	percentages, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		log.Warn().Err(err).Msg("获取CPU使用率失败")
		return 0.0
	}

	// percentages[0] 是所有CPU核心的平均使用率
	if len(percentages) == 0 {
		log.Warn().Msg("CPU使用率数据为空")
		return 0.0
	}

	return percentages[0]
}

// StopMonitoring 停止资源监控
func (rm *ResourceMonitor) StopMonitoring() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.isRunning && rm.cancelFunc != nil {
		rm.cancelFunc()
		rm.isRunning = false
		rm.cancelFunc = nil
	}
}

// CalculateMaxTabs 动态计算当前允许的最大标签页数
// T038 [EC2]: 添加缓存机制(每秒更新一次),提高性能
// 返回基于可用内存和CPU负载计算的上限
func (rm *ResourceMonitor) CalculateMaxTabs() int {
	// 检查缓存是否有效(1秒内)
	rm.cacheMu.RLock()
	if time.Since(rm.lastCacheTime) < time.Second && rm.cachedMaxTabs > 0 {
		cached := rm.cachedMaxTabs
		rm.cacheMu.RUnlock()
		return cached
	}
	rm.cacheMu.RUnlock()

	// 缓存失效,重新计算
	rm.mu.RLock()
	memStats := rm.lastMemStats
	rm.mu.RUnlock()

	// 计算可用内存
	allocatedMemory := memStats.Alloc
	availableMemory := int64(rm.totalMemory) - int64(allocatedMemory) - rm.config.SafetyReserveMemory

	// 基于内存计算上限
	maxTabsByMemory := 1 // 默认至少1个
	if availableMemory > rm.config.SafetyThreshold {
		surplus := availableMemory - rm.config.SafetyThreshold
		maxTabsByMemory = int(surplus / rm.config.TabMemoryUsage)
		if maxTabsByMemory < 1 {
			maxTabsByMemory = 1
		}
	}

	// 基于CPU计算上限
	maxTabsByCPU := runtime.NumCPU()

	// 取最小值
	result := maxTabsByMemory
	if maxTabsByCPU < result {
		result = maxTabsByCPU
	}
	if rm.config.MaxTabsLimit < result {
		result = rm.config.MaxTabsLimit
	}

	// 确保至少1个标签页
	if result < 1 {
		result = 1
	}

	// 更新缓存
	rm.cacheMu.Lock()
	rm.cachedMaxTabs = result
	rm.lastCacheTime = time.Now()
	rm.cacheMu.Unlock()

	return result
}

// CheckResourceAvailability 检查当前资源是否允许创建新标签页
// 返回canCreate(是否允许创建)和reason(不允许时的原因)
func (rm *ResourceMonitor) CheckResourceAvailability() (canCreate bool, reason string) {
	rm.mu.RLock()
	memStats := rm.lastMemStats
	rm.mu.RUnlock()

	// 计算可用内存
	allocatedMemory := memStats.Alloc
	availableMemory := int64(rm.totalMemory) - int64(allocatedMemory) - rm.config.SafetyReserveMemory

	// 检查内存
	if availableMemory < rm.config.SafetyThreshold {
		availableMemoryMB := availableMemory / (1024 * 1024)
		reasonStr := fmt.Sprintf("内存不足(当前%dMB)", availableMemoryMB)

		// 添加警告日志
		log.Warn().Msgf("可用内存不足(当前%dMB),标签页创建受限", availableMemoryMB)

		return false, reasonStr
	}

	// 检查CPU负载
	// 如果配置的阈值 >= 200, 则跳过CPU检查(视为禁用)
	if rm.config.CPULoadThreshold < 200 {
		// 获取缓存的CPU使用率
		rm.cpuUsageMu.RLock()
		cpuUsage := rm.lastCPUUsage
		rm.cpuUsageMu.RUnlock()

		// 检查CPU使用率是否超过阈值
		if cpuUsage > float64(rm.config.CPULoadThreshold) {
			return false, fmt.Sprintf("CPU负载过高(当前%.1f%%)", cpuUsage)
		}
	}

	return true, ""
}

// GetMemoryStatus 获取当前内存状态
func (rm *ResourceMonitor) GetMemoryStatus() MemoryStatus {
	rm.mu.RLock()
	memStats := rm.lastMemStats
	rm.mu.RUnlock()

	allocatedMemory := memStats.Alloc
	availableMemory := int64(rm.totalMemory) - int64(allocatedMemory) - rm.config.SafetyReserveMemory

	// 判断内存压力等级
	var pressure string
	availableMemoryMB := availableMemory / (1024 * 1024)
	switch {
	case availableMemoryMB < 200:
		pressure = "emergency"
	case availableMemoryMB < 300:
		pressure = "critical"
	case availableMemoryMB < 500:
		pressure = "warning"
	default:
		pressure = "normal"
	}

	return MemoryStatus{
		TotalMemory:     rm.totalMemory,
		AllocatedMemory: allocatedMemory,
		AvailableMemory: availableMemory,
		SafetyReserve:   rm.config.SafetyReserveMemory,
		SafetyThreshold: rm.config.SafetyThreshold,
		MemoryPressure:  pressure,
	}
}

// ShouldScaleDown 判断是否应该主动缩减标签页数量
// 返回shouldScale(是否应该缩减), targetCount(建议缩减到的数量), reason(原因)
func (rm *ResourceMonitor) ShouldScaleDown(currentTabs int) (shouldScale bool, targetCount int, reason string) {
	rm.mu.RLock()
	memStats := rm.lastMemStats
	rm.mu.RUnlock()

	// 计算可用内存
	allocatedMemory := memStats.Alloc
	availableMemory := int64(rm.totalMemory) - int64(allocatedMemory) - rm.config.SafetyReserveMemory
	availableMemoryMB := availableMemory / (1024 * 1024)

	// 渐进式降级策略
	switch {
	case availableMemoryMB < 200:
		// 紧急状态:缩减到1个标签页
		reasonStr := fmt.Sprintf("内存严重不足(当前%dMB),强制缩减至1个标签页", availableMemoryMB)

		// 添加错误日志
		log.Error().Msgf("内存紧急状态(当前%dMB),强制缩减标签页至1个", availableMemoryMB)

		return true, 1, reasonStr
	case availableMemoryMB < 300:
		// 严重不足:缩减50%
		targetCount = currentTabs / 2
		if targetCount < 1 {
			targetCount = 1
		}
		reasonStr := fmt.Sprintf("内存严重不足(当前%dMB),缩减标签页至%d个", availableMemoryMB, targetCount)

		// 添加警告日志
		log.Warn().Msgf("内存严重不足(当前%dMB),强制缩减标签页至%d个", availableMemoryMB, targetCount)

		return true, targetCount, reasonStr
	case availableMemoryMB < 500:
		// 警告:暂停创建但不缩减
		reasonStr := fmt.Sprintf("内存不足(当前%dMB),暂停创建新标签页", availableMemoryMB)

		// 添加警告日志
		log.Warn().Msgf("内存不足(当前%dMB),暂停创建新标签页", availableMemoryMB)

		return false, currentTabs, reasonStr
	default:
		// 正常
		return false, currentTabs, ""
	}
}
