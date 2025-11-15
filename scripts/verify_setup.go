package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	fmt.Println("==============================================")
	fmt.Println("  JsFIndcrack Go版本环境验证")
	fmt.Println("==============================================")
	fmt.Println()

	allOK := true

	// 检查Go版本
	goVersion := runtime.Version()
	fmt.Printf("✅ Go版本: %s\n", goVersion)

	// 检查Go版本是否满足要求
	if !strings.HasPrefix(goVersion, "go1.21") &&
	   !strings.HasPrefix(goVersion, "go1.22") &&
	   !strings.HasPrefix(goVersion, "go1.23") {
		fmt.Println("⚠️  警告: 建议使用Go 1.21+版本")
	}

	// 检查操作系统
	fmt.Printf("✅ 操作系统: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// 检查webcrack
	if checkCommand("webcrack", "--version") {
		fmt.Println("✅ webcrack已安装")
	} else {
		fmt.Println("❌ webcrack未安装 - 请运行: npm install -g webcrack")
		allOK = false
	}

	// 检查playwright
	if checkCommand("playwright", "--version") {
		fmt.Println("✅ Playwright已安装")
	} else {
		fmt.Println("⚠️  Playwright未安装 - 动态爬取功能将不可用")
		fmt.Println("   安装方法: npm install -g playwright && playwright install chromium")
	}

	// 检查Node.js
	if checkCommand("node", "--version") {
		nodeVersion := getCommandOutput("node", "--version")
		fmt.Printf("✅ Node.js已安装: %s\n", strings.TrimSpace(nodeVersion))
	} else {
		fmt.Println("❌ Node.js未安装 - webcrack依赖Node.js")
		allOK = false
	}

	// 检查npm
	if checkCommand("npm", "--version") {
		npmVersion := getCommandOutput("npm", "--version")
		fmt.Printf("✅ npm已安装: %s\n", strings.TrimSpace(npmVersion))
	} else {
		fmt.Println("❌ npm未安装")
		allOK = false
	}

	// 检查项目依赖
	fmt.Println()
	fmt.Println("检查Go模块依赖...")
	if _, err := os.Stat("go.mod"); err == nil {
		fmt.Println("✅ go.mod文件存在")

		// 运行go mod tidy
		fmt.Println("正在整理依赖...")
		cmd := exec.Command("go", "mod", "tidy")
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ go mod tidy失败: %v\n", err)
			allOK = false
		} else {
			fmt.Println("✅ 依赖整理完成")
		}

		// 运行go mod download
		fmt.Println("正在下载依赖...")
		cmd = exec.Command("go", "mod", "download")
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ go mod download失败: %v\n", err)
			allOK = false
		} else {
			fmt.Println("✅ 依赖下载完成")
		}
	} else {
		fmt.Println("❌ go.mod文件不存在")
		allOK = false
	}

	// 检查项目结构
	fmt.Println()
	fmt.Println("检查项目结构...")
	requiredDirs := []string{
		"cmd/jsfindcrack",
		"internal/core",
		"internal/crawlers",
		"internal/utils",
		"internal/models",
		"configs",
		"scripts",
		"tests",
	}

	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); err == nil {
			fmt.Printf("✅ %s/\n", dir)
		} else {
			fmt.Printf("❌ %s/ 不存在\n", dir)
			allOK = false
		}
	}

	fmt.Println()
	fmt.Println("==============================================")
	if allOK {
		fmt.Println("✅ 环境验证通过!可以开始开发了。")
		fmt.Println()
		fmt.Println("下一步:")
		fmt.Println("  1. 运行 'make install' 安装依赖")
		fmt.Println("  2. 运行 'make build' 构建项目")
		fmt.Println("  3. 运行 './jsfindcrack --help' 查看帮助")
		os.Exit(0)
	} else {
		fmt.Println("❌ 环境验证失败,请解决上述问题。")
		os.Exit(1)
	}
}

// checkCommand 检查命令是否可用
func checkCommand(name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	err := cmd.Run()
	return err == nil
}

// getCommandOutput 获取命令输出
func getCommandOutput(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(output)
}
