# MMemory - 架构文档

## 项目概述

**MMemory** 是一个基于 Telegram Bot 的智能提醒助手，通过对话式交互帮助用户管理日常习惯和任务提醒。

### 核心技术栈
- **编程语言**: Go 1.23+
- **数据库**: SQLite3
- **消息平台**: Telegram Bot API
- **监控**: Prometheus 指标收集
- **容器化**: Docker + Docker Compose
- **配置管理**: Viper (支持热更新)

### 核心特性
- 自然语言提醒设置
- 主动跟踪和关怀机制
- 灵活的提醒状态管理（完成/延期/跳过）
- 习惯养成和长期跟踪
- 实时监控和性能指标

## 项目结构

```
MMemory/
├── cmd/bot/                 # 主程序入口
│   └── main.go             # 应用启动和初始化
├── internal/               # 内部业务逻辑
│   ├── bot/               # Telegram Bot 处理层
│   │   ├── handlers/      # 消息和回调查询处理器
│   │   └── middleware/    # 中间件
│   ├── service/           # 业务服务层
│   │   ├── reminder.go    # 提醒服务
│   │   ├── scheduler.go   # 调度器服务
│   │   ├── notification.go # 通知服务
│   │   └── monitoring.go  # 监控服务
│   ├── repository/        # 数据访问层
│   │   └── sqlite/        # SQLite 实现
│   └── models/            # 数据模型
├── pkg/                   # 公共包
│   ├── config/            # 配置管理（支持热更新）
│   ├── logger/            # 日志工具
│   ├── metrics/           # Prometheus 指标定义
│   └── server/            # HTTP 服务器（指标端点）
├── configs/               # 配置文件
│   ├── config.yaml        # 主配置文件
│   └── alerts/            # 告警配置
├── scripts/               # 部署和管理脚本
├── test/                  # 测试文件
├── docs/                  # 项目文档
└── data/                  # 数据目录（SQLite 数据库）
```

## 核心功能模块

### 1. 提醒服务 (Reminder Service)
- 自然语言解析
- 提醒创建和管理
- 状态跟踪（活跃/完成/过期）

### 2. 调度器服务 (Scheduler Service)
- 基于 cron 的定时任务
- 多工作线程并发处理
- 失败重试机制

### 3. 通知服务 (Notification Service)
- Telegram 消息发送
- 主动关怀机制
- 超时处理

### 4. 监控服务 (Monitoring Service)
- Prometheus 指标收集
- 系统健康监控
- 性能指标追踪

## 数据库架构

### 主要数据表
- `users` - 用户信息
- `reminders` - 提醒配置
- `reminder_logs` - 提醒执行记录
- `conversations` - 对话上下文

### 数据模型
```go
type Reminder struct {
    ID          uint           `gorm:"primaryKey"`
    UserID      uint           `gorm:"not null;index"`
    Content     string         `gorm:"not null"`
    Schedule    string         `gorm:"not null"`
    Status      ReminderStatus `gorm:"default:'active'"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 扩展和定制

### 添加新提醒类型
1. 在 `internal/models/reminder.go` 中定义类型
2. 在 `internal/service/parser.go` 中实现解析逻辑
3. 在 `internal/service/reminder.go` 中处理业务逻辑

### 集成新通知渠道
1. 实现 `service.NotificationService` 接口
2. 在 `internal/service/notification.go` 中集成
3. 更新配置和依赖注入

## 相关文档

- [README.md](./README.md) - 项目介绍和使用指南
- [DEPLOYMENT.md](./DEPLOYMENT.md) - 详细部署说明
- [MMemory-Specs-v0.0.1.md](./MMemory-Specs-v0.0.1.md) - 技术规格文档
- [docs/](./docs/) - 开发文档目录