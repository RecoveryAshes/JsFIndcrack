# Feature Specification: 清理遗留Python文件

**Feature Branch**: `003-cleanup-legacy-files`
**Created**: 2025-11-15
**Status**: Draft
**Input**: User description: "现在py转go转好了,现在可以删除非必要文件包括py文件"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 清理Python源代码文件 (Priority: P1)

作为项目维护者,在完成Python到Go的迁移后,我需要删除所有Python源代码文件和相关配置,以保持代码库整洁,避免混淆,并减少仓库大小。

**Why this priority**: 这是核心需求,直接实现了用户描述的主要目标。删除Python文件是迁移完成后的必要清理步骤,可以立即减少代码库混乱和维护负担。

**Independent Test**: 可以通过验证所有`.py`文件、`requirements.txt`、`src/`目录及相关Python配置文件已被删除来独立测试,验证后项目仍能正常使用Go版本运行。

**Acceptance Scenarios**:

1. **Given** 项目中存在Python源代码文件(`.py`), **When** 执行清理操作, **Then** 所有Python源文件被删除,包括`main.py`、`src/`目录下的所有`.py`文件
2. **Given** 项目中存在Python配置文件(`requirements.txt`), **When** 执行清理操作, **Then** 所有Python依赖配置文件被删除
3. **Given** 清理完成后, **When** 使用Go版本工具运行, **Then** 工具正常工作,功能不受影响

---

### User Story 2 - 清理Python构建产物 (Priority: P2)

作为项目维护者,我需要删除所有Python构建产物、缓存文件和临时文件,以确保仓库只包含必要的文件。

**Why this priority**: 这是清理工作的补充部分。虽然不如源代码文件关键,但删除这些文件可以进一步减小仓库大小,避免遗留缓存文件造成困惑。

**Independent Test**: 可以通过验证所有`__pycache__`目录、`.pyc`文件、`.egg-info`目录已被删除来独立测试,不影响Go版本的正常运行。

**Acceptance Scenarios**:

1. **Given** 项目中存在`__pycache__`目录, **When** 执行清理操作, **Then** 所有`__pycache__`目录被递归删除
2. **Given** 项目中存在编译后的`.pyc`文件, **When** 执行清理操作, **Then** 所有`.pyc`文件被删除
3. **Given** 项目中存在`.egg-info`或其他Python包构建产物, **When** 执行清理操作, **Then** 这些目录和文件被删除

---

### User Story 3 - 保留必要的文档和配置 (Priority: P1)

作为项目维护者,在清理Python文件时,我需要保留与Python无关的重要文档、配置文件和其他必要资源,确保项目的完整性不受影响。

**Why this priority**: 这是关键的安全检查。错误地删除重要文件可能导致项目损坏,因此需要明确定义保留规则,与删除操作同等重要。

**Independent Test**: 可以通过验证Go源代码、配置文件、文档、测试文件、构建脚本等关键文件未被误删来独立测试。

**Acceptance Scenarios**:

1. **Given** 项目中存在Go源代码文件, **When** 执行清理操作, **Then** 所有`.go`文件和`go.mod`、`go.sum`保持不变
2. **Given** 项目中存在项目配置和构建脚本, **When** 执行清理操作, **Then** `Makefile`、构建脚本、配置目录保持不变
3. **Given** 项目中存在文档和规范, **When** 执行清理操作, **Then** `.md`文件、`specs/`目录、`.specify/`目录保持不变
4. **Given** 项目中存在测试文件和资源, **When** 执行清理操作, **Then** `tests/`目录及其内容保持不变

---

### Edge Cases

- 如果某个Python文件与重要功能相关但尚未完全迁移,如何处理?
- 如果清理过程中发现意外的Python文件类型(如`.pyx`、`.pyd`),如何处理?
- 如果src目录下混合了其他非Python文件,如何安全删除?
- 清理后如何验证Go版本的所有功能仍然正常工作?
- 如果在`.gitignore`中已忽略某些Python文件,清理操作是否需要处理它们?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须能够识别并删除所有Python源代码文件(`.py`扩展名)
- **FR-002**: 系统必须能够删除`src/`目录及其所有子目录和内容
- **FR-003**: 系统必须能够删除Python依赖配置文件(`requirements.txt`)
- **FR-004**: 系统必须能够删除所有`__pycache__`目录及其内容
- **FR-005**: 系统必须能够删除所有编译后的Python字节码文件(`.pyc`、`.pyo`)
- **FR-006**: 系统必须能够删除Python包构建产物(`.egg-info`目录、`dist/`中的Python包等)
- **FR-007**: 系统必须保留所有Go源代码文件和相关配置(`cmd/`、`internal/`、`go.mod`、`go.sum`)
- **FR-008**: 系统必须保留所有项目配置和构建文件(`Makefile`、`scripts/`、`configs/`、`config/`)
- **FR-009**: 系统必须保留所有文档和规范文件(`.md`文件、`specs/`、`.specify/`、`.github/`)
- **FR-010**: 系统必须保留测试文件和资源(`tests/`目录)
- **FR-011**: 清理操作必须提供清理前的确认提示,列出将要删除的文件和目录
- **FR-012**: 清理操作必须生成清理报告,记录已删除的文件列表

### Key Entities

- **Python源文件**: 所有`.py`扩展名的文件,包括`main.py`和`src/`目录下的所有模块
- **Python配置文件**: `requirements.txt`、`setup.py`等Python项目配置
- **Python构建产物**: `__pycache__`目录、`.pyc`文件、`.egg-info`目录等
- **保留资源**: Go代码、项目配置、文档、测试文件等必须保留的文件

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 清理后仓库中不存在任何`.py`文件(验证通过文件系统检查返回零结果)
- **SC-002**: 清理后仓库大小减少至少20%(对比清理前后的目录大小)
- **SC-003**: 清理后Go版本的所有现有功能测试100%通过(运行现有测试套件验证)
- **SC-004**: 清理操作可在30秒内完成(针对当前项目规模)
- **SC-005**: 清理报告准确列出所有已删除文件,准确率100%(人工抽查验证)
- **SC-006**: 清理后项目的核心文件(Go代码、配置、文档)完整性100%(验证关键文件列表)

## Assumptions

- 假设Python到Go的迁移已经完全完成,所有功能已在Go版本中实现并测试
- 假设不需要保留任何Python代码用于参考或回滚
- 假设`src/`目录专门用于Python代码,删除整个目录是安全的
- 假设项目使用Git进行版本控制,可以通过历史记录恢复误删的文件(如果需要)
- 假设清理操作将手动执行或通过脚本执行,不需要作为自动化流程的一部分

## Constraints

- 必须确保清理操作不影响当前正在使用的Go版本功能
- 必须保留足够的文档和历史记录以供未来参考
- 清理操作应该是可逆的(通过Git历史或备份)

## Dependencies

- 依赖于001-py-to-go-migration功能已完成并经过充分测试
- 依赖于现有的Git版本控制系统以确保可以恢复误删的文件
- 依赖于Go版本的功能测试套件以验证清理后的完整性
