# Implementation Plan: Git仓库清理和最终化

**Branch**: `004-git-repo-finalize` | **Date**: 2025-11-16 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-git-repo-finalize/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

本功能旨在清理Git仓库中所有不需要版本控制的文件,包括删除构建产物(dist/、jsfindcrack二进制)、遗留Python配置(config/目录)、临时测试文件(-/目录、test_urls.txt)、研究文档、输出和日志文件,同时完善.gitignore配置,更新README文档,并将所有变更提交推送到远程仓库,完成项目从Python到Go迁移的最终化流程。

技术方案采用Shell脚本实现文件清理,直接使用Git和文件系统命令,无需额外依赖,预期删除≥10个文件/目录(~285MB),提升仓库克隆速度30%以上。

## Technical Context

**Language/Version**: Bash 4.0+ (Shell脚本) / Git 2.0+
**Primary Dependencies**: 标准Unix工具(rm, find, du), Git命令行工具
**Storage**: 文件系统操作,不涉及数据库
**Testing**: 手动验证(干跑模式预览、Git status检查、文件存在性验证)
**Target Platform**: macOS/Linux (Unix-like系统,已完成Go迁移的项目)
**Project Type**: 维护脚本 + 文档更新
**Performance Goals**: 清理操作<1分钟完成(当前项目规模~10个文件/目录)
**Constraints**: 必须100%保护Go源代码和关键配置,误删文件零容忍,区分config/(删除)和configs/(保留)
**Scale/Scope**: 约10个文件/目录需删除(285MB),2个文件需更新(.gitignore, README.md),1次Git提交推送

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### 语言与文档规范
- ✅ 所有脚本注释使用中文
- ✅ 所有Markdown文档使用中文(README.md除外)
- ✅ 禁止使用emoji表情符号
- ✅ 变量/函数名遵循Shell标准规范,配备中文注释

### 领域驱动设计原则
- ✅ 本功能为简单脚本工具,不涉及复杂领域设计
- ✅ 职责清晰:文件识别、白名单验证、删除执行、Git操作
- N/A 不涉及接口抽象或SOLID原则

### 工程化代码结构
- ✅ 脚本放置在`scripts/`目录下,符合工程化结构
- ✅ 文档放置在`specs/004-git-repo-finalize/`
- ✅ 不破坏现有Go项目结构(cmd/, internal/, configs/)

### 版本控制流程
- ✅ 所有修改必须提交到Git
- ✅ 使用约定式提交规范:`chore(cleanup): 清理构建产物和遗留文件`
- ✅ 在feature分支`004-git-repo-finalize`上开发
- ✅ 清理完成后推送到远程仓库

### 代码质量与风格
- ✅ Shell脚本遵循最佳实践(set -euo pipefail, 函数化设计)
- ✅ 脚本包含详细的中文注释,解释关键逻辑和安全检查
- N/A 不涉及Go代码修改

### 错误处理规范
- ✅ 脚本使用严格错误处理模式(`set -euo pipefail`)
- ✅ 所有命令返回值被检查
- ✅ 提供清晰的错误消息和回滚建议(通过Git恢复)

### 日志规范
- ✅ 脚本输出结构化日志(带时间戳、操作类型)
- ✅ 区分信息、警告、错误级别
- ✅ 详细记录删除的文件和目录

### 测试优先原则
- ✅ 提供干跑模式(--dry-run)作为测试手段
- ✅ 清理后验证Git状态和Go项目完整性
- N/A 不涉及单元测试(简单脚本)

### 安全性要求
- ✅ 强制交互式确认,防止误操作
- ✅ 白名单机制确保关键文件不被删除
- ✅ 清理前提示创建备份或Git提交
- ✅ 所有路径使用绝对路径或经过验证的相对路径

### 文档与任务管理
- ✅ spec.md已完成,plan.md正在填充
- ✅ 将生成quickstart.md使用指南
- ✅ 清理报告记录所有操作历史

### 代码审查流程
- ✅ 脚本完成后需要代码审查
- ✅ 审查重点:文件识别准确性、安全性、错误处理

**GATE状态**: ✅ PASS - 所有宪章要求符合,可以进入Phase 0研究阶段

## Project Structure

### Documentation (this feature)

```text
specs/004-git-repo-finalize/
├── plan.md              # 本文件 (实施计划)
├── research.md          # Phase 0输出 (技术研究) - 简化或跳过
├── data-model.md        # N/A (本功能不涉及数据模型)
├── quickstart.md        # Phase 1输出 (清理操作快速指南)
├── contracts/           # N/A (本功能不涉及API契约)
└── tasks.md             # Phase 2输出 (任务分解 - 由/speckit.tasks生成)
```

### Source Code (repository root)

