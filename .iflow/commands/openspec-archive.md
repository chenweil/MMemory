---
name: openspec-archive
description: 归档完成的变更并更新主规范
license: MIT
workflow_trigger: /openspec-archive
agent_path: .iflow/agents/openspec
---

# OpenSpec 变更归档

## Command Activation

当用户调用 `/openspec-archive` 命令时，系统**必须**执行变更归档工作流。

### Environment Setup

**设置工作目录和环境:**
```bash
cd .iflow/agents/openspec
export NODE_PATH=$(pwd)/node_modules
```

### Workflow Decision Logic

**专注于变更归档工作流:**

| 用户请求 | 工作流动作 | 必需阅读 |
|-------------|-----------------|---------------------|
| 归档完成变更 | 执行 "archive workflow" 工作流 | 阅读变更文档和归档指南 |
| 更新主规范 | 执行 "spec merge" 工作流 | 阅读规范增量和主规范 |
| 清理变更目录 | 执行 "cleanup" 工作流 | 确认变更完成状态 |

### 自动依赖管理

**依赖检测和安装策略:**

该技能遵循"延迟安装"方法:
1. **假设依赖已安装** - 直接执行，无需预检查
2. **失败时安装** - 仅在命令失败时安装依赖
3. **一次性设置** - 安装后，依赖在会话间持续

## 关键执行规则

执行变更归档工作流时，系统**必须**遵循以下规则：

1. **验证完成状态** - 确认所有任务已完成且通过验证
2. **安全归档** - 确保变更内容正确合并到主规范
3. **保持历史记录** - 将变更移动到 archive/ 目录保存历史
4. **更新主规范** - 将规范增量合并到 openspec/specs/ 中
5. **清理工作区** - 确保活跃变更目录整洁
6. **验证结果** - 归档后验证主规范的完整性

## 归档完成的变更

### 归档前检查

1. **确认变更完成状态**:
   ```bash
   # 显示变更详情
   node bin/openspec.js show [change-id]
   
   # 验证变更正确性
   node bin/openspec.js validate [change-id] --strict
   ```

2. **检查任务完成情况**:
   - 确认 `tasks.md` 中所有任务都标记为 `[x]`
   - 验证相关代码已实施并通过测试
   - 确认功能符合规范要求

3. **审查规范增量**:
   - 检查 `specs/` 目录中的规范增量
   - 确认增量内容准确且完整
   - 验证规范格式符合 OpenSpec 标准

### 归档操作

#### 交互式归档
```bash
# 交互式归档（会提示确认）
node bin/openspec.js archive [change-id]
```

#### 非交互式归档
```bash
# 自动归档（跳过确认提示）
node bin/openspec.js archive [change-id] --yes
```

#### 特殊归档选项
```bash
# 仅工具变更归档（跳过规范更新）
# 适用于工具、文档、基础设施等不涉及功能规范的变更
node bin/openspec.js archive [change-id] --skip-specs --yes

# 跳过验证归档（不推荐）
node bin/openspec.js archive [change-id] --no-validate --yes
```

### 归档流程说明

1. **规范合并**:
   - 将 `openspec/changes/[change-id]/specs/` 中的增量合并到 `openspec/specs/`
   - 处理 ADDED、MODIFIED、REMOVED 类型的规范变更
   - 确保主规范保持一致性和完整性

2. **历史保存**:
   - 将整个变更目录移动到 `openspec/changes/archive/[change-id]/`
   - 保留完整的变更历史记录
   - 维护提案、任务、设计文档的可追溯性

3. **状态更新**:
   - 从活跃变更列表中移除已归档的变更
   - 更新项目状态和进度跟踪
   - 清理临时文件和工作目录

## 归档后验证

### 验证主规范
```bash
# 验证所有规范
node bin/openspec.js validate --specs --strict

# 验证特定规范
node bin/openspec.js validate [spec-name] --type spec --strict
```

### 检查归档状态
```bash
# 列出活跃变更（应该不包含已归档的变更）
node bin/openspec.js list

# 检查归档目录
ls openspec/changes/archive/

# 查看归档的变更
node bin/openspec.js show archive/[change-id]
```

## 常用命令

### 归档管理
```bash
# 列出可归档的变更
node bin/openspec.js list

# 批量归档多个变更
for change in change-1 change-2 change-3; do
  node bin/openspec.js archive $change --yes
done
```

### 状态检查
```bash
# 查看归档历史
ls -la openspec/changes/archive/

# 验证归档后的规范状态
node bin/openspec.js list --specs
node bin/openspec.js validate --all --strict
```

## 归档最佳实践

1. **及时归档** - 功能完成后立即归档，避免积累过多活跃变更
2. **验证先行** - 归档前必须通过严格验证
3. **保留历史** - 归档的变更提供重要的决策和演进历史
4. **批量处理** - 可以批量归档多个完成的变更提高效率
5. **定期清理** - 定期检查和维护归档目录的组织结构

## 故障排除

### 归档失败处理
```bash
# 如果归档失败，检查错误原因
node bin/openspec.js validate [change-id] --strict

# 手动检查任务完成状态
grep "- \[ \]" openspec/changes/[change-id]/tasks.md

# 检查规范增量格式
cat openspec/changes/[change-id]/specs/*/spec.md
```

### 回滚归档操作
如果需要撤销归档操作：
```bash
# 从归档目录移回活跃目录
mv openspec/changes/archive/[change-id] openspec/changes/
```

## 依赖项

必需依赖项（应该已安装）：

- **@fission-ai/openspec**: `npm install -g @fission-ai/openspec` (用于CLI功能)
- **typescript**: `npm install -g typescript` (用于TypeScript支持)
- **tsx**: `npm install -g tsx` (用于TypeScript执行)
- **Node.js**: v20.19.0+ (运行时环境)

开发依赖项在package.json中定义，包括测试框架和构建工具。