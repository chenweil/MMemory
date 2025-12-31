# MMemory - iFlow CLI 上下文指南

## 项目概述

**MMemory** 是一个基于 Telegram Bot 的 AI 智能提醒助手，通过自然对话帮助用户管理日常习惯和任务提醒，并支持生活活动记录和智能分析。

### 核心特性
- **AI 智能对话** - 自然语言理解，多意图识别，智能降级策略
- **多 AI 提供商支持** - OpenAI、Claude，智能提供商选择和成本控制
- **对话式提醒管理** - 创建、编辑、删除、暂停、恢复提醒
- **习惯养成跟踪** - 主动关怀、完成率统计、个性化建议
- **生活活动记录** - 自动记录日常活动（喝水、吃药、看书、运动等）
- **天气集成** - 基于位置的天气提醒触发
- **语音消息支持** - 语音转文字，支持语音创建提醒
- **高级监控** - Prometheus 指标、Grafana 仪表板、智能告警
- **成本控制** - 实时 AI 成本监控和预算管理

## 技术栈

- **编程语言**: Go 1.24+
- **数据库**: SQLite3 (GORM)
- **消息平台**: Telegram Bot API v5
- **AI 集成**: OpenAI GPT-4o-mini, Claude 3.5 Sonnet
- **监控**: Prometheus + Grafana + Alertmanager
- **配置管理**: Viper (支持热更新)
- **容器化**: Docker + Docker Compose
- **定时任务**: robfig/cron v3

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
│   │   ├── monitoring.go  # 监控服务
│   │   ├── voice_processor.go # 语音处理服务
│   │   ├── daily_activity.go # 日常活动记录服务
│   │   ├── activity_analysis.go # 活动分析服务
│   │   ├── activity_visualization.go # 活动可视化服务
│   │   ├── pattern_detector.go # 模式检测服务
│   │   ├── pattern_predictor.go # 模式预测服务
│   │   ├── condition_evaluator.go # 条件评估服务
│   │   ├── condition_monitor.go # 条件监控服务
│   │   ├── cost_monitor.go # 成本监控服务
│   │   ├── database_optimizer.go # 数据库优化服务
│   │   ├── transaction_manager.go # 事务管理器
│   │   ├── enhanced_services.go # 增强服务
│   │   ├── registry.go # 服务注册表
│   │   ├── interfaces.go # 服务接口定义
│   │   └── errors.go # 统一错误处理
│   ├── repository/        # 数据访问层
│   │   ├── interfaces/   # 仓储接口定义
│   │   └── sqlite/       # SQLite 实现
│   └── models/            # 数据模型
│       ├── reminder.go
│       ├── reminder_log.go
│       ├── user.go
│       ├── conversation.go
│       ├── conversation_context.go
│       ├── daily_activity.go
│       ├── weather.go
│       ├── condition.go
│       └── ai_parse_result.go
├── pkg/                   # 公共包
│   ├── ai/                # AI 配置、类型、提供商接口
│   │   ├── config.go
│   │   ├── types.go
│   │   ├── claude_provider.go
│   │   ├── cache.go
│   │   ├── circuit_breaker.go
│   │   ├── cost_controller.go
│   │   ├── fallback_strategy.go
│   │   ├── c3_integration.go
│   │   └── advanced_monitoring.go
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
│   ├── migrations/        # 数据库迁移脚本
│   ├── deploy-monitoring.sh
│   ├── test-monitoring.sh
│   ├── validate-config.sh
│   └── verify-monitoring.sh
├── docs/                  # 项目文档
├── test/                  # 测试目录
│   ├── e2e/               # 端到端测试
│   └── integration/       # 集成测试
├── migrations/            # 数据库迁移
└── data/                  # 数据目录（SQLite 数据库）
```

## 核心功能模块

### 1. AI 智能解析服务
- **多提供商支持**: OpenAI、Claude 智能选择
- **四层降级策略**: 主 AI → 备用 AI → 增强正则 → 智能回退
- **成本控制**: 实时预算监控、提供商成本优化
- **上下文记忆**: 30 天对话历史，支持多轮对话
- **意图识别**: 提醒、删除、编辑、暂停、恢复、查询、天气、活动记录等

### 2. 提醒服务
- 自然语言提醒创建和管理
- 灵活的提醒状态管理（活跃/完成/延期/跳过）
- 暂停/恢复功能，支持自定义时长
- 智能匹配和快速定位
- 条件触发支持（天气、时间等）

### 3. 调度器服务
- 基于 cron 的定时任务
- 多工作线程并发处理
- 失败重试机制
- 持久化和恢复
- 动态工作池优化

### 4. 通知服务
- Telegram 消息发送
- 主动关怀机制
- 交互式按钮（完成/延期/跳过）
- 超时处理
- 活动详情追问

### 5. 监控服务
- Prometheus 指标收集
- 系统健康监控
- 性能指标追踪
- 智能告警
- 成本监控

### 6. 天气服务
- 位置感知天气监控
- 条件触发提醒
- 多天气提供商支持

### 7. 语音处理服务
- 语音消息转文字
- 支持语音创建提醒
- 语音识别优化

### 8. 日常活动记录服务
- 自动记录用户活动（喝水、吃药、看书、运动等）
- AI 智能提取活动信息
- 活动详情管理
- 多来源记录（对话、手动、提醒关联）

### 9. 活动分析服务
- 活动模式检测
- 活动趋势分析
- 个性化建议生成
- 数据可视化支持

### 10. 成本监控服务
- AI 成本实时追踪
- 预算管理和告警
- 提供商成本对比
- 成本优化建议

### 11. 条件监控服务
- 条件评估引擎
- 动态条件触发
- 复杂条件组合
- 条件历史记录

## 数据库架构

### 主要数据表
- `users` - 用户信息和偏好设置
- `reminders` - 提醒配置
- `reminder_logs` - 提醒执行记录
- `conversations` - 对话上下文（30 天历史）
- `messages` - 会话消息记录
- `conversation_contexts` - 对话状态管理
- `daily_activities` - 日常活动记录
- `weather` - 天气数据缓存
- `conditions` - 条件配置

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

#### 语音处理配置
- `MMEMORY_VOICE_ENABLED` - 启用语音消息处理（默认：true）
- `MMEMORY_VOICE_MAX_DURATION` - 最大语音时长（默认：60秒）
- `MMEMORY_VOICE_LANGUAGE` - 语音识别语言（默认：zh-CN）

#### 活动记录配置
- `MMEMORY_ACTIVITY_ENABLED` - 启用活动记录功能（默认：true）
- `MMEMORY_ACTIVITY_RETENTION_DAYS` - 活动记录保留天数（默认：365）
- `MMEMORY_ACTIVITY_MAX_RECORDS` - 单用户最大记录数（默认：10000）

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

### 监控端点
- **Prometheus**: `http://localhost:9091`
- **Grafana**: `http://localhost:3000` (默认账号: admin/admin)
- **Alertmanager**: `http://localhost:9093`
- **应用指标**: `http://localhost:9090/metrics`
- **健康检查**: `http://localhost:8080/health`

