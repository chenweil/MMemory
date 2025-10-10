# C1: AI解析器接口设计实施方案 - 2025年9月29日

## 📋 项目概述

基于阶段3计划中的C1任务"AI解析器接口设计"，本文档详细说明了智能解析器的设计方案和实施计划。该方案将为MMemory项目引入强大的自然语言理解能力，支持复杂的提醒创建和智能对话功能。

## 🎯 核心目标

### 功能目标
- **智能提醒解析**: 理解自然语言中的时间、日期、待办事项
- **智能对话功能**: 支持书籍介绍、日常交流等对话场景
- **降级保障机制**: 确保服务高可用性的多层降级策略
- **上下文管理**: 30天对话历史，支持多轮对话理解

### 技术目标
- 设计统一的 `AIParserService` 接口
- 定义 `ReminderParseResult` 和 `TimeInfo` 数据结构
- 实现错误处理和重试机制
- 集成OpenAI API，支持GPT-4/3.5模型

## 🏗️ 技术架构设计

### 核心接口设计

```go
// AIParserService - 智能解析器服务接口
type AIParserService interface {
    // 智能解析 - 支持多种意图
    ParseMessage(ctx context.Context, userID string, message string) (*ParseResult, error)
    
    // 对话功能
    Chat(ctx context.Context, userID string, message string) (*ChatResponse, error)
    
    // 降级处理
    SetFallbackParser(parser TraditionalParser) error
}

// ParseResult - 解析结果统一结构
type ParseResult struct {
    // 解析意图
    Intent       ParseIntent     `json:"intent"`         // reminder/chat/summary等
    Confidence   float32         `json:"confidence"`     // 0.0-1.0
    
    // 提醒相关（当Intent为reminder时）
    Reminder     *ReminderInfo   `json:"reminder,omitempty"`
    
    // 对话相关（当Intent为chat时）  
    ChatResponse *ChatInfo       `json:"chat_response,omitempty"`
    
    // 元信息
    ParsedBy     string          `json:"parsed_by"`      // "openai-gpt-4"
    ProcessTime  time.Duration   `json:"process_time"`
}

// 意图类型定义
type ParseIntent string
const (
    IntentReminder     ParseIntent = "reminder"      // 创建提醒
    IntentChat         ParseIntent = "chat"          // 普通对话
    IntentSummary      ParseIntent = "summary"       // 总结请求
    IntentQuery        ParseIntent = "query"         // 查询提醒
    IntentUnknown      ParseIntent = "unknown"       // 未知意图
)
```

### 数据结构设计

```go
// ReminderInfo - 提醒信息结构
type ReminderInfo struct {
    Title           string                    `json:"title"`
    Type            models.ReminderType       `json:"type"`
    Time            TimeInfo                  `json:"time"`
    SchedulePattern models.SchedulePattern    `json:"schedule_pattern"`
    Description     string                    `json:"description,omitempty"`
}

// TimeInfo - 时间信息结构
type TimeInfo struct {
    Hour            int       `json:"hour"`
    Minute          int       `json:"minute"`
    Timezone        string    `json:"timezone"`
    ScheduleDetails string    `json:"schedule_details"` // "weekly:1,3,5"
    IsRelativeTime  bool      `json:"is_relative_time"` // 是否为相对时间
    RelativeDesc    string    `json:"relative_desc,omitempty"` // "明天", "下周一"
}

// ChatInfo - 对话信息结构
type ChatInfo struct {
    Response        string    `json:"response"`
    NeedFollowUp    bool      `json:"need_follow_up"`
    FollowUpPrompt  string    `json:"follow_up_prompt,omitempty"`
}
```

## 🔄 降级策略设计

### 四层降级机制

基于用户需求设计的降级策略：**主AI → 兜底AI → 正则 → 兜底对话**

