# MMemory - AI智能提醒助手

基于 Telegram Bot 的 AI 智能提醒工具，通过自然对话帮助你管理日常习惯和任务提醒。

## 🌟 核心特性

### 🤖 AI 智能对话
- **自然语言理解** - 直接说出你的需求，AI 自动理解并创建提醒
- **多意图识别** - 支持创建、编辑、删除、暂停、恢复、查询等多种操作
- **智能降级** - 四层解析策略（主AI → 备用AI → 正则 → 回退）确保稳定性
- **上下文记忆** - 30天对话历史，支持多轮对话和意图追踪

### 📋 提醒管理
- **对话式设置** - 通过自然语言添加提醒，无需复杂表单
- **灵活编辑** - 随时修改时间、标题、重复规则
- **暂停/恢复** - 临时暂停提醒，支持自定义时长
- **智能匹配** - 关键词快速定位和管理提醒

### 🎯 习惯养成
- **主动跟踪** - 超时后主动询问进度和完成情况
- **灵活反馈** - 支持完成、延期、跳过等多种状态
- **数据统计** - 完成率、连续天数、趋势分析
- **个性化建议** - 基于使用习惯推荐新提醒

## 🚀 快速开始

### 环境要求

- Go 1.21+
- SQLite 3
- Telegram Bot Token

### 安装

1. 克隆项目
```bash
git clone <repository>
cd mmemory
```

2. 安装依赖
```bash
go mod tidy
```

3. 配置
```bash
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml，设置必要的配置项
```

4. 运行
```bash
# 开发环境
go run cmd/bot/main.go

# 或使用 Makefile
make run
```

### 配置说明

#### 基础配置（必需）

在 `configs/config.yaml` 中设置：

```yaml
bot:
  token: "your_telegram_bot_token_here"  # 必填：你的Bot Token
  debug: false                           # 可选：调试模式

database:
  dsn: "./data/mmemory.db"              # 数据库文件路径

logging:
  level: "info"                         # 日志级别: debug/info/warn/error
  format: "json"                        # 日志格式: json/text

app:
  environment: "production"             # 运行环境: development/production

server:
  port: 8080                            # HTTP服务端口（健康检查）
```

#### AI 配置（可选）

启用 AI 智能对话功能：

```yaml
ai:
  enabled: true                         # 启用AI功能
  openai:
    api_key: "sk-..."                   # OpenAI API Key
    base_url: "https://api.openai.com/v1"  # API地址（支持第三方）
    primary_model: "gpt-4o-mini"        # 主模型
    backup_model: "gpt-4o-mini"         # 备用模型
    temperature: 0.1                    # 温度参数 (0-1)
    max_tokens: 1000                    # 最大token数
    timeout: "30s"                      # 请求超时
```

**环境变量方式**（推荐）：
```bash
export MMEMORY_BOT_TOKEN="your_bot_token"
export MMEMORY_AI_ENABLED=true
export MMEMORY_AI_OPENAI_API_KEY="sk-..."
```

> 💡 AI 功能为可选项，未配置时自动降级到传统正则解析模式。

## 💬 使用方法

### 支持的提醒格式

- **每日提醒**: "每天19点提醒我复盘工作"
- **每周提醒**: "每周一三五19点提醒我健身"  
- **一次性提醒**: "2024年10月1日提醒我交房租"
- **明天提醒**: "明天上午10点提醒我开会"

### Bot 命令

- `/start` - 开始使用，查看欢迎消息
- `/help` - 查看详细帮助信息
- `/list` - 查看提醒列表（带快捷操作按钮）
- `/stats` - 查看使用统计（完成率、连续天数等）
- `/delete <ID>` - 按ID删除指定提醒
- `/version` - 查看版本信息

### 使用示例

#### 创建提醒
```
你: "每天晚上8点提醒我复盘今天的工作"
Bot: "✅ 提醒已设置成功！
     📝 复盘今天的工作
     ⏰ 每天 20:00"
```

#### AI 智能编辑
```
你: "把健身提醒改到晚上7点"
Bot: "✅ 已成功修改提醒
     📝 健身
     ⏰ 每天 19:00"
```

