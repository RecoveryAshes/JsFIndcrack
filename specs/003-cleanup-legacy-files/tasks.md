# Tasks: 清理遗留Python文件

**Input**: 设计文档来自 `/specs/003-cleanup-legacy-files/`
**Prerequisites**: plan.md (已有), spec.md (已有), research.md (已有), quickstart.md (已有)

**Tests**: 本功能包含Shell脚本单元测试,用于验证文件识别逻辑的正确性和安全性。

**Organization**: 任务按用户故事分组,实现每个故事的独立实施和测试。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行(不同文件,无依赖)
- **[Story]**: 任务所属用户故事(US1, US2, US3)
- 描述中包含准确的文件路径

## Path Conventions

本项目为单一Go项目结构:
- Shell脚本: `scripts/`
- Shell测试: `tests/unit/`
- 报告输出: `specs/003-cleanup-legacy-files/reports/`
- 文档: `specs/003-cleanup-legacy-files/`

---

## Phase 1: Setup (项目初始化)

**Purpose**: 创建脚本框架和报告目录结构

- [X] T001 创建清理脚本文件 `scripts/cleanup-python.sh` 并设置执行权限
- [X] T002 在脚本中添加Shebang和基本错误处理 (`#!/usr/bin/env bash`, `set -euo pipefail`)
- [X] T003 [P] 创建报告目录 `specs/003-cleanup-legacy-files/reports/` (如不存在)
- [X] T004 [P] 在 `scripts/cleanup-python.sh` 中添加版本信息和帮助文档函数

**验证**: 脚本文件可执行,运行`--help`显示使用说明

---

## Phase 2: Foundational (基础功能 - 所有用户故事的前置依赖)

**Purpose**: 实现脚本的核心基础设施,包括参数解析、初始化检查、常量定义

**⚠️ CRITICAL**: 此阶段必须完成后才能开始任何用户故事的实现

### 基础设施任务

- [X] T005 实现命令行参数解析逻辑 (--dry-run, --preview, --execute, --force, --list-only) 在 `scripts/cleanup-python.sh`
- [X] T006 [P] 实现日志函数 (log_info, log_warn, log_error) 带时间戳和级别标识 在 `scripts/cleanup-python.sh`
- [X] T007 [P] 定义白名单常量 (WHITELIST_DIRS, WHITELIST_FILES) 在 `scripts/cleanup-python.sh`
- [X] T008 实现初始化检查函数 `check_prerequisites()`: 验证Git仓库、工作目录、权限 在 `scripts/cleanup-python.sh`
- [X] T009 实现Git状态检查函数 `check_git_status()`: 确保工作区干净 在 `scripts/cleanup-python.sh`
- [X] T010 [P] 创建临时文件管理机制: 用于存储待删除文件列表 在 `scripts/cleanup-python.sh`

**Checkpoint**: 基础功能就绪 - 用户故事实施现在可以并行开始

---

## Phase 3: User Story 1 - 清理Python源代码文件 (Priority: P1) 🎯 MVP

**Goal**: 识别、验证和删除所有Python源文件(.py)、src/目录和requirements.txt,同时确保Go代码和关键配置不被误删

**Independent Test**:
1. 运行脚本 `--dry-run` 模式,验证列出所有.py文件和src/目录
2. 验证白名单文件(go.mod, Makefile, cmd/, internal/)不在删除列表中
3. 执行删除后,确认所有.py文件和src/目录不存在
4. 运行 `go test ./...` 确认Go功能完整

### Tests for User Story 1

> **NOTE: 先编写测试,确保测试失败后再实现功能**

- [ ] T011 [P] [US1] 创建Shell测试文件 `tests/unit/cleanup-python.bats` 并配置bats-core框架
- [ ] T012 [P] [US1] 编写测试: 验证find_python_source_files()能识别所有.py文件 在 `tests/unit/cleanup-python.bats`
- [ ] T013 [P] [US1] 编写测试: 验证find_python_config_files()能识别requirements.txt 在 `tests/unit/cleanup-python.bats`
- [ ] T014 [P] [US1] 编写测试: 验证find_python_directories()能识别src/目录 在 `tests/unit/cleanup-python.bats`
- [ ] T015 [P] [US1] 编写测试: 验证白名单验证不包含go.mod, Makefile等 在 `tests/unit/cleanup-python.bats`
- [ ] T016 [P] [US1] 编写测试: 验证干跑模式不删除任何文件 在 `tests/unit/cleanup-python.bats`

### Implementation for User Story 1

