# MMemory 智能提醒助手技术规范 v0.1.0

> **版本**: v0.1.0
> **更新日期**: 2026-01-01
> **作者**: MMemory Team

## 1. 项目概述

### 1.1 产品简介

MMemory 是一个基于 Telegram Bot 的 AI 智能提醒工具，通过自然对话帮助用户管理日常习惯和任务提醒。用户可以通过对话的方式创建提醒、管理习惯，AI 会自动理解用户的意图并提供智能建议。

### 1.2 核心特性

- **AI 智能对话**: 支持 OpenAI GPT-4o-mini 和 Claude 3.5 Sonnet 多种 AI 提供商
- **多意图识别与智能降级**: C3 四层智能降级机制，确保服务高可用性
- **上下文记忆**: 30 天对话历史记录，智能上下文管理
- **活动追踪与习惯分析**: 自动记录用户活动（喝水、读书、运动等），分析习惯形成情况
- **天气集成与条件触发**: 根据天气情况智能调整提醒触发条件
- **高级监控与成本控制**: Prometheus 指标监控，AI 使用成本实时追踪

### 1.3 技术栈

| 类别 | 技术 |
|------|------|
| 开发语言 | Go 1.21+ |
| 数据库 | SQLite 3 + GORM |
| Bot 框架 | Telegram Bot API |
| AI 服务 | OpenAI GPT-4o-mini, Claude 3.5 Sonnet |
| 任务调度 | robfig/cron v3 |
| 监控 | Prometheus + Grafana + Alertmanager |
| 配置管理 | Viper + YAML |
| 日志 | Logrus (JSON 格式) |

## 2. 系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      Telegram Bot API                        │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                      Bot Layer                               │
│         internal/bot/handlers (message.go, callback.go)      │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                     Service Layer                            │
│         internal/service/ (16+ services)                     │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                     AI Layer                                 │
│         internal/ai/ + pkg/ai/ (多Provider管理)              │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                   Repository Layer                           │
│         internal/repository/sqlite/                          │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                   SQLite Database                            │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 核心服务列表

| 服务名 | 职责 | 关键方法 |
|--------|------|----------|
| UserService | 用户管理 | CreateUser, GetByTelegramID, UpdateUser |
| ReminderService | 提醒业务 | CreateReminder, GetUserReminders, EditReminder |
| ReminderLogService | 记录跟踪 | MarkAsCompleted, MarkAsSkipped, GetUserStatistics |
| SchedulerService | 任务调度 | Start, Stop, AddReminder, RemoveReminder |
| NotificationService | 通知发送 | SendReminder, SendFollowUp |
| ConversationService | 对话管理 | CreateConversation, GetConversation, ClearConversation |
| ContextManagerService | 上下文管理 | ProcessMessage, GetContext, UpdateContextState |
| AIParserService | AI解析 | ParseIntent, FallbackParse |
| IntelligentAIManager | AI管理 | GetAIResponse, SelectProvider |
| WeatherService | 天气服务 | GetWeather, GetForecast |
| DailyActivityService | 活动记录 | RecordActivity, GetActivities |
| ActivityAnalysisService | 活动分析 | DetectPatterns, GenerateSuggestions |
| ActivityVisualizationService | 可视化 | GetActivityTrendChart, GetActivityHeatmap |
| PatternDetectorService | 模式检测 | DetectPatterns, GetPatternSuggestions |
| SuggestionService | 智能建议 | GenerateSuggestions |
| CostMonitorService | 成本监控 | TrackCost, GetBudgetStatus |
| MonitoringService | 系统监控 | CollectMetrics, StartCollector |
| ConditionEvaluatorService | 条件评估 | EvaluateConditions |

## 3. 数据库设计

### 3.1 用户表 (users)

用户表存储所有用户的基本信息，是整个系统的基础表。

```go
type User struct {
    ID           uint      `gorm:"primaryKey;autoIncrement"`
    TelegramID   int64     `gorm:"uniqueIndex;not null"`
    Username     string    `gorm:"size:255"`
    FirstName    string    `gorm:"size:255"`
    LastName     string    `gorm:"size:255"`
    Timezone     string    `gorm:"size:50;default:'Asia/Shanghai'"`
    LanguageCode string    `gorm:"size:10;default:'zh-CN'"`
    IsActive     bool      `gorm:"default:true"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

