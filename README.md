# MMemory - 智能提醒助手

基于 Telegram Bot 的智能提醒工具，通过对话式交互帮助用户管理日常习惯和任务提醒。

## 🌟 特性

- **对话式设置** - 通过自然语言添加提醒，无需复杂表单
- **主动跟踪** - 超时后主动询问进度和完成情况  
- **灵活回复** - 支持完成、延期、跳过等多种状态
- **习惯养成** - 专门针对日常习惯的长期跟踪

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
# 编辑 configs/config.yaml，设置你的 TELEGRAM_BOT_TOKEN
```

4. 运行
```bash
go run cmd/bot/main.go
```

### 配置说明

在 `configs/config.yaml` 中设置：

```yaml
bot:
  token: "your_telegram_bot_token_here"  # 必填：你的Bot Token
  debug: false                           # 可选：调试模式

database:
  dsn: "./data/mmemory.db"              # 数据库文件路径

logging:
  level: "info"                         # 日志级别
  format: "json"                        # 日志格式

app:
  environment: "production"             # 运行环境
```

## 💬 使用方法

### 支持的提醒格式

- **每日提醒**: "每天19点提醒我复盘工作"
- **每周提醒**: "每周一三五19点提醒我健身"  
- **一次性提醒**: "2024年10月1日提醒我交房租"
- **明天提醒**: "明天上午10点提醒我开会"

### Bot 命令

- `/start` - 开始使用
- `/help` - 查看帮助
- `/list` - 查看提醒列表

### 交互示例

```
用户: "每天晚上8点提醒我复盘今天的工作"
Bot: "✅ 提醒已设置成功！
     📝 复盘今天的工作
     ⏰ 每天 20:00"

[20:00] Bot: "该复盘今天工作了，完成了吗？"
        [完成了] [延期1小时] [今天跳过]
```

## 🏗️ 项目结构

```
mmemory/
├── cmd/bot/                 # 主程序入口
├── internal/                # 内部包
│   ├── bot/handlers/        # Telegram 消息处理
│   ├── service/             # 业务逻辑层
│   ├── repository/          # 数据访问层
│   └── models/              # 数据模型
├── pkg/                     # 公共包
│   ├── config/              # 配置管理
│   └── logger/              # 日志工具
├── configs/                 # 配置文件
├── data/                    # 数据目录
└── docs/                    # 文档
```

## 🛠️ 开发

### 构建

```bash
# 开发环境运行
go run cmd/bot/main.go

# 构建生产版本
go build -o bin/mmemory cmd/bot/main.go
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./internal/service
```

## 📦 部署

### Docker 部署

```bash
# 构建镜像
docker build -t mmemory:v0.0.1 .

# 运行容器
docker run -d \
  -e TELEGRAM_BOT_TOKEN=your_token \
  -v /path/to/data:/app/data \
  mmemory:v0.0.1
```

### 云平台部署

支持部署到：
- Railway
- Render  
- Heroku

## 🗂️ 数据库

使用 SQLite 存储数据，包含以下表：

- `users` - 用户信息
- `reminders` - 提醒配置
- `reminder_logs` - 提醒记录
- `conversations` - 对话上下文

数据库会在首次运行时自动创建和迁移。

## 🔧 环境变量

| 变量名 | 说明 | 必填 |
|--------|------|------|
| `TELEGRAM_BOT_TOKEN` | Telegram Bot Token | 是 |
| `DATABASE_PATH` | 数据库文件路径 | 否 |
| `LOG_LEVEL` | 日志级别 | 否 |

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📞 支持

如有问题，请：
1. 查看 [文档](./docs/)
2. 提交 [Issue](../../issues)
3. 联系维护者

---

**版本**: v0.0.1  
**更新时间**: 2024年9月26日