# Tasks: 自定义HTTP请求头

**Input**: Design documents from `/specs/002-custom-http-headers/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: 本功能规范未明确要求TDD方法,测试任务仅包含核心模块的单元测试(覆盖率目标≥70%)和集成测试。

**Organization**: 任务按用户故事组织,每个用户故事可独立实现和测试。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行 (不同文件,无依赖)
- **[Story]**: 任务所属用户故事 (如 US1, US2, US3)
- 描述中包含准确的文件路径

## Path Conventions

项目采用标准Go工程结构:
- **命令行**: `cmd/jsfindcrack/`
- **内部包**: `internal/config/`, `internal/core/`, `internal/models/`, `internal/utils/`, `internal/crawlers/`
- **配置文件**: `configs/`
- **测试**: `tests/unit/`, `tests/integration/`

---

## Phase 1: Setup (项目初始化)

**Purpose**: 配置文件目录和默认模板准备

- [X] T001 创建配置文件目录 `configs/` (如不存在)
- [X] T002 设计并实现默认HTTP头部配置模板 (YAML格式,包含中文注释和示例)
- [X] T003 [P] 验证项目已引入依赖: viper v1.20.0-alpha.6 和 cobra (检查 go.mod)

---

## Phase 2: Foundational (阻塞性前置任务)

**Purpose**: 核心数据结构和接口定义,所有用户故事依赖的基础设施

**⚠️ CRITICAL**: 本阶段必须完成后才能开始任何用户故事实施

- [X] T004 [P] 定义 `HeaderConfig` 数据结构 in `internal/models/headers.go`
- [X] T005 [P] 定义 `CliHeaders` 类型及解析方法 in `internal/models/headers.go`
- [X] T006 [P] 定义 `HeaderProvider` 接口 in `internal/models/headers.go`
- [X] T007 [P] 定义 `ValidationError` 和 `ConfigError` 错误类型 in `internal/models/headers.go`
- [X] T008 实现 `HeaderValidator` 结构体和验证逻辑 (正则表达式验证) in `internal/utils/validator.go`
- [X] T009 实现敏感头部识别和脱敏逻辑 in `internal/utils/redactor.go`

**Checkpoint**: 基础数据模型和验证器就绪,用户故事实施可并行启动

---

## Phase 3: User Story 1 - 通过配置文件设置通用HTTP头部 (Priority: P1) 🎯 MVP

**Goal**: 用户能够在 `configs/headers.yaml` 中配置通用HTTP头部(如User-Agent、Referer),程序自动加载并应用到所有HTTP请求

**Independent Test**:
1. 编辑 `configs/headers.yaml` 设置 `User-Agent: MyCustomBot/1.0`
2. 运行爬虫抓取任意网站
3. 验证实际发送的HTTP请求头部包含自定义User-Agent (通过 `--log-level debug` 查看日志)

### Implementation for User Story 1

- [X] T010 [P] [US1] 实现配置文件自动生成逻辑: 首次运行时在 `configs/` 目录生成 `headers.yaml` 模板 in `internal/config/headers.go`
- [X] T011 [P] [US1] 实现配置文件加载逻辑: 使用 viper 解析 YAML 文件为 `HeaderConfig` in `internal/config/headers.go`
- [X] T012 [P] [US1] 实现配置文件验证逻辑: 检查文件大小(≤1MB)、YAML格式、头部合法性 in `internal/config/headers.go`
- [X] T013 [US1] 创建 `HeaderManager` 结构体,实现配置文件头部加载和默认头部合并 in `internal/core/header_manager.go`
- [X] T014 [US1] 实现 `HeaderManager.GetHeaders()` 方法,返回合并后的 http.Header (默认 < 配置文件) in `internal/core/header_manager.go`
- [X] T015 [US1] 实现 `HeaderManager.GetSafeHeaders()` 方法,返回脱敏后的头部用于日志输出 in `internal/core/header_manager.go`
- [X] T016 [US1] 单元测试: 配置文件加载和解析 in `tests/unit/config_headers_test.go`
- [X] T017 [US1] 单元测试: 头部验证器 (合法/非法头部名称和值) in `tests/unit/validator_test.go`
- [X] T018 [US1] 单元测试: 头部管理器合并逻辑 in `tests/unit/header_manager_test.go`

**Checkpoint**: 配置文件方式已完全可用,用户可通过编辑 `configs/headers.yaml` 自定义HTTP头部

---

## Phase 4: User Story 2 - 通过命令行参数传入临时认证头部 (Priority: P2)

**Goal**: 用户能够通过 `--header` 参数在运行时传入临时HTTP头部(如 `Authorization: Bearer token`),命令行参数优先级高于配置文件

**Independent Test**:
1. 运行 `jsfindcrack -u https://example.com --header "Authorization: Bearer abc123" --log-level debug`
2. 验证日志中显示请求头部包含 `Authorization: Bearer ***` (脱敏)
3. 验证实际请求携带完整的 `Authorization: Bearer abc123`

