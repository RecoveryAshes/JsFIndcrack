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

# 文件扫描函数 (T017)
find_python_source_files() {
    find "$PROJECT_ROOT" -type f -name "*.py" 2>/dev/null | grep -v "__pycache__" || true
}

# 配置文件扫描 (T018)
find_python_config_files() {
    local config_files=()

    [ -f "$PROJECT_ROOT/requirements.txt" ] && config_files+=("$PROJECT_ROOT/requirements.txt")
    [ -f "$PROJECT_ROOT/setup.py" ] && config_files+=("$PROJECT_ROOT/setup.py")
    [ -f "$PROJECT_ROOT/setup.cfg" ] && config_files+=("$PROJECT_ROOT/setup.cfg")
    [ -f "$PROJECT_ROOT/MANIFEST.in" ] && config_files+=("$PROJECT_ROOT/MANIFEST.in")
    [ -f "$PROJECT_ROOT/pyproject.toml" ] && config_files+=("$PROJECT_ROOT/pyproject.toml")

    if [ ${#config_files[@]} -gt 0 ]; then
        printf '%s\n' "${config_files[@]}"
    fi
}

# 目录识别 (T019)
find_python_directories() {
    local py_dirs=()

    # src目录是主要目标
    [ -d "$PROJECT_ROOT/src" ] && py_dirs+=("$PROJECT_ROOT/src")

    # 其他可能的Python目录
    [ -d "$PROJECT_ROOT/lib" ] && py_dirs+=("$PROJECT_ROOT/lib")
    [ -d "$PROJECT_ROOT/python" ] && py_dirs+=("$PROJECT_ROOT/python")

    if [ ${#py_dirs[@]} -gt 0 ]; then
        printf '%s\n' "${py_dirs[@]}"
    fi
}

# 构建产物扫描 (T029 - US2)
find_python_build_artifacts() {
    # __pycache__目录
    find "$PROJECT_ROOT" -type d -name "__pycache__" 2>/dev/null || true

    # .pyc和.pyo文件
    find "$PROJECT_ROOT" -type f \( -name "*.pyc" -o -name "*.pyo" \) 2>/dev/null || true

    # .egg-info目录
    find "$PROJECT_ROOT" -type d -name "*.egg-info" 2>/dev/null || true
}

# 白名单验证 (T020)
validate_against_whitelist() {
    local file_path="$1"
    local basename
    basename=$(basename "$file_path")
    local dirname
    dirname=$(dirname "$file_path")

    # 检查白名单目录
    for wl_dir in "${WHITELIST_DIRS[@]}"; do
        # 检查文件是否在保护目录内(路径开头匹配或包含/目录名/)
        if [[ "$file_path" == "$PROJECT_ROOT/$wl_dir" ]] || [[ "$file_path" == "$PROJECT_ROOT/$wl_dir/"* ]]; then
            log_error "⚠️ 白名单冲突: $file_path (位于保护目录: $wl_dir)"
            return 1
        fi
    done

    # 检查白名单文件
    for wl_file in "${WHITELIST_FILES[@]}"; do
        if [[ "$basename" == "$wl_file" ]]; then
            log_error "⚠️ 白名单冲突: $file_path (受保护文件: $wl_file)"
            return 1
        fi
    done

    # 检查白名单模式
    for pattern in "${WHITELIST_PATTERNS[@]}"; do
        case "$basename" in
            $pattern)
                # .py文件不应该被模式保护
                if [[ "$basename" == *.py ]]; then
                    continue
                fi
                log_error "⚠️ 白名单冲突: $file_path (匹配模式: $pattern)"
                return 1
                ;;
        esac
    done

    return 0
}

# 文件分类汇总 (T021)
categorize_files() {
    log_info "分类待删除文件..."

    PYTHON_SOURCE_FILES=()
    PYTHON_CONFIG_FILES=()
    PYTHON_BUILD_ARTIFACTS=()
    PYTHON_DIRECTORIES=()

    # 收集源文件
    while IFS= read -r file; do
        if [ -n "$file" ]; then
            PYTHON_SOURCE_FILES+=("$file")
        fi
    done < <(find_python_source_files)

    # 收集配置文件
    while IFS= read -r file; do
        if [ -n "$file" ]; then
            PYTHON_CONFIG_FILES+=("$file")
        fi
    done < <(find_python_config_files)

    # 收集构建产物
    while IFS= read -r file; do
        if [ -n "$file" ]; then
            PYTHON_BUILD_ARTIFACTS+=("$file")
        fi
    done < <(find_python_build_artifacts)

    # 收集目录
    while IFS= read -r dir; do
        if [ -n "$dir" ]; then
            PYTHON_DIRECTORIES+=("$dir")
        fi
    done < <(find_python_directories)

    # 验证白名单
    local has_conflicts=false
    local all_files=()

    # 合并所有文件和目录到一个数组进行验证
    if [ ${#PYTHON_SOURCE_FILES[@]} -gt 0 ]; then
        all_files+=("${PYTHON_SOURCE_FILES[@]}")
    fi
    if [ ${#PYTHON_CONFIG_FILES[@]} -gt 0 ]; then
        all_files+=("${PYTHON_CONFIG_FILES[@]}")
    fi
    if [ ${#PYTHON_DIRECTORIES[@]} -gt 0 ]; then
        all_files+=("${PYTHON_DIRECTORIES[@]}")
    fi

    if [ ${#all_files[@]} -gt 0 ]; then
        for file in "${all_files[@]}"; do
            if ! validate_against_whitelist "$file"; then
                has_conflicts=true
            fi
        done
    fi

    if [ "$has_conflicts" = true ]; then
        log_error "检测到白名单冲突 - 清理操作已中止"
        exit 1
    fi

    log_info "文件分类完成"
}

# 干跑模式显示 (T022)
display_cleanup_preview() {
    echo ""
    echo "=== 待删除文件清单 ==="
    echo ""

    # Python源文件
    if [ ${#PYTHON_SOURCE_FILES[@]} -gt 0 ]; then
        local total_size=0
        echo "【Python源文件】 (${#PYTHON_SOURCE_FILES[@]}个文件)"
        for file in "${PYTHON_SOURCE_FILES[@]}"; do
            if [ -f "$file" ]; then
                local size
                size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
                total_size=$((total_size + size))
                echo "  - $file ($(numfmt --to=iec --suffix=B "$size" 2>/dev/null || echo "${size}B"))"
            fi
        done
        echo "  小计: $(numfmt --to=iec --suffix=B "$total_size" 2>/dev/null || echo "${total_size}B")"
        echo ""
    fi

    # Python配置文件
    if [ ${#PYTHON_CONFIG_FILES[@]} -gt 0 ]; then
        echo "【Python配置文件】 (${#PYTHON_CONFIG_FILES[@]}个文件)"
        for file in "${PYTHON_CONFIG_FILES[@]}"; do
            echo "  - $file"
        done
        echo ""
    fi

    # 构建产物
    if [ ${#PYTHON_BUILD_ARTIFACTS[@]} -gt 0 ]; then
        echo "【Python构建产物】 (${#PYTHON_BUILD_ARTIFACTS[@]}个文件/目录)"
        local count=0
        for item in "${PYTHON_BUILD_ARTIFACTS[@]}"; do
            if [ $count -lt 10 ]; then
                echo "  - $item"
                count=$((count + 1))
            fi
        done
        if [ ${#PYTHON_BUILD_ARTIFACTS[@]} -gt 10 ]; then
            echo "  ... (还有 $((${#PYTHON_BUILD_ARTIFACTS[@]} - 10)) 个)"
        fi
        echo ""
    fi

    # Python目录
    if [ ${#PYTHON_DIRECTORIES[@]} -gt 0 ]; then
        echo "【目录】 (${#PYTHON_DIRECTORIES[@]}个目录)"
        for dir in "${PYTHON_DIRECTORIES[@]}"; do
            echo "  - $dir/"
        done
        echo ""
    fi

    # 总计
    local total_files=$((${#PYTHON_SOURCE_FILES[@]} + ${#PYTHON_CONFIG_FILES[@]} + ${#PYTHON_BUILD_ARTIFACTS[@]}))
    local total_dirs=${#PYTHON_DIRECTORIES[@]}

    echo "=== 清理摘要 ==="
    echo "总计: ${total_files}个文件, ${total_dirs}个目录"
    echo ""

    # 白名单验证状态
    echo "=== 白名单验证 ==="
    echo "✅ Go源代码: cmd/, internal/ (保留)"
    echo "✅ Go配置: go.mod, go.sum (保留)"
    echo "✅ 构建配置: Makefile (保留)"
    echo "✅ 文档: specs/, .specify/ (保留)"
    echo "✅ 测试: tests/ (保留)"
    echo ""
}

# 文件删除 (T023)
delete_python_files() {
    log_info "开始删除Python文件..."

    local deleted_count=0
    local failed_count=0

    # 删除源文件
    if [ ${#PYTHON_SOURCE_FILES[@]} -gt 0 ]; then
        for file in "${PYTHON_SOURCE_FILES[@]}"; do
            if [ -f "$file" ]; then
                if rm -f "$file" 2>/dev/null; then
                    log_info "删除: $file"
                    deleted_count=$((deleted_count + 1))
                else
                    log_error "删除失败: $file"
                    failed_count=$((failed_count + 1))
                fi
            fi
        done
    fi

    # 删除配置文件
    if [ ${#PYTHON_CONFIG_FILES[@]} -gt 0 ]; then
        for file in "${PYTHON_CONFIG_FILES[@]}"; do
            if [ -f "$file" ]; then
                if rm -f "$file" 2>/dev/null; then
                    log_info "删除: $file"
                    deleted_count=$((deleted_count + 1))
                else
                    log_error "删除失败: $file"
                    failed_count=$((failed_count + 1))
                fi
            fi
        done
    fi

    # 删除构建产物
    if [ ${#PYTHON_BUILD_ARTIFACTS[@]} -gt 0 ]; then
        for item in "${PYTHON_BUILD_ARTIFACTS[@]}"; do
            if [ -e "$item" ]; then
                if rm -rf "$item" 2>/dev/null; then
                    log_info "删除: $item"
                    deleted_count=$((deleted_count + 1))
                else
                    log_error "删除失败: $item"
                    failed_count=$((failed_count + 1))
                fi
            fi
        done
    fi

    # 删除目录(最后删除)
    if [ ${#PYTHON_DIRECTORIES[@]} -gt 0 ]; then
        for dir in "${PYTHON_DIRECTORIES[@]}"; do
            if [ -d "$dir" ]; then
                if rm -rf "$dir" 2>/dev/null; then
                    log_info "删除目录: $dir/"
                    deleted_count=$((deleted_count + 1))
                else
                    log_error "删除目录失败: $dir/"
                    failed_count=$((failed_count + 1))
                fi
            fi
        done
    fi

    log_info "删除完成: 成功 $deleted_count, 失败 $failed_count"

    if [ $failed_count -gt 0 ]; then
        log_error "部分文件删除失败,请检查权限"
        return 1
    fi

    return 0
}

# 主函数
main() {
    parse_arguments "$@"
    check_prerequisites
    check_git_status
    setup_temp_files

    log_info "Python文件清理工具初始化完成"
    log_info "运行模式: $MODE"

    # 扫描和分类文件
    categorize_files

    # 根据模式执行操作
    case "$MODE" in
        dry-run)
            log_info "=== 干跑模式 - 仅预览,不执行删除 ==="
            display_cleanup_preview
            log_info "干跑模式完成 - 未执行任何删除操作"
            log_info "如需执行清理,请运行: $SCRIPT_NAME --execute"
            ;;

        list-only)
            # 仅输出文件路径
            [ ${#PYTHON_SOURCE_FILES[@]} -gt 0 ] && printf '%s\n' "${PYTHON_SOURCE_FILES[@]}"
            [ ${#PYTHON_CONFIG_FILES[@]} -gt 0 ] && printf '%s\n' "${PYTHON_CONFIG_FILES[@]}"
            [ ${#PYTHON_BUILD_ARTIFACTS[@]} -gt 0 ] && printf '%s\n' "${PYTHON_BUILD_ARTIFACTS[@]}"
            [ ${#PYTHON_DIRECTORIES[@]} -gt 0 ] && printf '%s\n' "${PYTHON_DIRECTORIES[@]}"
            ;;

        preview)
            display_cleanup_preview
            read -r -p "是否继续执行清理? (yes/no): " response
            if [[ "$response" == "yes" ]]; then
                delete_python_files
            else
                log_info "清理操作已取消"
            fi
            ;;

        execute)
            display_cleanup_preview
            if [ "$FORCE" != "true" ]; then
                echo "⚠️  警告: 即将删除上述文件和目录!"
                echo "⚠️  此操作不可撤销 (除非通过Git恢复)"
                echo ""
                read -r -p "请输入 'yes' 确认继续: " confirmation
                if [[ "$confirmation" != "yes" ]]; then
                    log_info "清理操作已取消"
                    exit 0
                fi
            else
                log_warn "强制模式: 跳过确认提示"
            fi

            delete_python_files
            log_info "✅ 清理完成!"
            log_info "建议: 运行 'go test ./...' 验证Go功能完整性"
            ;;

        *)
            log_error "未知模式: $MODE"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
