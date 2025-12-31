---
name: openspec-apply
description: 实施变更提案中的任务和规范
license: MIT
workflow_trigger: /openspec-apply
agent_path: .iflow/agents/openspec
---

# OpenSpec 变更实施

## Command Activation

当用户调用 `/openspec-apply` 命令时，系统**必须**执行变更实施工作流。

### Environment Setup

**设置工作目录和环境:**
```bash
cd .iflow/agents/openspec
export NODE_PATH=$(pwd)/node_modules
```

### Workflow Decision Logic

**专注于变更实施工作流:**

| 用户请求 | 工作流动作 | 必需阅读 |
|-------------|-----------------|---------------------|
| 实施变更任务 | 执行 "apply implementation" 工作流 | 阅读变更的 tasks.md 和 specs |
| 执行具体任务 | 执行 "task execution" 工作流 | 阅读相关规范和设计文档 |
| 更新任务状态 | 执行 "task tracking" 工作流 | 阅读 tasks.md 进度 |

### 自动依赖管理

**依赖检测和安装策略:**

该技能遵循"延迟安装"方法:
1. **假设依赖已安装** - 直接执行，无需预检查
2. **失败时安装** - 仅在命令失败时安装依赖
3. **一次性设置** - 安装后，依赖在会话间持续

## 关键执行规则

执行变更实施工作流时，系统**必须**遵循以下规则：

1. **理解变更上下文** - 完整读取变更提案、任务清单和相关规范
2. **按步骤执行** - 严格按照 tasks.md 中的任务顺序执行
3. **更新任务状态** - 完成任务后及时更新 tasks.md 中的复选框状态
4. **遵循规范要求** - 确保实施符合 specs/ 目录中定义的技术规范
5. **验证实施结果** - 使用测试和验证确保代码质量
6. **记录实施过程** - 在必要时更新 design.md 记录技术决策

## 实施变更任务

### 准备工作

1. **读取变更信息**:
   ```bash
   # 显示变更详情
   node bin/openspec.js show [change-id]
   
   # 获取JSON格式用于脚本处理
   node bin/openspec.js show [change-id] --json
   ```

2. **分析任务结构**:
   - 读取 `openspec/changes/[change-id]/tasks.md`
   - 理解任务依赖关系和优先级
   - 确认前置条件已满足

3. **审查技术规范**:
   - 读取 `openspec/changes/[change-id]/specs/` 下的规范增量
   - 理解需要实现的功能要求
   - 检查设计约束和技术决策

### 实施流程

1. **逐步执行任务**:
   - 按 tasks.md 中的顺序执行每个任务
   - 实施代码时遵循项目的编码规范
   - 确保实现符合规范中的功能要求

2. **更新任务状态**:
   ```bash
   # 编辑 tasks.md 标记完成的任务
   # 将 - [ ] 改为 - [x]
   ```

3. **持续验证**:
   ```bash
   # 验证变更仍然有效
   node bin/openspec.js validate [change-id] --strict
   ```

4. **测试和质量保证**:
   - 运行相关测试确保功能正确
   - 执行代码检查和格式化
   - 验证性能和安全要求

### 任务跟踪

**任务状态管理:**
- `- [ ]` 未开始的任务
- `- [x]` 已完成的任务
- `- [~]` 进行中的任务（可选标记）
- `- [-]` 已跳过/不适用的任务

**进度报告:**
```bash
# 查看当前任务完成情况
grep -c "- \[x\]" openspec/changes/[change-id]/tasks.md
grep -c "- \[ \]" openspec/changes/[change-id]/tasks.md
```

## 常用命令

### 查看变更状态
```bash
# 列出活跃变更
node bin/openspec.js list

# 显示特定变更详情
node bin/openspec.js show [change-id]

# 验证变更正确性
node bin/openspec.js validate [change-id] --strict
```

### 管理实施过程
```bash
# 查看任务列表
cat openspec/changes/[change-id]/tasks.md

# 查看技术规范
cat openspec/changes/[change-id]/specs/*/spec.md

# 查看设计文档
cat openspec/changes/[change-id]/design.md
```

## 实施完成标准

变更实施完成需要满足以下条件：

1. **所有任务完成** - tasks.md 中所有任务都标记为 [x]
2. **通过验证** - `openspec validate [change-id] --strict` 无错误
3. **代码测试通过** - 相关单元测试和集成测试通过
4. **符合规范要求** - 实现满足 specs/ 中定义的所有功能需求
5. **文档更新** - 必要的技术文档已同步更新

完成所有任务后，变更就可以进行归档流程。

## 依赖项

必需依赖项（应该已安装）：

- **@fission-ai/openspec**: `npm install -g @fission-ai/openspec` (用于CLI功能)
- **typescript**: `npm install -g typescript` (用于TypeScript支持)
- **tsx**: `npm install -g tsx` (用于TypeScript执行)
- **Node.js**: v20.19.0+ (运行时环境)

开发依赖项在package.json中定义，包括测试框架和构建工具。