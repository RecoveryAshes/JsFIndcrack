# Tasks: Git仓库清理和最终化

**Input**: 设计文档来自 `/specs/004-git-repo-finalize/`
**Prerequisites**: plan.md (已有), spec.md (已有), quickstart.md (已有)

**Tests**: 本功能为简单文件清理和Git操作,采用手动验证方式,无需单元测试。

**Organization**: 任务按清理阶段分组,按依赖顺序执行。

## Format: `[ID] [P?] Description`

- **[P]**: 可并行执行(不同文件,无依赖)
- 描述中包含准确的文件路径和操作说明

## Path Conventions

本项目为单一Go项目结构:
- 项目根目录: `/Users/recovery/opt/课程/go/JsFIndcrack`
- 待删除文件: 根目录下的10个文件/目录
- 待更新文件: `.gitignore`, `README.md`
- 文档: `specs/004-git-repo-finalize/`

---

## Phase 1: 预清理检查和准备 (Pre-Cleanup Validation)

**Purpose**: 验证环境安全性,创建备份点,确保后续操作可恢复

- [ ] T001 验证Git工作区状态: 运行 `git status` 检查是否有未提交的更改
- [ ] T002 验证Go项目完整性: 运行 `go test ./...` 确保所有测试通过
- [ ] T003 [P] 验证Go项目可编译: 运行 `make build` 或 `go build ./cmd/jsfindcrack`
- [ ] T004 创建清理前Git备份标签: `git tag -a "pre-cleanup-004" -m "清理前备份点"`
- [ ] T005 [P] 记录待删除文件大小: 运行 `du -sh dist/ jsfindcrack config/ -/ output/ logs/` 统计空间占用
- [ ] T006 验证关键目录存在性: 检查 `configs/`, `cmd/`, `internal/`, `tests/` 目录存在

**验证**: Git工作区状态已知,Go测试通过,备份标签已创建

---

## Phase 2: 文件预览和验证 (Dry-Run Validation)

**Purpose**: 预览待删除文件,确认删除清单准确性,验证白名单保护

- [ ] T007 [P] 检查构建产物存在性: 验证 `dist/` 和 `jsfindcrack` 存在
- [ ] T008 [P] 检查遗留Python文件: 验证 `config/` 和 `install.sh` 存在
- [ ] T009 [P] 检查临时测试文件: 验证 `-/`, `test_urls.txt`, `Go并发和流式IO最佳实践研究.md` 存在
- [ ] T010 [P] 检查输出日志目录: 验证 `output/` 和 `logs/` 存在
- [ ] T011 [P] 检查系统文件: 运行 `find . -name ".DS_Store"` 查找系统文件
- [ ] T012 白名单验证: 确认 `configs/` (非 `config/`) 不在删除列表中
- [ ] T013 白名单验证: 确认 `cmd/`, `internal/`, `tests/` 不在删除列表中
- [ ] T014 白名单验证: 确认 `go.mod`, `go.sum`, `Makefile` 不在删除列表中

**验证**: 所有待删除文件存在,关键文件受白名单保护

---

## Phase 3: 执行清理操作 (File Deletion Execution)

**Purpose**: 删除构建产物、遗留Python文件、临时文件、输出日志、系统文件

**⚠️ CRITICAL**: 此阶段操作不可撤销(除非通过Git恢复),执行前务必确认T001-T014全部完成

### 删除构建产物

- [ ] T015 删除dist目录: `rm -rf dist/` (263MB - Go交叉编译产物)
- [ ] T016 删除根目录二进制: `rm -f jsfindcrack` (22MB - 根目录编译产物)

### 删除遗留Python文件

- [ ] T017 删除config目录: `rm -rf config/` (~10KB - Python配置目录)
- [ ] T018 删除Python安装脚本: `rm -f install.sh` (3.5KB - Python安装脚本)

### 删除临时和测试文件

- [ ] T019 删除临时测试目录: `rm -rf -/` (~7KB - 临时测试输出)
- [ ] T020 删除测试URL文件: `rm -f test_urls.txt` (<1KB - URL测试列表)
- [ ] T021 删除研究文档: `rm -f "Go并发和流式IO最佳实践研究.md"` (37KB - 研究文档)

### 删除输出和日志

- [ ] T022 删除output目录: `rm -rf output/` (2.4MB - 爬取结果)
- [ ] T023 删除logs目录: `rm -rf logs/` (856KB - 日志文件)

### 删除系统文件

- [ ] T024 删除.DS_Store文件: `find . -name ".DS_Store" -delete` (6KB - macOS系统文件)

**Checkpoint**: 所有待删除文件已删除,验证文件不存在

---

## Phase 4: 验证清理结果 (Post-Deletion Verification)

**Purpose**: 确认文件已删除,关键文件完整,Go项目功能正常