```go
// FallbackStrategy - 降级策略实现
type FallbackStrategy struct {
    PrimaryAI     Parser  // OpenAI GPT-4/3.5 (主要AI)
    BackupAI      Parser  // OpenAI GPT-3.5-turbo (兜底AI)
    RegexParser   Parser  // 传统正则解析
    ChatFallback  Parser  // 兜底对话: "我没理解你说的内容"
}

func (f *FallbackStrategy) Parse(ctx context.Context, userID string, message string) (*ParseResult, error) {
    parsers := []Parser{f.PrimaryAI, f.BackupAI, f.RegexParser, f.ChatFallback}
    
    var lastErr error
    for _, parser := range parsers {
        result, err := parser.Parse(ctx, userID, message)
        if err == nil && result != nil {
            result.ParsedBy = parser.GetName()
            return result, nil
        }
        lastErr = err
        logger.Warnf("Parser %s failed: %v", parser.GetName(), err)
    }
    
    return nil, fmt.Errorf("all parsers failed, last error: %w", lastErr)
}
```

### 错误处理机制

```go
// ErrorHandler - 错误处理器
type ErrorHandler struct {
    maxRetries     int
    retryDelay     time.Duration
    circuitBreaker *CircuitBreaker
}

func (h *ErrorHandler) HandleAIError(err error) (shouldRetry bool, shouldFallback bool) {
    switch {
    case isRateLimitError(err):
        return true, false  // 重试，不降级
    case isNetworkError(err):
        return true, false  // 重试，不降级
    case isAuthError(err):
        return false, true  // 不重试，直接降级
    case isModelError(err):
        return false, true  // 不重试，直接降级
    default:
        return false, true  // 未知错误，降级
    }
}
```

## 🤖 OpenAI集成方案

### 配置管理

```yaml
# configs/config.yaml - AI配置部分
ai:
  enabled: true
  
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    
    # 模型配置 - 基于成本效益考虑
    primary_model: "gpt-4o-mini"     # 主要模型：性价比好
    backup_model: "gpt-3.5-turbo"   # 兜底模型：成本低
    
    # 调用参数
    temperature: 0.1        # 低温度，更确定性的输出
    max_tokens: 1000       # 最大输出token数
    timeout: 30s           # 请求超时
    max_retries: 3         # 最大重试次数
    
  # Prompt模板配置
  prompts:
    reminder_parse: |
      你是MMemory的智能助手。请分析用户消息，识别意图并返回JSON格式结果。
      
      当前时间: {{.CurrentTime}}
      用户消息: "{{.Message}}"
      对话历史: {{.ConversationHistory}}
      
      支持的功能:
      1. 创建提醒 - 用户想要设置提醒、待办、日程
      2. 普通对话 - 用户想要聊天、询问信息  
      3. 查询总结 - 用户想要查看或总结某些内容
      
      时间格式说明:
      - 支持绝对时间: "明天8点", "下周一9点"
      - 支持相对时间: "1小时后", "明天"
      - 支持重复模式: "每天", "每周一三五", "工作日"
      
      请返回以下JSON格式(不要包含markdown代码块标记):
      {
        "intent": "reminder|chat|summary|query",
        "confidence": 0.95,
        "reminder": {
          "title": "具体要做的事情",
          "type": "habit|task",
          "time": {
            "hour": 8,
            "minute": 0,
            "timezone": "Asia/Shanghai",
            "is_relative_time": false,
            "relative_desc": ""
          },
          "schedule_pattern": "daily|weekly:1,3,5|monthly:1,15|once"
        },
        "chat_response": {
          "response": "如果是对话意图的回复内容",
          "need_follow_up": false
        }
      }
```

### OpenAI客户端封装

```go
// internal/ai/openai_client.go
type OpenAIClient struct {
    client      *openai.Client
    config      *AIConfig
    rateLimiter *rate.Limiter
    conversation ConversationService
}

func NewOpenAIClient(config *AIConfig, conversationService ConversationService) *OpenAIClient {
    client := openai.NewClient(config.OpenAI.APIKey)
    
    return &OpenAIClient{
        client:       client,
        config:       config,
        rateLimiter:  rate.NewLimiter(rate.Limit(10), 1), // 每秒最多10个请求
        conversation: conversationService,
    }
}

func (c *OpenAIClient) ParseMessage(ctx context.Context, userID, message string) (*ParseResult, error) {
    // 限流控制
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit: %w", err)
    }
    
    // 构建包含上下文的prompt
    prompt := c.buildContextPrompt(userID, message)
    
    // 调用OpenAI API
    resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:       c.config.OpenAI.PrimaryModel,
        Temperature: &c.config.OpenAI.Temperature,
        MaxTokens:   c.config.OpenAI.MaxTokens,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleUser,
                Content: prompt,
            },
        },
    })
    
    if err != nil {
        return nil, fmt.Errorf("openai api call failed: %w", err)
    }
    
    // 解析AI响应为结构化数据
    return c.parseAIResponse(resp.Choices[0].Message.Content)
}
```

