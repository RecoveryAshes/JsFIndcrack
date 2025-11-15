# Implementation Plan: 自定义HTTP请求头

**Branch**: `002-custom-http-headers` | **Date**: 2025-11-15 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-custom-http-headers/spec.md`

## Summary

实现灵活的HTTP请求头配置机制,支持通过YAML配置文件设置通用头部,通过命令行参数传入临时认证头部。程序首次运行时自动生成配置模板,命令行参数优先级高于配置文件。关键特性包括:配置验证、头部合法性检查、敏感信息脱敏。技术方案采用Go标准库+viper配置管理+cobra命令行解析,符合项目现有技术栈。

## Technical Context

**Language/Version**: Go 1.23+
**Primary Dependencies**:
- spf13/viper (已引入) - YAML配置文件解析
- spf13/cobra (已引入) - 命令行参数处理
- Go标准库 net/http - HTTP头部验证

**Storage**: 文件系统 (YAML配置文件存储在 `config/headers.yaml`)
**Testing**: Go标准testing包 + testify (建议引入用于断言)
**Target Platform**: macOS, Linux, Windows (跨平台命令行工具)
**Project Type**: 单一项目 (CLI工具)
**Performance Goals**:
- 配置文件解析时间 < 100ms
- 头部验证时间 < 10ms
- 程序启动开销 < 50ms

**Constraints**:
- 配置文件大小 < 1MB
- 单个头部值 < 8KB (HTTP协议限制)
- 支持最多100个自定义头部

**Scale/Scope**:
- 单用户CLI工具
- 配置文件行数通常 < 50行
- 并发场景:多个HTTP请求共享同一头部配置

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. 文档中文化 ✅
- **要求**: 所有文档(除README.md)使用中文,代码注释中文优先,禁用emoji
- **符合性**:
  - spec.md、plan.md均使用中文编写
  - 配置文件模板将包含中文注释
  - 错误信息和日志输出将提供中文
  - 代码注释将使用中文+英文技术术语

### II. 代码质量与安全 ✅
- **要求**: OWASP最佳实践,输入验证,敏感信息保护,30%注释率,SOLID原则
- **符合性**:
  - **输入验证**: 所有配置文件内容和命令行参数进行严格验证
  - **敏感信息保护**: FR-012要求自动脱敏认证头部(Authorization/Token/Key/Secret)
  - **错误处理**: 配置解析失败、文件权限不足等场景均有明确错误处理
  - **注释要求**: 关键模块(配置解析、头部合并、验证逻辑)将达到≥30%注释率
  - **SOLID原则**: 配置加载、头部验证、头部合并职责分离

### III. 模块化设计 ✅
- **要求**: 职责单一,独立可测试,接口通信,标准目录结构,使用internal保护
- **符合性**:
  - 配置管理模块独立 (`internal/config/headers.go`)
  - 命令行参数解析独立 (`cmd/jsfindcrack/root.go`扩展)
  - 头部合并逻辑独立 (`internal/core/header_manager.go`)
  - 使用internal目录保护内部实现
  - 通过接口抽象头部提供者 (`HeaderProvider` interface)

### IV. 用户体验优先 ✅
- **要求**: 清晰帮助文档,错误信息明确可操作
- **符合性**:
  - SC-002要求用户5分钟内理解配置格式(通过模板注释实现)
  - SC-003要求1秒内检测配置错误并显示
  - 错误信息包含文件路径、行号、具体原因、修复建议
  - `--help`输出包含`--header`和`--validate-config`使用示例

### V. 性能与可靠性 ✅
- **要求**: 合理超时,资源限制,context包管理生命周期
- **符合性**:
  - 配置文件大小限制1MB,防止滥用
  - 单个头部值限制8KB,符合HTTP协议
  - 配置解析超时100ms
  - 虽然此功能本身不涉及并发HTTP请求,但设计允许与现有爬虫模块集成时共享context

### VI. 领域驱动设计 ✅
- **要求**: DDD思想,小接口,依赖倒置,领域划分优于技术分层
- **符合性**:
  - 定义小接口 `HeaderProvider`,`HeaderValidator`
  - 依赖倒置: HTTP客户端依赖 `HeaderProvider` 接口,而非具体实现
  - 领域划分: `config/`(配置领域), `core/`(爬虫核心领域)

### VII. 版本控制纪律 ✅
- **要求**: 频繁提交,约定式提交,Git规范
- **符合性**:
  - 实施过程将遵循约定式提交 (feat: / fix: / docs:)
  - 按功能模块小步提交 (配置加载 → 命令行解析 → 合并逻辑 → 验证)

### 开发规范/代码风格 ✅
- **要求**: Go官方规范,gofmt/goimports,最小化接口,有意义命名
- **符合性**:
  - 所有代码将通过 `gofmt` 和 `goimports` 格式化
  - 接口命名遵循单一方法动词形式 (如 `Validate()`, `Provide()`)
  - 变量命名有意义 (如 `headerConfig`, `cliHeaders`, `mergedHeaders`)

### 开发规范/测试要求 ✅
- **要求**: 核心功能变更包含单元测试,Mock测试,覆盖率≥70%
- **符合性**:
  - 配置加载、头部合并、验证逻辑均将包含单元测试
  - 使用接口Mock测试 (如Mock `HeaderProvider`)
  - 目标覆盖率: config包和core/header_manager包≥70%

### 开发规范/依赖管理 ✅
- **要求**: Go Modules,避免不必要依赖
- **符合性**:
  - 使用已有依赖 viper 和 cobra,无需新增外部依赖
  - 建议引入 testify 用于测试断言 (可选,仅测试依赖)

**结论**: ✅ 所有宪章检查项通过,无违规,可进入Phase 0研究阶段。

## Project Structure

### Documentation (this feature)

```text
specs/002-custom-http-headers/
├── plan.md              # 本文件 (/speckit.plan命令输出)
├── spec.md              # 功能规格
├── research.md          # Phase 0输出 (技术调研)
├── data-model.md        # Phase 1输出 (数据模型)
├── quickstart.md        # Phase 1输出 (快速入门)
├── contracts/           # Phase 1输出 (接口契约)
│   └── header_provider_interface.md
└── checklists/
    └── requirements.md  # 规格质量检查清单