- [ ] T025 验证构建产物已删除: 确认 `dist/` 和 `jsfindcrack` 不存在
- [ ] T026 验证Python文件已删除: 确认 `config/` 和 `install.sh` 不存在
- [ ] T027 验证临时文件已删除: 确认 `-/`, `test_urls.txt`, 研究文档 不存在
- [ ] T028 验证输出日志已删除: 确认 `output/` 和 `logs/` 不存在
- [ ] T029 验证关键目录保留: 确认 `configs/`, `cmd/`, `internal/`, `tests/` 存在
- [ ] T030 验证关键文件保留: 确认 `go.mod`, `go.sum`, `Makefile`, `.gitignore`, `README.md` 存在
- [ ] T031 验证Go项目完整性: 运行 `go test ./...` 确保测试100%通过
- [ ] T032 验证Go项目可编译: 运行 `go build ./cmd/jsfindcrack` 成功编译

**Checkpoint**: 清理成功,Go项目功能完整

---

## Phase 5: 更新配置文件 (Configuration Updates)

**Purpose**: 更新.gitignore和README.md,确保配置准确反映当前仓库状态

### 更新.gitignore

- [ ] T033 读取当前.gitignore内容: 使用Read工具读取 `.gitignore`
- [ ] T034 检查现有忽略规则: 验证 `dist/`, `output/`, `logs/`, `.DS_Store` 规则是否存在
- [ ] T035 追加jsfindcrack忽略规则: 在.gitignore中添加 `/jsfindcrack` 和 `jsfindcrack-*` (如不存在)
- [ ] T036 追加特殊目录忽略规则: 在.gitignore中添加 `-/` (如不存在)
- [ ] T037 验证.gitignore完整性: 确认包含所有必要的忽略规则

### 更新README.md

- [ ] T038 读取当前README.md内容: 使用Read工具读取 `README.md`
- [ ] T039 移除install.sh引用: 删除README中关于Python安装脚本的说明
- [ ] T040 更新config目录说明: 移除 `config/` 引用,确保仅保留 `configs/` 说明
- [ ] T041 更新项目结构部分: 确保项目结构树不包含已删除文件
- [ ] T042 验证README准确性: 确认README不包含已删除文件引用

**Checkpoint**: .gitignore和README.md已更新

---

## Phase 6: Git提交和推送 (Git Commit & Push)

**Purpose**: 暂存所有变更,创建描述性提交,推送到远程仓库

- [ ] T043 暂存所有变更: `git add -A` 暂存删除和修改
- [ ] T044 检查暂存状态: `git status` 确认变更列表正确
- [ ] T045 创建提交: 使用约定式提交格式创建commit,包含清理摘要和文件列表
- [ ] T046 验证提交内容: `git show HEAD` 检查提交详情
- [ ] T047 推送到远程: `git push origin 004-git-repo-finalize` 推送到feature分支