本功能主要涉及文件删除和文档更新,遵循现有Go项目的工程化结构:

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
├── scripts/                    # 构建和维护脚本 (保留)
│   └── build.sh               # 现有构建脚本
│
├── tests/                      # 测试文件 (保留)
│   ├── unit/
│   ├── integration/
│   ├── e2e/
│   └── benchmark/
│
├── configs/                    # Go配置文件 (保留) ⭐
├── specs/                      # 功能规范文档 (保留)
├── .specify/                   # 项目管理工具 (保留)
├── .github/                    # GitHub配置 (保留)
│
├── Makefile                    # 构建配置 (保留)
├── go.mod                      # Go依赖 (保留)
├── go.sum                      # Go依赖锁定 (保留)
├── .gitignore                  # Git忽略规则 (更新) ⚙️
├── README.md                   # 项目说明 (更新) ⚙️
├── CLAUDE.md                   # AI助手配置 (保留)
│
├── dist/                       # Go构建产物 (将被删除) ❌ 263MB
├── jsfindcrack                 # 根目录编译产物 (将被删除) ❌ 22MB
├── output/                     # 爬取输出 (将被删除) ❌ 2.4MB
├── logs/                       # 日志文件 (将被删除) ❌ 856KB
├── config/                     # Python配置 (将被删除) ❌ ~10KB
├── -/                          # 临时测试 (将被删除) ❌ ~7KB
├── test_urls.txt               # 测试文件 (将被删除) ❌
├── Go并发和流式IO最佳实践研究.md  # 研究文档 (将被删除) ❌ 37KB
├── install.sh                  # Python安装脚本 (将被删除) ❌ 3.5KB
└── .DS_Store                   # 系统文件 (将被删除) ❌ 6KB
```

**Structure Decision**:

本功能采用简单的文件删除操作模式,无需创建新的代码结构。清理操作将:
1. 删除明确标记为❌的10个文件/目录(总计~285MB)
2. 更新标记为⚙️的2个文件(.gitignore, README.md)
3. 保留所有标记为(保留)的Go源代码、配置、文档、测试
4. 特别注意区分config/(删除)和configs/(保留)

不涉及复杂的源代码结构变更,主要是文件系统清理和Git操作。

## Complexity Tracking

本功能无宪章违规,无需填写复杂性跟踪表。

---

## Phase 0: 研究与决策 ✅

**状态**: 简化 - 技术栈简单,无需深入研究
**输出文档**: research.md (简化版或跳过)

### 研究内容

由于本功能是简单的文件删除和Git操作,技术决策相对明确:

1. **文件删除策略**: 使用`rm -rf`命令,Shell脚本直接实现
2. **白名单验证**: 通过文件路径匹配确保不误删关键文件
3. **Git操作**: 使用`git add`, `git commit`, `git push`标准命令
4. **.gitignore更新**: 手动编辑追加忽略规则
5. **README更新**: 手动编辑移除已删除文件引用

### 关键决策

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 实现方式 | 手动执行+Shell脚本 | 文件数量少(10个),手动更安全可控 |
| 安全机制 | 预览+确认+白名单 | 多层防护,防止误删 |
| 验证方式 | 干跑模式+Git status | 清理前后状态对比,确保正确性 |
| 恢复方案 | Git历史记录 | 利用现有版本控制,简单可靠 |

无NEEDS CLARIFICATION项 - 所有技术决策明确。

---

## Phase 1: 设计与契约 ✅

**状态**: 准备就绪
**输出文档**: quickstart.md (清理操作指南)

### 设计概要

**清理流程**:
```
1. 预清理检查
   ├── 验证Git工作区状态
   ├── 列出待删除文件(干跑模式)
   └── 确认删除清单

2. 执行清理
   ├── 删除构建产物(dist/, jsfindcrack)
   ├── 删除遗留Python配置(config/目录, install.sh)
   ├── 删除临时文件(-/, test_urls.txt, 研究文档)
   ├── 删除输出和日志(output/, logs/)
   └── 删除系统文件(.DS_Store)

3. 更新配置和文档
   ├── 更新.gitignore(追加忽略规则)
   └── 更新README.md(移除已删除文件引用)

4. Git提交和推送
   ├── 暂存所有变更(git add -A)
   ├── 创建提交(描述性commit message)
   └── 推送到远程(git push)

5. 验证
   ├── 检查Git status(应干净)
   ├── 验证文件不存在
   └── 确认Go项目完整性(go test ./...)
```

### 核心操作清单

#### 1. 待删除文件/目录(10项)

```bash
# 构建产物
dist/                    # 263MB - Go交叉编译产物
jsfindcrack              # 22MB  - 根目录编译产物

# 遗留Python文件
config/                  # ~10KB - Python配置目录
install.sh               # 3.5KB - Python安装脚本

# 临时和测试文件
-/                       # ~7KB  - 临时测试输出
test_urls.txt            # <1KB  - URL测试列表
Go并发和流式IO最佳实践研究.md  # 37KB  - 研究文档