```

### Source Code (repository root)

项目采用标准Go工程结构 (已存在):

```text
cmd/
└── jsfindcrack/
    ├── main.go             # 程序入口
    ├── root.go             # cobra根命令 (需扩展--header和--validate-config参数)
    └── version.go

internal/
├── config/
│   ├── config.go           # 现有通用配置
│   └── headers.go          # 新增: HTTP头部配置加载器
├── core/
│   ├── js_crawler.go       # 现有爬虫核心
│   └── header_manager.go   # 新增: 头部管理器 (合并、验证、脱敏)
├── crawlers/
│   ├── static_crawler.go   # 需修改: 集成HeaderProvider
│   └── dynamic_crawler.go  # 需修改: 集成HeaderProvider
├── models/
│   └── headers.go          # 新增: Header相关数据结构
└── utils/
    ├── logger.go           # 现有日志工具
    └── validator.go        # 新增: HTTP头部验证器

configs/                    # 配置文件目录 (已存在,但当前为空)
└── headers.yaml            # 新增: 默认HTTP头部配置模板 (首次运行生成)

tests/
├── unit/
│   ├── config_test.go      # 配置加载单元测试
│   ├── header_manager_test.go  # 头部管理器单元测试
│   └── validator_test.go   # 验证器单元测试
└── integration/
    └── headers_integration_test.go  # 端到端集成测试
```

**Structure Decision**:
采用选项1 (单一项目结构),因为这是CLI工具,无前后端分离需求。新增文件将融入现有 `internal/` 目录结构,遵循Go标准项目布局。配置文件生成到项目根目录的 `configs/` 子目录 (而非当前工作目录的 `config/`),以避免污染用户工作目录。

## Complexity Tracking

> 无宪章违规,此部分留空。