### Implementation for User Story 2

- [X] T019 [US2] 扩展 `cmd/jsfindcrack/root.go`: 添加 `--header` (短参数 `-H`) 命令行参数,类型为 StringSlice
- [X] T020 [US2] 实现 `CliHeaders.Parse()` 方法: 解析 `"Name: Value"` 格式字符串为 http.Header in `internal/models/headers.go`
- [X] T021 [US2] 扩展 `HeaderManager`: 集成命令行头部,实现三层合并 (默认 < 配置 < 命令行) in `internal/core/header_manager.go`
- [X] T022 [US2] 实现命令行参数格式错误处理: 缺少冒号、空名称等情况返回清晰错误 in `internal/core/header_manager.go`
- [X] T023 [US2] 单元测试: 命令行头部解析 (成功和失败场景) in `tests/unit/header_manager_test.go`
- [X] T024 [US2] 单元测试: 头部合并优先级 (命令行覆盖配置文件和默认) in `tests/unit/header_manager_test.go`
- [X] T025 [US2] 单元测试: 敏感头部脱敏逻辑 (Authorization/Token/Key等) in `tests/unit/redactor_test.go`

**Checkpoint**: 命令行参数方式已完全可用,用户可灵活传入临时头部并覆盖配置文件

---

## Phase 5: User Story 3 - 配置文件验证 (Priority: P3)

**Goal**: 用户能够使用 `--validate-config` 参数验证配置文件正确性,快速定位配置错误(格式错误、非法头部等),无需实际执行爬取

**Independent Test**:
1. 故意在 `configs/headers.yaml` 中引入格式错误 (如缺少冒号)
2. 运行 `jsfindcrack --validate-config`
3. 验证程序输出详细错误信息(行号、错误原因、修复建议)且返回非零退出码

### Implementation for User Story 3

- [X] T026 [US3] 扩展 `cmd/jsfindcrack/root.go`: 添加 `--validate-config` 命令行参数 (布尔类型)
- [X] T027 [US3] 实现配置验证命令逻辑: 加载配置文件 → 验证头部 → 输出结果 in `cmd/jsfindcrack/root.go`
- [X] T028 [US3] 实现验证成功输出: 显示配置文件路径、头部数量、头部列表(脱敏) in `cmd/jsfindcrack/root.go`
- [X] T029 [US3] 实现验证失败输出: 显示错误类型(解析错误/验证错误)、具体位置、修复建议 in `cmd/jsfindcrack/root.go`
- [X] T030 [US3] 单元测试: `--validate-config` 成功场景 in `tests/unit/config_headers_test.go`
- [X] T031 [US3] 单元测试: `--validate-config` 失败场景 (YAML语法错误、非法头部) in `tests/unit/config_headers_test.go`

**Checkpoint**: 配置验证功能完整,用户可快速验证配置正确性

---

## Phase 6: Integration (爬虫集成)

**Purpose**: 将头部管理器集成到现有爬虫模块