## 💬 对话管理设计

### 30天对话存储

```go
// internal/models/conversation.go
type Conversation struct {
    ID       string    `gorm:"primaryKey" json:"id"`
    UserID   string    `gorm:"index" json:"user_id"`
    Messages []Message `gorm:"foreignKey:ConversationID" json:"messages"`
    
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    ExpiresAt time.Time `gorm:"index" json:"expires_at"` // 30天后自动过期
}

type Message struct {
    ID             string    `gorm:"primaryKey" json:"id"`
    ConversationID string    `gorm:"index" json:"conversation_id"`
    
    Role           string    `json:"role"`    // "user" or "assistant"
    Content        string    `json:"content"` // 消息内容
    MessageType    string    `json:"type"`    // "text", "reminder", "chat"
    
    // 解析结果元信息
    ParseResult    *ParseResult `gorm:"embedded;embeddedPrefix:parse_" json:"parse_result,omitempty"`
    
    CreatedAt      time.Time `json:"created_at"`
}
```

### 上下文理解服务

```go
// ConversationService - 对话管理服务
type ConversationService interface {
    // 获取用户对话上下文（最近10条消息）
    GetContext(ctx context.Context, userID string) (*Conversation, error)
    
    // 保存消息和解析结果
    SaveMessage(ctx context.Context, userID, role, content string, parseResult *ParseResult) error
    
    // 清理过期对话（定时任务，每天执行）
    CleanupExpiredConversations(ctx context.Context) error
}

// 智能上下文构建
func (c *OpenAIClient) buildContextPrompt(userID, message string) string {
    conversation, _ := c.conversation.GetContext(context.Background(), userID)
    
    var contextStr strings.Builder
    if conversation != nil && len(conversation.Messages) > 0 {
        contextStr.WriteString("最近的对话历史:\n")
        // 只包含最近的5条消息作为上下文
        start := max(0, len(conversation.Messages)-5)
        for _, msg := range conversation.Messages[start:] {
            contextStr.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
        }
    }
    
    return c.renderPromptTemplate(message, contextStr.String())
}
```

## 📁 项目结构设计

```
internal/
├── ai/                           # AI相关模块
│   ├── openai_client.go         # OpenAI客户端封装
│   ├── prompt_builder.go        # Prompt模板构建
│   ├── response_parser.go       # AI响应解析
│   └── fallback_chain.go        # 降级链实现
├── service/
│   ├── ai_parser.go             # AI解析服务主实现
│   ├── conversation.go          # 对话管理服务
│   └── fallback_strategy.go    # 降级策略服务
├── models/
│   ├── conversation.go          # 对话数据模型
│   ├── ai_parse_result.go       # AI解析结果模型
│   └── ai_config.go             # AI配置模型
├── repository/
│   └── conversation.go          # 对话数据访问层
└── bot/handlers/
    └── ai_message_handler.go    # AI消息处理器

pkg/
└── ai/
    ├── config.go               # AI配置管理
    ├── types.go                # AI相关类型定义
    └── errors.go               # AI相关错误定义

configs/
├── prompts/                    # Prompt模板目录
│   ├── reminder_parse.tmpl     # 提醒解析模板
│   └── chat_response.tmpl      # 对话回复模板
└── config.yaml                 # 主配置文件（包含AI配置）
```

## ⏱️ 实施计划

### 第一周：C1任务核心实现

#### Day 1-2: 基础架构搭建
- [x] 创建AI配置管理模块 (`pkg/ai/config.go`) ✅
- [x] 实现OpenAI客户端基础封装 (`internal/ai/openai_client.go`) ✅
- [x] 设计并实现核心数据结构 (`internal/models/ai_parse_result.go`) ✅
- [x] 配置文件模板设计 (`configs/config.yaml`) ✅

