# Feature Specification: Git仓库清理和最终化

**Feature Branch**: `004-git-repo-finalize`
**Created**: 2025-11-16
**Status**: Draft
**Input**: User description: "删除所有不需要上传git的文件，让最后编辑好.gitignore文件和README文件，最后上传git"

## Clarifications

### Session 2025-11-16

- Q: `-/` 目录(包含测试输出deobfuscated.js)应该如何处理? → A: 删除 - 移除整个`-/`目录及其内容
- Q: 基于代码库全面排查,以下文件/目录是否需要删除? → A: 已识别需要删除的文件和目录:
  - `-/` 目录(临时测试输出,含deobfuscated.js) - 删除
  - `config/` 目录(Python版本配置,含.json/.bak文件和中文说明) - 删除
  - `jsfindcrack` 可执行文件(Go编译产物,22MB) - 删除
  - `test_urls.txt` (测试文件) - 删除
  - `Go并发和流式IO最佳实践研究.md` (研究文档) - 删除
  - `install.sh` (Python安装脚本) - 删除
  - `CLAUDE.md` (AI助手配置) - 保留(项目开发配置)
  - `configs/` 目录 - 保留(Go版本配置)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 清理构建产物和临时文件 (Priority: P1) 🎯 MVP

作为项目维护者,在完成Go版本开发和Python文件清理后,我需要删除所有构建产物、临时文件和不需要版本控制的文件,以确保仓库只包含必要的源代码和文档,减小仓库大小,提高克隆速度。

**Why this priority**: 这是核心清理需求,直接实现用户描述的主要目标。删除不必要的文件可以减小仓库大小(当前dist/目录就有263MB),避免将二进制文件和临时文件提交到版本控制系统。这是仓库最终化的第一步,也是最重要的一步。

**Independent Test**: 可以通过验证所有构建产物(dist/目录)、输出文件(output/目录)、日志文件(logs/目录)、系统临时文件(.DS_Store)、遗留Python配置(config/目录)、测试文件已被删除来独立测试,验证后仓库体积显著减小。

**Acceptance Scenarios**:

1. **Given** 项目中存在构建产物目录(dist/,包含263MB二进制文件), **When** 执行清理操作, **Then** dist/目录被完全删除,仓库体积减小263MB
2. **Given** 项目中存在输出目录(output/,包含2.4MB爬取结果), **When** 执行清理操作, **Then** output/目录被删除,同时.gitignore已配置忽略此目录
3. **Given** 项目中存在日志目录(logs/,包含856KB日志文件), **When** 执行清理操作, **Then** logs/目录被删除,同时.gitignore已配置忽略此目录
4. **Given** 项目中存在系统临时文件(.DS_Store), **When** 执行清理操作, **Then** 所有.DS_Store文件被删除,同时.gitignore已配置忽略此类文件
5. **Given** 项目中存在根目录编译产物(jsfindcrack可执行文件,22MB), **When** 执行清理操作, **Then** 该文件被删除,同时.gitignore已配置忽略
6. **Given** 项目中存在遗留Python配置(config/目录,含.json和.bak文件), **When** 执行清理操作, **Then** config/目录被完全删除
7. **Given** 项目中存在临时测试目录(-/目录), **When** 执行清理操作, **Then** -/目录及其内容被删除
8. **Given** 项目中存在测试和研究文件(test_urls.txt, Go并发研究.md), **When** 执行清理操作, **Then** 这些文件被删除
9. **Given** 项目中存在Python安装脚本(install.sh), **When** 执行清理操作, **Then** 该脚本被删除
10. **Given** 清理完成后, **When** 运行`git status`, **Then** 仅显示必要的源代码和文档变更,不包含任何临时文件

---

### User Story 2 - 完善.gitignore配置 (Priority: P1) 🎯 MVP

作为项目维护者,我需要完善.gitignore配置文件,确保所有不应纳入版本控制的文件类型都被正确忽略,防止将来误提交不必要的文件。

**Why this priority**: 这是预防性措施,与清理操作同等重要。正确配置.gitignore可以防止将来误提交构建产物、临时文件、IDE配置等,是保持仓库干净的长期保障。这是用户明确要求的功能("让最后编辑好.gitignore文件")。

**Independent Test**: 可以通过验证.gitignore文件包含所有必要的忽略规则(Go构建产物、输出目录、日志、系统文件、IDE配置等),并测试创建这些类型文件后`git status`不显示它们来独立验证。

**Acceptance Scenarios**:

1. **Given** .gitignore需要配置Go构建产物, **When** 编辑.gitignore, **Then** 文件包含dist/、bin/、build/、*.exe、*.test、jsfindcrack(根目录二进制)等规则
2. **Given** .gitignore需要配置输出和日志目录, **When** 编辑.gitignore, **Then** 文件包含output/、logs/、*.log等规则
3. **Given** .gitignore需要配置系统临时文件, **When** 编辑.gitignore, **Then** 文件包含.DS_Store、*.swp、*.tmp、-/(特殊目录名)等规则
4. **Given** .gitignore已更新, **When** 创建测试文件(如dist/test.exe、output/test.js、.DS_Store、jsfindcrack), **Then** `git status`不显示这些文件
5. **Given** .gitignore已更新, **When** 执行清理后的Git提交, **Then** 不会意外包含任何应忽略的文件

---

### User Story 3 - 更新README文档 (Priority: P2)

作为项目维护者,我需要更新README文档,确保它准确反映项目当前状态(Go版本已完成,Python版本已清理),提供清晰的使用说明和项目结构描述。

**Why this priority**: 这是文档完善需求,优先级次于清理和.gitignore配置。虽然当前README已经比较完善(包含Go版本说明),但需要更新项目结构部分,移除已删除Python文件的引用,确保文档与实际代码库状态一致。

**Independent Test**: 可以通过验证README中的项目结构、使用示例、快速开始指南与实际代码库状态一致,没有引用已删除的Python文件或目录来独立测试。

**Acceptance Scenarios**:

1. **Given** README包含项目结构说明, **When** 更新README, **Then** 项目结构部分不包含已删除的文件(main.py、src/目录、requirements.txt、config/目录、install.sh)
2. **Given** README包含快速开始指南, **When** 更新README, **Then** 快速开始部分仅包含Go版本的安装和使用说明,移除Python安装脚本引用
3. **Given** README包含功能特性说明, **When** 更新README, **Then** 所有功能特性都基于Go版本,准确反映当前实现状态
4. **Given** README需要反映仓库清理, **When** 更新README, **Then** 添加仓库清理说明,引导用户正确处理构建产物和输出文件
5. **Given** README已更新, **When** 新用户阅读文档, **Then** 能够清晰理解项目是Go实现,快速上手使用

---

### User Story 4 - 提交并推送到Git仓库 (Priority: P1) 🎯 MVP

作为项目维护者,我需要将所有清理后的变更(删除文件、更新.gitignore、更新README)提交到Git仓库并推送到远程,完成仓库最终化流程。

**Why this priority**: 这是整个流程的最终步骤,与前面的清理操作同等重要。用户明确要求"最后上传git",这个步骤确保所有变更被正确记录和同步,完成仓库从Python到Go的完整迁移。

**Independent Test**: 可以通过验证所有变更被正确提交(包含描述性commit message),远程仓库状态与本地一致,仓库历史记录清晰来独立测试。

**Acceptance Scenarios**:

1. **Given** 所有清理操作已完成, **When** 执行Git提交, **Then** 提交包含所有删除的文件、更新的.gitignore和README,commit message清晰描述变更内容
2. **Given** .gitignore已更新, **When** 执行Git提交, **Then** 不会意外暂存任何应忽略的文件(如.DS_Store、dist/中的残留文件)
3. **Given** 本地提交已完成, **When** 推送到远程仓库, **Then** 推送成功,远程仓库与本地状态一致
4. **Given** 推送已完成, **When** 在新环境克隆仓库, **Then** 克隆速度快,仓库体积小,不包含构建产物和临时文件
5. **Given** 整个流程完成后, **When** 查看Git历史, **Then** 可以清晰看到Python到Go迁移的完整过程(迁移→清理→最终化)

---

### Edge Cases

- 如果dist/目录中包含用户手动创建的重要文件,如何避免误删?
- 如果某些日志文件对调试很重要,是否应该保留部分日志?
- 如果output/目录中包含示例输出用于文档说明,如何处理?
- 清理操作是否应该强制要求Git工作区干净,还是允许合并到现有变更中?
- 如果.gitignore更新后仍有已跟踪的文件需要从Git索引中移除,如何处理?
- 是否需要在清理前创建Git标签或备份分支,以防需要回滚?
- 推送到远程时如果遇到冲突,如何处理?
- 是否需要同时更新所有feature分支,还是只处理main分支?
- 如果configs/目录中包含敏感配置(如headers.yaml),应该如何保护?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须能够识别并删除所有Go构建产物目录(dist/、bin/、build/)
- **FR-002**: 系统必须能够删除所有输出目录(output/)和日志目录(logs/)
- **FR-003**: 系统必须能够识别并删除所有系统临时文件(.DS_Store、*.swp、*.tmp、*.bak)
- **FR-004**: 系统必须能够删除所有Go测试覆盖率文件(coverage.out、*.coverprofile)
- **FR-005**: 系统必须能够删除根目录下的Go编译产物(jsfindcrack可执行文件)
- **FR-006**: 系统必须能够删除遗留Python配置目录(config/目录及其所有.json、.bak文件)
- **FR-007**: 系统必须能够删除临时测试目录(-/目录及其内容)
- **FR-008**: 系统必须能够删除测试文件(test_urls.txt)和研究文档(Go并发和流式IO最佳实践研究.md)
- **FR-009**: 系统必须能够删除Python安装脚本(install.sh)
- **FR-010**: 系统必须在删除前提供预览功能,显示待删除文件列表和总大小
- **FR-011**: 系统必须在.gitignore中配置Go构建产物忽略规则(dist/、bin/、*.exe、*.test、jsfindcrack等)
- **FR-012**: 系统必须在.gitignore中配置输出和日志目录忽略规则(output/、logs/、*.log)
- **FR-013**: 系统必须在.gitignore中配置系统临时文件忽略规则(.DS_Store、*.swp、*.tmp、-/)
- **FR-014**: 系统必须在.gitignore中配置IDE配置文件忽略规则(.vscode/、.idea/、*.swp)
- **FR-015**: 系统必须验证.gitignore配置有效性(测试文件创建后不被Git跟踪)
- **FR-016**: 系统必须更新README移除已删除文件的引用(install.sh、config/目录、Python相关说明)
- **FR-017**: 系统必须在README中添加仓库清理说明和构建产物管理指导
- **FR-018**: 系统必须创建描述性Git提交,清晰记录清理内容和原因
- **FR-019**: 系统必须验证推送前Git工作区状态,确保不包含未提交的关键变更
- **FR-020**: 系统必须在推送前确认远程仓库连接状态,避免推送失败