- [X] T032 [P] 修改 `internal/crawlers/static_crawler.go`: 注入 `HeaderProvider` 接口,在HTTP请求中应用头部
- [X] T033 [P] 修改 `internal/crawlers/dynamic_crawler.go`: 注入 `HeaderProvider` 接口,在浏览器自动化中应用头部
- [X] T034 修改 `cmd/jsfindcrack/root.go`: 在主程序初始化时创建 `HeaderManager` 实例,传递给爬虫
- [X] T035 集成测试: 端到端测试配置文件方式 in `tests/integration/headers_integration_test.go`
- [X] T036 集成测试: 端到端测试命令行参数方式 in `tests/integration/headers_integration_test.go`
- [X] T037 集成测试: 端到端测试优先级覆盖 (命令行 > 配置 > 默认) in `tests/integration/headers_integration_test.go`

**Checkpoint**: 所有爬虫模式 (static/dynamic/all) 均支持自定义HTTP头部

---

## Phase 7: Edge Cases (边缘场景处理)

**Purpose**: 处理边缘场景和错误情况

- [X] T038 [P] 实现配置文件权限不足错误处理: 无法创建 `configs/` 目录时给出明确提示 in `internal/config/headers.go`
- [X] T039 [P] 实现配置文件锁定错误处理: 文件被占用时优雅降级,使用默认配置并警告 in `internal/config/headers.go`
- [X] T040 [P] 实现超长头部值处理: 值超过8KB时截断或拒绝,并给出警告 in `internal/utils/validator.go`
- [X] T041 [P] 实现禁止头部过滤: 拒绝 `Host`/`Content-Length`/`Transfer-Encoding`/`Connection` 配置 in `internal/utils/validator.go`
- [X] T042 实现空配置文件处理: 配置文件存在但为空时使用默认头部,不报错 in `internal/config/headers.go`
- [X] T043 单元测试: 边缘场景覆盖 in `tests/unit/edge_cases_test.go`

**Checkpoint**: 所有边缘场景均有明确的错误处理和用户友好的提示

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: 文档、日志、代码质量优化

- [X] T044 [P] 更新 `--help` 输出: 添加 `--header` 和 `--validate-config` 使用示例 in `cmd/jsfindcrack/root.go`
- [X] T045 [P] 添加日志输出: 配置文件加载成功/失败、头部合并、验证错误详情 in `internal/core/header_manager.go`
- [X] T046 [P] 代码注释优化: 确保关键模块(config/core/utils)注释率≥30% in 相关文件
- [X] T047 添加 `.gitignore` 条目: 建议用户忽略 `configs/headers.yaml` (避免敏感信息泄露) in 项目根目录 `.gitignore`
- [X] T048 运行 `gofmt` 和 `goimports` 格式化所有新增代码
- [X] T049 运行 `go test -cover ./...` 验证覆盖率≥70% (针对 config、core、utils 包)
- [X] T050 验证 quickstart.md 中的所有使用示例均可正常运行

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖,可立即开始
- **Foundational (Phase 2)**: 依赖 Setup 完成 - **阻塞所有用户故事**
- **User Stories (Phase 3-5)**: 全部依赖 Foundational 完成
  - US1、US2、US3 可并行实施 (如有足够人力)
  - 或按优先级顺序实施 (P1 → P2 → P3)
- **Integration (Phase 6)**: 依赖 US1 和 US2 完成 (US3 可选)
- **Edge Cases (Phase 7)**: 依赖 Integration 完成
- **Polish (Phase 8)**: 依赖所有功能完成

### User Story Dependencies

- **User Story 1 (P1)**: Foundational 完成后即可开始 - 无其他依赖
- **User Story 2 (P2)**: Foundational 完成后即可开始 - 扩展 US1 但可独立测试
- **User Story 3 (P3)**: Foundational 完成后即可开始 - 依赖 US1 的配置加载逻辑

### Within Each User Story

- US1: 配置加载 (T010-T012) → HeaderManager (T013-T015) → 测试 (T016-T018)
- US2: 命令行解析 (T019-T020) → 合并逻辑 (T021-T022) → 测试 (T023-T025)
- US3: 验证命令 (T026-T029) → 测试 (T030-T031)