#### Day 3-4: 核心解析功能
- [x] 实现AIParserService接口 (`internal/service/ai_parser.go`) ✅
- [x] 构建智能Prompt模板 (`configs/prompts/`) ✅
- [x] 实现JSON响应解析逻辑 (`internal/ai/response_parser.go`) ✅
- [x] 集成对话上下文管理 (`internal/service/conversation.go`) ✅

#### Day 5: 降级机制和集成
- [x] 实现四层降级策略 (`internal/ai/fallback_chain.go`) ✅
- [x] 错误处理和重试逻辑 (`pkg/ai/errors.go`) ✅
- [x] 与现有正则解析器集成 ✅
- [x] Bot消息处理器集成 (`internal/bot/handlers/`) ✅

### 测试和验证
- [ ] 单元测试覆盖率 > 80%
- [ ] 集成测试：完整降级链测试
- [ ] 手动测试：复杂自然语言解析验证

## 🧪 测试用例设计

### 提醒解析测试用例

```go
// 测试用例示例
var testCases = []struct {
    input    string
    expected ParseIntent
    reminder *ReminderInfo
}{
    {
        input:    "每天早上8点提醒我喝水",
        expected: IntentReminder,
        reminder: &ReminderInfo{
            Title: "喝水",
            Type:  models.ReminderTypeHabit,
            Time:  TimeInfo{Hour: 8, Minute: 0, Timezone: "Asia/Shanghai"},
            SchedulePattern: models.SchedulePatternDaily,
        },
    },
    {
        input:    "工作日晚上8点提醒我复习英语",
        expected: IntentReminder,
        reminder: &ReminderInfo{
            Title: "复习英语",
            Type:  models.ReminderTypeHabit,
            Time:  TimeInfo{Hour: 20, Minute: 0, Timezone: "Asia/Shanghai"},
            SchedulePattern: "weekly:1,2,3,4,5", // 工作日
        },
    },
    {
        input:    "我在看《三体》",
        expected: IntentChat,
        // 期望AI回复关于三体的介绍
    },
}
```

## 🎯 预期成果

### 功能成果
1. **智能提醒创建**：支持复杂自然语言如"工作日早上醒来后提醒我看书"
2. **智能对话功能**：能够介绍书籍、回答问题、进行日常交流
3. **高可用性保障**：四层降级机制确保服务不中断
4. **上下文理解**：基于30天对话历史的智能上下文管理

### 技术成果
1. **标准化接口**：统一的AIParserService接口，易于扩展
2. **模块化设计**：清晰的架构分层，便于维护和测试
3. **配置化管理**：灵活的Prompt模板和参数配置
4. **监控就绪**：集成现有监控系统，支持性能和成本跟踪

## 📊 成功指标

### 技术指标
- **解析成功率**: > 90% (主AI解析)
- **降级成功率**: > 99.9% (整体可用性)
- **响应时间**: < 2秒 (包含AI调用)
- **上下文准确率**: > 85% (多轮对话理解)

### 业务指标
- **自然语言覆盖率**: > 80% (复杂表达支持)
- **用户满意度**: > 4.0/5 (AI回复质量)
- **降级透明度**: 用户无感知的降级体验

## 🔄 后续扩展计划

### C2任务准备（多AI服务提供商）
- Claude API集成准备
- 统一API适配器设计
- 智能路由和负载均衡策略

### C3任务准备（智能降级机制）
- 基于性能和成本的智能选择
- 动态降级策略调整
- A/B测试框架集成

### C4任务准备（双解析器架构部署）
- 性能监控和指标收集
- 解析器效果对比分析
- 生产环境部署策略

---

**文档版本**: v1.1
**创建日期**: 2025年9月29日
**更新日期**: 2025年10月10日
**负责人**: 开发团队
**状态**: ✅ **已完成实施** (2025-10-10)

**完成情况**:
- ✅ Day 1-2: 基础架构搭建 (100%)
- ✅ Day 3-4: 核心解析功能 (100%)
- ✅ Day 5: 降级机制和集成 (100%)
- **整体完成度**: 100%

**标签**: #MMemory #阶段3 #C1任务 #AI解析器 #实施方案 #OpenAI集成 #已完成