### Key Entities

- **构建产物**: dist/、bin/、build/目录及其内容,根目录jsfindcrack可执行文件(22MB),*.exe、*.test、*.out等二进制文件
- **输出文件**: output/目录(爬取结果,2.4MB)、logs/目录(日志文件,856KB)、*.log文件
- **系统临时文件**: .DS_Store(macOS)、*.swp/*.swo(Vim)、*.tmp/*.bak(临时备份)、Thumbs.db(Windows)、-/目录(测试输出)
- **遗留Python文件**: config/目录(含.json、.bak、配置说明.md,共约10KB)、install.sh(Python安装脚本)
- **测试和研究文件**: test_urls.txt、Go并发和流式IO最佳实践研究.md
- **保留配置**: configs/目录(Go版本配置,含config.yaml、headers.yaml)、CLAUDE.md(AI助手配置)
- **Git配置**: .gitignore文件,包含所有忽略规则
- **项目文档**: README.md,包含项目说明、使用指南、项目结构
- **Git提交记录**: 包含删除文件、.gitignore更新、README更新的原子提交

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 清理后仓库体积减少至少285MB(263MB dist/ + 22MB jsfindcrack + 其他临时文件)
- **SC-002**: 清理后`git status`仅显示必要的源代码和文档变更,不包含任何临时文件或构建产物
- **SC-003**: .gitignore配置完整性达到100%(创建测试文件后`git status`不显示应忽略的文件类型)
- **SC-004**: README文档准确性100%(不包含任何已删除文件的引用,项目结构描述与实际一致)
- **SC-005**: Git推送成功率100%(所有变更成功推送到远程仓库,无冲突或错误)
- **SC-006**: 清理操作可在1分钟内完成(包括文件扫描、预览、删除、提交)
- **SC-007**: 新环境克隆仓库速度提升30%以上(由于仓库体积减小)
- **SC-008**: Git历史清晰度100%(commit message准确描述变更,易于追溯)
- **SC-009**: 清理后删除文件数量≥10个(包括目录、二进制文件、测试文件、配置文件)

## Assumptions

- 假设dist/、output/、logs/、config/、-/目录中的所有内容都是可再生的,可以安全删除
- 假设用户不需要保留任何构建产物、历史日志文件或Python遗留配置
- 假设当前.gitignore配置基本完善,只需要补充和验证
- 假设README.md已经包含Go版本的基本说明,只需要微调和移除已删除文件引用
- 假设项目使用Git进行版本控制,且有权限推送到远程仓库
- 假设清理操作在feature分支上执行,完成后可以合并到main分支
- 假设不需要处理其他feature分支(001、002、003),只处理当前分支
- 假设configs/目录包含Go版本的有效配置,需要保留
- 假设CLAUDE.md是项目开发配置,需要保留

## Constraints

- 必须确保清理操作不删除任何源代码、测试文件、Go配置文件(configs/)、文档
- 必须确保.gitignore配置不会误忽略重要的配置文件或测试资源
- 必须区分Python遗留配置(config/)和Go配置(configs/),只删除前者
- 清理操作应该是可逆的(通过Git历史或手动重建)
- 提交message必须遵循项目的约定式提交规范
- 推送操作必须在确认所有变更正确的情况下执行

## Dependencies

- 依赖于001-py-to-go-migration功能已完成(Go版本代码可用)
- 依赖于003-cleanup-legacy-files功能已完成(Python文件已清理)
- 依赖于Git版本控制系统和远程仓库访问权限
- 依赖于用户对清理范围和.gitignore配置的确认
