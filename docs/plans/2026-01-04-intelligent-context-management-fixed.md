# Intelligent Context Management Implementation Plan (Fixed)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Critical Fixes Applied:**
1. ✅ Fixed infinite context growth loop - messages now persist back to database
2. ✅ Fixed performance issue - AI summarization is async (non-blocking)
3. ✅ Fixed missing userID - updated method signatures
4. ✅ Fixed "Ask User" strategy - downgraded to soft warning for MVP
5. ✅ Added safe buffer for token estimation (120k config for 128k model)

**Goal:** 解决AI对话时上下文超限问题,实现智能的对话历史管理,包括token估算、话题切换检测、对话压缩和用户交互式存档。

**Architecture:**
1. **Token管理**: 使用简单估算(中文×2 + 英文词数)监控上下文使用率
2. **分级处理**: 80%触发警告,95%强制清理
3. **智能清理**: 删除不重要对话(已入库提醒) + AI压缩讨论型对话
4. **异���存档**: 清理后立即返回,存档在后台goroutine中执行
5. **状态持久化**: 清理后的消息列表必须写回数据库

**Tech Stack:** Go, GORM, SQLite, OpenAI API(用于生成摘要), Telegram Bot API

---

## Task 1: 添加Token估算器 (不变)

**Files:**
- Create: `internal/service/token_estimator.go`
- Test: `internal/service/token_estimator_test.go`

*(代码与原计划相同,省略以节省篇幅)*

**Step 4: 提交代码**

```bash
git add internal/service/token_estimator.go internal/service/token_estimator_test.go
git commit -m "feat: 添加Token估算器,支持中英文混合文本的token数估算"
```

---

## Task 2: 添加话题切换检测器 (不变)

**Files:**
- Create: `internal/service/topic_switch_detector.go`
- Test: `internal/service/topic_switch_detector_test.go`

*(代码与原计划相同,省略以节省篇幅)*

**Step 4: 提交代码**

```bash
git add internal/service/topic_switch_detector.go internal/service/topic_switch_detector_test.go
git commit -m "feat: 添加话题切换检测器,支持时间间隔和意图变化检测"
```

---

## Task 3: 创建存档数据模型 (不变)

**Files:**
- Create: `internal/models/conversation_archive.go`
- Modify: `internal/repository/interfaces/repository.go`
- Create: `internal/repository/sqlite/conversation_archive.go`
- Test: `internal/models/conversation_archive_test.go`

*(代码与原计划相同,省略以节省篇幅)*

**Step 7: 提交代码**

```bash
git add internal/models/conversation_archive.go internal/models/conversation_archive_test.go
git add internal/repository/interfaces/repository.go
git add internal/repository/sqlite/conversation_archive.go
git add internal/repository/sqlite/database.go
git commit -m "feat: 添加对话存档模型和Repository,支持完整内容和摘要存档"
```

---

## Task 4: 实现存档服务 (修改 - 添加异步方法)

**Files:**
- Create: `internal/service/conversation_archive_service.go`
- Test: `internal/service/conversation_archive_service_test.go`
- Modify: `internal/service/interfaces.go`

**Step 1: 更新服务接口**

在 `internal/service/interfaces.go` 中添加:

```go
// ConversationArchiveService 存档服务接口
type ConversationArchiveService interface {
    // CreateArchive 创建存档(同步)
    CreateArchive(ctx context.Context, userID uint, messages []models.ConversationMessage, archiveType models.ArchiveType) (*models.ConversationArchive, error)

    // CreateArchiveAsync 异步创建存档(fire-and-forget)
    CreateArchiveAsync(userID uint, messages []models.ConversationMessage, archiveType models.ArchiveType)

    // GenerateSummary 生成摘要(使用AI)
    GenerateSummary(ctx context.Context, messages []models.ConversationMessage) (string, error)

    // ExtractKeyEntities 提取关键实体
    ExtractKeyEntities(messages []models.ConversationMessage) (*models.KeyEntities, error)

    // GetUserArchives 获取用户存档
    GetUserArchives(ctx context.Context, userID uint, limit int) ([]*models.ConversationArchive, error)

    // CleanupExpiredArchives 清理过期存档
    CleanupExpiredArchives(ctx context.Context) (int64, error)
}
```

