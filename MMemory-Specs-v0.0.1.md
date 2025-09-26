# MMemory - 智能提醒助手技术方案 v0.0.1

## 项目概述

MMemory 是一个基于 Telegram Bot 的智能提醒工具，通过对话式交互帮助用户管理日常习惯和任务提醒。

### 核心特色
- **对话式设置** - 通过自然语言添加提醒，无需复杂表单
- **主动跟踪** - 超时后主动询问进度和完成情况
- **灵活回复** - 支持完成、延期、跳过等多种状态
- **习惯养成** - 专门针对日常习惯的长期跟踪

## 用户需求分析

### 核心功能需求
1. **习惯提醒管理**
   - 每日定时提醒（如：每天19点提醒我复盘工作）
   - 每周固定时间提醒（如：每周三提醒我看书）
   - 自定义频率提醒

2. **任务提醒管理**
   - 一次性任务提醒（如：2024年10月1日提醒我交房租）
   - 带截止日期的任务跟踪

3. **智能交互**
   - 自然语言解析设置提醒
   - 多种回复方式：完成/延期/跳过/取消
   - 超时未回复时的主动关怀

4. **进度跟踪**
   - 记录每次提醒的完成状态
   - 统计习惯完成率
   - 提供进度反馈

### 交互流程示例
```
用户: "每天晚上8点提醒我复盘今天的工作"
Bot: "好的，已设置每天20:00的工作复盘提醒 ✅"

[20:00] Bot: "该复盘今天工作了，完成了吗？"
         [完成了] [延期1小时] [今天跳过]

用户点击: "完成了"
Bot: "太棒了！已记录今天的工作复盘完成 🎉"

[超时未回复场景]
[21:00] Bot: "工作复盘还没完成，需要帮助吗？或者遇到什么困难？"
```

## 技术架构方案

### 技术栈选择
- **后端语言**: Go 1.21+
- **Web框架**: Gin (轻量高性能)
- **数据库**: SQLite + GORM ORM
- **消息平台**: Telegram Bot API
- **定时任务**: robfig/cron v3
- **配置管理**: Viper
- **日志系统**: Logrus
- **部署方式**: Docker + Railway/Render

### 技术栈优势
- **Go语言**: 并发性能优秀，部署简单，适合学习实践
- **SQLite**: 轻量级，无需额外数据库服务
- **Telegram**: 免费稳定，支持丰富交互
- **Docker**: 一键部署，环境一致性

### 系统架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Telegram      │    │   Web Server    │    │   Scheduler     │
│   Bot API       │◄──►│   (Gin)         │◄──►│   (Cron)        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Business      │
                       │   Logic Layer   │
                       └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Repository    │
                       │   Layer (GORM)  │
                       └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   SQLite        │
                       │   Database      │
                       └─────────────────┘
```

### 项目目录结构
```
mmemory/
├── cmd/
│   └── bot/main.go              # 应用启动入口
├── internal/                    # 内部包，不对外暴露
│   ├── bot/                     # Telegram Bot 相关
│   │   ├── handlers/            # 消息处理器
│   │   │   ├── message.go       # 文本消息处理
│   │   │   ├── callback.go      # 按钮回调处理
│   │   │   └── command.go       # 命令处理
│   │   └── middleware/          # 中间件
│   │       └── auth.go          # 用户认证
│   ├── service/                 # 业务逻辑层
│   │   ├── reminder.go          # 提醒业务逻辑
│   │   ├── scheduler.go         # 定时任务管理
│   │   ├── parser.go            # 自然语言解析
│   │   └── notification.go      # 通知发送
│   ├── repository/              # 数据访问层
│   │   ├── interfaces.go        # 接口定义
│   │   └── sqlite/              # SQLite 实现
│   │       ├── user.go
│   │       ├── reminder.go
│   │       └── log.go
│   └── models/                  # 数据模型
│       ├── user.go
│       ├── reminder.go
│       └── log.go
├── pkg/                         # 对外公开的包
│   ├── config/                  # 配置管理
│   │   └── config.go
│   └── logger/                  # 日志工具
│       └── logger.go
├── migrations/                  # 数据库迁移文件
│   └── 001_initial.sql
├── configs/                     # 配置文件
│   └── config.yaml
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── scripts/                     # 部署脚本
│   ├── build.sh
│   └── deploy.sh
├── docs/                        # 文档
│   └── api.md
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

## 数据库设计

### 核心表结构

#### 用户表 (users)
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
    language_code VARCHAR(10) DEFAULT 'zh-CN',
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### 提醒配置表 (reminders)
```sql
CREATE TABLE reminders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL,  -- 'habit' | 'task'
    schedule_pattern VARCHAR(100) NOT NULL,  -- 'daily', 'weekly:1,3,5', 'once:2024-10-01'
    target_time TIME NOT NULL,  -- '19:00:00'
    timezone VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

#### 提醒记录表 (reminder_logs)
```sql
CREATE TABLE reminder_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reminder_id INTEGER NOT NULL,
    scheduled_time DATETIME NOT NULL,
    sent_time DATETIME,
    status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'sent', 'completed', 'skipped', 'overdue', 'cancelled'
    user_response TEXT,
    response_time DATETIME,
    follow_up_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reminder_id) REFERENCES reminders(id)
);
```

#### 对话上下文表 (conversations)
```sql
CREATE TABLE conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    context_type VARCHAR(50) NOT NULL,  -- 'creating_reminder', 'responding_reminder'
    context_data TEXT,  -- JSON 格式存储上下文信息
    expires_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### 关键字段说明