### 关键指标
- **业务指标**:
  - 提醒创建/完成/跳过数量
  - 活动记录数量
  - AI 解析成功率
  - 用户活跃度

- **性能指标**:
  - API 响应时间
  - 数据库查询时间
  - 内存使用率
  - CPU 使用率

- **成本指标**:
  - AI 调用次数
  - Token 使用量
  - 按提供商成本统计
  - 预算使用率

- **系统指标**:
  - 调度器队列长度
  - 数据库连接数
  - Goroutine 数量
  - 错误率

### 告警规则
- 提醒失败率 > 10%
- AI 解析失败率 > 20%
- 数据库响应时间 > 1s
- API 响应时间 > 2s
- AI 成本超预算
- 系统错误率 > 5%

## 开发规范

### 代码风格
- 使用 Go 标准格式化 (`gofmt`)
- 错误处理遵循 Go 惯用方式
- 接口定义优先设计
- 包名使用小写单词
- 导出函数使用大写开头
- 常量使用大写或驼峰命名

### 测试规范
- 单元测试覆盖核心业务逻辑
- 集成测试验证端到端流程
- 测试文件与源码文件对应（`_test.go`）
- 目标测试覆盖率 >80%
- 使用表驱动测试（Table-Driven Tests）
- 测试函数命名：`Test<FunctionName>`

### 日志规范
- 结构化日志 (JSON 格式)
- 分级日志输出 (debug/info/warn/error)
- 上下文信息丰富
- 使用 `logger.Infof`, `logger.Errorf` 等方法
- 避免在日志中记录敏感信息