**Step 2: 更新服务实现 - 添加异步方法**

在 `internal/service/conversation_archive_service.go` 中添加异步方法:

```go
// CreateArchiveAsync 异步创建存档(fire-and-forget)
func (s *conversationArchiveService) CreateArchiveAsync(
    userID uint,
    messages []models.ConversationMessage,
    archiveType models.ArchiveType,
) {
    // 在后台goroutine中执行
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        logger.Infof("开始异步存档: userID=%d, messages=%d, type=%s", userID, len(messages), archiveType)

        archive, err := s.CreateArchive(ctx, userID, messages, archiveType)
        if err != nil {
            logger.Errorf("异步存档失败: userID=%d, error=%v", userID, err)
            return
        }

        logger.Infof("异步存档成功: userID=%d, archiveID=%d", userID, archive.ID)
    }()
}
```

**Step 3: 运行测试**

```bash
go test ./internal/service -run TestConversationArchiveService -v
```

**Step 4: 提交代码**

```bash
git add internal/service/conversation_archive_service.go internal/service/conversation_archive_service_test.go
git add internal/service/interfaces.go
git commit -m "feat: 实现存档服务,支持同步和异步创建存档、生成摘要"
```

---

## Task 5: 实现上下文Token管理器 (重构 - 改为返回kept和archived消息)

**Files:**
- Create: `internal/service/context_token_manager.go`
- Test: `internal/service/context_token_manager_test.go`

**Step 1: 编写测试**

创建 `internal/service/context_token_manager_test.go`:

```go
package service

import (
    "context"
    "testing"
    "time"

    "mmemory/internal/models"
    "mmemory/pkg/ai"
)

func TestContextTokenManager_PruneMessages(t *testing.T) {
    config := &ai.AIConfig{
        Providers: []ai.AIProviderConfig{
            {MaxContextTokens: 1000},
        },
        ContextManagement: ai.ContextManagementConfig{
            WarningThreshold: 0.8,
            HardLimit:        0.95,
            KeepRecentMessages: 8,
        },
    }

    manager := NewContextTokenManager(config, nil)

    t.Run("低于阈值-不清理", func(t *testing.T) {
        messages := []models.ConversationMessage{
            {Content: "短消息"},
        }

        kept, archived, strategy := manager.PruneMessages(messages)
        if len(kept) != 1 || len(archived) != 0 {
            t.Errorf("不应该清理消息")
        }
        if strategy != StrategyNone {
            t.Errorf("策略应该是None, got %v", strategy)
        }
    })

    t.Run("超过硬限制-强制清理", func(t *testing.T) {
        // 构造长消息(>950 tokens)
        longText := string(make([]byte, 500))
        messages := make([]models.ConversationMessage, 20)
        for i := range messages {
            messages[i] = models.ConversationMessage{
                Content:   longText,
                Timestamp: time.Now(),
            }
        }

        kept, archived, strategy := manager.PruneMessages(messages)

        if len(kept) > 8 {
            t.Errorf("保留消息应该<=8, got %d", len(kept))
        }
        if len(archived) == 0 {
            t.Errorf("应该有消息被归档")
        }
        if strategy != StrategyForceClean {
            t.Errorf("策略应该是ForceClean, got %v", strategy)
        }
    })
}

func TestContextTokenManager_NeedsPruning(t *testing.T) {
    config := &ai.AIConfig{
        Providers: []ai.AIProviderConfig{
            {MaxContextTokens: 1000},
        },
        ContextManagement: ai.ContextManagementConfig{
            WarningThreshold: 0.8,
        },
    }

    manager := NewContextTokenManager(config, nil)

    t.Run("低于阈值", func(t *testing.T) {
        messages := []models.ConversationMessage{
            {Content: "短消息"},
        }

        needs, ratio := manager.NeedsPruning(messages)
        if needs {
            t.Errorf("不需要清理")
        }
        if ratio > 0.8 {
            t.Errorf("使用率应该<0.8, got %v", ratio)
        }
    })

    t.Run("超过警告阈值", func(t *testing.T) {
        longText := string(make([]byte, 500)) // 约1000 token
        messages := []models.ConversationMessage{
            {Content: longText},
        }

        needs, ratio := manager.NeedsPruning(messages)
        if !needs {
            t.Errorf("应该需要清理")
        }
        if ratio < 0.8 {
            t.Errorf("使用率应该>0.8, got %v", ratio)
        }
    })
}
```

