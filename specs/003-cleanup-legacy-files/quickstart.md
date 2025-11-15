# 快速入门: Python文件清理脚本

**功能**: 003-cleanup-legacy-files
**脚本路径**: `scripts/cleanup-python.sh`
**更新日期**: 2025-11-15

## 概述

`cleanup-python.sh`是一个安全的Python遗留文件清理工具,用于在Python到Go迁移完成后删除所有Python相关文件,同时确保Go代码和关键配置不受影响。

## 前置条件

在运行清理脚本前,请确保:

1. ✅ Python到Go的迁移已完全完成
2. ✅ 所有Go功能测试已通过: `make test` 或 `go test ./...`
3. ✅ Git工作区干净(无未提交的更改): `git status`
4. ✅ 已在`feature/003-cleanup-legacy-files`分支或创建了备份分支

## 基本用法

### 第一步: 预览待删除文件 (推荐)

使用干跑模式查看将要删除的文件,不执行任何实际删除操作:

```bash
cd /path/to/JsFIndcrack
./scripts/cleanup-python.sh --dry-run
```

**输出示例**:
```
[INFO] 2025-11-15 10:30:00 - Python文件清理工具 (干跑模式)
[INFO] 扫描Python文件...

=== 待删除文件清单 ===

【Python源文件】 (18个文件, 45.2 KB)
  - ./main.py (3.5 KB)
  - ./src/core/config.py (2.1 KB)
  - ./src/core/js_crawler.py (8.3 KB)
  ... (更多文件)

【Python配置文件】 (1个文件, 0.8 KB)
  - ./requirements.txt (0.8 KB)

【Python构建产物】 (0个文件, 0 KB)
  (未发现缓存文件)

【目录】 (4个目录)
  - ./src/
  - ./src/core/
  - ./src/crawlers/
  - ./src/utils/

=== 清理摘要 ===
总计: 19个文件, 4个目录, 约46.0 KB

=== 白名单验证 ===
✅ Go源代码: cmd/, internal/ (保留)
✅ Go配置: go.mod, go.sum (保留)
✅ 构建配置: Makefile (保留)
✅ 文档: specs/, .specify/ (保留)
✅ 测试: tests/ (保留)

[INFO] 干跑模式完成,未执行任何删除操作
[INFO] 如需执行清理,请运行: ./scripts/cleanup-python.sh --execute
```

### 第二步: 创建Git备份点

在执行清理前,创建Git标签作为恢复点:

```bash
# 确保工作区干净
git status

# 如有未提交更改,先提交
git add -A
git commit -m "chore: 清理Python文件前的保存点"

# 创建备份标签
git tag -a "before-python-cleanup-$(date +%Y%m%d)" -m "Python清理前备份"
```

### 第三步: 执行清理

使用执行模式运行清理,脚本会要求确认:

```bash
./scripts/cleanup-python.sh --execute
```

**交互流程**:
```
[INFO] 2025-11-15 10:35:00 - Python文件清理工具 (执行模式)
[WARN] ⚠️  即将删除以下文件和目录:
       - 19个Python文件
       - 4个目录
       - 总大小: 46.0 KB

[WARN] ⚠️  此操作不可撤销! (除非通过Git恢复)

[提示] 是否继续清理? 请输入 'yes' 确认,其他任何输入取消: yes

[INFO] 开始清理...
[INFO] 删除文件: ./main.py
[INFO] 删除文件: ./src/core/config.py
...
[INFO] 删除目录: ./src/
[INFO] 清理完成!

[INFO] 生成清理报告: ./specs/003-cleanup-legacy-files/reports/cleanup-report-20251115-103500.json
[INFO] 生成摘要文件: ./specs/003-cleanup-legacy-files/reports/cleanup-summary-20251115-103500.md
```

### 第四步: 验证清理结果

清理完成后,验证Go项目功能完整性:

```bash
# 1. 验证关键文件存在
ls cmd/ internal/ go.mod Makefile

# 2. 运行测试套件
make test
# 或
go test -v ./...

# 3. 尝试构建项目
make build
# 或
go build ./cmd/jsfindcrack

# 4. 查看清理报告
cat specs/003-cleanup-legacy-files/reports/cleanup-summary-*.md
```

## 高级选项

### 详细预览模式

显示更详细的文件信息(大小、修改时间):

```bash
./scripts/cleanup-python.sh --preview
```

### 静默执行(CI/CD用)

跳过交互确认,直接执行(谨慎使用):

```bash
./scripts/cleanup-python.sh --execute --force
```

**警告**: `--force`选项会跳过确认,仅在自动化环境中使用,且确保已有备份。

### 仅显示文件列表

只显示文件路径,不显示详细信息(用于管道处理):

```bash
./scripts/cleanup-python.sh --list-only
```

