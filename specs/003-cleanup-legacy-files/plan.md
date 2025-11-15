# Implementation Plan: 清理遗留Python文件

**Branch**: `003-cleanup-legacy-files` | **Date**: 2025-11-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-cleanup-legacy-files/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

本功能旨在清理Python到Go迁移完成后的遗留文件,包括删除所有Python源代码文件(.py)、src/目录、Python配置文件(requirements.txt)、Python构建产物(__pycache__、.pyc等),同时确保保留所有Go代码、项目配置、文档和测试资源。

清理过程需要提供预览和确认机制,生成详细的清理报告,并确保清理后Go版本的所有功能测试通过。技术方案采用Shell脚本实现,提供干跑模式、交互式确认和完整的文件清单记录。

## Technical Context

**Language/Version**: Shell Script (Bash 4.0+) / Go 1.21+
**Primary Dependencies**: 标准Unix工具(find, rm, du, tree), Git
**Storage**: 文件系统操作,不涉及数据库
**Testing**: Shell脚本单元测试(bats-core), Go测试套件验证
**Target Platform**: macOS/Linux (Unix-like系统)
**Project Type**: 维护工具脚本 + 文档更新
**Performance Goals**: 清理操作<30秒完成(当前项目规模约20个Python文件)
**Constraints**: 必须100%保留Go代码和关键配置,误删文件零容忍
**Scale/Scope**: 约20个.py文件,1个src目录(约6个子目录),潜在的构建产物文件

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### 语言与文档规范
- ✅ 所有代码注释使用中文
- ✅ 所有Markdown文档使用中文命名(README.md除外)
- ✅ 禁止使用emoji表情符号
- ✅ 变量/函数名遵循语言标准规范,配备中文注释

### 领域驱动设计原则
- ✅ 本功能为工具脚本,不涉及复杂业务领域,遵循单一职责原则
- ✅ 脚本设计清晰分离:文件识别、确认交互、删除执行、报告生成

### 工程化代码结构
- ✅ 脚本放置在`scripts/`目录下,符合工程化结构
- ✅ 相关文档放置在`specs/003-cleanup-legacy-files/`
- ⚠️ 注意: 删除`src/`目录本身不违反工程化原则,因为Go项目已迁移到`internal/`和`cmd/`

### 版本控制流程
- ✅ 所有修改必须提交到Git
- ✅ 使用约定式提交: `feat: 添加Python文件清理脚本`
- ✅ 在feature分支`003-cleanup-legacy-files`上开发
- ✅ 清理前必须确保工作区干净,建议创建Git提交点

### 代码质量与风格
- ✅ Shell脚本遵循ShellCheck规范
- ✅ 脚本包含详细的中文注释,解释关键逻辑和安全检查
- N/A Go代码不涉及修改,无需检查

### 错误处理规范
- ✅ 脚本必须检查所有命令返回值
- ✅ 使用`set -euo pipefail`确保错误立即退出
- ✅ 提供清晰的错误消息和回滚建议

### 日志规范
- ✅ 脚本输出结构化日志(带时间戳、操作类型)
- ✅ 区分信息、警告、错误级别
- ✅ 生成详细的清理报告文件

### 测试优先原则
- ✅ 提供干跑模式(-d/--dry-run)作为测试手段
- ✅ 清理后运行Go测试套件验证完整性
- ✅ 编写Shell脚本单元测试验证文件识别逻辑

### 安全性要求
- ✅ 强制交互式确认,防止误操作
- ✅ 白名单机制确保关键文件不被删除
- ✅ 删除前生成备份建议(Git提交或手动备份)
- ✅ 所有路径使用绝对路径或经过验证的相对路径

### 文档与任务管理
- ✅ spec.md已完成,plan.md正在填充
- ✅ 将生成tasks.md任务分解
- ✅ 清理报告将记录所有操作历史

### 代码审查流程
- ✅ 脚本完成后需要代码审查
- ✅ 审查重点:文件识别准确性、安全性、错误处理

**GATE状态**: ✅ PASS - 所有宪章要求符合,可以进入Phase 0研究阶段

## Project Structure

### Documentation (this feature)

```text
specs/003-cleanup-legacy-files/
├── plan.md              # 本文件 (实施计划)
├── research.md          # Phase 0输出 (技术研究)
├── data-model.md        # N/A (本功能不涉及数据模型)
├── quickstart.md        # Phase 1输出 (脚本使用快速入门)
├── contracts/           # N/A (本功能不涉及API契约)
└── tasks.md             # Phase 2输出 (任务分解 - 由/speckit.tasks生成)
```

### Source Code (repository root)

本功能主要涉及脚本工具和文档,遵循现有Go项目的工程化结构:

```text
JsFIndcrack/
├── cmd/                        # Go主程序入口 (保留)
│   └── jsfindcrack/
│       └── main.go
│
├── internal/                   # Go私有代码 (保留)
│   ├── config/                # 配置管理
│   ├── core/                  # 核心业务逻辑
│   ├── crawlers/              # 爬虫实现
│   ├── models/                # 数据模型
│   └── utils/                 # 工具函数
│
├── scripts/                    # 构建和维护脚本 (保留并添加)
│   ├── build.sh               # 现有构建脚本
│   └── cleanup-python.sh      # 新增: Python文件清理脚本 ⭐
│
├── tests/                      # 测试文件 (保留)
│   ├── unit/
│   ├── integration/
│   ├── e2e/
│   └── benchmark/
│
├── configs/                    # 配置文件 (保留)
├── specs/                      # 功能规范文档 (保留)
├── .specify/                   # 项目管理工具 (保留)
├── .github/                    # GitHub配置 (保留)
│
├── Makefile                    # 构建配置 (保留)
├── go.mod                      # Go依赖 (保留)
├── go.sum                      # Go依赖锁定 (保留)
├── .gitignore                  # Git忽略规则 (保留)
│
├── src/                        # Python源代码 (将被删除) ❌
│   ├── core/
│   ├── crawlers/
│   └── utils/
│
├── main.py                     # Python主程序 (将被删除) ❌
├── requirements.txt            # Python依赖 (将被删除) ❌
└── __pycache__/                # Python缓存 (将被删除,如果存在) ❌
```

**Structure Decision**:

本功能采用维护脚本模式,新增文件仅为`scripts/cleanup-python.sh`清理脚本。该脚本将:
1. 识别并列出所有待删除的Python相关文件和目录
2. 提供干跑模式预览待删除内容
3. 交互式确认后执行删除操作
4. 生成清理报告记录删除历史

清理目标:
- **删除**: src/目录、所有.py文件、requirements.txt、Python构建产物
- **保留**: 所有Go代码(cmd/、internal/)、配置(configs/、Makefile、go.mod)、文档(specs/、.specify/)、测试(tests/)

不涉及复杂的源代码结构变更,主要是文件系统清理操作。

## Complexity Tracking

本功能无宪章违规,无需填写复杂性跟踪表。

---

## Phase 0: 研究与决策 ✅

**状态**: 已完成
**输出文档**: [research.md](./research.md)

### 研究成果总结

完成了6个关键技术领域的研究:

1. **Shell脚本最佳实践**: 确定使用`set -euo pipefail`错误处理机制
2. **文件识别策略**: 采用白名单+黑名单双重机制,确保安全性
3. **交互式确认设计**: 三级确认机制(干跑、预览、执行)
4. **清理报告生成**: JSON + Markdown双格式报告
5. **回滚和恢复机制**: 基于Git标签的恢复方案
6. **测试策略**: 单元测试(bats) + 集成测试 + Go测试验证

所有NEEDS CLARIFICATION项已解决,无遗留技术问题。

### 关键决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 脚本语言 | Bash | 适合文件操作,无需额外依赖 |
| 安全机制 | 三级确认 + 白名单 | 多层防护,防止误删 |
| 报告格式 | JSON + Markdown | 兼顾程序化处理和人类阅读 |
| 恢复方案 | Git标签 | 利用现有版本控制,简单可靠 |
| 测试方法 | bats + 真实环境 | 确保逻辑和实际效果双重验证 |

---

## Phase 1: 设计与契约 ✅

**状态**: 已完成
**输出文档**: [quickstart.md](./quickstart.md)

### 设计概要

**脚本架构**:
```
cleanup-python.sh
├── 初始化检查 (Git状态、工作目录验证)
├── 文件扫描模块 (识别Python文件和目录)
├── 白名单验证模块 (确保关键文件不被包含)
├── 交互确认模块 (干跑/预览/执行模式)
├── 删除执行模块 (安全删除文件和目录)
└── 报告生成模块 (JSON + Markdown输出)
```

### 核心模块设计

#### 1. 文件扫描模块

**职责**: 识别所有待删除的Python文件

**输入**: 项目根目录路径
**输出**: 分类文件列表(源文件、配置、构建产物、目录)

**关键函数**:
- `find_python_source_files()`: 查找所有.py文件
- `find_python_config_files()`: 查找requirements.txt等配置
- `find_python_build_artifacts()`: 查找__pycache__、.pyc等
- `find_python_directories()`: 识别src/等待删除目录

#### 2. 白名单验证模块

**职责**: 确保关键文件不被误删

**输入**: 待删除文件列表
**输出**: 验证结果(通过/失败) + 冲突文件列表

**白名单定义**:
```bash
WHITELIST_DIRS=(
  "cmd" "internal" "tests" "configs" "specs"
  ".specify" ".github" "scripts" "dist"
)

WHITELIST_FILES=(
  "go.mod" "go.sum" "Makefile" ".gitignore"
  "README.md" "*.md" "*.sh" "*.go"
)
```

**验证逻辑**:
- 遍历待删除列表
- 检查是否与白名单匹配
- 如有冲突,立即报错退出

#### 3. 交互确认模块

**职责**: 根据运行模式提供相应的用户交互

**模式定义**:

| 模式 | 参数 | 行为 |
|------|------|------|
| 干跑模式 | `--dry-run` / `-d` | 仅显示待删除文件,不执行 |
| 预览模式 | `--preview` / `-p` | 显示详细信息 + 询问是否继续 |
| 执行模式 | `--execute` / `-e` | 要求输入"yes"确认后执行 |
| 强制模式 | `--force` | 跳过确认(CI/CD用) |
| 列表模式 | `--list-only` | 仅输出文件路径 |

**确认流程**:
```
执行模式 (--execute):
  1. 显示待删除文件统计
  2. 显示警告信息(红色)
  3. 提示输入"yes"
  4. 验证输入 == "yes"
  5. 执行删除或取消
```

#### 4. 删除执行模块

**职责**: 安全地删除文件和目录

**执行策略**:
- 先删除文件,后删除目录(避免非空目录错误)
- 每次删除前验证路径有效性
- 使用`rm -rf`删除目录,`rm -f`删除文件
- 记录每次删除操作到日志

**错误处理**:
- 任何删除失败立即退出(`set -e`)
- 记录失败的文件/目录路径
- 提供回滚建议

#### 5. 报告生成模块

**职责**: 生成详细的清理报告

**报告内容**:
- 时间戳和运行模式
- 删除文件清单(按类别)
- 文件数量和大小统计
- 白名单验证结果
- Git状态快照(清理前后)

**输出路径**: `specs/003-cleanup-legacy-files/reports/`

### 使用流程

详细使用说明见[quickstart.md](./quickstart.md),包括:
- 前置条件检查
- 基本用法(干跑→备份→执行→验证)
- 高级选项
- 恢复指南
- 故障排查
- FAQ

### 代理上下文更新

已执行`.specify/scripts/bash/update-agent-context.sh claude`,更新内容:
- 语言: Shell Script (Bash 4.0+) / Go 1.21+
- 依赖: 标准Unix工具(find, rm, du, tree), Git
- 项目类型: 维护工具脚本 + 文档更新

---

## Phase 1 宪章复查 ✅

**复查时间**: 2025-11-15
**复查结果**: ✅ 全部通过

### 设计符合性验证

#### 语言与文档规范
- ✅ quickstart.md使用中文编写
- ✅ 脚本中将包含中文注释
- ✅ 无emoji使用

#### 领域驱动设计
- ✅ 脚本模块划分清晰(扫描、验证、确认、删除、报告)
- ✅ 单一职责原则:每个模块专注一个功能

#### 工程化结构
- ✅ 脚本放置在`scripts/`目录
- ✅ 报告输出到`specs/003-cleanup-legacy-files/reports/`
- ✅ 文档结构完整(spec, plan, research, quickstart)

#### 错误处理
- ✅ 使用`set -euo pipefail`
- ✅ 白名单验证提供安全屏障
- ✅ 详细错误消息和回滚建议

#### 测试策略
- ✅ 干跑模式作为测试手段
- ✅ 计划编写bats单元测试
- ✅ 清理后运行Go测试验证

#### 安全性
- ✅ 多级确认机制
- ✅ Git备份点强制建议
- ✅ 白名单防误删

**结论**: 设计完全符合项目宪章要求,可以进入Phase 2任务分解阶段。

---

## 下一步行动

1. **运行 `/speckit.tasks`** 命令生成tasks.md任务分解文档
2. **执行任务** 根据tasks.md实施清理脚本
3. **代码审查** 完成后提交代码审查
4. **合并主分支** 审查通过后合并到main分支

---

## 附录

### 预计时间线

| 阶段 | 任务 | 预计耗时 |
|------|------|----------|
| Phase 0 | 研究与决策 | ✅ 已完成 |
| Phase 1 | 设计与文档 | ✅ 已完成 |
| Phase 2 | 任务分解 | 待执行 (/speckit.tasks) |
| 实施 | 编写清理脚本 | 2-3小时 |
| 实施 | 编写测试 | 1-2小时 |
| 测试 | 运行测试和验证 | 1小时 |
| 审查 | 代码审查和修正 | 1小时 |
| **总计** | | **5-7小时** |

### 交付物清单

- [x] spec.md - 功能规范
- [x] plan.md - 实施计划(本文档)
- [x] research.md - 技术研究
- [x] quickstart.md - 使用指南
- [ ] tasks.md - 任务分解(待生成)
- [ ] scripts/cleanup-python.sh - 清理脚本(待实施)
- [ ] tests/unit/cleanup-python.bats - 单元测试(待实施)
- [ ] reports/ - 清理报告(执行时生成)

### 风险管理

| 风险 | 概率 | 影响 | 缓解措施 | 状态 |
|------|------|------|----------|------|
| 误删重要文件 | 低 | 高 | 白名单+三级确认+Git备份 | ✅ 已缓解 |
| 脚本跨平台兼容性 | 中 | 中 | 测试macOS/Linux,文档说明Windows需WSL | ✅ 已缓解 |
| 用户误操作 | 低 | 高 | 强制输入"yes",红色警告 | ✅ 已缓解 |
| 清理后测试失败 | 低 | 中 | 强制前置条件:Go测试通过 | ✅ 已缓解 |

---

**计划完成时间**: 2025-11-15
**负责人**: AI Agent (Claude Code)
**审核状态**: 待审核