**Step 2: 实现重构后的管理器**

创建 `internal/service/context_token_manager.go`:

```go
package service

import (
    "fmt"

    "mmemory/internal/models"
    "mmemory/pkg/ai"
)

// ContextTokenManager 上下文Token管理器
type ContextTokenManager struct {
    config         *ai.AIConfig
    tokenEstimator *TokenEstimator
    topicDetector  *TopicSwitchDetector
}

// NewContextTokenManager 创建管理器
func NewContextTokenManager(
    config *ai.AIConfig,
) *ContextTokenManager {
    return &ContextTokenManager{
        config:         config,
        tokenEstimator: NewTokenEstimator(),
        topicDetector:  NewTopicSwitchDetector(config.ContextManagement),
    }
}

// NeedsPruning 检查是否需要清理
// 返回: (是否需要清理, token使用率)
func (m *ContextTokenManager) NeedsPruning(messages []models.ConversationMessage) (bool, float64) {
    if len(messages) == 0 {
        return false, 0
    }

    maxTokens := m.getMaxContextTokens()
    if maxTokens <= 0 {
        return false, 0
    }

    ratio := m.tokenEstimator.EstimateUsageRatio(messages, maxTokens)
    needsCleanup := ratio >= m.config.ContextManagement.WarningThreshold

    return needsCleanup, ratio
}

// PruneMessages 清理消息(状态无关,只返回决策结果)
// 返回: (保留的消息, 需要归档的消息, 使用的策略)
func (m *ContextTokenManager) PruneMessages(messages []models.ConversationMessage) (
    []models.ConversationMessage,
    []models.ConversationMessage,
    CleanupStrategy,
) {
    if len(messages) == 0 {
        return messages, nil, StrategyNone
    }

    maxTokens := m.getMaxContextTokens()
    ratio := m.tokenEstimator.EstimateUsageRatio(messages, maxTokens)

    // 根据使用率选择策略
    var strategy CleanupStrategy
    if ratio >= m.config.ContextManagement.HardLimit {
        strategy = StrategyForceClean
    } else if ratio >= m.config.ContextManagement.WarningThreshold {
        strategy = StrategySmartClean
    } else {
        return messages, nil, StrategyNone
    }

    // 执行清理
    kept, archived := m.executePrune(messages, strategy)

    logger.Infof("Token管理: 使用率=%.2f%%, 策略=%s, 保留=%d, 归档=%d",
        ratio*100, strategy, len(kept), len(archived))

    return kept, archived, strategy
}

// executePrune 执行清理策略
func (m *ContextTokenManager) executePrune(
    messages []models.ConversationMessage,
    strategy CleanupStrategy,
) ([]models.ConversationMessage, []models.ConversationMessage) {
    switch strategy {
    case StrategyForceClean:
        return m.forceClean(messages)
    case StrategySmartClean:
        return m.smartClean(messages)
    default:
        return messages, nil
    }
}

// forceClean 强制清理
func (m *ContextTokenManager) forceClean(messages []models.ConversationMessage) (
    []models.ConversationMessage,
    []models.ConversationMessage,
) {
    keepRecent := m.config.ContextManagement.KeepRecentMessages
    if keepRecent <= 0 {
        keepRecent = 8
    }

    if len(messages) <= keepRecent {
        return messages, nil
    }

    // 分割为保留和归档
    recent := messages[len(messages)-keepRecent:]
    oldMessages := messages[:len(messages)-keepRecent]

    // 过滤出重要的旧消息
    important := m.filterImportantMessages(oldMessages)

    return recent, important
}

// smartClean 智能清理
func (m *ContextTokenManager) smartClean(messages []models.ConversationMessage) (
    []models.ConversationMessage,
    []models.ConversationMessage,
) {
    var important []models.ConversationMessage
    var unimportant []models.ConversationMessage

    for _, msg := range messages {
        if m.isMessageImportant(&msg) {
            important = append(important, msg)
        } else {
            unimportant = append(unimportant, msg)
        }
    }

    if len(unimportant) == 0 {
        return messages, nil
    }

    logger.Infof("智能清理: 删除 %d 条不重要消息", len(unimportant))
    return important, unimportant
}

// filterImportantMessages 过滤重要消息
func (m *ContextTokenManager) filterImportantMessages(messages []models.ConversationMessage) []models.ConversationMessage {
    var important []models.ConversationMessage

    for _, msg := range messages {
        if m.isMessageImportant(&msg) {
            important = append(important, msg)
        }
    }

    return important
}

// isMessageImportant 判断消息是否重要
func (m *ContextTokenManager) isMessageImportant(msg *models.ConversationMessage) bool {
    // 不重要的意图
    unimportantIntents := map[string]bool{
        "reminder":       true, // 已入库的提醒
        "query_activity": true, // 查询
    }

    // 如果是不重要的意图,可以删除
    if unimportantIntents[msg.Intent] {
        return false
    }

    // 聊天和记录活动通常重要
    return true
}

// getMaxContextTokens 获取最大上下文token数
func (m *ContextTokenManager) getMaxContextTokens() int {
    if len(m.config.Providers) == 0 {
        return 0
    }

    // 使用第一个provider的配置
    return m.config.Providers[0].MaxContextTokens
}

// CleanupStrategy 清理策略
type CleanupStrategy int

const (
    StrategyNone      CleanupStrategy = iota
    StrategySmartClean                 // 智能清理: 删除不重要
    StrategyForceClean                 // 强制清理: 保留最近N条,其余归档
)

func (s CleanupStrategy) String() string {
    switch s {
    case StrategySmartClean:
        return "smart_clean"
    case StrategyForceClean:
        return "force_clean"
    default:
        return "none"
    }
}
```