输出格式:
```
./main.py
./src/core/config.py
./src/core/js_crawler.py
...
```

## 恢复指南

如果清理后发现问题,使用以下方法恢复:

### 方法1: 恢复到备份标签(推荐)

```bash
# 查看备份标签
git tag -l "before-python-cleanup-*"

# 恢复到备份点(硬重置,丢弃清理后的更改)
git reset --hard before-python-cleanup-20251115

# 或创建新分支查看备份内容
git checkout -b restore-python before-python-cleanup-20251115
```

### 方法2: 恢复特定文件

```bash
# 查看Git历史
git log --oneline --all --graph

# 恢复特定文件或目录
git checkout <commit-hash> -- src/ main.py requirements.txt
```

### 方法3: 使用Git Reflog

如果忘记创建标签,使用reflog查找清理前的commit:

```bash
git reflog
# 找到清理前的commit ID,例如: abc1234

git reset --hard abc1234
```

## 故障排查

### 问题1: 脚本执行失败 "permission denied"

**原因**: 脚本没有执行权限

**解决方案**:
```bash
chmod +x scripts/cleanup-python.sh
```

### 问题2: Git工作区不干净警告

**原因**: 存在未提交的更改

**解决方案**:
```bash
# 查看未提交的更改
git status

# 选项A: 提交更改
git add -A
git commit -m "保存当前工作"

# 选项B: 暂存更改
git stash push -m "临时暂存"
```

### 问题3: 白名单验证失败

**原因**: 关键文件缺失或路径不正确

**解决方案**:
```bash
# 检查当前目录
pwd
# 应该在项目根目录

# 验证关键文件
ls -la go.mod Makefile cmd/ internal/

# 如果缺失,说明可能在错误的目录或项目结构异常
```

### 问题4: 清理后测试失败

**原因**: 可能误删了测试依赖文件或Go代码有问题

**解决方案**:
```bash
# 1. 查看测试错误详情
go test -v ./...

# 2. 如果是文件缺失,恢复Git
git reset --hard before-python-cleanup-*

# 3. 如果是Go代码问题,说明清理前测试可能未通过
# 先修复Go代码,再执行清理
```

## 清理报告说明

### JSON报告

路径: `specs/003-cleanup-legacy-files/reports/cleanup-report-<timestamp>.json`

用途: 程序化处理、自动化验证、CI/CD集成

示例字段:
```json
{
  "timestamp": "2025-11-15T10:35:00Z",
  "operation": "python-cleanup",
  "summary": {
    "files_deleted": 19,
    "directories_deleted": 4,
    "total_size_kb": 46.0
  },
  "categories": { ... },
  "preserved_files_verified": true
}
```

### Markdown摘要

路径: `specs/003-cleanup-legacy-files/reports/cleanup-summary-<timestamp>.md`

用途: 人类阅读、归档记录、审计追踪

包含内容:
- 执行时间和模式
- 删除文件清单(按类别)
- 验证结果
- 后续操作建议

## 最佳实践

1. **始终先运行干跑模式**: 确认清理范围符合预期
2. **创建Git备份点**: 确保可以快速恢复
3. **在非主分支操作**: 建议在feature分支执行,验证无误后合并
4. **保留清理报告**: 记录操作历史,便于审计
5. **团队协作提醒**: 如果多人协作,通知团队成员清理计划

## 常见问题 (FAQ)

**Q: 是否可以在主分支直接执行清理?**

A: 不推荐。建议在feature分支执行,验证后再合并到主分支。

**Q: 清理后可以重新添加Python文件吗?**

A: 可以,但这违背了迁移到Go的初衷。如确需添加,确保新Python文件用途明确。

**Q: 如果只想删除部分Python文件怎么办?**

A: 本脚本设计为完整清理。如需部分清理,请手动删除或修改脚本的黑名单规则。

**Q: Windows系统可以使用吗?**

A: 本脚本为Bash编写,需要在Git Bash、WSL或Cygwin环境下运行。

**Q: 是否会删除tests/目录下的Python测试文件?**

A: 脚本设计会检查tests/目录。如果tests/下有.py文件,需要确认:
   - 如果是Python工具的测试(已废弃),可以删除
   - 如果是Go项目的Python测试工具(如集成测试脚本),应保留
   - 建议查看干跑模式输出,手动确认

## 支持与反馈

如遇到问题或有改进建议,请:
1. 查看清理报告中的错误信息
2. 检查Git状态和reflog
3. 在项目issues中提交问题,附带清理报告和错误日志

## 相关文档

- [功能规范](./spec.md) - 了解清理需求和验收标准
- [实施计划](./plan.md) - 了解技术方案和设计决策
- [研究文档](./research.md) - 了解技术选型依据
- [任务清单](./tasks.md) - 了解实施步骤(由`/speckit.tasks`生成)