- **schedule_pattern**: 调度模式
  - `daily`: 每天
  - `weekly:1,3,5`: 每周一、三、五
  - `monthly:1,15`: 每月1号、15号
  - `once:2024-10-01`: 一次性，指定日期
- **status**: 提醒状态跟踪
- **context_data**: JSON格式存储对话状态，支持复杂交互流程

## 开发计划

### 第一阶段 - MVP基础版 (2-3天)
**目标**: 完成核心提醒功能

#### 任务列表
1. **项目初始化**
   - Go模块初始化 (`go mod init mmemory`)
   - 创建目录结构
   - 配置文件设置

2. **数据库层开发**
   - GORM模型定义
   - SQLite连接配置
   - 数据库迁移脚本
   - 基础CRUD操作

3. **Telegram Bot基础**
   - Bot API集成
   - Webhook设置
   - 基本消息接收和发送
   - 用户注册流程

4. **核心提醒服务**
   - 简单的提醒创建逻辑
   - 基础的定时发送功能
   - 用户回复处理

#### 预期结果
- 用户可以通过简单命令创建提醒
- Bot能够定时发送提醒消息
- 用户可以回复提醒状态

### 第二阶段 - 智能交互版 (2天)
**目标**: 提升用户体验和智能化程度

#### 任务列表
1. **定时调度系统**
   - robfig/cron集成
   - 调度任务持久化
   - 任务重启恢复机制

2. **自然语言解析**
   - 正则表达式匹配常见模式
   - 时间解析（支持中文时间描述）
   - 提醒类型自动识别

3. **状态管理系统**
   - 多种回复状态处理（完成/延期/跳过）
   - 延期逻辑实现
   - 状态统计功能

4. **超时处理机制**
   - 未回复检测
   - 自动关怀消息
   - 多次提醒逻辑

#### 预期结果
- 支持自然语言创建提醒
- 完整的状态流转
- 智能的超时关怀

### 第三阶段 - 完善优化版 (1天)
**目标**: 完善功能和部署

#### 任务列表
1. **功能完善**
   - 帮助命令系统
   - 提醒列表查看
   - 提醒编辑/删除功能
   - 统计数据展示

2. **Docker化部署**
   - Dockerfile编写
   - docker-compose配置
   - 环境变量配置

3. **云端部署**
   - Railway/Render部署配置
   - 持久化存储设置
   - 监控和日志配置

#### 预期结果
- 完整的功能体验
- 生产环境部署
- 稳定的服务运行

## 核心技术实现要点

### 1. 自然语言解析策略
```go
// 支持的模式示例
patterns := []Pattern{
    {
        Regex: `每天(\d{1,2})[点:](\d{2})?提醒我(.+)`,
        Type:  "daily",
    },
    {
        Regex: `每周([一二三四五六日,，\s]+)(\d{1,2})[点:](\d{2})?提醒我(.+)`,
        Type:  "weekly",
    },
    {
        Regex: `(\d{4})[年-](\d{1,2})[月-](\d{1,2})日?(\d{1,2})[点:](\d{2})?提醒我(.+)`,
        Type:  "once",
    },
}
```

### 2. 定时任务持久化
- 服务启动时从数据库恢复所有有效提醒
- 动态添加/删除cron任务
- 任务执行失败时的重试机制

### 3. 并发处理设计
- 每个用户消息使用独立的goroutine处理
- 使用context进行超时控制
- 合理使用channel进行组件间通信

### 4. 错误处理和监控
- 分层错误处理机制
- 关键操作的日志记录
- 用户友好的错误提示

## 部署方案

### 开发环境
```bash
# 启动开发环境
git clone <repository>
cd mmemory
go mod tidy
cp configs/config.example.yaml configs/config.yaml
# 配置 TELEGRAM_BOT_TOKEN
go run cmd/bot/main.go
```

### 生产环境部署
```bash
# Docker 部署
docker build -t mmemory:v0.0.1 .
docker run -d \
  -e TELEGRAM_BOT_TOKEN=your_token \
  -v /path/to/data:/app/data \
  mmemory:v0.0.1
```

### 云平台部署
- **Railway**: 支持Git自动部署
- **Render**: 免费额度，支持SQLite持久化
- **Heroku**: 需要配置PostgreSQL插件

## 测试策略

### 单元测试
- 核心业务逻辑测试
- 数据库操作测试
- 消息解析测试

### 集成测试
- Telegram Bot API集成测试
- 定时任务执行测试
- 端到端流程测试

### 性能测试
- 并发用户处理能力
- 数据库查询性能
- 内存使用情况

## 风险评估

### 技术风险
- **Go学习曲线**: 边学边做，可能影响开发进度
- **定时任务可靠性**: 需要确保重启后任务恢复
- **SQLite并发限制**: 用户量大时可能需要升级到PostgreSQL

### 业务风险
- **Telegram政策变化**: Bot API可能调整
- **用户习惯培养**: 需要足够的用户粘性
- **自然语言理解**: 初期识别能力有限

### 解决方案
- 采用渐进式开发，降低技术风险
- 设计可扩展的架构，便于后期升级
- 收集用户反馈，持续优化体验

## 后续发展规划

### v0.1.0 功能扩展
- 支持更复杂的重复模式
- 添加语音消息支持
- 团队协作功能

### v0.2.0 智能化升级
- AI驱动的自然语言理解
- 智能提醒时间推荐
- 个性化习惯分析

### v0.3.0 平台扩展
- 微信机器人支持
- Web端管理界面
- 移动端APP

---

**文档版本**: v0.0.1  
**创建日期**: 2024年9月26日  
**更新日期**: 2024年9月26日  
**作者**: 开发团队