| 字段 | 类型 | 说明 |
|-----|------|------|
| ID | uint | 主键，自增 |
| TelegramID | int64 | Telegram 用户 ID，唯一索引，必填 |
| Username | string | 用户名 |
| FirstName | string | 名字 |
| LastName | string | 姓氏 |
| Timezone | string | 时区，默认 Asia/Shanghai |
| LanguageCode | string | 语言代码，默认 zh-CN |
| IsActive | bool | 是否激活，默认 true |
| CreatedAt | time.Time | 创建时间 |
| UpdatedAt | time.Time | 更新时间 |

### 3.2 提醒配置表 (reminders)

提醒配置表存储用户创建的提醒规则，包括习惯型和任务型两种类型。

```go
type Reminder struct {
    ID              uint           `gorm:"primaryKey;autoIncrement"`
    UserID          uint           `gorm:"not null;index"`
    Title           string         `gorm:"size:500;not null"`
    Description     string         `gorm:"type:text"`
    Type            ReminderType   `gorm:"size:20;not null"` // habit/task
    SchedulePattern string         `gorm:"size:100;not null"` // daily, weekly:1,3,5, once:2024-10-01
    TargetTime      string         `gorm:"size:8;not null"` // HH:MM:SS
    Timezone        string         `gorm:"size:50"`
    IsActive        bool           `gorm:"default:true"`
    PausedUntil     *time.Time     `gorm:"index"`
    PauseReason     string         `gorm:"type:text"`
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

| 字段 | 类型 | 说明 |
|-----|------|------|
| ID | uint | 主键，自增 |
| UserID | uint | 关联用户 ID，索引 |
| Title | string | 提醒标题，必填，最大 500 字符 |
| Description | string | 提醒描述，文本类型 |
| Type | ReminderType | 提醒类型，habit 或 task |
| SchedulePattern | string | 调度模式，daily/weekly:1,3,5/monthly:1,15/once:2024-10-01 |
| TargetTime | string | 目标时间，格式 HH:MM:SS |
| Timezone | string | 时区 |
| IsActive | bool | 是否激活，默认 true |
| PausedUntil | *time.Time | 暂停截止时间，索引 |
| PauseReason | string | 暂停原因 |
| CreatedAt | time.Time | 创建时间 |
| UpdatedAt | time.Time | 更新时间 |

### 3.3 提醒记录表 (reminder_logs)

提醒记录表追踪每次提醒的执行情况，包括发送状态和用户响应。

```go
type ReminderLog struct {
    ID            uint           `gorm:"primaryKey;autoIncrement"`
    ReminderID    uint           `gorm:"not null;index"`
    ScheduledTime time.Time      `gorm:"not null"`
    SentTime      *time.Time
    Status        ReminderStatus `gorm:"size:20;default:'pending'` // pending/sent/completed/skipped/overdue/cancelled
    UserResponse  string         `gorm:"type:text"`
    ResponseTime  *time.Time
    FollowUpCount int            `gorm:"default:0"`
    CreatedAt     time.Time
}
```

| 字段 | 类型 | 说明 |
|-----|------|------|
| ID | uint | 主键，自增 |
| ReminderID | uint | 关联提醒 ID，索引 |
| ScheduledTime | time.Time | 计划发送时间，必填 |
| SentTime | *time.Time | 实际发送时间 |
| Status | ReminderStatus | 状态，pending/sent/completed/skipped/overdue/cancelled |
| UserResponse | string | 用户响应内容 |
| ResponseTime | *time.Time | 用户响应时间 |
| FollowUpCount | int | 跟进次数，默认 0 |
| CreatedAt | time.Time | 创建时间 |

### 3.4 对话上下文表 (conversations)

对话上下文表管理多轮对话的上下文状态，支持创建提醒、编辑提醒等交互流程。

```go
type Conversation struct {
    ID          uint        `gorm:"primaryKey;autoIncrement"`
    UserID      uint        `gorm:"not null;index"`
    ContextType ContextType `gorm:"size:50;not null"` // creating_reminder/responding_reminder/editing_reminder/chat
    ContextData string      `gorm:"type:text"` // JSON
    ExpiresAt   *time.Time
    CreatedAt   time.Time
}
```

| 字段 | 类型 | 说明 |
|-----|------|------|
| ID | uint | 主键，自增 |
| UserID | uint | 关联用户 ID，索引 |
| ContextType | ContextType | 上下文类型，creating_reminder/responding_reminder/editing_reminder/chat |
| ContextData | string | 上下文数据，JSON 格式 |
| ExpiresAt | *time.Time | 过期时间 |
| CreatedAt | time.Time | 创建时间 |

### 3.5 日常活动表 (daily_activities)

日常活动表记录用户的各种活动，用于习惯追踪和分析。

```go
type DailyActivity struct {
    ID           uint           `gorm:"primaryKey;autoIncrement"`
    UserID       uint           `gorm:"not null;index"`
    ActivityType ActivityType   `gorm:"size:50;not null;index"` // drink_water/take_medicine/read_book/exercise/sleep/eat/custom
    OccurredAt   time.Time      `gorm:"not null;index"`
    Details      string         `gorm:"type:text"` // JSON
    Source       ActivitySource `gorm:"size:20;not null;default:'conversation'` // conversation/manual/reminder
    Metadata     string         `gorm:"type:text"` // JSON
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

| 字段 | 类型 | 说明 |
|-----|------|------|
| ID | uint | 主键，自增 |
| UserID | uint | 关联用户 ID，索引 |
| ActivityType | ActivityType | 活动类型，drink_water/take_medicine/read_book/exercise/sleep/eat/custom |
| OccurredAt | time.Time | 发生时间，索引 |
| Details | string | 活动详情，JSON 格式 |
| Source | ActivitySource | 来源，conversation/manual/reminder |
| Metadata | string | 元数据，JSON 格式 |
| CreatedAt | time.Time | 创建时间 |
| UpdatedAt | time.Time | 更新时间 |

### 3.6 对话状态表 (conversation_contexts)

对话状态表存储对话的详细状态信息，支持 30 天对话历史的智能管理。

```go
type ConversationContext struct {
    ID           uint       `gorm:"primaryKey;autoIncrement"`
    UserID       uint       `gorm:"not null;index"`
    SessionID    string     `gorm:"size:64;not null"`
    State        string     `gorm:"size:64;not null"`
    Intent       string     `gorm:"size:64;not null"`
    Channel      string     `gorm:"size:32"`
    Locale       string     `gorm:"size:10"`
    MessagesJSON string     `gorm:"type:text;not null"`
    EntitiesJSON string     `gorm:"type:text"`
    TTLSeconds   int64      `gorm:"not null;default:0`
    LastActivity time.Time  `gorm:"not null"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
    ExpiresAt    *time.Time
}
```

| 字段 | 类型 | 说明 |
|-----|------|------|
| ID | uint | 主键，自增 |
| UserID | uint | 关联用户 ID，索引 |
| SessionID | string | 会话 ID |
| State | string | 当前状态 |
| Intent | string | 用户意图 |
| Channel | string | 通信渠道 |
| Locale | string | 区域设置 |
| MessagesJSON | string | 消息列表，JSON 格式，必填 |
| EntitiesJSON | string | 实体列表，JSON 格式 |
| TTLSeconds | int64 | TTL 秒数 |
| LastActivity | time.Time | 最后活动时间 |
| CreatedAt | time.Time | 创建时间 |
| UpdatedAt | time.Time | 更新时间 |
| ExpiresAt | *time.Time | 过期时间 |

## 4. AI 集成架构

### 4.1 多 Provider 支持

MMemory 支持多种 AI 提供商，实现智能选择和故障转移：

| Provider | 主模型 | 备用模型 | 特点 |
|----------|--------|----------|------|
| OpenAI | gpt-4o-mini | gpt-3.5-turbo | 成本效益高，响应快速 |
| Claude | Claude 3.5 Sonnet | - | 推理能力强，支持长上下文 |

### 4.2 C3 智能降级机制

C3（Cascading Cost-optimized Confidence-based）智能降级机制是 MMemory 的核心创新，提供四层降级保障：

1. **Primary AI**: OpenAI GPT-4o-mini，作为主解析器处理大部分请求
2. **Secondary AI**: Claude 3.5 Sonnet，当主 AI 出现问题或置信度不足时切换
3. **Enhanced Regex Parser**: 上下文感知的传统模式匹配，作为第三层降级
4. **Intelligent Fallback**: 保留上下文的简单聊天响应，确保用户始终能得到回复

### 4.3 AI 配置结构

```go
type AIConfig struct {
    Enabled           bool          // 是否启用 AI 功能
    OpenAI            OpenAIConfig  // OpenAI 配置
    Claude            ClaudeConfig  // Claude 配置（可选）
    Temperature       float64       // 温度参数
    MaxTokens         int           // 最大 token 数
    Timeout           time.Duration // 超时时间
    MaxRetries        int           // 最大重试次数
}

type OpenAIConfig struct {
    APIKey        string        // API 密钥
    BaseURL       string        // API 地址
    PrimaryModel  string        // 主模型（gpt-4o-mini）
    BackupModel   string        // 备用模型（gpt-3.5-turbo）
    Temperature   float64       // 温度（0.1）
    MaxTokens     int           // 最大 token（1000）
    Timeout       time.Duration // 超时（30s）
    MaxRetries    int           // 最大重试（3）
}
```

### 4.4 成本控制配置

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| MMEMORY_AI_C3_ENABLED | 启用 C3 智能降级 | true |
| MMEMORY_AI_C3_BUDGET_LIMIT_PER_USER | 每用户每日预算（USD） | 10.0 |
| MMEMORY_AI_C3_GLOBAL_BUDGET_LIMIT | 全局每日预算（USD） | 100.0 |
| MMEMORY_AI_C3_DEGRADATION_THRESHOLD | 降级置信度阈值 | 0.7 |
| MMEMORY_AI_FALLBACK_CONFIDENCE_THRESHOLD | 最小置信度阈值 | 0.6 |

## 5. 开发阶段 (C1-C6)

### 5.1 C1: AI Parser 接口设计

**完成时间**: 2025-10-10

C1 阶段完成了 AI Parser 的基础架构设计，包括：

- OpenAI 客户端集成，支持 GPT-4o-mini 和 GPT-3.5-turbo
- Fallback 链实现，确保 AI 服务不可用时的降级处理
- 对话历史管理，支持上下文相关的意图理解
- 统一的 AI 解析接口设计

### 5.2 C2: Multi-AI Provider Support

**完成时间**: 2025-10-15

C2 阶段引入了多 AI 提供商支持：

- Claude AI 集成，新增 Claude 3.5 Sonnet Provider
- 动态 Provider 选择机制，根据请求类型和成本选择最佳 Provider
- 成本跟踪系统，记录每个 AI 请求的 token 消耗和费用
- Provider 健康检查，自动检测并移除不可用的 Provider

### 5.3 C3: Intelligent Degradation

**完成时间**: 2025-10-20

C3 阶段实现了智能降级机制：

- 成本控制，每用户每日预算和全局预算限制
- 置信度降级，当 AI 返回置信度低于阈值时自动降级
- 实时监控，Prometheus 指标收集和 Grafana 仪表板
- 四层降级策略：OpenAI -> Claude -> Regex Parser -> Fallback Response

### 5.4 C4: Enhanced Features

**完成时间**: 2025-11-09

C4 阶段增强了系统的功能特性：

- 天气服务集成，支持根据天气条件触发提醒
- 上下文感知建议，基于用户行为模式提供个性化建议
- Bot API 改进，优化消息处理和回调响应
- 用户偏好学习，个性化提醒时间和频率

### 5.5 C5: Performance Optimization

**完成时间**: 2025-12-30

C5 阶段进行了全面的性能优化：

- 数据库连接池优化，GORM 连接池参数调优
- 动态 Worker 池，根据负载自动调整并发工作线程数
- AI 成本降低，优化 Prompt 减少 token 消耗
- 缓存机制，减少重复的 AI 请求

### 5.6 C6: User Life Recording

**完成时间**: 2025-12-31

C6 阶段实现了用户生活记录系统：

- 活动追踪，自动识别和记录用户活动（喝水、读书、运动等）
- AI 驱动的活动提取，从对话中智能提取活动信息
- 模式检测，识别用户行为模式和习惯形成情况
- ASCII 可视化，活动趋势图表和热力图展示
- 习惯形成分析，评估用户习惯的稳定性和发展阶段

## 6. 配置说明

### 6.1 基础配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| MMEMORY_BOT_TOKEN | Telegram Bot Token | 必填 |
| MMEMORY_DATABASE_DSN | SQLite 文件路径 | ./data/mmemory.db |
| MMEMORY_SERVER_PORT | HTTP 服务端口 | 8080 |
| MMEMORY_LOG_LEVEL | 日志级别 | info |
| MMEMORY_LOG_FORMAT | 日志格式 | json |
| MMEMORY_MONITORING_ENABLED | 启用监控 | false |

### 6.2 AI 配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| MMEMORY_AI_ENABLED | 启用 AI 功能 | false |
| MMEMORY_AI_OPENAI_API_KEY | OpenAI API Key | 必填（AI 启用时） |
| MMEMORY_AI_OPENAI_BASE_URL | API 端点 | https://api.openai.com/v1 |
| MMEMORY_AI_OPENAI_PRIMARY_MODEL | 主模型 | gpt-4o-mini |
| MMEMORY_AI_OPENAI_BACKUP_MODEL | 备用模型 | gpt-3.5-turbo |
| MMEMORY_AI_CLAUDE_API_KEY | Claude API Key | 可选 |
| MMEMORY_AI_TEMPERATURE | 温度参数 | 0.1 |
| MMEMORY_AI_MAX_TOKENS | 最大 token 数 | 1000 |
| MMEMORY_AI_TIMEOUT | AI 请求超时 | 30s |
| MMEMORY_AI_MAX_RETRIES | 最大重试次数 | 3 |

### 6.3 天气服务配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| MMEMORY_WEATHER_API_KEY | 天气 API Key | 可选 |
| MMEMORY_WEATHER_BASE_URL | API 端点 | 可选 |
| MMEMORY_WEATHER_TIMEOUT | 请求超时 | 10s |

## 7. 开发命令

### 7.1 Makefile 命令

```bash
make help           # 查看所有可用命令
make build          # 构建项目（输出到 bin/mmemory）
make run            # 运行应用
make test           # 运行测试
make test-cover     # 运行测试并生成覆盖率报告
make clean          # 清理构建产物
make docker-build   # 构建 Docker 镜像
make docker-up      # 启动容器
make docker-down    # 停止容器
make docker-rebuild # 重新构建并启动
make docker-logs    # 查看日志
make fmt            # 格式化代码
make tidy           # 整理依赖
make lint           # 代码质量检查
make version        # 显示版本信息
```

### 7.2 手动命令

```bash
go run cmd/bot/main.go              # 运行应用
go test ./...                       # 运行所有测试
go test -v ./internal/service       # 运行服务层测试
go test ./pkg/config -run TestConfig # 运行配置测试
go build -o bin/mmemory cmd/bot/main.go  # 手动构建
go run -race cmd/bot/main.go        # 运行并检测竞态条件
```

## 8. 核心数据结构

### 8.1 UserStatistics

用户统计信息，用于展示用户的提醒完成情况。

```go
type UserStatistics struct {
    TotalReminders   int // 总提醒数
    ActiveReminders  int // 活跃提醒数
    CompletedToday   int // 今日完成数
    CompletedWeek    int // 本周完成数
    CompletedMonth   int // 本月完成数
    SkippedToday     int // 今日跳过数
    CompletionRate   int // 完成率（百分比）
    LongestStreak    int // 最长连续天数
    CurrentStreak    int // 当前连续天数
}
```

### 8.2 ActivityStatistics

活动统计信息，用于展示用户活动记录的分析结果。

```go
type ActivityStatistics struct {
    TotalActivities  int64            // 总活动数
    ByType           map[string]int64 // 按类型统计
    ByDay            map[string]int64 // 按天统计
    MostActiveDay    string           // 最活跃日
    MostActiveType   string           // 最活跃类型
    DailyAverage     float64          // 日均
    Trend            string           // 趋势（up/down/stable）
    WeeklyData       []DailyData      // 周数据
}
```

### 8.3 ActivityPattern

活动模式信息，描述用户的习惯形成情况。

```go
type ActivityPattern struct {
    PatternID           string              // 模式 ID
    ActivityType        string              // 活动类型
    PatternType         string              // 模式类型（daily/weekly/monthly）
    Frequency           float64             // 完成频率
    AverageTime         string              // 平均执行时间
    TimeVariance        float64             // 时间方差
    Streak              int                 // 当前连续天数
    LongestStreak       int                 // 最长连续天数
    ConsistencyScore    float64             // 一致性评分（0-100）
    FirstRecorded       time.Time           // 首次记录
    LastRecorded        time.Time           // 最后记录
    WeeklyDistribution  map[string]float64  // 每周分布
    HourlyDistribution  map[int]float64     // 每小时分布
}
```

### 8.4 HabitFormationReport

习惯形成报告，评估用户习惯的发展阶段。

```go
type HabitFormationReport struct {
    ActivityType      string   // 活动类型
    TotalDays         int      // 总天数
    CompletionRate    float64  // 完成率（0.0-1.0）
    CurrentStreak     int      // 当前连续天数
    LongestStreak     int      // 最长连续天数
    Stage             string   // 阶段（initiating/forming/established/master）
    StageProgress     float64  // 阶段进度（0.0-1.0）
    BestDayOfWeek     string   // 一周最佳日
    BestTimeOfDay     string   // 一天最佳时间
    ConsistencyScore  float64  // 一致性评分（0-100）
    Recommendations   []string // 建议列表
}
```

### 8.5 ActivityAnomaly

活动异常信息，检测用户的异常行为模式。

```go
type ActivityAnomaly struct {
    AnomalyID        string     // 异常 ID
    UserID           uint       // 用户 ID
    ActivityType     string     // 活动类型
    AnomalyType      string     // 异常类型（missing/late/early/overdue/frequency_drop）
    Severity         string     // 严重程度（low/medium/high）
    Description      string     // 描述
    ExpectedTime     string     // 期望时间
    ActualTime       string     // 实际时间
    ExpectedCount    int        // 期望数量
    ActualCount      int        // 实际数量
    DetectedAt       time.Time  // 检测时间
    Resolved         bool       // 是否已解决
    ResolvedAt       *time.Time // 解决时间
}
```

### 8.6 ActivityInsights

活动洞察，提供综合的活动分析结果。

```go
type ActivityInsights struct {
    UserID            uint                  // 用户 ID
    Period            string                // 时间周期
    MostConsistent    []ActivityPattern     // 最一致的活动
    NeedsAttention    []ActivityAnomaly     // 需要关注的异常
    TopSuggestions    []ActivitySuggestion  // 顶部建议
    OverallScore      float64               // 综合评分（0-100）
    Summary           string                // 摘要
    GeneratedAt       time.Time             // 生成时间
}
```

## 9. 监控与告警

### 9.1 Prometheus 指标

系统提供以下核心指标：

- **提醒相关**: reminder_total, reminder_sent, reminder_completed, reminder_skipped
- **AI 相关**: ai_requests_total, ai_tokens_used, ai_cost_usd, ai_latency_seconds
- **活动相关**: activity_total, activity_by_type
- **系统指标**: goroutines, memory_usage, database_connections

### 9.2 监控端点

| 端点 | 用途 |
|------|------|
| /metrics | Prometheus 指标采集 |
| /health | 健康检查 |
| /ready | 就绪检查 |

### 9.3 Grafana 仪表板

- 系统概览仪表板：CPU、内存、goroutine 数量
- AI 成本仪表板：每日/每周/每月 AI 成本趋势
- 提醒统计仪表板：完成率、跳过率统计
- 用户活动仪表板：活动追踪和习惯分析

## 10. 数据库操作

### 10.1 自动迁移

系统启动时自动执行数据库迁移，创建所需的表结构：

```go
// 自动迁移 schema
db.AutoMigrate(&User{}, &Reminder{}, &ReminderLog{}, &Conversation{}, &DailyActivity{}, &ConversationContext{})
```

### 10.2 数据库索引

| 表名 | 索引字段 |
|------|----------|
| users | TelegramID |
| reminders | UserID, PausedUntil |
| reminder_logs | ReminderID |
| conversations | UserID |
| daily_activities | UserID, ActivityType, OccurredAt |
| conversation_contexts | UserID |

## 附录 A: 与 v0.0.1 差异对比

| 维度 | v0.0.1 | v0.1.0 |
|------|--------|--------|
| AI 集成 | 无（计划 v0.2.0） | OpenAI + Claude（C1-C3） |
| 服务数量 | 4 | 16+ |
| 监控 | 无 | Prometheus + Grafana |
| 活动追踪 | 无 | C6 完整支持 |
| 降级策略 | 无 | C3 四层智能降级 |
| 对话历史 | 基础 | 30 天 + 状态机 |
| 数据库表 | 4 | 6+ |
| 天气集成 | 无 | C4 支持 |
| 成本控制 | 无 | C3 预算管理 |
| 测试覆盖率 | - | ~60% |
| 版本号 | v0.0.1 | v0.4.0-dev |

## 附录 B: 文件结构

```
MMemory/
├── cmd/
│   └── bot/
│       └── main.go              # 应用入口
├── internal/
│   ├── ai/
│   │   ├── client.go            # OpenAI 客户端
│   │   └── parser.go            # AI 解析器
│   ├── bot/
│   │   ├── handlers/
│   │   │   ├── message.go       # 消息处理器
│   │   │   ├── callback.go      # 回调处理器
│   │   │   └── commands.go      # 命令处理器
│   │   └── bot.go               # Bot 初始化
│   ├── models/
│   │   ├── user.go              # 用户模型
│   │   ├── reminder.go          # 提醒模型
│   │   ├── activity.go          # 活动模型
│   │   └── context.go           # 上下文模型
│   ├── repository/
│   │   ├── user.go              # 用户仓储
│   │   ├── reminder.go          # 提醒仓储
│   │   └── sqlite/
│   │       └── sqlite.go        # SQLite 实现
│   └── service/
│       ├── reminder.go          # 提醒服务
│       ├── scheduler.go         # 调度服务
│       ├── notification.go      # 通知服务
│       ├── ai_parser.go         # AI 解析服务
│       ├── context_manager.go   # 上下文管理服务
│       ├── activity.go          # 活动服务
│       └── weather.go           # 天气服务
├── pkg/
│   ├── ai/
│   │   ├── manager.go           # AI 管理器
│   │   ├── provider.go          # Provider 接口
│   │   ├── openai.go            # OpenAI Provider
│   │   └── claude.go            # Claude Provider
│   ├── config/
│   │   ├── config.go            # 配置管理
│   │   └── config_test.go       # 配置测试
│   ├── logger/
│   │   └── logger.go            # 日志工具
│   ├── metrics/
│   │   └── metrics.go           # Prometheus 指标
│   └── server/
│       └── server.go            # HTTP 服务器
├── configs/
│   ├── config.example.yaml      # 配置示例
│   └── config.full.yaml         # 完整配置
├── bin/
│   └── mmemory                  # 编译产物
├── data/                        # 数据目录
├── docker-compose.yml           # Docker 编排
├── Dockerfile                   # Docker 构建
├── Makefile                     # 构建脚本
├── go.mod                       # Go 模块
├── go.sum                       # Go 依赖校验
└── MMemory-Specs-v0.1.0.md      # 本技术规范文档
```

## 附录 C: 快速入门

### C.1 环境准备

```bash
# 1. 安装 Go 1.21+
go version

# 2. 克隆项目
git clone https://github.com/your-org/MMemory.git
cd MMemory

# 3. 安装依赖
go mod tidy

# 4. 复制配置
cp configs/config.example.yaml configs/config.yaml
```

### C.2 配置 Bot Token

编辑 `configs/config.yaml`：

```yaml
bot:
  token: "${TELEGRAM_BOT_TOKEN}"  # 或直接填写 Token
```

### C.3 运行应用

```bash
# 开发模式运行
make run

# 或构建后运行
make build
./bin/mmemory
```

### C.4 测试

```bash
# 运行所有测试
make test

# 运行测试并查看覆盖率
make test-cover

# 查看覆盖率报告
open cover.html
```

## 变更日志

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| v0.1.0 | 2026-01-01 | 初始版本，包含 C1-C6 完整功能 |

---

**文档版本**: v0.1.0
**最后更新**: 2026-01-01
**维护者**: MMemory Team