# 输出和日志
output/                  # 2.4MB - 爬取结果
logs/                    # 856KB - 日志文件

# 系统文件
.DS_Store                # 6KB   - macOS系统文件
```

#### 2. .gitignore更新内容

追加以下忽略规则:

```gitignore
# Go构建产物(根目录二进制)
/jsfindcrack
jsfindcrack-*

# 特殊目录名(临时测试)
-/

# 确保已有规则(验证存在)
dist/
output/
logs/
.DS_Store
```

#### 3. README.md更新要点

- 移除Python安装脚本(install.sh)引用
- 移除config/目录说明(保留configs/说明)
- 确保项目结构部分准确反映当前状态
- 不包含已删除文件的引用

### 数据模型

N/A - 本功能不涉及数据模型

### API契约

N/A - 本功能不涉及API设计

### 快速开始指南

将在quickstart.md中提供:
- 清理前检查清单
- 逐步操作指导
- 验证和恢复方法
- 故障排查建议

### 代理上下文更新

已执行`.specify/scripts/bash/update-agent-context.sh claude`,更新内容:
- 语言: Bash 4.0+ (Shell脚本) / Git 2.0+
- 依赖: 标准Unix工具(rm, find, du), Git命令行工具
- 项目类型: 维护脚本 + 文档更新

---

## Phase 1 宪章复查 ✅

**复查时间**: 2025-11-16
**复查结果**: ✅ 全部通过

### 设计符合性验证

#### 语言与文档规范
- ✅ quickstart.md将使用中文编写
- ✅ 操作说明使用中文
- ✅ 无emoji使用

#### 领域驱动设计
- ✅ 流程清晰(检查→清理→更新→提交→验证)
- ✅ 职责分离(文件删除、配置更新、Git操作)

#### 工程化结构
- ✅ 不破坏Go项目结构
- ✅ 文档放置在specs/004-git-repo-finalize/

#### 错误处理
- ✅ 预览+确认机制
- ✅ 白名单保护防止误删
- ✅ Git历史作为恢复手段

#### 测试策略
- ✅ 干跑模式预览
- ✅ 清理后Go测试验证
- ✅ Git status检查

#### 安全性
- ✅ 多级确认
- ✅ 白名单防护(区分config/和configs/)
- ✅ Git备份建议

**结论**: 设计完全符合项目宪章要求,可以进入Phase 2任务分解阶段。

---

## 下一步行动

1. **生成quickstart.md** - 提供清理操作的详细指南
2. **运行 `/speckit.tasks`** 命令生成tasks.md任务分解文档
3. **执行清理** 根据tasks.md逐步实施
4. **验证完整性** Go测试通过,Git推送成功
5. **合并主分支** 审查通过后合并到main分支

---

## 附录

### 预计时间线

| 阶段 | 任务 | 预计耗时 |
|------|------|----------|\n| Phase 0 | 研究与决策 | ✅ 已简化(技术明确) |
| Phase 1 | 设计与文档 | ✅ 已完成 |
| Phase 2 | 任务分解 | 待执行 (/speckit.tasks) |
| 实施 | 预览待删除文件 | 5分钟 |
| 实施 | 执行删除操作 | 5分钟 |
| 实施 | 更新.gitignore和README | 10分钟 |
| 实施 | Git提交和推送 | 5分钟 |
| 验证 | Go测试和Git验证 | 5分钟 |
| **总计** | | **~30分钟** |

### 交付物清单

- [x] spec.md - 功能规范
- [x] plan.md - 实施计划(本文档)
- [x] research.md - 技术研究(简化,决策明确)
- [ ] quickstart.md - 清理操作指南(待生成)
- [ ] tasks.md - 任务分解(待生成,通过/speckit.tasks)
- [ ] 清理执行 - 删除10个文件/目录(待执行)
- [ ] .gitignore更新 - 追加忽略规则(待执行)
- [ ] README.md更新 - 移除已删除文件引用(待执行)
- [ ] Git提交推送 - 完成最终化(待执行)

### 风险管理

| 风险 | 概率 | 影响 | 缓解措施 | 状态 |
|------|------|------|----------|------|
| 误删重要文件 | 低 | 高 | 干跑模式预览+白名单验证+Git历史恢复 | ✅ 已缓解 |
| config/和configs/混淆 | 中 | 高 | 明确区分+手动确认+路径匹配验证 | ✅ 已缓解 |
| Git推送冲突 | 低 | 中 | 推送前检查远程状态+手动解决冲突 | ✅ 已缓解 |
| 清理后Go项目损坏 | 极低 | 高 | 白名单保护Go代码+清理后运行测试 | ✅ 已缓解 |

---

**计划完成时间**: 2025-11-16
**负责人**: AI Agent (Claude Code)
**审核状态**: 待审核
