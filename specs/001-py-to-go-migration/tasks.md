# 实施任务清单: Python到Go语言迁移

**功能分支**: `001-py-to-go-migration`
**生成日期**: 2025-11-15
**规格文档**: [spec.md](spec.md)
**实施计划**: [plan.md](plan.md)

---

## 任务概览

**总任务数**: 51 (原58,已删除4个相似度分析任务和3个断点续爬任务)
**用户故事**: 3个 (US1: P1核心功能, US2: P2性能, US3: P3部署)
**并行机会**: 29个可并行任务 (原35个)
**MVP范围**: Phase 3 (US1 - 命令行工具功能保持)

### 任务分布

| 阶段                  | 任务数 | 描述                                                      |
| --------------------- | ------ | --------------------------------------------------------- |
| Phase 1: Setup        | 8      | 项目初始化和环境搭建                                      |
| Phase 2: Foundational | 10     | 基础设施(日志、模型、工具)                                |
| Phase 3: US1 (P1)     | 18     | 核心功能等价性实现 (已删除4个相似度任务和3个断点续爬任务) |
| Phase 4: US2 (P2)     | 8      | 性能优化                                                  |
| Phase 5: US3 (P3)     | 5      | 交叉编译和部署                                            |
| Phase 6: Polish       | 2      | 文档和最终验证                                            |

---

## Phase 1: Setup (项目初始化)

**目标**: 建立Go项目结构,配置依赖和开发环境

**任务清单**:

- [X] T001 初始化Go模块 `go mod init github.com/RecoveryAshes/JsFIndcrack`
- [X] T002 创建项目目录结构 (cmd/, internal/{core,crawlers,utils,models}, tests/, configs/, scripts/)
- [X] T003 [P] 添加核心Go依赖到go.mod (colly, rod, zerolog, cobra, viper等)
- [X] T004 [P] 创建Makefile with build, test, clean, cross-compile targets
- [X] T005 [P] 创建默认配置文件模板 configs/config.yaml
- [X] T006 [P] 创建环境验证脚本 scripts/verify_setup.go
- [X] T007 [P] 创建交叉编译脚本 scripts/build.sh for Linux/macOS/Windows
- [X] T008 配置GitHub Actions CI workflow (.github/workflows/go.yml)

**完成标准**:

- `go mod download` 成功
- `make build` 可生成空的可执行文件
- 所有目录结构就绪

---

## Phase 2: Foundational (基础设施)

**目标**: 实现所有用户故事共享的基础组件

**任务清单**:

### 日志系统 (FR-010, FR-011)

- [X] T009 [P] 实现Zerolog日志初始化 internal/utils/logger.go
- [X] T010 [P] 配置多输出目标 (控制台彩色+主日志+错误日志) in logger.go
- [X] T011 [P] 配置日志轮转 (使用Lumberjack) in logger.go
- [X] T012 测试日志系统 (中文输出、级别过滤、文件轮转)

### 数据模型 (所有实体)

- [X] T013 [P] 实现CrawlTask和CrawlConfig模型 internal/models/task.go
- [X] T014 [P] 实现JSFile和MapFile模型 internal/models/file.go
- [X] T015 [P] 实现Checkpoint模型 internal/models/checkpoint.go
- [X] T016 [P] 实现SimilarityGroup模型 internal/models/similarity.go
- [X] T017 [P] 实现CrawlReport模型 internal/models/report.go
- [X] T018 实现所有模型的JSON序列化和验证函数 (Validate, ToJSON, FromJSON)

---

## Phase 3: US1 - 命令行工具功能保持 (P1)

**用户故事**: 用户能够使用Go版本的JsFIndcrack工具执行与当前Python版本完全相同的JavaScript文件爬取和反混淆操作,包括单URL爬取、批量爬取、静态/动态模式切换等所有核心功能。

**独立测试标准**:

1. 运行 `./jsfindcrack -u https://example.com -d 3` 并对比Python版本的输出文件数量、内容、目录结构
2. 批量爬取对比: Python vs Go,每个URL的处理结果一致
3. 反混淆对比: decode文件内容一致