#### 提醒触发交互
```
[20:00] Bot: "该复盘今天工作了，完成了吗？"
        [✅ 完成了] [⏰ 延期1小时] [😴 今天跳过]

你: [点击"完成了"]
Bot: "✅ 太棒了！
     📝 复盘今天的工作
     🎉 已记录完成，继续保持！"
```

#### 查看统计
```
你: /stats
Bot: "📊 你的使用统计

     📝 提醒总数: 5 个
     ✅ 活跃提醒: 4 个

     📅 今日数据:
       ✅ 完成: 3 个
       😴 跳过: 0 个

     📈 本月数据:
       ✅ 完成: 45 个
       🎯 完成率: 85%

     🌟 今天做得很棒！继续保持！"
```

## 🏗️ 项目结构

```
mmemory/
├── cmd/bot/                 # 主程序入口
├── internal/                # 内部包
│   ├── ai/                  # AI客户端（OpenAI集成）
│   ├── bot/handlers/        # Telegram 消息处理
│   ├── service/             # 业务逻辑层
│   ├── repository/          # 数据访问层
│   └── models/              # 数据模型
├── pkg/                     # 公共包
│   ├── ai/                  # AI配置和类型
│   ├── config/              # 配置管理
│   ├── logger/              # 日志工具
│   ├── metrics/             # Prometheus监控
│   ├── server/              # HTTP服务器
│   └── version/             # 版本管理
├── configs/                 # 配置文件
├── docs/                    # 技术文档
├── scripts/                 # 构建和部署脚本
└── data/                    # 数据目录（SQLite）
```

## 🛠️ 开发

### 构建

```bash
# 使用 Makefile（推荐）
make build          # 构建到 bin/mmemory
make run            # 运行应用
make test           # 运行测试
make test-cover     # 运行测试并生成覆盖率报告
make clean          # 清理构建产物

# 手动构建
go build -o bin/mmemory cmd/bot/main.go

# 带版本信息的构建
make build VERSION=v0.4.0
```

### 测试

```bash
# 运行所有测试
go test ./...
make test

# 运行特定模块测试
go test ./internal/service -v
go test ./internal/ai -v

# 生成覆盖率报告
make test-cover

# 查看覆盖率详情
go tool cover -html=coverage.out

# 运行竞态检测
go test -race ./...
```

## 📦 部署

### Docker 部署

```bash
# 使用 Makefile
make docker-build      # 构建镜像
make docker-up         # 启动容器
make docker-down       # 停止容器
make docker-rebuild    # 重新构建并启动
make docker-logs       # 查看日志

# 手动构建和运行
docker build -t mmemory:latest .

docker run -d \
  -e MMEMORY_BOT_TOKEN=your_token \
  -e MMEMORY_AI_ENABLED=true \
  -e MMEMORY_AI_OPENAI_API_KEY=sk-... \
  -v /path/to/data:/app/data \
  --name mmemory \
  mmemory:latest
```

### Docker Compose 部署

```bash
# 基础部署
docker-compose up -d

# 带监控栈部署（Prometheus + Grafana）
docker-compose -f docker-compose.monitoring.yml up -d

# 查看服务
docker-compose ps
docker-compose logs -f mmemory
```

**环境变量配置** (`.env` 文件)：
```bash
MMEMORY_BOT_TOKEN=your_bot_token
MMEMORY_AI_ENABLED=true
MMEMORY_AI_OPENAI_API_KEY=sk-...
MMEMORY_AI_OPENAI_BASE_URL=https://api.openai.com/v1
MMEMORY_DATABASE_DSN=/app/data/mmemory.db
```

## 🗂️ 数据库

使用 SQLite 存储数据，包含以下表：

- `users` - 用户信息和偏好设置
- `reminders` - 提醒配置（时间、重复规则等）
- `reminder_logs` - 提醒执行记录和状态
- `conversations` - 对话上下文（30天历史）
- `messages` - 会话消息记录
- `conversation_contexts` - 对话状态管理（C4阶段新增）

数据库会在首次运行时自动创建和迁移。

### 调度模式说明

- `daily` - 每天固定时间
- `weekly:1,3,5` - 每周指定日期（1=周一, 7=周日）
- `monthly:1,15` - 每月指定日期
- `once:2024-10-15` - 一次性提醒

