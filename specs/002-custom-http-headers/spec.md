# Feature Specification: 自定义HTTP请求头

**Feature Branch**: `002-custom-http-headers`
**Created**: 2025-11-15
**Status**: Draft
**Input**: User description: "自定义header头（放在config文件里），自定义认证头（可以在终端条件中传入），config文件首次运行程序生成到当前目录的config目录中"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 通过配置文件设置通用HTTP头部 (Priority: P1)

用户需要访问某些网站时携带特定的HTTP请求头（如User-Agent、Referer等），这些头部在每次爬取时都保持不变。用户希望将这些通用头部预先配置在文件中，避免每次运行都手动指定。

**Why this priority**: 这是最基础的功能，许多目标网站会检查User-Agent等基本请求头，不配置会导致爬取失败。这是用户能够成功爬取受保护网站的前提。

**Independent Test**: 用户配置config文件中的User-Agent和Referer，运行爬虫后，检查实际发送的HTTP请求是否包含这些头部。可以通过抓包工具或目标网站的请求日志验证。

**Acceptance Scenarios**:

1. **Given** 首次运行程序，**When** 程序启动，**Then** 自动在当前目录下创建 `config/` 目录和默认配置文件 `config/headers.yaml`（或其他配置格式）
2. **Given** 配置文件中设置了 `User-Agent: MyCustomBot/1.0`，**When** 爬取任意网站，**Then** 所有HTTP请求都携带该User-Agent头部
3. **Given** 配置文件中设置了多个头部（User-Agent、Referer、Accept-Language等），**When** 爬取网站，**Then** 所有配置的头部都出现在请求中
4. **Given** 配置文件不存在或为空，**When** 运行程序，**Then** 使用系统默认的HTTP头部，程序正常运行不报错

---

### User Story 2 - 通过命令行参数传入临时认证头部 (Priority: P2)

用户需要访问需要认证的网站（如Bearer Token、API Key），但这些凭证是临时的或敏感的，不适合写入配置文件。用户希望通过命令行参数在运行时传入认证头部。

**Why this priority**: 认证头部通常包含敏感信息，写入配置文件存在安全风险。命令行传参方式更灵活、更安全，适合临时爬取和自动化脚本。

**Independent Test**: 用户通过命令行参数传入 `--header "Authorization: Bearer <token>"`，运行爬虫后，验证请求中包含该认证头部，且未配置时不会携带该头部。

**Acceptance Scenarios**:

1. **Given** 命令行指定 `--header "Authorization: Bearer abc123"`，**When** 爬取需要认证的网站，**Then** 请求头中包含 `Authorization: Bearer abc123`
2. **Given** 命令行指定多个头部 `--header "Authorization: Bearer abc123" --header "X-API-Key: xyz789"`，**When** 爬取网站，**Then** 所有指定的头部都出现在请求中
3. **Given** 同时配置了config文件头部和命令行头部，**When** 爬取网站，**Then** 命令行头部优先级更高，会覆盖config文件中的同名头部
4. **Given** 命令行头部格式错误（如缺少冒号），**When** 运行程序，**Then** 显示清晰的错误提示，说明正确的格式，程序退出

---

### User Story 3 - 配置文件热更新和验证 (Priority: P3)

用户修改配置文件后，希望能够验证配置是否正确，以及在长时间运行的任务中能够重新加载配置而无需重启程序。

**Why this priority**: 提升用户体验和调试效率，避免因配置错误导致爬取失败，减少重启程序的时间成本。

**Independent Test**: 用户修改配置文件后，使用验证命令或查看日志确认配置已生效，无需重启程序。

**Acceptance Scenarios**:

1. **Given** 程序提供 `--validate-config` 参数，**When** 用户运行 `jsfindcrack --validate-config`，**Then** 程序读取配置文件并输出验证结果（成功/失败及错误详情）
2. **Given** 配置文件中存在格式错误，**When** 程序启动或验证配置，**Then** 显示具体的错误位置和原因，帮助用户快速定位问题
3. **Given** 配置文件中包含不支持的头部名称或非法字符，**When** 验证配置，**Then** 给出警告或错误提示

---

### Edge Cases