**提交消息模板**:
```
chore(004): 清理Python遗留文件和构建产物

- 删除构建产物: dist/, jsfindcrack (285MB)
- 删除遗留Python配置: config/, install.sh
- 删除临时文件: -/, test_urls.txt, 研究文档
- 删除输出日志: output/, logs/
- 删除系统文件: .DS_Store
- 更新.gitignore: 添加jsfindcrack和-/忽略规则
- 更新README.md: 移除已删除文件引用

完成Python到Go迁移的最终清理,仓库体积减少285MB

Related: specs/004-git-repo-finalize/

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Checkpoint**: 所有变更已提交并推送到远程仓库

---

## Phase 7: 最终验证和文档 (Final Verification & Documentation)

**Purpose**: 全面验证清理成果,确保所有成功标准达成

- [ ] T048 验证Git工作区干净: `git status` 应显示 "nothing to commit, working tree clean"
- [ ] T049 验证远程推送成功: `git log origin/004-git-repo-finalize` 确认提交已推送
- [ ] T050 运行完整Go测试: `go test -v ./...` 确保100%通过
- [ ] T051 验证Go构建成功: `make build` 或 `go build ./cmd/jsfindcrack` 成功
- [ ] T052 统计清理成果: 运行 `du -sh .` 对比清理前大小,确认减少~285MB
- [ ] T053 验证文件数量: 确认删除≥10个文件/目录
- [ ] T054 [P] 更新tasks.md: 标记所有任务为已完成
- [ ] T055 [P] 创建清理报告: (可选) 在 `specs/004-git-repo-finalize/reports/` 创建清理摘要

**Checkpoint**: 所有功能完成,清理成果符合预期

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (预清理检查)**: 无依赖 - 可立即开始
- **Phase 2 (文件预览)**: 依赖Phase 1完成 - 需要环境验证通过
- **Phase 3 (执行清理)**: 依赖Phase 1和Phase 2完成 - **阻塞**后续所有阶段
- **Phase 4 (验证清理)**: 依赖Phase 3完成 - 验证删除结果
- **Phase 5 (更新配置)**: 依赖Phase 3完成 - 可与Phase 4并行
- **Phase 6 (Git提交)**: 依赖Phase 4和Phase 5完成 - 需要验证通过
- **Phase 7 (最终验证)**: 依赖Phase 6完成 - 全面验证成果

### Task Dependencies

#### Phase 1内部依赖
- T001 → T004 (Git状态检查后再创建标签)
- T002, T003 无依赖,可并行

#### Phase 2内部依赖
- T007-T011 无依赖,可并行
- T012-T014 依赖T007-T011完成

#### Phase 3内部依赖
- T015-T024 可按分类并行(每个子类别内顺序执行)
  - 构建产物: T015, T016
  - Python文件: T017, T018
  - 临时文件: T019, T020, T021
  - 输出日志: T022, T023
  - 系统文件: T024

#### Phase 4内部依赖
- T025-T030 可并行(验证操作)
- T031, T032 依赖T025-T030完成

#### Phase 5内部依赖
- .gitignore更新: T033 → T034 → T035 → T036 → T037 (顺序执行)
- README更新: T038 → T039 → T040 → T041 → T042 (顺序执行)
- 两个子类别可并行

#### Phase 6内部依赖
- T043 → T044 → T045 → T046 → T047 (严格顺序执行)

#### Phase 7内部依赖
- T048-T053 可并行(验证操作)
- T054, T055 可并行(文档操作)

### Parallel Opportunities

**Phase 1并行**:
- T002 (Go测试) || T003 (Go编译) || T005 (统计大小) || T006 (验证目录)

**Phase 2并行**:
- T007 || T008 || T009 || T010 || T011 (所有文件检查)

**Phase 3并行** (分类内部并行):
- (T015 || T016) && (T017 || T018) && (T019 || T020 || T021) && (T022 || T023) && T024

**Phase 4并行**:
- T025 || T026 || T027 || T028 || T029 || T030 (所有验证检查)

**Phase 5并行**:
- (.gitignore更新序列) || (README更新序列)

**Phase 7并行**:
- T048 || T049 || T050 || T051 || T052 || T053 (所有最终验证)
- T054 || T055 (文档更新)

---

## Implementation Strategy

### 推荐执行顺序 (Sequential - 适用于单人操作)

1. **Phase 1**: 完成所有预清理检查 (T001-T006)
   - 关键: 创建Git备份标签
2. **Phase 2**: 完成所有文件预览和验证 (T007-T014)
   - 关键: 白名单验证通过
3. **Phase 3**: 执行清理操作 (T015-T024)
   - 关键: 谨慎操作,不可撤销
4. **CHECKPOINT**: 验证文件已删除
5. **Phase 4**: 验证清理结果 (T025-T032)
   - 关键: Go测试必须100%通过
6. **Phase 5**: 更新配置文件 (T033-T042)
   - 可与Phase 4部分并行
7. **Phase 6**: Git提交和推送 (T043-T047)
   - 关键: 提交消息描述准确
8. **Phase 7**: 最终验证 (T048-T055)
   - 关键: 所有成功标准达成

### 快速执行路径 (MVP - 仅必要任务)

如需快速完成,可跳过以下可选任务:
- T005 (记录文件大小 - 可选)
- T006 (验证目录存在 - 可选,已在T029验证)
- T046 (验证提交内容 - 可选)
- T052 (统计清理成果 - 可选)
- T055 (创建清理报告 - 可选)

**最小MVP任务集**: T001-T004, T007-T014, T015-T024, T025-T032, T033-T042, T043-T045, T047-T051, T054

---

## Notes

- **[P]** 标记 = 不同文件或独立操作,可并行执行
- 所有文件路径均为项目根目录相对路径
- Phase 3操作不可撤销,执行前务必完成Phase 1-2验证
- Git备份标签 `pre-cleanup-004` 可用于紧急回滚
- 每个Phase完成后建议提交一次(可选)
- T043-T047为Git操作,必须严格顺序执行
- 最终验证(Phase 7)确保所有4个用户故事的成功标准达成

---

## Summary

**总任务数**: 55个任务
**任务分布**:
- Phase 1 预清理检查: 6个任务
- Phase 2 文件预览: 8个任务
- Phase 3 执行清理: 10个任务
- Phase 4 验证清理: 8个任务
- Phase 5 更新配置: 10个任务 (5 .gitignore + 5 README)
- Phase 6 Git提交: 5个任务
- Phase 7 最终验证: 8个任务

**并行机会**: 约20个任务标记为[P],可在各自Phase内并行执行

**关键检查点**:
1. Phase 1完成: 环境安全,备份已创建
2. Phase 2完成: 白名单验证通过
3. Phase 3完成: 文件已删除
4. Phase 4完成: Go项目功能完整
5. Phase 5完成: 配置文件已更新
6. Phase 6完成: 变更已推送
7. Phase 7完成: 所有成功标准达成

**预计耗时**: 约30分钟(手动执行),或10分钟(使用自动化脚本)

**成功标准**:
- ✅ 删除≥10个文件/目录(~285MB)
- ✅ .gitignore和README.md更新准确
- ✅ Go测试100%通过
- ✅ Git工作区干净
- ✅ 变更已推送到远程

**格式验证**: ✅ 所有任务遵循清单格式 (checkbox + TaskID + [P]? + 描述 + 文件路径/命令)
