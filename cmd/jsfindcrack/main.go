package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/RecoveryAshes/JsFIndcrack/internal/core"
	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

// 命令行参数
var (
	// 全局参数
	configFile string
	verbose    bool
	logLevel   string

	// HTTP头部参数
	headers        []string // 自定义HTTP请求头
	validateConfig bool     // 验证配置文件

	// 爬取参数
	targetURL           string
	urlFile             string
	depth               int
	waitTime            int
	mode                string
	maxWorkers          int
	playwrightTabs      int
	headless            bool
	resume              bool
	similarityEnabled   bool
	similarityThreshold float64
	outputDir           string

	// 批量处理参数
	batchDelay      int
	continueOnError bool
)

var rootCmd = &cobra.Command{
	Use:   "jsfindcrack",
	Short: "JavaScript文件爬取和反混淆工具",
	Long: `JsFIndcrack - 强大的JavaScript文件爬取和反混淆工具 (Go版本)

这是一个专门用于自动化爬取和分析JavaScript文件的工具,支持:
  • 静态和动态爬取模式
  • 自动检测和反混淆
  • 智能去重和相似度分析
  • 断点续爬功能
  • 批量URL处理
  • 自定义HTTP请求头

HTTP头部配置示例:
  # 通过配置文件 (configs/headers.yaml)
  jsfindcrack -u https://example.com

  # 通过命令行参数
  jsfindcrack -u https://example.com -H "User-Agent: MyBot/1.0" -H "Authorization: Bearer token"

  # 验证配置文件
  jsfindcrack --validate-config

版本: ` + Version + `
构建时间: ` + BuildTime,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 加载配置
		config, err := core.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("加载配置失败: %w", err)
		}

		// 初始化日志系统
		logConfig := utils.LogConfig{
			Level:      config.Logging.Level,
			LogDir:     config.Logging.LogDir,
			MaxSize:    config.Logging.Rotation.MaxSize,
			MaxBackups: config.Logging.Rotation.MaxBackups,
			MaxAge:     config.Logging.Rotation.MaxAge,
			Compress:   config.Logging.Rotation.Compress,
		}

		// 命令行参数覆盖配置文件
		if logLevel != "" {
			logConfig.Level = logLevel
		}

		if err := utils.InitLogger(logConfig); err != nil {
			return fmt.Errorf("初始化日志系统失败: %w", err)
		}

		if verbose {
			utils.Info("详细模式已启用")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// 设置信号处理(Ctrl+C优雅退出)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			sig := <-sigChan
			utils.Warnf("\n收到中断信号: %v, 正在优雅关闭...", sig)
			os.Exit(0)
		}()

		// 重新加载配置(从PersistentPreRunE中获取)
		appConfig, err := core.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("加载配置失败: %w", err)
		}

		// 创建HTTP头部管理器
		headerManager, err := core.NewHeaderManager(configFile, headers)
		if err != nil {
			return fmt.Errorf("创建HTTP头部管理器失败: %w", err)
		}

		// 如果用户请求验证配置
		if validateConfig {
			utils.Info("🔍 验证HTTP头部配置...")
			if err := headerManager.LoadConfig(); err != nil {
				return fmt.Errorf("加载配置失败: %w", err)
			}
			if err := headerManager.Validate(); err != nil {
				return fmt.Errorf("配置验证失败: %w", err)
			}

			// 显示合并后的头部(脱敏)
			safeHeaders := headerManager.GetSafeHeaders()
			utils.Info("✅ 配置验证通过!")
			utils.Infof("当前有效的HTTP头部 (%d个):", len(safeHeaders))
			for name, value := range safeHeaders {
				utils.Infof("  %s: %s", name, value)
			}
			return nil
		}

		// 如果没有提供任何参数,显示帮助信息
		if targetURL == "" && urlFile == "" {
			return cmd.Help()
		}

		// 验证参数
		if err := ValidateFlags(
			targetURL,
			depth,
			waitTime,
			maxWorkers,
			playwrightTabs,
			similarityThreshold,
			mode,
		); err != nil {
			return err
		}

		// 创建爬取配置
		crawlConfig := models.CrawlConfig{
			Depth:               depth,
			WaitTime:            waitTime,
			MaxWorkers:          maxWorkers,
			PlaywrightTabs:      playwrightTabs,
			Headless:            headless,
			Resume:              resume,
			SimilarityEnabled:   similarityEnabled,
			SimilarityThreshold: similarityThreshold,
			AllowCrossDomain:    appConfig.Crawl.AllowCrossDomain, // 从配置文件加载
			// 资源配置
			SafetyReserveMemory: appConfig.Resource.SafetyReserveMemory,
			SafetyThreshold:     appConfig.Resource.SafetyThreshold,
			CPULoadThreshold:    appConfig.Resource.CPULoadThreshold,
			MaxTabsLimit:        appConfig.Resource.MaxTabsLimit,
		}

		// 检查是否为批量处理模式
		if urlFile != "" {
			// 批量处理模式
			urls, err := utils.ReadURLsFromFile(urlFile)
			if err != nil {
				return fmt.Errorf("读取URL文件失败: %w", err)
			}

			// 创建批量爬取器
			batchCrawler := core.NewBatchCrawler(crawlConfig, outputDir, mode, batchDelay, continueOnError, headerManager)

			// 执行批量爬取
			if _, err := batchCrawler.CrawlBatch(urls); err != nil {
				return fmt.Errorf("批量爬取失败: %w", err)
			}

			utils.Info("✨ 批量爬取任务完成!")
			return nil
		}

		// 单URL爬取模式
		crawler, err := core.NewCrawler(targetURL, crawlConfig, outputDir, mode, headerManager)
		if err != nil {
			return fmt.Errorf("创建爬取器失败: %w", err)
		}

		// 执行爬取
		if err := crawler.Crawl(); err != nil {
			return fmt.Errorf("爬取失败: %w", err)
		}

		// 显示统计结果
		stats := crawler.GetStats()
		fmt.Println("\n==================================================")
		fmt.Println("📊 爬取统计")
		fmt.Println("==================================================")
		fmt.Printf("✅ 访问URL数: %d\n", stats.VisitedURLs)
		fmt.Printf("✅ 静态爬取文件: %d\n", stats.StaticFiles)
		fmt.Printf("✅ 动态爬取文件: %d\n", stats.DynamicFiles)
		fmt.Printf("✅ 总文件数(去重): %d\n", stats.TotalFiles)
		fmt.Printf("✅ Source Map文件: %d\n", stats.MapFiles)
		fmt.Printf("❌ 失败文件: %d\n", stats.FailedFiles)
		fmt.Printf("📦 总大小: %.2f MB\n", float64(stats.TotalSize)/(1024*1024))
		fmt.Printf("⏱️  总耗时: %.2f秒\n", stats.Duration)
		fmt.Println("==================================================")

		utils.Info("✨ 爬取任务完成!")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("JsFIndcrack %s\n", Version)
		fmt.Printf("构建时间: %s\n", BuildTime)
		fmt.Println("Go实现版本 - 高性能JavaScript爬取工具")
	},
}

