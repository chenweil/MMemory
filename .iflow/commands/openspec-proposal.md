---
name: openspec-proposal
description: 创建新的变更提案和技术规范
license: MIT
workflow_trigger: /openspec-proposal
agent_path: .iflow/agents/openspec
---

# OpenSpec 变更提案创建

## Command Activation

当用户调用 `/openspec-proposal` 命令时，系统**必须**执行变更提案创建工作流。

### Environment Setup

**设置工作目录和环境:**
```bash
cd .iflow/agents/openspec
export NODE_PATH=$(pwd)/node_modules
```

### Workflow Decision Logic

**专注于变更提案创建工作流:**

| 用户请求 | 工作流动作 | 必需阅读 |
|-------------|-----------------|---------------------|
| 创建变更提案 | 执行 "proposal creation" 工作流 | 阅读 `AGENTS.md` (完整文件, ~455行) |
| 创建新功能规范 | 执行 "new spec creation" 工作流 | 阅读 `AGENTS.md` 模板部分 |

### 自动依赖管理

**依赖检测和安装策略:**

该技能遵循"延迟安装"方法:
1. **假设依赖已安装** - 直接执行，无需预检查
2. **失败时安装** - 仅在命令失败时安装依赖
3. **一次性设置** - 安装后，依赖在会话间持续

**按平台安装命令:**

```bash
# Node.js 包
npm install -g @fission-ai/openspec

# 开发依赖（在智能体目录内）
npm install

# TypeScript 全局工具
npm install -g typescript tsx
```

## 关键执行规则

执行变更提案创建工作流时，系统**必须**遵循以下规则：

1. **总是阅读完整文档** - 阅读 `AGENTS.md` 时不使用 `offset` 或 `limit`，确保获取完整上下文。
2. **设计原则优先** - 新变更提案使用 proposal creation，始终考虑内容驱动的设计选择并在编写代码前解释。
3. **生成并执行脚本** - 不要尝试直接操作文件，而应生成并运行适当的命令。
4. **优雅处理依赖** - 遵循延迟安装策略。
5. **设置环境变量** - 确保在执行脚本前正确设置 `NODE_PATH` 和工作目录。
6. **验证输出** - 使用验证命令确保结果正确。

## 创建新的变更提案

当从头创建新的变更提案时，使用**OpenSpec工作流**来创建结构化的提案。

### 设计原则

**关键**: 在创建任何变更提案前，分析内容并选择适当的设计元素：
1. **考虑主题**: 这个变更是关于什么的？它暗示什么语调、行业或情绪？
2. **检查品牌**: 如果用户提到公司/组织，考虑他们的品牌和身份
3. **匹配内容范围**: 选择反映主题的结构和格式
4. **说明您的方法**: 在编写代码前解释您的设计选择

### 工作流

1. **强制 - 阅读完整文件**: 从头到尾完整阅读 [`AGENTS.md`](AGENTS.md)。**阅读此文件时不设任何范围限制。** 在继续创建变更提案之前，阅读完整文件内容以了解详细语法、关键格式规则和最佳实践。
2. **探索现有状态**: 使用 `openspec list` 和 `openspec list --specs` 了解当前上下文
3. **选择唯一的变更ID**: 使用kebab-case，动词开头 (`add-`, `update-`, `remove-`, `refactor-`)
4. **创建目录结构**: 
   ```bash
   mkdir -p openspec/changes/[change-id]/{specs/[capability]}
   ```
5. **编写提案文件**:
   - `proposal.md`: 为什么、改变什么、影响
   - `tasks.md`: 实现检查清单
   - `design.md`: 技术决策(可选)
   - `specs/[capability]/spec.md`: 增量变更
6. **验证**: 运行 `node bin/openspec.js validate [change-id] --strict` 并修复问题

## 常用命令

### 探索现有状态
```bash
# 列出活跃变更
node bin/openspec.js list

# 列出所有规范
node bin/openspec.js list --specs

# 显示变更详情
node bin/openspec.js show [change-id]
```

### 验证新提案
```bash
# 验证创建的变更提案
node bin/openspec.js validate [change-id] --strict

# JSON输出用于脚本处理
node bin/openspec.js show [change-id] --json
```

## 依赖项

必需依赖项（应该已安装）：

- **@fission-ai/openspec**: `npm install -g @fission-ai/openspec` (用于CLI功能)
- **typescript**: `npm install -g typescript` (用于TypeScript支持)
- **tsx**: `npm install -g tsx` (用于TypeScript执行)
- **Node.js**: v20.19.0+ (运行时环境)

开发依赖项在package.json中定义，包括测试框架和构建工具。