**Step 3: 运行测试**

```bash
go test ./internal/service -run TestContextTokenManager -v
```

**Step 4: 提交代码**

```bash
git add internal/service/context_token_manager.go internal/service/context_token_manager_test.go
git commit -m "feat: 实现上下文Token管理器(重构版),支持智能清理策略"
```

---

## Task 6: 扩展ContextManager支持消息持久化 (新增关键任务)

**Files:**
- Modify: `internal/service/context_manager.go`
- Test: `internal/service/context_manager_test.go`

**Step 1: 添加OverwriteMessages方法**

在 `internal/service/context_manager.go` 中添加:

```go
// OverwriteMessages 覆盖消息列表并持久化
// CRITICAL: 这是修复无限增长循环的关键方法
func (m *ContextManager) OverwriteMessages(ctx context.Context, userID uint, sessionID string, messages []models.ConversationMessage) error {
    if userID == 0 {
        return fmt.Errorf("userID is required")
    }

    // 1. 获取当前状态
    currentState, err := m.getOrCreateState(ctx, userID, sessionID)
    if err != nil {
        return fmt.Errorf("failed to get state: %w", err)
    }

    // 2. 替换消息列表
    currentState.Messages = messages

    // 3. 更新时间戳
    currentState.UpdatedAt = m.nowFunc()

    // 4. 序列化状态
    stateJSON, err := json.Marshal(currentState)
    if err != nil {
        return fmt.Errorf("failed to marshal state: %w", err)
    }

    // 5. 持久化到数据库
    if err := m.repo.Update(ctx, currentState); err != nil {
        return fmt.Errorf("failed to persist state: %w", err)
    }

    logger.Infof("覆盖用户 %d 的消息列表: %d 条消息", userID, len(messages))
    return nil
}
```

**Step 2: 更新接口**

在 `internal/service/interfaces.go` 的 `ContextManager` 接口中添加:

```go
type ContextManager interface {
    // ... 现有方法 ...

    // OverwriteMessages 覆盖消息列表并持久化
    OverwriteMessages(ctx context.Context, userID uint, sessionID string, messages []models.ConversationMessage) error
}
```

**Step 3: 添加测试**

在 `internal/service/context_manager_test.go` 中添加:

```go
func TestContextManager_OverwriteMessages(t *testing.T) {
    repo := mocks.NewMockConversationContextRepository()
    config := ContextManagerConfig{
        MaxMessages: 20,
    }

    manager := NewContextManager(repo, nil, nil, config)

    ctx := context.Background()
    userID := uint(123)
    sessionID := "test-session"

    // 创建初始状态
    originalMessages := []models.ConversationMessage{
        {Role: "user", Content: "消息1"},
        {Role: "assistant", Content: "回复1"},
        {Role: "user", Content: "消息2"},
    }

    state, err := manager.ProcessMessage(ctx, ProcessMessageInput{
        UserID:    userID,
        SessionID: sessionID,
        Role:      "user",
        Message:   "测试消息",
    })
    assert.NoError(t, err)
    assert.Equal(t, 1, len(state.Messages))

    // 覆盖消息列表
    newMessages := []models.ConversationMessage{
        {Role: "user", Content: "新消息1"},
    }

    err = manager.OverwriteMessages(ctx, userID, sessionID, newMessages)
    assert.NoError(t, err)

    // 验证消息被替换
    updatedState, err := manager.GetContext(ctx, userID, sessionID)
    assert.NoError(t, err)
    assert.Equal(t, 1, len(updatedState.Messages))
    assert.Equal(t, "新消息1", updatedState.Messages[0].Content)
}
```

**Step 4: 运行测试**

```bash
go test ./internal/service -run TestContextManager_OverwriteMessages -v
```

**Step 5: 提交代码**

```bash
git add internal/service/context_manager.go internal/service/interfaces.go
git add internal/service/context_manager_test.go
git commit -m "feat: 添加ContextManager.OverwriteMessages方法,支持消息列表持久化"
```

---

## Task 7: 集成到消息处理流程 (修复版本)

**Files:**
- Modify: `internal/bot/handlers/message.go`
- Modify: `internal/service/interfaces.go`

**Step 1: 更新MessageHandler结构**

在 `internal/bot/handlers/message.go` 中修改(约第50-60行):

```go
type MessageHandler struct {
    // ... 现有字段 ...

    contextTokenManager service.ContextTokenManagerService
    archiveService      service.ConversationArchiveService
}
```

**Step 2: 更新构造函数**

修改 `NewMessageHandler` 函数签名(约第100-150行):

```go
func NewMessageHandler(
    reminderService service.ReminderService,
    userService service.UserService,
    logService service.ReminderLogService,
    aiParserService service.AIParserService,
    activityService service.DailyActivityService,
    contextManager service.ContextManager,
    suggestionService service.SuggestionService,
    weatherService service.WeatherService,
    notificationService service.NotificationService,
    contextTokenManager service.ContextTokenManagerService,
    archiveService service.ConversationArchiveService,
) *MessageHandler {
    return &MessageHandler{
        // ... 现有字段 ...
        contextTokenManager: contextTokenManager,
        archiveService:      archiveService,
    }
}
```

**Step 3: 修改handleWithAI方法 - 关键修复**

在 `internal/bot/handlers/message.go:361` 的 `handleWithAI` 方法中:

```go
func (h *MessageHandler) handleWithAI(ctx context.Context, bot botinterface.BotAPI, message *tgbotapi.Message, user *models.User) error {
    // 获取会话历史上下文
    var contextState *models.ConversationContextState
    if h.contextManager != nil {
        var err error
        sessionID := fmt.Sprintf("%d", message.Chat.ID)

        contextState, err = h.contextManager.GetContext(ctx, user.ID, sessionID)
        if err == nil && contextState != nil {
            logger.Infof("获取到用户 %d 的会话历史，包含 %d 条消息", user.ID, len(contextState.Messages))
        } else {
            logger.Debugf("获取用户 %d 的会话上下文失败或为空: %v", user.ID, err)
            contextState = &models.ConversationContextState{
                Messages: []models.ConversationMessage{},
            }
        }

        // ===== 关键修复: Token管理和清理 =====
        if h.contextTokenManager != nil && len(contextState.Messages) > 0 {
            needsCleanup, ratio := h.contextTokenManager.NeedsPruning(contextState.Messages)

            if needsCleanup {
                logger.Warnf("用户 %d 的对话使用率达到 %.2f%%, 触发清理", user.ID, ratio*100)

                // 执行清理
                kept, archived, strategy := h.contextTokenManager.PruneMessages(contextState.Messages)

                // 如果有消息被归档
                if len(archived) > 0 {
                    // 1. CRITICAL: 立即持久化清理后的消息列表
                    sessionID := fmt.Sprintf("%d", message.Chat.ID)
                    if err := h.contextManager.OverwriteMessages(ctx, user.ID, sessionID, kept); err != nil {
                        logger.Errorf("持久化清理后的消息列表失败: %v", err)
                        // 继续处理,使用内存中的kept
                    } else {
                        logger.Infof("已持久化清理后的消息列表: %d 条消息", len(kept))
                    }

                    // 2. 异步归档被删除的消息(fire-and-forget,不阻塞)
                    if h.archiveService != nil {
                        archiveType := models.ArchiveTypeSummary
                        if strategy == StrategyForceClean {
                            archiveType = models.ArchiveTypeSummary // 压缩成摘要
                        }

                        h.archiveService.CreateArchiveAsync(user.ID, archived, archiveType)
                        logger.Infof("已触发异步归档: %d 条消息", len(archived))
                    }

                    // 3. 更新本地变量,用于当前AI prompt
                    contextState.Messages = kept
                }

                // 4. 记录指标
                userStr := fmt.Sprintf("%d", user.ID)
                metrics.ContextCleanupTotal.WithLabelValues(userStr, strategy.String()).Inc()
                metrics.ContextTokenUsageGauge.WithLabelValues(userStr).Set(ratio)
            }
        }
    }

    // 格式化会话历史为字符串
    var conversationHistory string
    if contextState != nil && len(contextState.Messages) > 0 {
        conversationHistory = h.formatConversationHistory(contextState.Messages)
    }

    // 调用AI解析服务（带上下文）
    userIDStr := fmt.Sprintf("%d", user.TelegramID)
    parseResult, err := h.aiParserService.ParseMessageWithContext(ctx, userIDStr, message.Text, conversationHistory)

    // ... 其余代码保持不变 ...
}
```

**Step 4: 简化formatConversationHistory**

修改 `internal/bot/handlers/message.go:1761`:

```go
func (h *MessageHandler) formatConversationHistory(messages []models.ConversationMessage) string {
    if len(messages) == 0 {
        return ""
    }

    // 最多保留最近20条消息(兜底逻辑,实际清理已由TokenManager处理)
    recentMessages := messages
    if len(recentMessages) > 20 {
        recentMessages = recentMessages[len(recentMessages)-20:]
    }

    var history strings.Builder
    history.WriteString("\n【对话历史（请记住关键信息）】\n")
    history.WriteString("以下是最近的对话记录，请记住用户提到的书名、章节、人物等关键实体：\n\n")

    for i, msg := range recentMessages {
        role := msg.Role
        if role == "" {
            role = "user"
        }

        history.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, role, msg.Content))
    }

    return history.String()
}
```

**Step 5: 更新依赖注入**

在 `cmd/bot/main.go` 中:

```go
// 创建Token管理器
var contextTokenManager service.ContextTokenManagerService
if aiConfig.ContextManagement.Enabled {
    contextTokenManager = service.NewContextTokenManager(aiConfig)
}

// 创建消息处理器
handler := handlers.NewMessageHandler(
    reminderService,
    userService,
    logService,
    aiParserService,
    activityService,
    contextManager,
    suggestionService,
    weatherService,
    notificationService,
    contextTokenManager, // 新增
    archiveService,      // 新增
)
```

**Step 6: 编译检查**

```bash
go build ./internal/bot/handlers
go build ./cmd/bot
```

预期: 无编译错误

**Step 7: 提交代码**

```bash
git add internal/bot/handlers/message.go cmd/bot/main.go
git commit -m "feat: 集成Token管理器到消息处理流程(修复版)

关键修复:
1. 清理后立即调用OverwriteMessages持久化到数据库
2. 异步归档不阻塞主流程(fire-and-forget)
3. 更新本地变量确保AI prompt使用清理后的消息
4. 移除formatConversationHistory中的清理逻辑(移到handleWithAI)"
```

---

## Task 8: 添加配置文件 (更新 - 添加安全buffer)

**Files:**
- Modify: `pkg/config/ai_config.go`
- Modify: `configs/config.example.yaml`

**Step 1: 配置说明更新**