- **配置文件权限不足**：当程序无法在当前目录创建config目录时（如只读文件系统），应给出明确错误提示，建议用户手动创建或更改工作目录
- **配置文件被锁定**：当配置文件被其他进程占用时，程序应优雅降级，使用默认配置并警告用户
- **头部名称冲突**：当config文件和命令行参数指定了相同的头部名称时，明确优先级规则（建议：命令行 > config文件 > 系统默认）
- **超长头部值**：当头部值超过HTTP协议限制（通常8KB）时，应截断或拒绝，并给出警告
- **非ASCII字符**：当头部值包含非ASCII字符时，应进行适当的编码（如URL编码或Base64），避免请求失败
- **空配置文件**：config文件存在但为空时，程序使用系统默认头部，不报错
- **配置文件格式错误**：YAML/JSON解析失败时，显示具体行号和错误原因
- **敏感信息泄露**：日志输出时，应脱敏认证相关头部（如只显示 `Authorization: Bearer ***`）

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 程序首次运行时，MUST自动在当前工作目录下创建 `config/` 目录
- **FR-002**: 程序首次运行时，MUST在 `config/` 目录中生成默认配置文件模板，包含常用头部示例和注释说明
- **FR-003**: 配置文件MUST支持至少一种标准格式（YAML、JSON或TOML），建议使用YAML以便于人工编辑
- **FR-004**: 用户MUST能够在配置文件中定义任意数量的自定义HTTP头部（键值对形式）
- **FR-005**: 程序启动时MUST读取配置文件中的头部设置，并应用到所有HTTP请求中
- **FR-006**: 程序MUST提供命令行参数（如 `--header` 或 `-H`）允许用户在运行时传入临时头部
- **FR-007**: 命令行参数MUST支持多次使用以指定多个头部（如 `--header "A: 1" --header "B: 2"`）
- **FR-008**: 当config文件和命令行同时指定相同头部名称时，命令行参数MUST具有更高优先级
- **FR-009**: 程序MUST验证头部名称和值的合法性（符合HTTP协议规范），拒绝非法头部
- **FR-010**: 配置文件格式错误时，程序MUST显示具体的错误信息（文件路径、行号、错误原因）并退出
- **FR-011**: 程序MUST提供配置验证命令（如 `--validate-config`），用于验证配置文件正确性而不实际执行爬取
- **FR-012**: 日志输出包含认证头部时，MUST自动脱敏敏感信息（如Token、API Key等）
- **FR-013**: 配置文件不存在或为空时，程序MUST使用合理的默认HTTP头部（如标准User-Agent）正常运行

### Key Entities

- **配置文件 (Config File)**: 存储用户自定义的通用HTTP头部，格式为YAML/JSON/TOML，包含头部名称和值的键值对，位于 `config/` 目录下
- **命令行参数 (CLI Arguments)**: 用户通过命令行传入的临时头部，格式为 `--header "Name: Value"`，优先级高于配置文件
- **HTTP请求头部 (HTTP Headers)**: 应用到所有爬取请求的键值对，来源包括：系统默认、配置文件、命令行参数

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 用户无需修改任何代码，仅通过配置文件即可设置所有常用HTTP头部（如User-Agent、Referer、Accept-Language等）
- **SC-002**: 用户能够在5分钟内理解配置文件格式并成功配置自定义头部（通过查看生成的模板和注释）
- **SC-003**: 配置文件格式错误时，程序在1秒内检测并显示错误，不执行任何爬取操作
- **SC-004**: 100%的认证头部在日志中被自动脱敏，不会泄露敏感信息
- **SC-005**: 命令行参数传入的头部能够成功覆盖配置文件中的同名头部，验证优先级机制正确实施
- **SC-006**: 90%的常见目标网站能够通过配置User-Agent和Referer成功绕过基础反爬虫检测

## Assumptions

1. **配置文件格式**: 默认使用YAML格式，因为其易读性强且支持注释，适合人工编辑
2. **配置文件位置**: 默认生成在当前工作目录的 `config/` 子目录，用户也可通过环境变量或命令行参数指定其他路径
3. **头部优先级**: 命令行参数 > 配置文件 > 系统默认，确保临时需求优先且不破坏已有配置
4. **敏感头部识别**: 自动识别包含 `Authorization`、`Token`、`Key`、`Secret` 等关键字的头部进行脱敏
5. **默认头部**: 当无任何配置时，使用浏览器标准User-Agent（如Chrome或Firefox的User-Agent字符串）
6. **配置热更新**: 初始版本不支持运行时热更新，需重启程序生效（P3故事可选实现）
