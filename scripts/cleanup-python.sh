#!/usr/bin/env bash
# cleanup-python.sh - Python文件清理脚本
# 用途: 在Python到Go迁移完成后,删除所有Python遗留文件
# 版本: 1.0.0
# 作者: JsFIndcrack项目团队

# 严格错误处理模式
set -euo pipefail

# 脚本版本信息
readonly SCRIPT_VERSION="1.0.0"
readonly SCRIPT_NAME="cleanup-python.sh"

# 显示帮助信息
show_help() {
    cat << EOF
用法: $SCRIPT_NAME [选项]

Python文件清理工具 - 删除迁移后的Python遗留文件

选项:
  -d, --dry-run     干跑模式 - 仅显示待删除文件,不执行删除操作
  -p, --preview     预览模式 - 显示详细文件信息并询问是否继续
  -e, --execute     执行模式 - 执行清理操作(需要确认)
  -f, --force       强制模式 - 跳过确认直接执行(CI/CD用)
  -l, --list-only   列表模式 - 仅输出文件路径列表
  -h, --help        显示此帮助信息
  -v, --version     显示版本信息

示例:
  $SCRIPT_NAME --dry-run              # 预览待删除文件
  $SCRIPT_NAME --execute              # 执行清理(需确认)
  $SCRIPT_NAME --execute --force      # 强制执行(跳过确认)

更多信息: 参见 specs/003-cleanup-legacy-files/quickstart.md
EOF
}

# 显示版本信息
show_version() {
    echo "$SCRIPT_NAME 版本 $SCRIPT_VERSION"
}

# 日志函数 (T006)
log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $*"
}

log_warn() {
    echo "[WARN] $(date '+%Y-%m-%d %H:%M:%S') - $*" >&2
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $*" >&2
}

# 白名单常量定义 (T007)
declare -a WHITELIST_DIRS=(
    "cmd"
    "internal"
    "tests"
    "configs"
    "config"
    "specs"
    ".specify"
    ".github"
    "scripts"
    "dist"
    "output"
    "logs"
)

declare -a WHITELIST_FILES=(
    "go.mod"
    "go.sum"
    "Makefile"
    ".gitignore"
    "README.md"
)

# 白名单文件模式 (通配符)
declare -a WHITELIST_PATTERNS=(
    "*.md"
    "*.sh"
    "*.go"
    ".git"
)

# 全局变量
MODE=""              # 运行模式: dry-run, preview, execute, list-only
FORCE=false          # 强制模式标志
PROJECT_ROOT=""      # 项目根目录
TEMP_FILE=""         # 临时文件路径

# 参数解析 (T005)
parse_arguments() {
    if [ $# -eq 0 ]; then
        show_help
        exit 0
    fi

    while [ $# -gt 0 ]; do
        case "$1" in
            -d|--dry-run)
                MODE="dry-run"
                shift
                ;;
            -p|--preview)
                MODE="preview"
                shift
                ;;
            -e|--execute)
                MODE="execute"
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -l|--list-only)
                MODE="list-only"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--version)
                show_version
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # 检查模式是否设置
    if [ -z "$MODE" ]; then
        log_error "必须指定运行模式 (--dry-run, --preview, --execute, 或 --list-only)"
        show_help
        exit 1
    fi
}

# 初始化检查 (T008)
check_prerequisites() {
    # 获取项目根目录
    PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

    # 验证Git仓库
    if ! git -C "$PROJECT_ROOT" rev-parse --git-dir > /dev/null 2>&1; then
        log_error "当前目录不是Git仓库: $PROJECT_ROOT"
        exit 1
    fi

    # 验证关键目录存在
    if [ ! -d "$PROJECT_ROOT/cmd" ] || [ ! -d "$PROJECT_ROOT/internal" ]; then
        log_error "Go项目目录结构不完整 (缺少cmd/或internal/)"
        log_error "请确认Python到Go迁移已完成"
        exit 1
    fi

    # 验证写入权限
    if [ ! -w "$PROJECT_ROOT" ]; then
        log_error "没有项目根目录的写入权限: $PROJECT_ROOT"
        exit 1
    fi

    log_info "初始化检查通过 - 项目根目录: $PROJECT_ROOT"
}

# Git状态检查 (T009)
check_git_status() {
    local git_status
    git_status=$(git -C "$PROJECT_ROOT" status --porcelain 2>&1)

    if [ -n "$git_status" ]; then
        log_warn "Git工作区不干净 - 检测到未提交的更改"
        log_warn "建议: git add -A && git commit -m '清理前保存点'"
        log_warn "或使用: git stash"

        if [ "$FORCE" != "true" ]; then
            log_error "为安全起见,请先提交或暂存更改"
            log_error "或使用 --force 选项跳过此检查 (不推荐)"
            exit 1
        fi
    else
        log_info "Git工作区干净 ✓"
    fi
}

# 临时文件管理 (T010)
setup_temp_files() {
    TEMP_FILE=$(mktemp "${TMPDIR:-/tmp}/cleanup-python.XXXXXX")

    # 确保退出时清理临时文件
    trap 'rm -f "$TEMP_FILE"' EXIT

    log_info "临时文件创建: $TEMP_FILE"
}

# 主函数
main() {
    parse_arguments "$@"
    check_prerequisites
    check_git_status
    setup_temp_files

    log_info "Python文件清理工具初始化完成"
    log_info "运行模式: $MODE"

    # TODO: 实现核心清理逻辑
    log_warn "核心清理逻辑尚未实现"
}

# 执行主函数
main "$@"