- [ ] T017 [P] [US1] 实现文件扫描函数 `find_python_source_files()`: 使用find查找所有.py文件 在 `scripts/cleanup-python.sh`
- [ ] T018 [P] [US1] 实现配置扫描函数 `find_python_config_files()`: 查找requirements.txt, setup.py 在 `scripts/cleanup-python.sh`
- [ ] T019 [P] [US1] 实现目录识别函数 `find_python_directories()`: 识别src/及其子目录 在 `scripts/cleanup-python.sh`
- [ ] T020 [US1] 实现白名单验证函数 `validate_against_whitelist()`: 检查待删除列表不包含白名单文件 在 `scripts/cleanup-python.sh`
- [ ] T021 [US1] 实现文件分类汇总函数 `categorize_files()`: 按类型(源文件、配置、目录)分组 在 `scripts/cleanup-python.sh`
- [ ] T022 [US1] 实现干跑模式显示逻辑: 格式化输出待删除文件清单和统计 在 `scripts/cleanup-python.sh`
- [ ] T023 [US1] 实现文件删除函数 `delete_python_files()`: 先删除文件,后删除目录 在 `scripts/cleanup-python.sh`
- [ ] T024 [US1] 添加删除操作的详细日志记录和错误处理 在 `scripts/cleanup-python.sh`

**Checkpoint**: 用户故事1完成 - 可以识别、验证和删除Python源文件,同时保护Go代码

---

## Phase 4: User Story 2 - 清理Python构建产物 (Priority: P2)

**Goal**: 识别和删除Python构建产物,包括__pycache__目录、.pyc/.pyo文件、.egg-info目录

**Independent Test**:
1. 创建测试用的__pycache__目录和.pyc文件
2. 运行脚本 `--dry-run`,验证列出所有构建产物
3. 执行删除后,确认所有构建产物不存在
4. 验证不影响Go构建产物(dist/目录保留)

### Tests for User Story 2

- [ ] T025 [P] [US2] 编写测试: 验证find_python_build_artifacts()能识别__pycache__目录 在 `tests/unit/cleanup-python.bats`
- [ ] T026 [P] [US2] 编写测试: 验证能识别.pyc和.pyo文件 在 `tests/unit/cleanup-python.bats`
- [ ] T027 [P] [US2] 编写测试: 验证能识别.egg-info目录 在 `tests/unit/cleanup-python.bats`
- [ ] T028 [P] [US2] 编写测试: 验证不会误删Go构建产物(dist/中的Go binary) 在 `tests/unit/cleanup-python.bats`

### Implementation for User Story 2

- [ ] T029 [P] [US2] 实现构建产物扫描函数 `find_python_build_artifacts()`: 查找__pycache__, .pyc, .pyo 在 `scripts/cleanup-python.sh`
- [ ] T030 [P] [US2] 扩展`find_python_build_artifacts()`支持.egg-info目录识别 在 `scripts/cleanup-python.sh`
- [ ] T031 [US2] 将构建产物集成到`categorize_files()`的分类逻辑中 在 `scripts/cleanup-python.sh`
- [ ] T032 [US2] 更新白名单验证确保dist/目录中的Go binary不被删除 在 `scripts/cleanup-python.sh`
- [ ] T033 [US2] 更新删除函数`delete_python_files()`支持递归删除缓存目录 在 `scripts/cleanup-python.sh`

**Checkpoint**: 用户故事1和2完成 - Python源文件和构建产物都能正确清理

---

## Phase 5: User Story 3 - 保留必要的文档和配置 (Priority: P1)

**Goal**: 实现严格的白名单验证机制,确保Go代码、配置文件、文档、测试资源绝对不被误删

**Independent Test**:
1. 运行脚本 `--dry-run`,确认白名单验证输出中显示所有关键文件被保留
2. 创建模拟的误删场景(如.go文件被加入删除列表),验证脚本报错退出
3. 执行完整清理后,验证go.mod, Makefile, cmd/, internal/, tests/, specs/全部存在
4. 运行 `make build` 和 `go test ./...` 验证项目完整性

### Tests for User Story 3

- [ ] T034 [P] [US3] 编写测试: 验证白名单包含所有关键目录(cmd, internal, tests, specs等) 在 `tests/unit/cleanup-python.bats`
- [ ] T035 [P] [US3] 编写测试: 验证白名单包含所有关键文件(go.mod, Makefile, .gitignore) 在 `tests/unit/cleanup-python.bats`
- [ ] T036 [P] [US3] 编写测试: 验证如果待删除列表包含白名单文件则脚本退出并报错 在 `tests/unit/cleanup-python.bats`
- [ ] T037 [P] [US3] 编写测试: 验证.md文件(除README外)被正确保留 在 `tests/unit/cleanup-python.bats`