### Git 提交规范
- 提交信息格式：`<type>(<scope>): <subject>`
- type 类型：
  - `feat`: 新功能
  - `fix`: 修复 bug
  - `docs`: 文档更新
  - `style`: 代码格式调整
  - `refactor`: 重构
  - `test`: 测试相关
  - `chore`: 构建/工具相关

### 代码审查清单
- [ ] 代码符合 Go 规范
- [ ] 添加了必要的注释
- [ ] 编写了单元测试
- [ ] 测试覆盖率达标
- [ ] 更新了相关文档
- [ ] 没有引入安全漏洞
- [ ] 性能影响可接受
- [ ] 错误处理完善

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

- 📋 **C6**: 用户生活记录系统（2025-12-30 规划）
  - **目标**: 实现用户日常活动记录和智能分析
  - **核心功能**:
    1. 活动记录功能（喝水、吃药、看书、运动等）
    2. AI 智能提取（从对话中自动识别活动）
    3. 主动关怀增强（基于活动记录的追问）
    4. 查询与统计功能
  - **详细计划**:
    - [C6-用户生活记录系统需求文档_20251230.md](./docs/C6-用户生活记录系统需求文档_20251230.md)
    - [C6-用户生活记录系统实现计划_20251230.md](./docs/C6-用户生活记录系统实现计划_20251230.md)

### 测试覆盖率详情

| 模块 | 覆盖率 |
|------|--------|
| **总体** | **~60%** |
| pkg/version | 100.0% |
| pkg/server | 94.7% |
| pkg/logger | 94.4% |
| pkg/config | 76.8% |
| internal/ai | 73.9% |
| internal/repository/sqlite | 73.5% |
| internal/service | 47.4% |
| pkg/ai | 51.2% |
| pkg/metrics | 43.2% |
| internal/bot/handlers | 40.4% |

#### 测试覆盖说明
- 核心基础包（version、server、logger）覆盖率 >90%
- AI 和数据库层覆盖率 >70%
- 服务层和处理器层覆盖率 40-50%
- 需要持续提升服务层和处理器层测试覆盖率

#### 测试类型
- 单元测试：覆盖核心业务逻辑
- 集成测试：验证端到端流程
- E2E 测试：完整场景验证
- 竞态检测：并发安全性验证

#### 运行测试
```bash
# 运行所有测试（需要 CGO）
CGO_ENABLED=1 go test ./...

# 运行特定模块测试
CGO_ENABLED=1 go test ./internal/service -v
CGO_ENABLED=1 go test ./internal/ai -v

# 生成覆盖率报告
CGO_ENABLED=1 go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行竞态检测
CGO_ENABLED=1 go test -race ./...
```

## 相关文档

- [README.md](./README.md) - 项目介绍和使用指南
- [DEPLOYMENT.md](./DEPLOYMENT.md) - 详细部署说明
- [CLAUDE.md](./CLAUDE.md) - 开发指南和架构说明
- [AGENTS.md](./AGENTS.md) - Agent 配置说明
- [docs/](./docs/) - 开发文档目录

### 阶段实施文档
- [C1-AI-Parser-Implementation-20250929.md](./docs/C1-AI-Parser-Implementation-20250929.md) - AI Parser 接口设计
- [C2-AI-Provider-Implementation-20250930.md](./docs/C2-AI-Provider-Implementation-20250930.md) - 多 AI 提供商支持
- [C3-Critical-Fixes-And-Enhancements-20251010.md](./docs/C3-Critical-Fixes-And-Enhancements-20251010.md) - 关键修复和增强
- [C3-智能降级机制使用指南_20251109.md](./docs/C3-智能降级机制使用指南_20251109.md) - 智能降级机制使用指南
- [C4-Week4-Implementation-Complete-Report_20251109.md](./docs/C4-Week4-Implementation-Complete-Report_20251109.md) - C4 阶段完整报告
- [C5-Performance-Optimization-Plan_20251230.md](./docs/C5-Performance-Optimization-Plan_20251230.md) - C5 性能优化计划
- [C6-用户生活记录系统需求文档_20251230.md](./docs/C6-用户生活记录系统需求文档_20251230.md) - C6 需求文档
- [C6-用户生活记录系统实现计划_20251230.md](./docs/C6-用户生活记录系统实现计划_20251230.md) - C6 实现计划

