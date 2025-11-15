# Research Document: 清理遗留Python文件

**Feature**: 003-cleanup-legacy-files
**Date**: 2025-11-15
**Status**: 完成

## 研究目的

为Python文件清理脚本的实现提供技术决策支持,确保清理操作的安全性、可靠性和可验证性。

## 研究任务与决策

### 1. Shell脚本最佳实践研究

**研究问题**: 如何编写安全可靠的文件删除脚本,避免误删和错误处理不当?

**决策**: 采用Bash脚本,遵循以下最佳实践

**理由**:
- **错误处理**: 使用`set -euo pipefail`确保任何命令失败时脚本立即退出
  - `-e`: 遇到错误立即退出
  - `-u`: 使用未定义变量时报错
  - `-o pipefail`: 管道中任一命令失败即返回失败状态
- **路径安全**: 所有路径操作使用引号包裹,防止空格和特殊字符问题
- **变量验证**: 在使用变量前验证其非空且符合预期格式
- **原子性操作**: 使用临时文件记录待删除列表,确保操作可追溯

**替代方案考虑**:
- Python脚本: 更容易跨平台,但本项目正在清理Python,使用Shell更合适
- Go程序: 需要编译,对于简单的文件清理过于复杂
- 手动删除: 容易遗漏文件,缺乏记录和验证机制

### 2. 文件识别策略

**研究问题**: 如何准确识别所有需要删除的Python文件,同时避免误删?

**决策**: 采用白名单+黑名单双重机制

**文件识别规则**:

**删除目标(黑名单)**:
```bash
# Python源文件
find . -name "*.py" -type f

# Python配置文件
./requirements.txt
./setup.py (如果存在)

# Python目录
./src/ (整个目录)

# Python构建产物
find . -name "__pycache__" -type d
find . -name "*.pyc" -type f
find . -name "*.pyo" -type f
find . -name "*.egg-info" -type d
```

**保留规则(白名单)**:
```bash
# Go源代码和配置
./cmd/
./internal/
./go.mod
./go.sum

# 项目配置和构建
./Makefile
./scripts/ (除了待删除脚本自身)
./configs/
./config/

# 文档和规范
./specs/
./.specify/
./.github/
./README.md
./*.md

# 测试文件
./tests/

# Git和版本控制
./.git/
./.gitignore

# 输出和日志(用户数据)
./output/
./logs/
./dist/
```

**验证机制**:
- 生成待删除文件列表前,先检查白名单文件是否存在
- 对比待删除列表,确保没有白名单文件被包含
- 提供详细的文件分类统计(源文件、配置、构建产物)

**理由**: 双重机制提供多层保护,防止误删关键文件。白名单确保重要文件不被触碰,黑名单明确删除范围。

**替代方案考虑**:
- 仅使用黑名单: 风险高,可能误删新增的重要文件
- 仅使用白名单: 需要列举所有保留文件,维护成本高
- 模式匹配: 灵活性高但容易出错,不如明确的列表清晰

### 3. 交互式确认设计

**研究问题**: 如何设计用户确认流程,平衡安全性和易用性?

**决策**: 三级确认机制 + 干跑模式

**实施方案**:

**Level 1: 干跑模式(默认)**
```bash
./cleanup-python.sh --dry-run  # 或 -d
# 输出:
# - 列出所有待删除文件(分类显示)
# - 显示文件数量和总大小
# - 显示白名单文件状态(确认未被误包含)
# - 不执行任何删除操作
```

**Level 2: 详细预览**
```bash
./cleanup-python.sh --preview  # 或 -p
# 输出:
# - 详细的文件树视图
# - 每个文件的大小和最后修改时间
# - 清理前后的仓库大小对比估算
# - 询问是否继续
```

**Level 3: 执行确认**
```bash
./cleanup-python.sh --execute  # 或 -e
# 输出:
# - 最终确认提示: "即将删除X个文件,总大小Y MB,是否继续? (yes/no)"
# - 要求输入完整的"yes"才执行(防止误操作)
# - 执行前创建Git状态快照建议
```