在 `configs/config.example.yaml` 中添加注释:

```yaml
ai:
  enabled: false

  providers:
    - name: "openai-gpt-4o-mini"
      provider: "openai"
      api_key: "${MMEMORY_AI_OPENAI_API_KEY}"
      base_url: "https://api.openai.com/v1"
      model: "gpt-4o-mini"
      backup_model: "gpt-3.5-turbo"
      # CRITICAL: 设置为120k而非128k,留7%安全buffer
      # 原因: token估算使用简单算法(中文×2),存在误差
      max_context_tokens: 120000  # 128k模型 × 0.94 ≈ 120k
      temperature: 0.1
      max_tokens: 1000

    - name: "zai-glm-4.6"
      provider: "openai"
      api_key: "${MMEMORY_AI_ZAI_API_KEY}"
      base_url: "https://api.siliconflow.cn/v1"
      model: "zai-org/GLM-4.6"
      # 同样留安全buffer
      max_context_tokens: 120000  # 128k模型 × 0.94
      temperature: 0.1
      max_tokens: 1000

  # 上下文管理配置
  context_management:
    enabled: true
    keep_recent_messages: 8

    # Token管理阈值(基于max_context_tokens计算)
    warning_threshold: 0.8   # 80% = 96k tokens时警告
    hard_limit: 0.95         # 95% = 114k tokens时强制清理

    # ... 其余配置与原计划相同 ...
```

**Step 2: 提交配置**

```bash
git add configs/config.example.yaml
git commit -m "docs: 更新配置示例,添加token安全buffer说明"
```

---

## Task 9: 添加监控指标 (不变)

**Files:**
- Modify: `pkg/metrics/metrics.go`
- Modify: `internal/service/context_token_manager.go`

*(代码与原计划相同,省略以节省篇幅)*

**Step 4: 提交代码**

```bash
git add pkg/metrics/metrics.go internal/service/context_token_manager.go
git commit -m "feat: 添加上下文管理的Prometheus监控指标"
```

---

## Task 10: 集成测试和文档 (更新)

**Files:**
- Test: `internal/bot/handlers/message_integration_test.go`
- Modify: `CLAUDE.md`
- Modify: `docs/plans/2026-01-04-intelligent-context-management.md`

**Step 1: 添加端到端集成测试**

在 `internal/bot/handlers/message_integration_test.go` 中添加:

```go
func TestMessageHandler_ContextCleanupIntegration(t *testing.T) {
    // 创建mock服务
    mockTokenManager := new(MockContextTokenManagerService)
    mockArchiveService := new(MockConversationArchiveService)
    mockContextManager := new(MockContextManager)

    handler := NewMessageHandler(
        nil, nil, nil, nil, nil,
        mockContextManager, nil, nil, nil,
        mockTokenManager,
        mockArchiveService,
    )

    t.Run("Token超限-触发清理并持久化", func(t *testing.T) {
        user := &models.User{ID: 123, TelegramID: 456}
        message := &tgbotapi.Message{
            Chat: &tgbotapi.Chat{ID: 456},
            Text: "新消息",
        }

        // 原始消息列表(20条)
        originalMessages := make([]models.ConversationMessage, 20)
        for i := range originalMessages {
            originalMessages[i] = models.ConversationMessage{
                Role:    "user",
                Content: fmt.Sprintf("消息%d", i),
            }
        }

        // Mock context
        contextState := &models.ConversationContextState{
            Messages: originalMessages,
        }
        mockContextManager.On("GetContext", mock.Anything, uint(123), "456").Return(contextState, nil)

        // Mock token manager
        mockTokenManager.On("NeedsPruning", originalMessages).Return(true, 0.9)
        keptMessages := originalMessages[12:] // 保留8条
        archivedMessages := originalMessages[:12]
        mockTokenManager.On("PruneMessages", originalMessages).Return(
            keptMessages,
            archivedMessages,
            StrategyForceClean,
        )

        // Mock overwrite
        mockContextManager.On("OverwriteMessages", mock.Anything, uint(123), "456", keptMessages).Return(nil)

        // Mock async archive (fire-and-forget,不等待)
        mockArchiveService.On("CreateArchiveAsync", uint(123), archivedMessages, models.ArchiveTypeSummary).Return()

        // 执行
        err := handler.handleWithAI(context.Background(), nil, message, user)

        // 验证
        assert.NoError(t, err)
        mockContextManager.AssertExpectations(t)
        mockTokenManager.AssertExpectations(t)
        mockArchiveService.AssertExpectations(t)

        // 验证OverwriteMessages被调用(关键:防止无限循环)
        mockContextManager.AssertCalled(t, "OverwriteMessages", mock.Anything, uint(123), "456", keptMessages)
    })
}
```