### 其他文档
- [CONFIG_MANAGEMENT.md](./docs/CONFIG_MANAGEMENT.md) - 配置管理说明
- [MONITORING.md](./docs/MONITORING.md) - 监控配置说明
- [version-management.md](./docs/version-management.md) - 版本管理说明
- [mmemory-development-roadmap-2025.md](./docs/mmemory-development-roadmap-2025.md) - 2025 年开发路线图

## 故障排除

### 常见问题

1. **Bot 无响应**
   - 检查 Token 配置是否正确
   - 验证网络连接是否正常
   - 查看日志文件：`tail -f data/mmemory.log`
   - 检查 Telegram API 状态

2. **数据库错误**
   - 验证数据目录权限：`chmod 755 data/`
   - 检查数据库文件路径配置
   - 确认 CGO 已启用：`CGO_ENABLED=1`

3. **调度器异常**
   - 检查时区设置（默认使用系统时区）
   - 验证 cron 表达式格式
   - 查看调度器日志输出

4. **AI 解析失败**
   - 检查 API 密钥配置
   - 验证网络连接和代理设置
   - 查看降级日志了解失败原因
   - 检查 API 配额是否用尽

5. **测试失败**
   - 确保使用 CGO：`CGO_ENABLED=1 go test`
   - 检查数据库是否正确初始化
   - 清理测试数据后重试

6. **内存占用过高**
   - 检查对话历史保留策略
   - 优化数据库查询
   - 检查是否有内存泄漏

7. **监控数据缺失**
   - 确认监控服务已启动
   - 检查 Prometheus 配置
   - 验证指标端口是否开放

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
- **最后更新**: 2025-12-31
- **当前阶段**: C5 性能优化与监控增强（计划中）, C6 用户生活记录系统（规划中）
- **Git 分支**: main
- **测试覆盖率**: ~60%

### 版本历史
- v0.4.0-dev - 当前开发版本
  - 新增：语音消息支持
  - 新增：日常活动记录服务（规划中）
  - 新增：活动分析和模式检测
  - 优化：AI 成本监控
  - 优化：数据库连接池
- v0.3.0 - C4 阶段完成
  - 测试覆盖率提升至 60%
  - 对话上下文优化
  - 智能建议系统
- v0.2.0 - C3 阶段完成
  - 智能降级机制
  - 多 AI 提供商支持
- v0.1.0 - C1/C2 阶段完成
  - AI Parser 接口设计
  - 基础提醒功能

---

## 快速参考

### 常用命令速查
```bash
# 开发
make run              # 运行应用
make test             # 运行测试
make test-cover       # 生成覆盖率报告
make fmt              # 格式化代码
make lint             # 代码检查
make tidy             # 整理依赖

# 构建
make build            # 构建二进制
make clean            # 清理构建产物
make version          # 查看版本信息

# Docker
make docker-build     # 构建镜像
make docker-up        # 启动容器
make docker-down      # 停止容器
make docker-logs      # 查看日志
```

### 关键文件位置
- **配置文件**: `configs/config.yaml`
- **数据库**: `data/mmemory.db`
- **日志文件**: `data/mmemory.log`
- **主程序**: `cmd/bot/main.go`
- **服务层**: `internal/service/`
- **模型定义**: `internal/models/`
- **测试目录**: `test/`

### 环境变量速查
```bash
# 必需
export MMEMORY_BOT_TOKEN="your_bot_token"

# AI 功能（可选）
export MMEMORY_AI_ENABLED=true
export MMEMORY_AI_OPENAI_API_KEY="sk-..."
export MMEMORY_AI_OPENAI_BASE_URL="https://api.openai.com/v1"

# 监控（可选）
export MMEMORY_MONITORING_ENABLED=true
export MMEMORY_MONITORING_PORT=9090
```

### 重要端口
- **HTTP 服务**: 8080
- **监控指标**: 9090
- **Prometheus**: 9091
- **Alertmanager**: 9093
- **Grafana**: 3000

### 调度模式
- `daily` - 每天固定时间
- `weekly:1,3,5` - 每周指定日期（1=周一, 7=周日）
- `monthly:1,15` - 每月指定日期
- `once:2024-10-15` - 一次性提醒

### AI 意图类型
- `reminder` - 创建提醒
- `delete` - 删除提醒
- `edit` - 编辑提醒
- `pause` - 暂停提醒
- `resume` - 恢复提醒
- `query` - 查询提醒
- `summary` - 统计信息
- `weather` - 天气查询
- `chat` - 普通对话
- `record_activity` - 记录活动（规划中）
- `query_activity` - 查询活动（规划中）