**安全措施**:
- 默认为干跑模式,必须显式指定`--execute`才真正删除
- 确认提示使用红色高亮,增强视觉警示
- 提供`--force`选项用于CI/CD自动化,但记录警告日志

**理由**: 三级确认确保用户充分了解删除内容,干跑模式允许安全测试。强制输入"yes"防止按键误触。

**替代方案考虑**:
- 单次确认: 风险高,用户可能习惯性按"y"
- 图形界面: 不适合命令行工具,增加复杂度
- 无确认+强制备份: 依赖备份恢复,用户体验差

### 4. 清理报告生成

**研究问题**: 如何记录清理操作,提供可追溯性和验证依据?

**决策**: 生成结构化JSON报告 + 人类可读的Markdown摘要

**报告格式**:

**JSON报告** (`cleanup-report-{timestamp}.json`):
```json
{
  "timestamp": "2025-11-15T10:30:45Z",
  "operation": "python-cleanup",
  "dry_run": false,
  "summary": {
    "files_deleted": 18,
    "directories_deleted": 4,
    "total_size_kb": 1234
  },
  "categories": {
    "python_source": {
      "count": 18,
      "files": ["./main.py", "./src/core/config.py", ...]
    },
    "python_config": {
      "count": 1,
      "files": ["./requirements.txt"]
    },
    "python_cache": {
      "count": 0,
      "files": []
    },
    "directories": {
      "count": 4,
      "paths": ["./src/", "./src/core/", ...]
    }
  },
  "preserved_files_verified": true,
  "git_status_before": "clean",
  "git_status_after": "modified: (new files deleted)"
}
```

**Markdown摘要** (`cleanup-summary-{timestamp}.md`):
```markdown
# Python文件清理报告

**执行时间**: 2025-11-15 10:30:45
**操作模式**: 执行 (Execute)

## 清理摘要

- 删除文件数: 18个
- 删除目录数: 4个
- 释放空间: 1.2 MB

## 文件分类

### Python源文件 (18个)
- ./main.py
- ./src/core/config.py
- ...

### Python配置文件 (1个)
- ./requirements.txt

### Python缓存 (0个)
(无缓存文件)

## 验证结果

- ✅ 白名单文件完整性: 100%
- ✅ Go代码未被删除: 通过
- ✅ 配置文件保留: 通过

## 后续操作建议

1. 运行Go测试套件: `make test` 或 `go test ./...`
2. 提交Git变更: `git add -A && git commit -m "chore: 删除Python遗留文件"`
3. 验证构建: `make build`
```

**报告存放位置**: `./specs/003-cleanup-legacy-files/reports/`

**理由**:
- JSON格式便于程序化处理和自动化验证
- Markdown格式便于人类阅读和归档
- 时间戳确保每次执行都有独立记录
- 分类统计帮助理解清理范围

**替代方案考虑**:
- 仅日志输出: 不利于后续查阅,缺乏结构化
- 数据库记录: 过于复杂,不适合一次性清理任务
- CSV格式: 结构化程度低,不如JSON灵活

### 5. 回滚和恢复机制

**研究问题**: 如果清理后发现问题,如何快速恢复?

**决策**: 依赖Git版本控制 + 清理前提示创建备份点

**实施方案**:

**清理前检查**:
```bash
# 检查Git状态
if [ -n "$(git status --porcelain)" ]; then
  echo "警告: 工作区有未提交的更改"
  echo "建议先提交或暂存: git add -A && git commit -m '清理前保存点'"
  echo "或使用: git stash"
  exit 1
fi

# 创建清理前的Git标签
git tag -a "before-python-cleanup-$(date +%Y%m%d-%H%M%S)" -m "Python清理前备份点"
```

**恢复方案**:
```bash
# 如果需要恢复
git reset --hard before-python-cleanup-YYYYMMDD-HHMMSS
# 或
git checkout <commit-hash> -- src/ main.py requirements.txt
```

