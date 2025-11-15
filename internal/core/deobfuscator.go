package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/RecoveryAshes/JsFIndcrack/internal/models"
	"github.com/RecoveryAshes/JsFIndcrack/internal/utils"
)

// Deobfuscator 反混淆器
type Deobfuscator struct {
	webcrackAvailable bool
	timeout           time.Duration
}

// NewDeobfuscator 创建反混淆器
func NewDeobfuscator() *Deobfuscator {
	d := &Deobfuscator{
		timeout: 30 * time.Second,
	}

	// 检测webcrack是否可用
	d.webcrackAvailable = d.checkWebcrackAvailable()

	if d.webcrackAvailable {
		utils.Info("✅ webcrack已检测到,将使用高级反混淆功能")
	} else {
		utils.Warn("⚠️  未检测到webcrack,将使用基础清理功能")
		utils.Info("💡 提示: 安装webcrack获得更好效果: npm install -g webcrack")
	}

	return d
}

// checkWebcrackAvailable 检查webcrack是否可用
func (d *Deobfuscator) checkWebcrackAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "webcrack", "--version")
	if err := cmd.Run(); err != nil {
		utils.Debugf("webcrack检测失败: %v", err)
		return false
	}

	return true
}

// Deobfuscate 反混淆JavaScript文件
func (d *Deobfuscator) Deobfuscate(jsFile *models.JSFile, outputDir string) error {
	// 读取混淆代码
	obfuscatedCode, err := os.ReadFile(jsFile.FilePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 检测是否混淆
	if !d.isObfuscated(string(obfuscatedCode)) {
		utils.Debugf("文件未混淆,跳过: %s", jsFile.URL)
		return nil
	}

	jsFile.IsObfuscated = true
	utils.Infof("🔍 检测到混淆文件: %s", filepath.Base(jsFile.FilePath))

	var deobfuscatedCode string

	// 尝试使用webcrack
	if d.webcrackAvailable {
		deobfuscatedCode, err = d.deobfuscateWithWebcrack(string(obfuscatedCode))
		if err != nil {
			utils.Warnf("webcrack反混淆失败,降级到简单清理: %v", err)
			deobfuscatedCode = d.simpleCleanup(string(obfuscatedCode))
		}
	} else {
		// 使用简单清理
		deobfuscatedCode = d.simpleCleanup(string(obfuscatedCode))
	}

	// 保存反混淆后的代码
	decodePath := d.generateDecodePath(jsFile, outputDir)
	if err := os.MkdirAll(filepath.Dir(decodePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.WriteFile(decodePath, []byte(deobfuscatedCode), 0644); err != nil {
		return fmt.Errorf("写入反混淆文件失败: %w", err)
	}

	utils.Infof("✨ 反混淆完成: %s", filepath.Base(decodePath))
	return nil
}

// deobfuscateWithWebcrack 使用webcrack反混淆
func (d *Deobfuscator) deobfuscateWithWebcrack(code string) (string, error) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "webcrack-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建临时输入文件
	inputFile := filepath.Join(tmpDir, "input.js")
	if err := os.WriteFile(inputFile, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 创建上下文和超时
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	// 调用webcrack,输出到临时目录
	outputDir := filepath.Join(tmpDir, "output")
	cmd := exec.CommandContext(ctx, "webcrack", inputFile, "-o", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("webcrack执行失败: %w, output: %s", err, string(output))
	}

	// 读取反混淆后的文件
	deobfuscatedFile := filepath.Join(outputDir, "deobfuscated.js")
	deobfuscatedCode, err := os.ReadFile(deobfuscatedFile)
	if err != nil {
		return "", fmt.Errorf("读取反混淆文件失败: %w", err)
	}

	return string(deobfuscatedCode), nil
}

// simpleCleanup 简单清理(Go实现的降级方案)
func (d *Deobfuscator) simpleCleanup(code string) string {
	utils.Debugf("使用简单清理模式")

	// 1. 十六进制数字转十进制
	code = d.convertHexNumbers(code)

	// 2. 字符串解码
	code = d.decodeStrings(code)

	// 3. 移除多余空行
	code = d.removeExtraNewlines(code)

	// 4. 基础格式化
	code = d.basicFormat(code)

	return code
}

// convertHexNumbers 将十六进制数字转为十进制
func (d *Deobfuscator) convertHexNumbers(code string) string {
	hexPattern := regexp.MustCompile(`0x([0-9a-fA-F]+)`)

	return hexPattern.ReplaceAllStringFunc(code, func(match string) string {
		hexStr := strings.TrimPrefix(match, "0x")
		if num, err := strconv.ParseInt(hexStr, 16, 64); err == nil {
			return strconv.FormatInt(num, 10)
		}
		return match
	})
}

// decodeStrings 解码转义字符串
func (d *Deobfuscator) decodeStrings(code string) string {
	// 解码 \x 十六进制编码
	hexEscapePattern := regexp.MustCompile(`\\x([0-9a-fA-F]{2})`)

	code = hexEscapePattern.ReplaceAllStringFunc(code, func(match string) string {
		hexStr := strings.TrimPrefix(match, `\x`)
		if num, err := strconv.ParseInt(hexStr, 16, 32); err == nil {
			return string(rune(num))
		}
		return match
	})

	// 解码 \u Unicode编码
	unicodePattern := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

	code = unicodePattern.ReplaceAllStringFunc(code, func(match string) string {
		hexStr := strings.TrimPrefix(match, `\u`)
		if num, err := strconv.ParseInt(hexStr, 16, 32); err == nil {
			return string(rune(num))
		}
		return match
	})

	return code
}

// removeExtraNewlines 移除多余空行
func (d *Deobfuscator) removeExtraNewlines(code string) string {
	// 将多个连续空行替换为单个空行
	multipleNewlines := regexp.MustCompile(`\n{3,}`)
	return multipleNewlines.ReplaceAllString(code, "\n\n")
}

// basicFormat 基础格式化
func (d *Deobfuscator) basicFormat(code string) string {
	// 在运算符周围添加空格
	operators := map[string]string{
		"=": " = ",
		"+": " + ",
		"-": " - ",
		"*": " * ",
		"/": " / ",
		">": " > ",
		"<": " < ",
	}

	for op, formatted := range operators {
		// 避免重复添加空格
		pattern := regexp.MustCompile(fmt.Sprintf(`\s*%s\s*`, regexp.QuoteMeta(op)))
		code = pattern.ReplaceAllString(code, formatted)
	}

	return code
}

// isObfuscated 检测代码是否被混淆
func (d *Deobfuscator) isObfuscated(code string) bool {
	// 多个启发式规则检测混淆

	// 1. 检查是否有大量单字符变量名
	singleCharVars := regexp.MustCompile(`\b[a-zA-Z]\b`)
	singleCharCount := len(singleCharVars.FindAllString(code, -1))
	if float64(singleCharCount)/float64(len(code)) > 0.01 {
		return true
	}

	// 2. 检查是否有十六进制数字编码
	hexPattern := regexp.MustCompile(`0x[0-9a-fA-F]+`)
	if len(hexPattern.FindAllString(code, -1)) > 10 {
		return true
	}

	// 3. 检查是否有字符串转义编码
	escapePattern := regexp.MustCompile(`\\x[0-9a-fA-F]{2}|\\u[0-9a-fA-F]{4}`)
	if len(escapePattern.FindAllString(code, -1)) > 5 {
		return true
	}

	// 4. 检查eval或Function构造
	evalPattern := regexp.MustCompile(`\beval\s*\(|Function\s*\(`)
	if evalPattern.MatchString(code) {
		return true
	}

	// 5. 检查常见混淆器特征
	obfuscatorPatterns := []string{
		`_0x[0-9a-f]+`,           // 常见混淆器变量名
		`\['push'\]`,             // 数组方法字符串化
		`\['length'\]`,           // 属性访问字符串化
		`String\['fromCharCode`, // 字符串构造
	}

	for _, pattern := range obfuscatorPatterns {
		if matched, _ := regexp.MatchString(pattern, code); matched {
			return true
		}
	}

	return false
}

// generateDecodePath 生成反混淆文件路径
func (d *Deobfuscator) generateDecodePath(jsFile *models.JSFile, outputDir string) string {
	// 从encode/js路径转换到decode/js路径
	// 例如: output/domain/encode/js/file.js -> output/domain/decode/js/file.js

	filename := filepath.Base(jsFile.FilePath)
	domain := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(jsFile.FilePath))))

	return filepath.Join(outputDir, domain, "decode", "js", filename)
}

// DeobfuscateAll 批量反混淆所有文件
func (d *Deobfuscator) DeobfuscateAll(jsFiles []*models.JSFile, outputDir string) (int, int, error) {
	successCount := 0
	failCount := 0

	utils.Infof("🔧 开始批量反混淆: %d个文件", len(jsFiles))

	for _, jsFile := range jsFiles {
		if err := d.Deobfuscate(jsFile, outputDir); err != nil {
			utils.Errorf("反混淆失败 [%s]: %v", jsFile.URL, err)
			failCount++
			continue
		}

		if jsFile.IsObfuscated {
			successCount++
		}
	}

	utils.Infof("✅ 反混淆完成: 成功 %d, 失败 %d", successCount, failCount)
	return successCount, failCount, nil
}
