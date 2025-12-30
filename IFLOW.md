# MMemory - iFlow CLI 上下文指南

## 项目概述

**MMemory** 是一个基于 Telegram Bot 的 AI 智能提醒助手，通过自然对话帮助用户管理日常习惯和任务提醒。

### 核心特性
- **AI 智能对话** - 自然语言理解，多意图识别，智能降级策略
- **多 AI 提供商支持** - OpenAI、Claude，智能提供商选择和成本控制
- **对话式提醒管理** - 创建、编辑、删除、暂停、恢复提醒
- **习惯养成跟踪** - 主动关怀、完成率统计、个性化建议
- **天气集成** - 基于位置的天气提醒触发
- **高级监控** - Prometheus 指标、Grafana 仪表板、智能告警

## 技术栈

- **编程语言**: Go 1.24+
- **数据库**: SQLite3 (GORM)
- **消息平台**: Telegram Bot API v5
- **AI 集成**: OpenAI GPT-4o-mini, Claude 3.5 Sonnet
- **监控**: Prometheus + Grafana + Alertmanager
- **配置管理**: Viper (支持热更新)
- **容器化**: Docker + Docker Compose

## 项目结构

```
MMemory/
├── cmd/bot/                 # 主程序入口
│   └── main.go             # 应用启动和初始化
├── internal/               # 内部业务逻辑
│   ├── ai/                # AI 客户端（OpenAI 集成、降级链）
│   ├── bot/               # Telegram Bot 处理层
│   │   ├── handlers/      # 消息和回调查询处理器
│   │   └── middleware/    # 中间件
│   ├── service/           # 业务服务层
│   │   ├── ai_parser.go   # AI 解析服务
│   │   ├── reminder.go    # 提醒服务
│   │   ├── scheduler.go   # 调度器服务
│   │   ├── notification.go # 通知服务
│   │   ├── weather.go     # 天气服务
│   │   ├── conversation.go # 对话服务
│   │   ├── context_manager.go # 上下文管理
│   │   ├── suggestion.go  # 智能建议
│   │   └── monitoring.go  # 监控服务
│   ├── repository/        # 数据访问层
│   │   └── sqlite/        # SQLite 实现
│   └── models/            # 数据模型
├── pkg/                   # 公共包
│   ├── ai/                # AI 配置、类型、提供商接口
│   ├── config/            # 配置管理（支持热更新）
│   ├── logger/            # 日志工具
│   ├── metrics/           # Prometheus 指标定义
│   ├── server/            # HTTP 服务器（指标端点）
│   └── version/           # 版本管理
├── configs/               # 配置文件
│   ├── config.yaml        # 主配置文件
│   ├── config.full.yaml   # 完整配置示例
│   ├── alerts/            # 告警配置
│   └── grafana/           # Grafana 仪表板
├── scripts/               # 部署和管理脚本
├── docs/                  # 项目文档
├── migrations/            # 数据库迁移
└── data/                  # 数据目录（SQLite 数据库）
```

## 核心功能模块

### 1. AI 智能解析服务
- **多提供商支持**: OpenAI、Claude 智能选择
- **四层降级策略**: 主 AI → 备用 AI → 增强正则 → 智能回退
- **成本控制**: 实时预算监控、提供商成本优化
- **上下文记忆**: 30 天对话历史，支持多轮对话

### 2. 提醒服务
- 自然语言提醒创建和管理
- 灵活的提醒状态管理（活跃/完成/延期/跳过）
- 暂停/恢复功能，支持自定义时长
- 智能匹配和快速定位

### 3. 调度器服务
- 基于 cron 的定时任务
- 多工作线程并发处理
- 失败重试机制
- 持久化和恢复

### 4. 通知服务
- Telegram 消息发送
- 主动关怀机制
- 交互式按钮（完成/延期/跳过）
- 超时处理

### 5. 监控服务
- Prometheus 指标收集
- 系统健康监控
- 性能指标追踪
- 智能告警

### 6. 天气服务
- 位置感知天气监控
- 条件触发提醒
- 多天气提供商支持

## 数据库架构

### 主要数据表
- `users` - 用户信息和偏好设置
- `reminders` - 提醒配置
- `reminder_logs` - 提醒执行记录
- `conversations` - 对话上下文（30 天历史）
- `messages` - 会话消息记录
- `conversation_contexts` - 对话状态管理

### 调度模式
- `daily` - 每天固定时间
- `weekly:1,3,5` - 每周指定日期（1=周一, 7=周日）
- `monthly:1,15` - 每月指定日期
- `once:2024-10-15` - 一次性提醒