**Step 2: 更新CLAUDE.md**

添加关键修复说明:

```markdown
## Context Management (Fixed)

**Critical Implementation Details:**

1. **State Persistence**: Messages are persisted to database after pruning
   - Uses `ContextManager.OverwriteMessages()` to save pruned list
   - Prevents infinite growth loop on subsequent requests

2. **Async Archiving**: AI summarization runs in background
   - Does not block the current user turn
   - Fire-and-forget pattern with error logging

3. **Safe Token Buffer**: Config uses 120k for 128k model
   - Accommodates estimation errors (Chinese ×2 heuristic)
   - 80% warning = 96k, 95% hard limit = 114k

**Workflow:**
```
User message → GetContext → Check tokens
  → If >80%: PruneMessages
    → OverwriteMessages (CRITICAL: persist to DB)
    → CreateArchiveAsync (non-blocking)
    → Use pruned list for AI prompt
  → Proceed with AI parsing
```
```

**Step 3: 运行测试**

```bash
make test
```

**Step 4: 提交文档**

```bash
git add CLAUDE.md internal/bot/handlers/message_integration_test.go
git commit -m "docs: 更新上下文管理文档,添加关键实现细节说明"
```

---

## 修复总结

### 关键修复

1. **✅ 无限增长循环**
   - 添加 `ContextManager.OverwriteMessages()` 方法
   - 清理后立即持久化到数据库
   - 下次请求加载的是清理后的列表

2. **✅ 性能问题**
   - AI摘要改为异步执行(`CreateArchiveAsync`)
   - Fire-and-forget模式,不阻塞主流程
   - 用户立即收到回复,存档在后台完成

3. **✅ 集成错误**
   - `formatConversationHistory` 不再需要userID
   - 清理逻辑移到 `handleWithAI`
   - 保留正确的sessionID用于持久化

4. **✅ 安全Buffer**
   - 配置120k for 128k model (94%)
   - 预留token估算误差空间

### 数据流(修复后)

```
请求1: 用户消息(第1条)
  → GetContext: [msg1]
  → NeedsPruning: false (1%)
  → AI处理msg1
  → ProcessMessage: 保存msg1
  → 数据库: [msg1]

请求2: 用户消息(第50条)
  → GetContext: [msg1, msg2, ..., msg50] (从DB加载)
  → NeedsPruning: true (95%)
  → PruneMessages: kept=[msg43-50], archived=[msg1-42]
  → OverwriteMessages: 保存[msg43-50]到DB ✅
  → CreateArchiveAsync: 异步归档[msg1-42]
  → AI处理msg50 (使用[msg43-50])
  → 数据库: [msg43-50] ✅ 下次请求加载这个

请求3: 用户消息(第51条)
  → GetContext: [msg43-50, msg51] (从DB加载,不是1-51!) ✅
  → NeedsPruning: false (20%)
  → AI处理msg51
```

### 测试验证清单

**单元测试:**
- [ ] TokenEstimator准确估算中英文混合文本
- [ ] TopicSwitchDetector正确检测话题切换
- [ ] ContextTokenManager.PruneMessages返回正确的kept/archived
- [ ] ContextManager.OverwriteMessages正确持久化

**集成测试:**
- [ ] 清理后数据库消息数减少
- [ ] 异步存档不阻塞主流程
- [ ] 第二次请求加载的是清理后的消息
- [ ] Prometheus指标正确上报

**手动测试:**
1. 发送50条消息,触发清理
2. 检查数据库中conversation_context表
3. 验证只有最近8-12条消息
4. 验证archives表有新的归档记录
5. 发送第51条消息,检查日志显示使用率<30%

### 预计完成时间

每个任务 15-30 分钟,总计 3-4 小时
