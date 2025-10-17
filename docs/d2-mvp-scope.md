# D2 MVP 范围说明 [[D]]

## 🎯 项目目标 [[D]]
- 建立**多轮对话上下文**，支撑消息理解与后续 D1 智能建议。
- 提供**实体与意图缓存**，为解析与个性化逻辑提供高可信参考。
- 在 **500ms 内** 完成上下文处理，保障 Bot 交互体验。

## 🧱 功能边界 [[D]]
- **会话追踪**：保存最近 `N=20` 条消息，维护 `state` 与最新 `intent`。
- **实体缓存**：支持 `datetime`、`recurrence`、`task`、`tone`、`preference` 等核心实体。
- **意图识别**：规则+关键词，覆盖 `create_reminder`、`modify_reminder`、`list_reminder`、`small_talk`、`unknown`。
- **上下文过期**：默认 `TTL=30m`，超时自动重置；活跃用户按照 `user_id` 定义上下文。
- **持久化策略**：SQLite `conversation_contexts` 表存储摘要，配合内存缓存快速读取。

## 🧩 核心组件 [[D]]
- `ConversationContext`：字段包含 `UserID`、`SessionID`、`Messages`、`Entities`、`Intent`、`State`、`LastActivity`、`TTL`。
- `ContextManager`：提供 `ProcessMessage`、`GetContext`、`UpdateContextState` 等接口。
- `ContextStore`：封装持久化读写，支持 SQLite 实现与可选内存缓存装饰器。
- `EntityExtractor`：正则/规则解析时间、频率、任务描述，预留 AI fallback 钩子。
- `IntentTracker`：基于关键词和上下文状态判断意图，为未来 ML 模型预留扩展。

## 🔗 系统整合 [[D]]
- **与 D1 建议**：`ReminderSuggester` 通过 `ContextManager.GetContext` 获取最新意图与实体。
- **与 Handler**：消息入口调用 `ProcessMessage` 更新上下文，再驱动业务分支。
- **与解析链路**：传统/AI 解析失败时优先回退上下文信息补全缺失参数。

## 🗄️ 数据结构与表设计 [[D]]
- 新增模型文件 `internal/models/conversation_context.go` 描述上下文结构。
- 新建表 `conversation_contexts`（字段：`user_id`、`session_id`、`state`、`intent`、`entities_json`、`messages_json`、`updated_at`）。
- 视性能需求增加 `conversation_entities` 索引表（初版可留空，待优化时引入）。

## 🚀 最小实现路径 [[D]]
1. 定义模型与接口签名。
2. 实现 SQLite `ContextStore` 与可选内存缓存。
3. 编写 `EntityExtractor` 规则解析（时间短语、频率、任务关键词）。
4. 实现 `IntentTracker` 规则集与状态机。
5. 在消息 Handler 接入上下文调用链，验证多轮对话流程。
6. 编写单元测试覆盖实体提取、意图判断、上下文 TTL 逻辑。

## ✅ 验收标准 [[D]]
- 多轮对话测试：Bot 记住上一轮待补信息并正确续问。
- D1 可读取上下文并生成包含意图和任务关键字的建议。
- TTL 失效后上下文自动清理，无陈旧数据干扰。
- `go test ./internal/bot/handlers ./internal/service` 全部通过。

## 🔮 扩展预留 [[D]]
- 数据结构预留 `Channel`、`Locale` 字段支撑 D4 多模态/多语言。
- `entities` 子项包含 `confidence`，方便未来引入 ML 模型。
- 在 `configs/config.yaml` 新增 `context.max_messages`、`context.default_ttl`、`context.entity_confidence_threshold` 配置项。