### Implementation for User Story 3

- [ ] T038 [P] [US3] 扩展白名单常量定义,确保覆盖所有关键路径 在 `scripts/cleanup-python.sh`
- [ ] T039 [P] [US3] 实现白名单冲突检测函数 `detect_whitelist_conflicts()`: 返回冲突文件列表 在 `scripts/cleanup-python.sh`
- [ ] T040 [US3] 增强`validate_against_whitelist()`在发现冲突时立即错误退出 在 `scripts/cleanup-python.sh`
- [ ] T041 [US3] 实现白名单验证报告功能: 显示被保护的关键文件状态 在 `scripts/cleanup-python.sh`
- [ ] T042 [US3] 在干跑和预览模式中显示白名单验证结果 在 `scripts/cleanup-python.sh`

**Checkpoint**: 所有3个用户故事完成 - 清理安全可靠,关键文件受保护

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 交互确认、报告生成、文档完善等跨故事功能

### 交互确认功能

- [ ] T043 [P] 实现预览模式 `--preview`: 显示文件详细信息(大小、修改时间) 在 `scripts/cleanup-python.sh`
- [ ] T044 [P] 实现执行确认流程: 显示警告、要求输入"yes" 在 `scripts/cleanup-python.sh`
- [ ] T045 [P] 实现强制模式 `--force`: 跳过确认(添加警告日志) 在 `scripts/cleanup-python.sh`
- [ ] T046 [P] 实现列表模式 `--list-only`: 仅输出文件路径(用于管道) 在 `scripts/cleanup-python.sh`

### 报告生成功能

- [ ] T047 [P] 实现JSON报告生成函数 `generate_json_report()`: 输出到specs/003-cleanup-legacy-files/reports/ 在 `scripts/cleanup-python.sh`
- [ ] T048 [P] 实现Markdown摘要生成函数 `generate_markdown_summary()`: 人类可读格式 在 `scripts/cleanup-python.sh`
- [ ] T049 实现Git状态快照功能: 记录清理前后的Git状态 在 `scripts/cleanup-python.sh`
- [ ] T050 在报告中包含文件数量、大小统计、白名单验证结果 在 `scripts/cleanup-python.sh`

### 错误处理和回滚

- [ ] T051 [P] 实现Git备份点建议功能: 清理前提示创建Git标签 在 `scripts/cleanup-python.sh`
- [ ] T052 [P] 添加详细的错误消息和回滚指导 在 `scripts/cleanup-python.sh`
- [ ] T053 实现清理失败时的部分回滚逻辑(如果可能) 在 `scripts/cleanup-python.sh`

### 文档和最终验证

- [ ] T054 [P] 更新 `specs/003-cleanup-legacy-files/quickstart.md` 中的实际脚本命令示例
- [ ] T055 [P] 创建示例报告文件展示JSON和Markdown格式 在 `specs/003-cleanup-legacy-files/reports/`
- [ ] T056 运行ShellCheck静态分析工具检查脚本质量
- [ ] T057 执行所有Shell单元测试 `bats tests/unit/cleanup-python.bats`
- [ ] T058 执行干跑模式测试: `./scripts/cleanup-python.sh --dry-run` 并验证输出
- [ ] T059 在测试环境执行完整清理流程并验证
- [ ] T060 清理后运行Go测试套件: `go test -v ./...` 确认100%通过
- [ ] T061 验证清理后Go项目可正常构建: `make build` 或 `go build ./cmd/jsfindcrack`

**Checkpoint**: 所有功能完成,脚本经过完整测试和验证

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **Foundational (Phase 2)**: 依赖Setup完成 - **阻塞**所有用户故事
- **User Stories (Phase 3-5)**: 全部依赖Foundational完成
  - 用户故事可并行进行(如有多人)
  - 或按优先级顺序: P1 (US1, US3) → P2 (US2)
- **Polish (Phase 6)**: 依赖所有需要的用户故事完成

### User Story Dependencies

- **User Story 1 (P1)**: Foundational完成后可开始 - 无其他依赖
- **User Story 2 (P2)**: Foundational完成后可开始 - 扩展US1的删除功能
- **User Story 3 (P1)**: Foundational完成后可开始 - 增强US1的验证逻辑

**注意**: US1和US3都是P1优先级,可以并行开发,或先做US3(安全优先),再做US1

### Within Each User Story