## 🔧 环境变量

### 基础配置

| 变量名 | 说明 | 必填 | 默认值 |
|--------|------|------|--------|
| `MMEMORY_BOT_TOKEN` | Telegram Bot Token | ✅ 是 | - |
| `MMEMORY_DATABASE_DSN` | 数据库文件路径 | 否 | `./data/mmemory.db` |
| `MMEMORY_SERVER_PORT` | HTTP服务端口 | 否 | `8080` |
| `MMEMORY_LOG_LEVEL` | 日志级别 | 否 | `info` |
| `MMEMORY_LOG_FORMAT` | 日志格式 (json/text) | 否 | `json` |
| `MMEMORY_APP_ENVIRONMENT` | 运行环境 | 否 | `production` |

### AI 配置（可选）

| 变量名 | 说明 | 必填 | 默认值 |
|--------|------|------|--------|
| `MMEMORY_AI_ENABLED` | 启用AI功能 | 否 | `false` |
| `MMEMORY_AI_OPENAI_API_KEY` | OpenAI API密钥 | AI启用时必填 | - |
| `MMEMORY_AI_OPENAI_BASE_URL` | API端点地址 | 否 | `https://api.openai.com/v1` |
| `MMEMORY_AI_OPENAI_PRIMARY_MODEL` | 主模型名称 | 否 | `gpt-4o-mini` |
| `MMEMORY_AI_OPENAI_BACKUP_MODEL` | 备用模型名称 | 否 | `gpt-4o-mini` |
| `MMEMORY_AI_OPENAI_TEMPERATURE` | 温度参数 (0-1) | 否 | `0.1` |
| `MMEMORY_AI_OPENAI_MAX_TOKENS` | 最大token数 | 否 | `1000` |
| `MMEMORY_AI_OPENAI_TIMEOUT` | 请求超时 | 否 | `30s` |
| `MMEMORY_AI_OPENAI_MAX_RETRIES` | 最大重试次数 | 否 | `3` |

### 监控配置（可选）

| 变量名 | 说明 | 必填 | 默认值 |
|--------|------|------|--------|
| `MMEMORY_MONITORING_ENABLED` | 启用Prometheus监控 | 否 | `true` |
| `MMEMORY_MONITORING_PORT` | 监控指标端口 | 否 | `9090` |

> 💡 **提示**：所有配置项都可以通过环境变量覆盖YAML配置文件中的值。环境变量使用 `MMEMORY_` 前缀。

## 📄 许可证

MIT License

## 🎯 开发路线图

- [x] **C1**: AI Parser 接口设计（已完成）
  - OpenAI 集成
  - 四层降级策略
  - 对话历史管理
- [x] **C2**: 多AI提供商支持（已完成）
- [x] **C3**: 关键功能增强（已完成）
  - 编辑/暂停/恢复功能
  - 用户交互优化
  - 版本管理系统
- [x] **C4**: 对话上下文优化（已完成）
  - 会话状态管理
  - 智能建议系统
  - 测试覆盖率提升
- [ ] **C5**: 性能优化与监控增强（计划中）
- [ ] **C6**: 多语言支持（计划中）

详见 `docs/` 目录中的各阶段实施报告。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

### 开发规范

- 遵循 Go 代码规范
- 编写单元测试（覆盖率 >80%）
- 更新相关文档
- 运行 `make test` 确保测试通过

## 📞 支持

如有问题，请：
1. 查看 [文档](./docs/)
2. 查阅 [CLAUDE.md](./CLAUDE.md) 了解项目架构
3. 提交 [Issue](../../issues)
4. 联系维护者

## 🔗 相关链接

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [OpenAI API](https://platform.openai.com/docs)
- [项目文档](./docs/)
- [开发指南](./CLAUDE.md)

---

**版本**: v0.4.0-dev
**最后更新**: 2025-10-15
**作者**: chenwl
**许可证**: MIT

<div align="center">
  <p>⭐ 如果这个项目对你有帮助，请给一个 Star！</p>
  <p>Made with ❤️ in Beijing, China</p>
</div>