### 命令行接口 (FR-001)

- [X] T019 [P] [US1] 使用Cobra实现根命令和所有标志 cmd/jsfindcrack/main.go
- [X] T020 [P] [US1] 实现参数验证函数 (ValidateURL, ValidateFlags) in cmd/jsfindcrack/validate.go
- [X] T021 [P] [US1] 实现配置文件加载 (Viper) in internal/core/config.go
- [X] T022 [US1] 实现中文帮助信息和版本输出

### 静态爬取器 (FR-004, FR-012)

- [X] T023 [P] [US1] 实现StaticCrawler基础结构 internal/crawlers/static.go
- [X] T024 [P] [US1] 集成Colly,实现HTML解析和script标签提取 in static.go
- [X] T025 [P] [US1] 实现JavaScript文件下载和保存 in static.go
- [X] T026 [P] [US1] 实现Source Map文件识别和下载 in static.go
- [X] T027 [P] [US1] 实现并发下载控制 (errgroup + SetLimit) in static.go
- [X] T028 [US1] 测试静态爬取器 (单URL,深度2,对比Python输出)

### 动态爬取器 (FR-004)

- [X] T029 [P] [US1] 实现DynamicCrawler基础结构 internal/crawlers/dynamic.go
- [X] T030 [P] [US1] 集成Rod,实现浏览器启动和页面导航 in dynamic.go
- [X] T031 [P] [US1] 实现网络请求拦截 (HijackRequests) in dynamic.go
- [X] T032 [P] [US1] 实现JavaScript文件捕获和保存 in dynamic.go
- [X] T033 [P] [US1] 实现多标签页并发 (PagePool) in dynamic.go
- [X] T034 [US1] 测试动态爬取器 (单URL,对比Python输出)

### 主爬取器协调 (FR-002, FR-003)

- [X] T035 [US1] 实现主Crawler协调逻辑 internal/core/crawler.go
- [X] T036 [US1] 实现模式选择 (static/dynamic/all) in crawler.go
- [X] T037 [US1] 实现输出目录管理 (domain分离,encode/decode结构) in crawler.go
- [X] ✅ T038 [US1] 实现文件去重 (SHA-256哈希,跨模式去重) in crawler.go - 已通过全局哈希表实现跨爬取器去重

### 反混淆功能 (FR-005)

- [X] T039 [P] [US1] 实现Deobfuscator结构 internal/core/deobfuscator.go
- [X] T040 [P] [US1] 实现webcrack外部命令调用 + Go简单清理降级 (os/exec + context) in deobfuscator.go
- [X] T041 [US1] 实现混淆检测逻辑 in deobfuscator.go
- [X] T042 [US1] 测试反混淆功能 (webcrack和Go降级方案均通过)

### ~~相似度分析 (FR-006)~~ - 已删除

~~相似度分析功能已从项目中移除。文件去重已通过SHA-256哈希在爬取器中实现。~~

### ~~断点续爬 (FR-007)~~ - 已删除

~~断点续爬功能已从项目中移除。~~

### 批量处理 (FR-008)

- [X] T050 [P] [US1] 实现URL文件解析 internal/utils/helpers.go
- [X] T051 [P] [US1] 实现批量爬取循环 in internal/core/batch.go
- [X] T052 [US1] 实现批量错误容错 (--continue-on-error) in batch.go
- [X] T053 [US1] 实现批量延迟控制 (--batch-delay) in batch.go

### 报告生成 (FR-003, FR-013)

- [X] T054 [P] [US1] 实现CrawlReport生成 internal/utils/reporter.go
- [X] T055 [P] [US1] 实现进度条显示 (schollz/progressbar) in reporter.go
- [X] T056 [P] [US1] 生成JSON报告文件 (crawl_report, success_files, failed_files) in reporter.go
- [X] T057 [US1] 测试报告格式 (JSON schema对比Python版本)

### 端到端测试