func init() {
	// 全局参数
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出模式")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "日志级别 (trace|debug|info|warn|error)")

	// HTTP头部参数
	rootCmd.PersistentFlags().StringSliceVarP(&headers, "header", "H", []string{}, "自定义HTTP头部,格式: 'Name: Value',可多次指定")
	rootCmd.PersistentFlags().BoolVar(&validateConfig, "validate-config", false, "验证配置文件正确性")

	// 爬取参数
	rootCmd.Flags().StringVarP(&targetURL, "url", "u", "", "目标URL (必需,除非使用 --url-file)")
	rootCmd.Flags().StringVarP(&urlFile, "url-file", "f", "", "包含URL列表的文件路径")
	rootCmd.Flags().IntVarP(&depth, "depth", "d", 2, "爬取深度 (1-10)")
	rootCmd.Flags().IntVarP(&waitTime, "wait", "w", 3, "页面等待时间(秒)")
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "all", "爬取模式 (all|static|dynamic)")
	rootCmd.Flags().IntVar(&maxWorkers, "threads", 2, "静态爬取并发线程数")
	rootCmd.Flags().IntVar(&playwrightTabs, "tabs", 4, "Playwright标签页数量")
	rootCmd.Flags().BoolVar(&headless, "headless", true, "无头浏览器模式")
	rootCmd.Flags().BoolVar(&resume, "resume", false, "从检查点恢复")
	rootCmd.Flags().BoolVar(&similarityEnabled, "similarity", true, "启用相似度分析")
	rootCmd.Flags().Float64Var(&similarityThreshold, "similarity-threshold", 0.8, "相似度阈值 (0.0-1.0)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "output", "输出目录")

	// 批量处理参数
	rootCmd.Flags().IntVar(&batchDelay, "batch-delay", 1, "批量处理URL间延迟(秒)")
	rootCmd.Flags().BoolVar(&continueOnError, "continue-on-error", true, "遇到错误继续处理")

	// 添加子命令
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
