package core

import (
	"net/http"

	"github.com/RecoveryAshes/JsFIndcrack/internal/config"
	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
)

const (
	// DefaultUserAgent 默认User-Agent
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/120.0.0.0 Safari/537.36"
)

// HeaderManager 管理HTTP请求头部的生命周期
// 实现 HeaderProvider 接口
type HeaderManager struct {
	// configFile 配置文件路径
	configFile string

	// defaults 系统默认头部 (硬编码)
	defaults http.Header

	// config 从配置文件加载的头部
	config http.Header

	// cli 从命令行参数解析的头部
	cli http.Header

	// validator 头部验证器
	validator *utils.HeaderValidator

	// redactor 头部脱敏器
	redactor *utils.HeaderRedactor

	// configLoader 配置文件加载器
	configLoader *config.HeaderConfigLoader

	// loaded 标记配置是否已加载
	loaded bool
}

// NewHeaderManager 创建头部管理器
// 参数:
//   - configFile: 配置文件路径 (如为空则使用默认路径)
//   - cliHeaders: 命令行传递的头部字符串列表
//
// 返回:
//   - *HeaderManager: 头部管理器实例
//   - error: 如果命令行参数解析失败
func NewHeaderManager(configFile string, cliHeaders []string) (*HeaderManager, error) {
	hm := &HeaderManager{
		configFile:   configFile,
		defaults:     getDefaultHeaders(),
		validator:    utils.NewHeaderValidator(),
		redactor:     utils.NewHeaderRedactor(),
		configLoader: config.NewHeaderConfigLoader(configFile),
		loaded:       false,
	}

	// 解析命令行头部
	if len(cliHeaders) > 0 {
		cliHeadersParsed, err := models.CliHeaders(cliHeaders).Parse()
		if err != nil {
			return nil, err
		}
		hm.cli = cliHeadersParsed
	} else {
		hm.cli = make(http.Header)
	}

	return hm, nil
}

// getDefaultHeaders 返回系统默认头部
func getDefaultHeaders() http.Header {
	return http.Header{
		"User-Agent":      []string{DefaultUserAgent},
		"Accept":          []string{"*/*"},
		"Accept-Encoding": []string{"gzip, deflate, br"},
	}
}

// LoadConfig 加载配置文件
// 如果已加载则跳过
func (hm *HeaderManager) LoadConfig() error {
	if hm.loaded {
		return nil
	}

	// 加载配置文件
	headerConfig, err := hm.configLoader.LoadConfig()
	if err != nil {
		utils.Errorf("加载HTTP头部配置失败: %v", err)
		return err
	}

	// 将map[string]string转换为http.Header
	hm.config = make(http.Header)
	for name, value := range headerConfig.Headers {
		hm.config.Set(name, value)
	}

	hm.loaded = true

	// 记录加载成功 (脱敏后的头部)
	if len(headerConfig.Headers) > 0 {
		safeHeaders := hm.redactor.Redact(hm.config)
		utils.Debugf("成功加载%d个HTTP头部配置: %v", len(safeHeaders), safeHeaders)
	}

	return nil
}

// Validate 验证所有头部的合法性
// 验证顺序: 默认 → 配置 → 命令行
func (hm *HeaderManager) Validate() error {
	// 验证默认头部 (理论上应该总是合法的)
	if err := hm.validator.Validate(hm.defaults); err != nil {
		utils.Errorf("默认头部验证失败: %v", err)
		return err
	}

	// 验证配置文件头部
	if err := hm.validator.Validate(hm.config); err != nil {
		utils.Errorf("配置文件头部验证失败: %v", err)
		return err
	}

	// 验证命令行头部
	if err := hm.validator.Validate(hm.cli); err != nil {
		utils.Errorf("命令行头部验证失败: %v", err)
		return err
	}

	utils.Debugf("所有HTTP头部验证通过")
	return nil
}

// GetMergedHeaders 按优先级合并头部 (default < config < cli)
// 返回: 合并后的http.Header
func (hm *HeaderManager) GetMergedHeaders() http.Header {
	result := make(http.Header)

	// 1. 首先应用默认头部
	for name, values := range hm.defaults {
		result[name] = values
	}

	// 2. 配置文件覆盖默认
	for name, values := range hm.config {
		result[name] = values
	}

	// 3. 命令行覆盖配置文件
	for name, values := range hm.cli {
		result[name] = values
	}

	return result
}

// GetSafeHeaders 返回脱敏后的头部 (用于日志)
// 返回: map[string]string 脱敏后的头部
func (hm *HeaderManager) GetSafeHeaders() map[string]string {
	merged := hm.GetMergedHeaders()
	return hm.redactor.Redact(merged)
}

// GetHeaders 实现 HeaderProvider 接口
// 返回当前有效的HTTP请求头部
func (hm *HeaderManager) GetHeaders() (http.Header, error) {
	// 1. 确保配置已加载
	if err := hm.LoadConfig(); err != nil {
		return nil, err
	}

	// 2. 验证所有头部
	if err := hm.Validate(); err != nil {
		return nil, err
	}

	// 3. 合并并返回
	return hm.GetMergedHeaders(), nil
}
