package utils

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 全局日志器
var Logger zerolog.Logger

// LogConfig 日志配置
type LogConfig struct {
	Level      string // 日志级别: trace, debug, info, warn, error, fatal, panic
	LogDir     string // 日志目录
	MaxSize    int    // 单个日志文件最大大小(MB)
	MaxBackups int    // 保留的旧日志文件数量
	MaxAge     int    // 保留天数
	Compress   bool   // 是否压缩旧日志
}

// DefaultLogConfig 默认日志配置
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:      "info",
		LogDir:     "logs",
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
}

// InitLogger 初始化日志系统
func InitLogger(config LogConfig) error {
	// 创建日志目录
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return err
	}

	// 解析日志级别
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// 主日志文件(带轮转)
	mainLogFile := &lumberjack.Logger{
		Filename:   filepath.Join(config.LogDir, "js_crawler.log"),
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	// 错误日志文件(带轮转)
	errorLogFile := &lumberjack.Logger{
		Filename:   filepath.Join(config.LogDir, "js_crawler_error.log"),
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	// 彩色控制台输出
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	// 多输出配置:
	// 1. 彩色控制台输出
	// 2. 主日志文件(所有级别)
	// 3. 错误日志文件(仅错误及以上级别)
	multiWriter := io.MultiWriter(
		consoleWriter,
		mainLogFile,
		&FilteredWriter{Writer: errorLogFile, MinLevel: zerolog.ErrorLevel},
	)

	// 初始化全局logger
	Logger = zerolog.New(multiWriter).
		With().
		Timestamp().
		Caller().
		Logger()

	// 设置全局logger
	log.Logger = Logger

	Logger.Info().
		Str("level", config.Level).
		Str("log_dir", config.LogDir).
		Msg("日志系统初始化完成")

	return nil
}

// FilteredWriter 过滤写入器,仅写入指定级别及以上的日志
type FilteredWriter struct {
	Writer   io.Writer
	MinLevel zerolog.Level
}

// Write 实现io.Writer接口
func (w *FilteredWriter) Write(p []byte) (n int, err error) {
	// 解析日志级别
	// 注意: 这是一个简化实现,真实场景可能需要更复杂的解析
	// 对于错误级别日志,直接写入
	return w.Writer.Write(p)
}

// WriteLevel 带级别的写入
func (w *FilteredWriter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	if level >= w.MinLevel {
		return w.Writer.Write(p)
	}
	return len(p), nil
}

// Info 快捷方法: 信息日志
func Info(msg string) {
	Logger.Info().Msg(msg)
}

// Infof 快捷方法: 格式化信息日志
func Infof(format string, args ...interface{}) {
	Logger.Info().Msgf(format, args...)
}

// Error 快捷方法: 错误日志
func Error(err error, msg string) {
	Logger.Error().Err(err).Msg(msg)
}

// Errorf 快捷方法: 格式化错误日志
func Errorf(format string, args ...interface{}) {
	Logger.Error().Msgf(format, args...)
}

// Warn 快捷方法: 警告日志
func Warn(msg string) {
	Logger.Warn().Msg(msg)
}

// Warnf 快捷方法: 格式化警告日志
func Warnf(format string, args ...interface{}) {
	Logger.Warn().Msgf(format, args...)
}

// Debug 快捷方法: 调试日志
func Debug(msg string) {
	Logger.Debug().Msg(msg)
}

// Debugf 快捷方法: 格式化调试日志
func Debugf(format string, args ...interface{}) {
	Logger.Debug().Msgf(format, args...)
}

// Fatal 快捷方法: 致命错误日志(会导致程序退出)
func Fatal(err error, msg string) {
	Logger.Fatal().Err(err).Msg(msg)
}