**文档说明**: 在quickstart.md中提供详细的恢复步骤和常见问题解答

**理由**:
- Git是项目已有的版本控制系统,无需引入额外工具
- Git标签提供明确的恢复点,易于识别
- 强制清理前工作区干净,确保回滚不丢失其他更改

**替代方案考虑**:
- 手动tar备份: 占用额外空间,管理复杂
- 复制到备份目录: 不如Git优雅,需要手动清理备份
- 系统快照: 依赖特定平台,不通用

### 6. 测试策略

**研究问题**: 如何验证清理脚本的正确性和安全性?

**决策**: 单元测试(bats) + 集成测试(真实环境模拟) + Go测试套件验证

**测试方案**:

**单元测试 (使用bats-core)**:
```bash
# tests/unit/cleanup-python.bats

@test "识别所有.py文件" {
  run find_python_files
  [ "$status" -eq 0 ]
  [[ "$output" == *"main.py"* ]]
  [[ "$output" == *"src/core/config.py"* ]]
}

@test "白名单文件不被包含" {
  run generate_deletion_list
  [[ "$output" != *"go.mod"* ]]
  [[ "$output" != *"Makefile"* ]]
  [[ "$output" != *"specs/"* ]]
}

@test "干跑模式不删除文件" {
  initial_count=$(find . -name "*.py" | wc -l)
  run ./cleanup-python.sh --dry-run
  final_count=$(find . -name "*.py" | wc -l)
  [ "$initial_count" -eq "$final_count" ]
}
```

**集成测试 (测试环境)**:
```bash
# 创建测试目录结构
mkdir -p test-cleanup/{src/core,cmd,internal}
touch test-cleanup/main.py
touch test-cleanup/src/core/test.py
touch test-cleanup/go.mod

# 运行清理脚本
cd test-cleanup
../cleanup-python.sh --execute --force

# 验证结果
[ ! -f main.py ]              # Python文件已删除
[ ! -d src ]                  # src目录已删除
[ -f go.mod ]                 # Go配置保留
```

**清理后验证**:
```bash
# 运行Go测试套件
make test || go test -v ./...

# 验证构建
make build || go build ./cmd/jsfindcrack

# 检查关键文件存在
test -f go.mod
test -f Makefile
test -d internal
test -d cmd
```

**CI/CD集成**: 在GitHub Actions中运行测试,确保脚本在不同环境下正常工作

**理由**:
- 多层测试确保脚本可靠性
- 单元测试验证逻辑正确性
- 集成测试验证实际效果
- Go测试验证清理后项目完整性

**替代方案考虑**:
- 仅手动测试: 不可重复,容易遗漏边界情况
- 仅在生产环境测试: 风险高,出错代价大
- Mock文件系统: 过于复杂,不如真实环境直观

## 技术栈总结

| 组件 | 技术选择 | 版本要求 |
|------|---------|----------|
| 脚本语言 | Bash | 4.0+ |
| 错误处理 | set -euo pipefail | Bash内置 |
| 测试框架 | bats-core | 1.5+ (可选) |
| 报告格式 | JSON + Markdown | 标准格式 |
| 版本控制 | Git | 2.0+ |
| 验证工具 | ShellCheck | 0.7+ (开发时) |

## 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 误删重要文件 | 高 | 白名单机制、三级确认、干跑模式 |
| 脚本执行失败 | 中 | set -e错误退出、详细日志 |
| 无法恢复 | 高 | Git标签备份、清理前状态检查 |
| 跨平台兼容性 | 低 | 使用POSIX标准命令、测试macOS/Linux |
| 用户误操作 | 中 | 强制输入"yes"、红色警告 |

## 未解决问题

无。所有技术决策已明确,可以进入Phase 1设计阶段。

## 参考资料

- Google Shell Style Guide: https://google.github.io/styleguide/shellguide.html
- Bash Strict Mode: http://redsymbol.net/articles/unofficial-bash-strict-mode/
- ShellCheck: https://www.shellcheck.net/
- bats-core: https://bats-core.readthedocs.io/