- [X] T058 [US1] E2E测试: 单URL爬取完整流程,对比Python版本所有输出
- [X] T059 [US1] E2E测试: 批量爬取10个URL,对比统计数据
- [X] T060 [US1] E2E测试: 反混淆功能,对比decode文件内容
- [X] T061 [US1] 边界情况测试: 超大文件(>50MB), 网络超时, 编码异常, Ctrl+C中断

**US1完成标准**:

- ✅ 所有命令行参数100%兼容Python版本
- ✅ 输出目录结构、JSON报告格式100%一致
- ✅ 单URL爬取结果文件数量和内容MD5哈希一致
- ✅ 批量爬取统计数据(成功/失败数)一致
- ✅ 所有日志和错误信息为中文

---

## Phase 4: US2 - 性能提升 (P2)

**用户故事**: 用户在执行大规模JavaScript文件爬取任务时,能够感受到明显的性能提升,包括更快的并发处理速度、更低的内存占用和更好的系统资源利用率。

**独立测试标准**:

1. 批量爬取100个URL: Go版本执行时间 < Python版本 × 70%
2. 并发10线程: Go版本内存峰值 < Python版本 × 60%
3. 反混淆1000文件: Go版本时间 < Python版本 × 50%

**任务清单**:

### 并发优化

- [X] T064 [P] [US2] 优化静态爬取并发数 (动态调整,基于CPU核心数) in static.go
- [X] T065 [P] [US2] 优化动态爬取标签页池 (复用策略,减少浏览器启动) in dynamic.go
- [X] T066 [P] [US2] 优化反混淆并发 (内存配额管理,避免OOM) in deobfuscator.go

### 内存优化

- [X] T067 [P] [US2] 实现流式文件读写 (bufio + io.TeeReader) in utils/fileutil.go
- [X] T068 [P] [US2] 实现对象池 (sync.Pool,复用缓冲区) in utils/pool.go
- [X] T069 [US2] 优化大文件哈希计算 (边读边算,不全文加载)

### 性能基准测试

- [X] T070 [US2] 创建性能基准测试脚本 tests/benchmark/performance_test.sh
- [X] T071 [US2] 执行100 URL批量爬取对比 (Python vs Go,记录时间和内存)

**US2完成标准**:

- ✅ 批量爬取100 URL: 时间减少≥30%
- ✅ 内存峰值占用: 降低≥40%
- ✅ 反混淆1000文件: 速度提升≥50%
- ✅ 无内存泄漏(长时间运行稳定)

---

## Phase 5: US3 - 部署便利性 (P3)

**用户故事**: 用户能够获得单个可执行文件(无需安装Python运行时、依赖库等),支持跨平台部署(Linux、macOS、Windows),简化安装和分发流程。

**独立测试标准**:

1. 全新Linux服务器(无Python): 下载二进制文件直接运行基本爬取任务
2. 三平台交叉编译: Linux/macOS/Windows可执行文件均能正常运行
3. 用户分发场景: 单文件分发,无需安装依赖

**任务清单**:

- [X] T072 [P] [US3] 配置Linux AMD64交叉编译 in scripts/build.sh
- [X] T073 [P] [US3] 配置macOS AMD64/ARM64交叉编译 in scripts/build.sh
- [X] T074 [P] [US3] 配置Windows AMD64交叉编译 in scripts/build.sh
- [X] T075 [US3] 测试macOS AMD64二进制文件
- [X] T076 [US3] 创建发布包 (包含README、使用说明、外部依赖安装指南)

**US3完成标准**:

- ✅ 生成Linux/macOS/Windows三平台二进制文件
- ✅ 每个平台的二进制文件可独立运行(除webcrack/Playwright外)
- ✅ 文件大小合理(<50MB)
- ✅ 启动时间<1秒

---

## Phase 6: Polish & Cross-Cutting Concerns

**目标**: 文档更新,最终验证和发布准备

**任务清单**:

- [ ] T077 更新README.md (Go安装说明,构建步骤,使用示例,迁移指南)
- [ ] T078 最终端到端验证 (运行完整测试套件,确认所有成功标准达成)