## 开发命令

### 快速开始
```bash
# 安装依赖
go mod tidy

# 配置环境
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml，设置必要的配置项

# 运行应用
go run cmd/bot/main.go

# 或使用 Makefile（推荐）
make run
```

### Makefile 命令
```bash
make help              # 显示所有可用命令
make build             # 构建到 bin/mmemory
make run               # 运行应用
make test              # 运行测试
make test-cover        # 运行测试并生成覆盖率报告
make clean             # 清理构建产物
make fmt               # 格式化代码
make tidy              # 整理依赖
make lint              # 代码质量检查
make version           # 显示版本信息
```

### Docker 命令
```bash
make docker-build      # 构建镜像
make docker-up         # 启动容器
make docker-down       # 停止容器
make docker-rebuild    # 重新构建并启动
make docker-logs       # 查看日志
```

### 测试命令
```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./internal/service -v
go test ./internal/ai -v

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行竞态检测
go test -race ./...
```

## 配置管理

### 环境变量
所有配置项都可以通过环境变量覆盖，使用 `MMEMORY_` 前缀：

#### 基础配置
- `MMEMORY_BOT_TOKEN` - Telegram Bot Token（必填）
- `MMEMORY_DATABASE_DSN` - 数据库文件路径（默认：`./data/mmemory.db`）
- `MMEMORY_SERVER_PORT` - HTTP 服务端口（默认：8080）
- `MMEMORY_LOG_LEVEL` - 日志级别（默认：info）
- `MMEMORY_LOG_FORMAT` - 日志格式 json/text（默认：json）

#### AI 配置
- `MMEMORY_AI_ENABLED` - 启用 AI 功能（默认：false）
- `MMEMORY_AI_OPENAI_API_KEY` - OpenAI API 密钥
- `MMEMORY_AI_OPENAI_BASE_URL` - API 端点地址（默认：https://api.openai.com/v1）
- `MMEMORY_AI_OPENAI_PRIMARY_MODEL` - 主模型名称（默认：gpt-4o-mini）
- `MMEMORY_AI_OPENAI_BACKUP_MODEL` - 备用模型名称（默认：gpt-3.5-turbo）
- `MMEMORY_AI_OPENAI_TEMPERATURE` - 温度参数（默认：0.1）
- `MMEMORY_AI_OPENAI_MAX_TOKENS` - 最大 token 数（默认：1000）
- `MMEMORY_AI_OPENAI_TIMEOUT` - 请求超时（默认：30s）
- `MMEMORY_AI_OPENAI_MAX_RETRIES` - 最大重试次数（默认：3）

#### C3 降级配置
- `MMEMORY_AI_C3_ENABLED` - 启用 C3 智能降级（默认：true）
- `MMEMORY_AI_C3_BUDGET_LIMIT_PER_USER` - 每用户每日预算 USD（默认：10.0）
- `MMEMORY_AI_C3_GLOBAL_BUDGET_LIMIT` - 全局每日预算 USD（默认：100.0）
- `MMEMORY_AI_C3_DEGRADATION_THRESHOLD` - 降级置信度阈值（默认：0.7）

#### 监控配置
- `MMEMORY_MONITORING_ENABLED` - 启用 Prometheus 监控（默认：true）
- `MMEMORY_MONITORING_PORT` - 监控指标端口（默认：9090）

## 部署

### Docker 部署
```bash
# 使用部署脚本
./deploy.sh start

# 手动部署
docker-compose up -d

# 带监控栈部署
docker-compose -f docker-compose.monitoring.yml up -d
```

### 环境变量配置（.env 文件）
```bash
MMEMORY_BOT_TOKEN=your_bot_token
MMEMORY_AI_ENABLED=true
MMEMORY_AI_OPENAI_API_KEY=sk-...
MMEMORY_AI_OPENAI_BASE_URL=https://api.openai.com/v1
MMEMORY_DATABASE_DSN=/app/data/mmemory.db
```

## 监控和告警

- **Prometheus**: `http://localhost:9091`
- **Grafana**: `http://localhost:3000`
- **Alertmanager**: `http://localhost:9093`
- **应用指标**: `http://localhost:9090/metrics`

## 开发规范

### 代码风格
- 使用 Go 标准格式化 (`gofmt`)
- 错误处理遵循 Go 惯用方式
- 接口定义优先设计

### 测试规范
- 单元测试覆盖核心业务逻辑
- 集成测试验证端到端流程
- 测试文件与源码文件对应
- 目标测试覆盖率 >80%