### Parallel Opportunities

- **Phase 1**: T001、T002、T003 可并行 (不同操作)
- **Phase 2**: T004-T007 (数据结构定义) 可并行, T008-T009 (验证器) 可并行
- **Phase 3**: T010-T012 (配置逻辑) 可并行, T016-T018 (测试) 可并行
- **Phase 4**: T023-T025 (测试) 可并行
- **Phase 6**: T032-T033 (爬虫修改) 可并行, T035-T037 (集成测试) 可并行
- **Phase 7**: T038-T041 (边缘场景) 可并行
- **Phase 8**: T044-T046 (文档和注释) 可并行

**跨用户故事并行**: 一旦 Foundational 完成,US1、US2、US3 可由不同开发者并行实施

---

## Parallel Example: User Story 1

```bash
# 并行执行配置逻辑开发 (不同文件):
Task T010: "实现配置文件自动生成逻辑 in internal/config/headers.go"
Task T011: "实现配置文件加载逻辑 in internal/config/headers.go"
Task T012: "实现配置文件验证逻辑 in internal/config/headers.go"

# 并行执行单元测试 (不同测试文件):
Task T016: "单元测试: 配置文件加载和解析 in tests/unit/config_headers_test.go"
Task T017: "单元测试: 头部验证器 in tests/unit/validator_test.go"
Task T018: "单元测试: 头部管理器合并逻辑 in tests/unit/header_manager_test.go"
```

---

## Implementation Strategy

### MVP First (仅 User Story 1)

1. 完成 Phase 1: Setup (配置目录和模板)
2. 完成 Phase 2: Foundational (数据模型和验证器) - **关键阻塞点**
3. 完成 Phase 3: User Story 1 (配置文件方式)
4. **STOP and VALIDATE**: 独立测试 US1 - 验证配置文件加载和应用
5. 如果 MVP 就绪,可先部署/演示基础功能

### Incremental Delivery

1. Setup + Foundational → 基础设施就绪
2. 添加 US1 → 独立测试 → 部署/演示 (MVP: 配置文件方式)
3. 添加 US2 → 独立测试 → 部署/演示 (增强: 命令行参数)
4. 添加 US3 → 独立测试 → 部署/演示 (完整: 配置验证)
5. 添加 Integration → 端到端测试 → 部署/演示 (全面集成)
6. 每个故事增加价值,不破坏已有功能

### Parallel Team Strategy

如果有多个开发者:

1. 团队共同完成 Setup + Foundational
2. Foundational 完成后:
   - Developer A: User Story 1 (T010-T018)
   - Developer B: User Story 2 (T019-T025)
   - Developer C: User Story 3 (T026-T031)
3. 各故事独立完成并测试后,共同进行 Integration (Phase 6)

---

## Notes

- [P] 标记表示可并行任务 (不同文件,无依赖关系)
- [Story] 标签映射任务到具体用户故事,确保可追溯性
- 每个用户故事应独立可完成、可测试、可演示
- 验证测试在实施前编写 (TDD 风格)
- 每个任务或逻辑组完成后提交代码 (遵循约定式提交)
- 在任何 Checkpoint 停止并独立验证用户故事
- 避免: 模糊任务描述、同文件冲突、破坏故事独立性的跨故事依赖

---

## 总任务数统计

- **总任务数**: 50 (T001-T050)
- **Phase 1 (Setup)**: 3 任务
- **Phase 2 (Foundational)**: 6 任务
- **Phase 3 (US1)**: 9 任务
- **Phase 4 (US2)**: 7 任务
- **Phase 5 (US3)**: 6 任务
- **Phase 6 (Integration)**: 6 任务
- **Phase 7 (Edge Cases)**: 6 任务
- **Phase 8 (Polish)**: 7 任务

**并行机会**: 约 25 个任务标记为 [P],可显著缩短总实施时间

**建议 MVP 范围**: Phase 1 + Phase 2 + Phase 3 (共 18 任务) - 实现配置文件方式的基础功能