---

## 依赖关系图

### 用户故事依赖顺序

```
Phase 1 (Setup)
    ↓
Phase 2 (Foundational) - 必须完成
    ↓
Phase 3 (US1 - P1) - MVP,必须完成
    ↓
Phase 4 (US2 - P2) - 基于US1,性能优化
    ↓
Phase 5 (US3 - P3) - 基于US1,交叉编译
    ↓
Phase 6 (Polish)
```

**关键路径**: Setup → Foundational → US1 → US2 → US3 → Polish

**并行机会**:

- Phase 1: T003-T007 可并行
- Phase 2: T009-T017 可并行 (不同文件)
- Phase 3内:
  - 静态爬取器 (T023-T027) vs 动态爬取器 (T029-T033) 可并行
  - 反混淆 (T039-T041) vs 批量处理 (T050-T051) 可并行
- Phase 4: T064-T069 可并行
- Phase 5: T072-T074 可并行

---

## 并行执行示例

### US1 (Phase 3) 并行策略

**阶段1**: CLI + 静态爬取器 + 动态爬取器 (并行)

```
T019-T022 (CLI) || T023-T027 (Static) || T029-T033 (Dynamic)
```

**阶段2**: 主协调器 + 辅助功能 (部分并行)

```
T035-T038 (Crawler) → T039-T042 (Deobf)
```

**阶段3**: 批量处理 + 报告 (并行)

```
T050-T053 (Batch) || T054-T057 (Report)
```

**阶段4**: E2E测试 (串行,依赖所有前置)

```
T058 → T059 → T060 → T061
```

---

## 实施策略

### MVP优先 (最小可行产品)

**MVP范围**: Phase 1 + Phase 2 + Phase 3 (US1)

**理由**:

- US1是核心功能等价性要求,没有US1迁移就失去意义
- US2和US3是增量改进,可在US1稳定后逐步优化

**交付里程碑**:

1. **Week 1-2**: Setup + Foundational (T001-T018)
2. **Week 3-5**: US1 Core (T019-T057)
3. **Week 6**: US1 E2E测试 (T058-T061)
4. **Week 7**: US2 性能优化 (T064-T071)
5. **Week 8**: US3 交叉编译 (T072-T076)
6. **Week 9**: Polish + 发布 (T077-T078)

### 增量交付

每完成一个Phase,立即进行集成测试:

- Phase 3 (US1) → 功能对比测试 vs Python
- Phase 4 (US2) → 性能基准测试
- Phase 5 (US3) → 跨平台验证

### 风险缓解

| 风险           | 任务       | 缓解措施                        |
| -------------- | ---------- | ------------------------------- |
| 数据格式不兼容 | T058-T061  | E2E测试早期执行,对比JSON schema |
| 性能目标未达成 | T071       | 基准测试尽早执行,识别瓶颈       |
| Go库学习曲线   | T023-T033  | 参考research.md示例代码,POC验证 |
| 外部依赖问题   | T040, T031 | 早期测试webcrack和Rod集成       |

---

## 任务执行指南

### 任务标记说明

- `[P]`: 可并行任务 (操作不同文件或无依赖)
- `[US1]`, `[US2]`, `[US3]`: 所属用户故事
- 无标记: 串行任务,依赖前置任务完成

### 验收标准

**每个任务完成时**:

- [ ] 代码通过 `go fmt` 和 `go vet`
- [ ] 单元测试覆盖率 ≥ 70% (关键模块)
- [ ] 如有中文输出,确保UTF-8编码正确

**Phase完成时**:

- [ ] 所有任务checkbox已勾选
- [ ] Phase独立测试标准全部通过
- [ ] 无遗留TODO或FIXME

### 下一步

1. 执行 `make install` 安装依赖
2. 开始 Phase 1 Setup 任务 (T001-T008)
3. 完成Setup后,运行 `go run scripts/verify_setup.go` 验证环境

---

**文档版本**: 1.0
**最后更新**: 2025-11-15
**下一步**: 开始执行 Phase 1 Setup 任务