### 日志规范
- 结构化日志 (JSON 格式)
- 分级日志输出 (debug/info/warn/error)
- 上下文信息丰富

## 项目阶段

### 已完成阶段
- ✅ **C1**: AI Parser 接口设计（2025-10-10）
- ✅ **C2**: 多 AI 提供商支持（2025-10-15）
- ✅ **C3**: 智能降级机制（2025-10-20）
- ✅ **C4**: 增强功能和测试（2025-11-09）
- ✅ **测试覆盖率提升**: 整体覆盖率从 30.9% 提升至 60.0%（2025-12-30）

### 计划中阶段
- 📋 **C5**: 性能优化与监控增强（2025-12-30 开始）
  - **目标**: 数据库查询性能提升 30%+，AI 成本降低 20%+
  - **周期**: 2-3 周
  - **核心改进**:
    1. 数据库连接池优化（动态调整、健康检查）
    2. 调度器性能提升（动态工作池、队列监控）
    3. AI 成本控制增强（预测模型、优化规则）
    4. 监控指标扩展（新增 10+ 关键指标）
    5. 查询优化与缓存增强
  - **详细计划**: [C5-Performance-Optimization-Plan_20251230.md](./docs/C5-Performance-Optimization-Plan_20251230.md)
- 📋 **C6**: 多语言支持

### 测试覆盖率详情

| 模块 | 覆盖率 |
|------|--------|
| **总体** | **60.0%** |
| pkg/metrics | 100.0% |
| pkg/version | 100.0% |
| pkg/logger | 94.4% |
| pkg/server | 94.7% |
| pkg/config | 80.6% |
| internal/ai | 73.9% |
| internal/repository/sqlite | 70.8% |
| internal/service | 62.4% |
| internal/bot/handlers | 50.4% |
| pkg/ai | 58.8% |

#### 新增测试文件
- `internal/service/parser_test.go` - Parser 服务测试（16个测试函数）
- `internal/service/conversation_test.go` - Conversation 服务测试（扩展边界测试）
- `internal/bot/handlers/message_handler_test.go` - Message Handler 测试扩展
- `pkg/ai/provider_test.go` - Provider 类型测试
- `pkg/ai/cache_test.go` - 缓存测试
- `pkg/ai/ratelimiter_test.go` - 限流器测试
- `pkg/ai/metrics_test.go` - 指标测试
- `pkg/ai/circuit_breaker_test.go` - 熔断器测试
- `pkg/ai/manager_test.go` - Manager 测试

#### 修复的问题
- 修复 `pkg/ai/advanced_monitoring.go` slice out of bounds bug

## 相关文档

- [README.md](./README.md) - 项目介绍和使用指南
- [DEPLOYMENT.md](./DEPLOYMENT.md) - 详细部署说明
- [CLAUDE.md](./CLAUDE.md) - 开发指南和架构说明
- [docs/](./docs/) - 开发文档目录
  - [C1-AI-Parser-Implementation-20250929.md](./docs/C1-AI-Parser-Implementation-20250929.md)
  - [C2-AI-Provider-Implementation-20250930.md](./docs/C2-AI-Provider-Implementation-20250930.md)
  - [C3-Critical-Fixes-And-Enhancements-20251010.md](./docs/C3-Critical-Fixes-And-Enhancements-20251010.md)
  - [C3-智能降级机制使用指南_20251109.md](./docs/C3-智能降级机制使用指南_20251109.md)
  - [C4-Week4-Implementation-Complete-Report_20251109.md](./docs/C4-Week4-Implementation-Complete-Report_20251109.md)
  - [C5-Performance-Optimization-Plan_20251230.md](./docs/C5-Performance-Optimization-Plan_20251230.md) - C5 阶段详细实施计划

## 故障排除

### 常见问题
1. **Bot 无响应**: 检查 Token 配置和网络连接
2. **数据库错误**: 验证文件权限和路径
3. **调度器异常**: 检查时区设置和 cron 表达式
4. **AI 解析失败**: 检查 API 密钥和网络连接，查看降级日志

### 调试技巧
```bash
# 启用调试模式
将 config.yaml 中的 bot.debug 设为 true

# 查看详细日志
tail -f data/mmemory.log

# 检查指标
curl http://localhost:9090/metrics

# 查看 Docker 日志
docker-compose logs -f mmemory
```

## 版本信息

- **当前版本**: v0.4.0-dev
- **Go 版本**: 1.24+
- **最后更新**: 2025-12-30
- **当前阶段**: C5 性能优化与监控增强（计划中）