- Tests必须先编写并**失败** → 实现功能 → Tests通过
- 扫描函数 → 验证函数 → 删除函数
- 单元测试 → 集成测试 → 完整流程测试

### Parallel Opportunities

**Setup阶段并行**:
- T003 (创建报告目录) || T004 (添加帮助文档)

**Foundational阶段并行**:
- T006 (日志函数) || T007 (白名单常量) || T010 (临时文件管理)

**User Story 1测试并行**:
- T012, T013, T014, T015, T016 可同时编写(不同测试用例)

**User Story 1实现并行**:
- T017, T018, T019 可同时实现(不同扫描函数)

**User Story 2测试并行**:
- T025, T026, T027, T028 可同时编写

**User Story 2实现并行**:
- T029, T030 可同时实现

**User Story 3测试并行**:
- T034, T035, T036, T037 可同时编写

**User Story 3实现并行**:
- T038, T039 可同时实现

**Polish阶段并行**:
- T043, T044, T045, T046 (交互确认) 可并行
- T047, T048 (报告生成) 可并行
- T051, T052 (错误处理) 可并行
- T054, T055 (文档) 可并行

---

## Parallel Example: User Story 1

```bash
# 并行编写所有US1测试(不同文件或不同测试函数):
Task T012: "测试find_python_source_files() - tests/unit/cleanup-python.bats"
Task T013: "测试find_python_config_files() - tests/unit/cleanup-python.bats"
Task T014: "测试find_python_directories() - tests/unit/cleanup-python.bats"
Task T015: "测试白名单验证 - tests/unit/cleanup-python.bats"
Task T016: "测试干跑模式 - tests/unit/cleanup-python.bats"

# 并行实现US1的扫描函数(不同函数定义):
Task T017: "find_python_source_files() - scripts/cleanup-python.sh"
Task T018: "find_python_config_files() - scripts/cleanup-python.sh"
Task T019: "find_python_directories() - scripts/cleanup-python.sh"
```

---

## Implementation Strategy

### MVP First (仅User Story 1 + US3安全验证)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational (**关键** - 阻塞所有故事)
3. 完成 Phase 3: User Story 1 (清理Python源文件)
4. 完成 Phase 5: User Story 3 (白名单保护)
5. **STOP 并验证**: 测试US1和US3独立工作
6. 部署/演示(如果准备好)

### Incremental Delivery

1. Setup + Foundational → 基础就绪
2. 添加 US1 + US3 → 测试独立性 → 部署/演示 (MVP - 安全清理源文件!)
3. 添加 US2 → 测试独立性 → 部署/演示 (增强 - 清理构建产物)
4. 添加 Polish功能 → 最终测试 → 生产就绪
5. 每个故事增加价值而不破坏已有故事

### Parallel Team Strategy

多个开发者协作:

1. 团队共同完成 Setup + Foundational
2. Foundational完成后:
   - Developer A: User Story 1 (T011-T024)
   - Developer B: User Story 3 (T034-T042)
   - Developer C: User Story 2 (T025-T033)
3. 故事独立完成并集成

---

## Notes

- **[P]** 标记 = 不同文件或无依赖,可并行
- **[Story]** 标签将任务映射到特定用户故事,便于追溯
- 每个用户故事应可独立完成和测试
- 先验证测试失败再实现功能
- 每个任务或逻辑组完成后提交
- 可在任何checkpoint停止以独立验证故事
- **避免**: 模糊任务、相同文件冲突、破坏独立性的跨故事依赖

---

## Summary

**总任务数**: 61个任务
**任务分布**:
- Phase 1 Setup: 4个任务
- Phase 2 Foundational: 6个任务 (阻塞所有故事)
- Phase 3 US1: 14个任务 (6测试 + 8实现)
- Phase 4 US2: 9个任务 (4测试 + 5实现)
- Phase 5 US3: 9个任务 (4测试 + 5实现)
- Phase 6 Polish: 19个任务 (交互4 + 报告4 + 错误3 + 文档/验证8)

**并行机会**: 约30个任务标记为[P],可在各自阶段内并行执行

**独立测试标准**:
- US1: 所有.py文件和src/目录删除,Go测试通过
- US2: 所有Python构建产物删除,不影响Go构建
- US3: 所有Go代码和配置文件完整保留

**建议MVP范围**: Phase 1 + Phase 2 + Phase 3 (US1) + Phase 5 (US3) = 核心安全清理功能

**格式验证**: ✅ 所有任务遵循清单格式 (checkbox + TaskID + [P]? + [Story]? + 描述 + 文件路径)
