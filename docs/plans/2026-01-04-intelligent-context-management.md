# Intelligent Context Management Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 解决AI对话时上下文超限问题,实现智能的对话历史管理,包括token估算、话题切换检测、对话压缩和用户交互式存档。

**Architecture:**
1. **Token管理**: 使用简单估算(中文×2 + 英文词数)监控上下文使用率
2. **分级处理**: 80%触发警告,95%强制清理
3. **智能清理**: 删除不重要对话(已入库提醒) + AI压缩讨论型对话
4. **话题切换检测**: 单一话题立即询问,频繁切换异步询问
5. **存档机制**: 单表存储(full/summary),保留最近8条对话

**Tech Stack:** Go, GORM, SQLite, OpenAI API(用于生成摘要), Telegram Bot API

---

## Task 1: 添加Token估算器

**Files:**
- Create: `internal/service/token_estimator.go`
- Test: `internal/service/token_estimator_test.go`

**Step 1: 编写测试文件**

创建 `internal/service/token_estimator_test.go`:

```go
package service

import (
    "testing"
    "mmemory/internal/models"
)

func TestTokenEstimator_EstimateToken(t *testing.T) {
    estimator := NewTokenEstimator()

    tests := []struct {
        name     string
        text     string
        expected int // 期望的token数范围
    }{
        {
            name:     "纯中文",
            text:     "这是一段中文文本",
            expected: 28, // 14字符 × 2
        },
        {
            name:     "纯英文",
            text:     "This is English text",
            expected: 4, // 4个词
        },
        {
            name:     "中英混合",
            text:     "Hello 世界",
            expected: 6, // 1英文词 + 2中文×2
        },
        {
            name:     "空字符串",
            text:     "",
            expected: 0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := estimator.EstimateToken(tt.text)
            // 允许10%误差
            min := tt.expected * 9 / 10
            max := tt.expected * 11 / 10
            if result < min || result > max {
                t.Errorf("EstimateToken() = %v, expected range [%v, %v]", result, min, max)
            }
        })
    }
}

func TestTokenEstimator_EstimateMessagesToken(t *testing.T) {
    estimator := NewTokenEstimator()

    messages := []models.ConversationMessage{
        {Role: "user", Content: "你好"},
        {Role: "assistant", Content: "Hello"},
    }

    result := estimator.EstimateMessagesToken(messages)
    expected := 2*2 + 1 + 20*2 // 中文4 + 英文1 + 元数据40
    if result < expected*9/10 || result > expected*11/10 {
        t.Errorf("EstimateMessagesToken() = %v, expected around %v", result, expected)
    }
}

func TestTokenEstimator_EstimateUsageRatio(t *testing.T) {
    estimator := NewTokenEstimator()

    messages := []models.ConversationMessage{
        {Role: "user", Content: "这是一段测试文本"},
    }

    ratio := estimator.EstimateUsageRatio(messages, 1000)
    expected := 8.0 / 1000.0 // 4字符×2=8token
    if ratio < expected*0.9 || ratio > expected*1.1 {
        t.Errorf("EstimateUsageRatio() = %v, expected around %v", ratio, expected)
    }
}
```

**Step 2: 实现Token估算器**

创建 `internal/service/token_estimator.go`:

```go
package service

import (
    "strings"
    "unicode"

    "mmemory/internal/models"
)

// TokenEstimator Token估算器
type TokenEstimator struct {
    // 中文: 1字符 ≈ 2 token
    // 英文: 1词 ≈ 1 token
    // 代码/数字: 1字符 ≈ 0.5 token
}

// NewTokenEstimator 创建估算器
func NewTokenEstimator() *TokenEstimator {
    return &TokenEstimator{}
}

// EstimateToken 估算文本的token数量
func (e *TokenEstimator) EstimateToken(text string) int {
    if text == "" {
        return 0
    }

    totalTokens := 0

    // 按行分割
    lines := strings.Split(text, "\n")
    for _, line := range lines {
        totalTokens += e.estimateLineTokens(line)
    }

    return totalTokens
}

// EstimateMessagesToken 估算多条消息的总token数
func (e *TokenEstimator) EstimateMessagesToken(messages []models.ConversationMessage) int {
    total := 0
    for _, msg := range messages {
        total += e.EstimateToken(msg.Content)
        // 加上元数据的token开销(约20 token)
        total += 20
    }
    return total
}

// estimateLineTokens 估算单行的token数
func (e *TokenEstimator) estimateLineTokens(line string) int {
    if len(line) == 0 {
        return 0
    }

    chineseChars := 0
    englishWords := 0
    otherChars := 0

    inWord := false
    for _, r := range line {
        if unicode.Is(unicode.Han, r) {
            // 中文字符
            chineseChars++
        } else if unicode.IsLetter(r) || r == '\'' || r == '-' {
            // 英文字母
            inWord = true
        } else {
            // 分隔符或数字
            if inWord {
                englishWords++
                inWord = false
            }
            if unicode.IsDigit(r) || unicode.IsPunct(r) {
                otherChars++
            }
        }
    }

    // 最后一个词
    if inWord {
        englishWords++
    }

    // 计算token
    // 中文: 1字符 = 2 token
    // 英文: 1词 = 1 token
    // 其他(数字、标点): 2字符 = 1 token
    tokens := chineseChars*2 + englishWords + otherChars/2

    if tokens == 0 && len(line) > 0 {
        return 1 // 至少1 token
    }

    return tokens
}

// EstimateUsageRatio 估算使用率
func (e *TokenEstimator) EstimateUsageRatio(messages []models.ConversationMessage, maxTokens int) float64 {
    if maxTokens <= 0 {
        return 0
    }

    used := e.EstimateMessagesToken(messages)
    return float64(used) / float64(maxTokens)
}
```

**Step 3: 运行测试**

```bash
go test ./internal/service -run TestTokenEstimator -v
```

预期: 所有测试通过

**Step 4: 提交代码**

```bash
git add internal/service/token_estimator.go internal/service/token_estimator_test.go
git commit -m "feat: 添加Token估算器,支持中英文混合文本的token数估算"
```

---

## Task 2: 添加话题切换检测器

**Files:**
- Create: `internal/service/topic_switch_detector.go`
- Test: `internal/service/topic_switch_detector_test.go`

**Step 1: 编写测试**

创建 `internal/service/topic_switch_detector_test.go`:

```go
package service

import (
    "testing"
    "time"

    "mmemory/internal/models"
)

func TestTopicSwitchDetector_DetectSwitch(t *testing.T) {
    config := ContextManagementConfig{
        TopicSwitch: struct {
            Enabled               bool `mapstructure:"enabled" yaml:"enabled"`
            TimeThresholdMinutes  int  `mapstructure:"time_threshold_minutes" yaml:"time_threshold_minutes"`
            IntentChangeThreshold int  `mapstructure:"intent_change_threshold" yaml:"intent_change_threshold"`
        }{
            Enabled:               true,
            TimeThresholdMinutes:  30,
            IntentChangeThreshold: 2,
        },
    }

    detector := NewTopicSwitchDetector(config)

    t.Run("时间间隔检测", func(t *testing.T) {
        now := time.Now()
        messages := []models.ConversationMessage{
            {Intent: "chat", Timestamp: now.Add(-40 * time.Minute)},
            {Intent: "chat", Timestamp: now},
        }

        switched, frequent := detector.DetectSwitch(messages)
        if !switched {
            t.Errorf("期望检测到话题切换(时间间隔)")
        }
        if frequent {
            t.Errorf("不应该是频繁切换")
        }
    })

    t.Run("意图变化检测", func(t *testing.T) {
        now := time.Now()
        messages := []models.ConversationMessage{
            {Intent: "reminder", Timestamp: now.Add(-5 * time.Minute)},
            {Intent: "chat", Timestamp: now.Add(-3 * time.Minute)},
            {Intent: "reminder", Timestamp: now},
        }

        switched, frequent := detector.DetectSwitch(messages)
        if !switched {
            t.Errorf("期望检测到话题切换(意图变化)")
        }
        if !frequent {
            t.Errorf("应该检测到频繁切换")
        }
    })

    t.Run("无切换", func(t *testing.T) {
        now := time.Now()
        messages := []models.ConversationMessage{
            {Intent: "chat", Timestamp: now.Add(-5 * time.Minute)},
            {Intent: "chat", Timestamp: now},
        }

        switched, _ := detector.DetectSwitch(messages)
        if switched {
            t.Errorf("不应该检测到话题切换")
        }
    })
}
```

**Step 2: 实现检测器**

创建 `internal/service/topic_switch_detector.go`:

```go
package service

import (
    "time"

    "mmemory/internal/models"
)

// TopicSwitchDetector 话题切换检测器
type TopicSwitchDetector struct {
    timeThreshold   time.Duration
    intentThreshold int
}

// NewTopicSwitchDetector 创建检测器
func NewTopicSwitchDetector(config ContextManagementConfig) *TopicSwitchDetector {
    return &TopicSwitchDetector{
        timeThreshold:   time.Duration(config.TopicSwitch.TimeThresholdMinutes) * time.Minute,
        intentThreshold: config.TopicSwitch.IntentChangeThreshold,
    }
}

// DetectSwitch 检测话题是否切换
// 返回: (是否切换, 是否频繁切换)
func (d *TopicSwitchDetector) DetectSwitch(messages []models.ConversationMessage) (bool, bool) {
    if len(messages) < 2 {
        return false, false
    }

    // 1. 检测时间间隔
    timeSwitch := d.detectTimeInterval(messages)

    // 2. 检测意图变化
    intentSwitch, frequentSwitch := d.detectIntentChange(messages)

    // 3. 综合判断
    switched := timeSwitch || intentSwitch

    return switched, frequentSwitch
}

// detectTimeInterval 检测时间间隔
func (d *TopicSwitchDetector) detectTimeInterval(messages []models.ConversationMessage) bool {
    if len(messages) < 2 {
        return false
    }

    // 检查最近两条消息的时间间隔
    last := messages[len(messages)-1]
    secondLast := messages[len(messages)-2]

    if last.Timestamp.Sub(secondLast.Timestamp) > d.timeThreshold {
        return true
    }

    return false
}

// detectIntentChange 检测意图变化
func (d *TopicSwitchDetector) detectIntentChange(messages []models.ConversationMessage) (bool, bool) {
    if len(messages) < d.intentThreshold+1 {
        return false, false
    }

    // 统计最近的意图变化次数
    changes := 0
    recent := messages[len(messages)-d.intentThreshold-1:]

    for i := 1; i < len(recent); i++ {
        if recent[i].Intent != recent[i-1].Intent {
            changes++
        }
    }

    // 至少有1次意图变化才算切换
    switched := changes >= 1

    // 达到阈值才算频繁切换
    frequent := changes >= d.intentThreshold

    return switched, frequent
}
```

**Step 3: 运行测试**

```bash
go test ./internal/service -run TestTopicSwitchDetector -v
```

预期: 所有测试通过

**Step 4: 提交代码**

```bash
git add internal/service/topic_switch_detector.go internal/service/topic_switch_detector_test.go
git commit -m "feat: 添加话题切换检测器,支持时间间隔和意图变化检测"
```

---

## Task 3: 创建存档数据模型

**Files:**
- Create: `internal/models/conversation_archive.go`
- Modify: `internal/repository/interfaces/repository.go`
- Create: `internal/repository/sqlite/conversation_archive.go`
- Test: `internal/models/conversation_archive_test.go`

**Step 1: 创建模型**

创建 `internal/models/conversation_archive.go`:

```go
package models

import (
    "encoding/json"
    "time"
)

// ArchiveType 存档类型
type ArchiveType string

const (
    ArchiveTypeFull    ArchiveType = "full"
    ArchiveTypeSummary ArchiveType = "summary"
)

// ConversationArchive 对话存档
type ConversationArchive struct {
    ID              uint        `gorm:"primarykey"`
    UserID          uint        `gorm:"not null;index:idx_archives_user_id"`
    ArchiveType     ArchiveType `gorm:"not null;index:idx_archives_type;size:20"`

    // 内容
    Content         string      `gorm:"type:text"` // 完整内容
    Summary         string      `gorm:"type:text"` // AI摘要

    // 关键信息 (JSON)
    KeyEntities     string      `gorm:"type:text"` // JSON: 关键实体

    // 元信息
    ImportanceScore float64     `gorm:"default:0.5"`
    MessageCount    int         `gorm:"default:1"`
    DateRangeStart  *time.Time  `gorm:"index"`
    DateRangeEnd    *time.Time  `gorm:"index"`

    // 时间戳
    CreatedAt       time.Time   `gorm:"autoCreateTime;index:idx_archives_created"`
    ExpiresAt       *time.Time  `gorm:"index"`

    // 关联
    User            *User       `gorm:"foreignKey:UserID"`
}

// KeyEntities 关键实体结构
type KeyEntities struct {
    BookName   string   `json:"book_name,omitempty"`
    Topic      string   `json:"topic,omitempty"`
    Insights   []string `json:"insights,omitempty"`
    People     []string `json:"people,omitempty"`
    Keywords   []string `json:"keywords,omitempty"`
}

// GetKeyEntities 获取关键实体
func (a *ConversationArchive) GetKeyEntities() (*KeyEntities, error) {
    if a.KeyEntities == "" {
        return &KeyEntities{}, nil
    }

    var entities KeyEntities
    err := json.Unmarshal([]byte(a.KeyEntities), &entities)
    if err != nil {
        return nil, err
    }

    return &entities, nil
}

// SetKeyEntities 设置关键实体
func (a *ConversationArchive) SetKeyEntities(entities *KeyEntities) error {
    data, err := json.Marshal(entities)
    if err != nil {
        return err
    }

    a.KeyEntities = string(data)
    return nil
}

// IsExpired 检查是否过期
func (a *ConversationArchive) IsExpired() bool {
    if a.ExpiresAt == nil {
        return false
    }
    return time.Now().After(*a.ExpiresAt)
}

// SetExpiry 设置过期时间
func (a *ConversationArchive) SetExpiry(duration time.Duration) {
    if duration <= 0 {
        a.ExpiresAt = nil
        return
    }

    expiry := time.Now().Add(duration)
    a.ExpiresAt = &expiry
}

// TableName 指定表名
func (ConversationArchive) TableName() string {
    return "conversation_archives"
}
```

**Step 2: 编写模型测试**

创建 `internal/models/conversation_archive_test.go`:

```go
package models

import (
    "testing"
    "time"

   "github.com/stretchr/testify/assert"
)

func TestConversationArchive_KeyEntities(t *testing.T) {
    archive := &ConversationArchive{}

    entities := &KeyEntities{
        BookName: "沉默的大多数",
        Topic:    "社会观察",
        Insights: []string{"关于社会现象的思考"},
    }

    // 测试设置
    err := archive.SetKeyEntities(entities)
    assert.NoError(t, err)
    assert.NotEmpty(t, archive.KeyEntities)

    // 测试获取
    retrieved, err := archive.GetKeyEntities()
    assert.NoError(t, err)
    assert.Equal(t, "沉默的大多数", retrieved.BookName)
    assert.Equal(t, "社会观察", retrieved.Topic)
    assert.Len(t, retrieved.Insights, 1)
}

func TestConversationArchive_IsExpired(t *testing.T) {
    t.Run("已过期", func(t *testing.T) {
        archive := &ConversationArchive{}
        archive.SetExpiry(-1 * time.Hour) // 1小时前过期

        assert.True(t, archive.IsExpired())
    })

    t.Run("未过期", func(t *testing.T) {
        archive := &ConversationArchive{}
        archive.SetExpiry(1 * time.Hour) // 1小时后过期

        assert.False(t, archive.IsExpired())
    })

    t.Run("永不过期", func(t *testing.T) {
        archive := &ConversationArchive{}

        assert.False(t, archive.IsExpired())
    })
}
```

**Step 3: 添加Repository接口**

在 `internal/repository/interfaces/repository.go` 中添加:

```go
// ConversationArchiveRepository 存档仓库接口
type ConversationArchiveRepository interface {
    // Create 创建存档
    Create(ctx context.Context, archive *models.ConversationArchive) error

    // GetByUserID 获取用户的所有存档
    GetByUserID(ctx context.Context, userID uint, limit int, offset int) ([]*models.ConversationArchive, error)

    // GetByID 根据ID获取存档
    GetByID(ctx context.Context, id uint) (*models.ConversationArchive, error)

    // Delete 删除存档
    Delete(ctx context.Context, id uint) error

    // DeleteExpired 删除过期存档
    DeleteExpired(ctx context.Context) (int64, error)

    // CountByUserID 统计用户存档数量
    CountByUserID(ctx context.Context, userID uint) (int64, error)
}
```

**Step 4: 实现SQLite Repository**

创建 `internal/repository/sqlite/conversation_archive.go`:

```go
package sqlite

import (
    "context"
    "mmemory/internal/models"
    "mmemory/internal/repository/interfaces"
)

type conversationArchiveRepository struct {
    db *gorm.DB
}

// NewConversationArchiveRepository 创建存档仓库
func NewConversationArchiveRepository(db *gorm.DB) interfaces.ConversationArchiveRepository {
    return &conversationArchiveRepository{db: db}
}

// Create 创建存档
func (r *conversationArchiveRepository) Create(ctx context.Context, archive *models.ConversationArchive) error {
    return r.db.WithContext(ctx).Create(archive).Error
}

// GetByUserID 获取用户存档
func (r *conversationArchiveRepository) GetByUserID(ctx context.Context, userID uint, limit int, offset int) ([]*models.ConversationArchive, error) {
    var archives []*models.ConversationArchive
    err := r.db.WithContext(ctx).
        Where("user_id = ?", userID).
        Order("created_at DESC").
        Limit(limit).
        Offset(offset).
        Find(&archives).Error
    return archives, err
}

// GetByID 根据ID获取
func (r *conversationArchiveRepository) GetByID(ctx context.Context, id uint) (*models.ConversationArchive, error) {
    var archive models.ConversationArchive
    err := r.db.WithContext(ctx).First(&archive, id).Error
    return &archive, err
}

// Delete 删除存档
func (r *conversationArchiveRepository) Delete(ctx context.Context, id uint) error {
    return r.db.WithContext(ctx).Delete(&models.ConversationArchive{}, id).Error
}

// DeleteExpired 删除过期存档
func (r *conversationArchiveRepository) DeleteExpired(ctx context.Context) (int64, error) {
    result := r.db.WithContext(ctx).
        Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
        Delete(&models.ConversationArchive{})
    return result.RowsAffected, result.Error
}

// CountByUserID 统计数量
func (r *conversationArchiveRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).
        Model(&models.ConversationArchive{}).
        Where("user_id = ?", userID).
        Count(&count).Error
    return count, err
}
```

**Step 5: 在数据库迁移中添加表**

修改 `internal/repository/sqlite/database.go` 中的 `AutoMigrate`:

```go
func (d *Database) AutoMigrate(ctx context.Context) error {
    return d.db.WithContext(ctx).AutoMigrate(
        // ... 现有模型 ...
        &models.ConversationArchive{}, // 新增
    )
}
```

**Step 6: 运行测试**

```bash
go test ./internal/models -run TestConversationArchive -v
```

预期: 所有测试通过

**Step 7: 提交代码**

```bash
git add internal/models/conversation_archive.go internal/models/conversation_archive_test.go
git add internal/repository/interfaces/repository.go
git add internal/repository/sqlite/conversation_archive.go
git add internal/repository/sqlite/database.go
git commit -m "feat: 添加对话存档模型和Repository,支持完整内容和摘要存档"
```

---

## Task 4: 实现存档服务

**Files:**
- Create: `internal/service/conversation_archive_service.go`
- Test: `internal/service/conversation_archive_service_test.go`
- Modify: `internal/service/interfaces.go`

**Step 1: 添加服务接口**

在 `internal/service/interfaces.go` 中添加:

```go
// ConversationArchiveService 存档服务接口
type ConversationArchiveService interface {
    // CreateArchive 创建存档
    CreateArchive(ctx context.Context, userID uint, messages []models.ConversationMessage, archiveType models.ArchiveType) (*models.ConversationArchive, error)

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

**Step 2: 编写服务测试**

创建 `internal/service/conversation_archive_service_test.go`:

```go
package service

import (
    "context"
    "testing"
    "time"

    "mmemory/internal/models"
    "mmemory/pkg/ai"
)

// MockAIClient for testing
type MockAIClientForArchive struct{}

func (m *MockAIClientForArchive) GenerateResponse(ctx context.Context, prompt string, history string) (*ai.ParseResult, error) {
    return &ai.ParseResult{}, nil
}

func (m *MockAIClientForArchive) GenerateChatResponse(ctx context.Context, message string, history string) (string, error) {
    return "这是摘要", nil
}

func TestConversationArchiveService_ExtractKeyEntities(t *testing.T) {
    messages := []models.ConversationMessage{
        {
            Role:    "user",
            Content: "我在看《沉默的大多数》",
            Entities: map[string]models.ConversationEntityRef{
                "book": {Name: "book", Value: "沉默的大多数"},
            },
        },
    }

    config := &ai.AIConfig{}
    service := NewConversationArchiveService(nil, nil, config)

    entities, err := service.ExtractKeyEntities(messages)
    if err != nil {
        t.Fatalf("ExtractKeyEntities failed: %v", err)
    }

    if entities.BookName != "沉默的大多数" {
        t.Errorf("Expected BookName '沉默的大多数', got '%s'", entities.BookName)
    }
}

func TestConversationArchiveService_GenerateSummary(t *testing.T) {
    mockAI := &MockAIClientForArchive{}
    config := &ai.AIConfig{
        Enabled: true,
    }

    service := NewConversationArchiveService(mockAI, nil, config)

    messages := []models.ConversationMessage{
        {Role: "user", Content: "讨论了一本书"},
        {Role: "assistant", Content: "好的"},
    }

    summary, err := service.GenerateSummary(context.Background(), messages)
    if err != nil {
        t.Fatalf("GenerateSummary failed: %v", err)
    }

    if summary != "这是摘要" {
        t.Errorf("Expected summary '这是摘要', got '%s'", summary)
    }
}
```

**Step 3: 实现服务**

创建 `internal/service/conversation_archive_service.go`:

```go
package service

import (
    "context"
    "fmt"
    "strings"
    "time"

    "mmemory/internal/models"
    "mmemory/internal/repository/interfaces"
    "mmemory/pkg/ai"
)

type conversationArchiveService struct {
    archiveRepo interfaces.ConversationArchiveRepository
    aiClient    ai.Client
    config      *ai.AIConfig
}

// NewConversationArchiveService 创建存档服务
func NewConversationArchiveService(
    archiveRepo interfaces.ConversationArchiveRepository,
    aiClient ai.Client,
    config *ai.AIConfig,
) ConversationArchiveService {
    return &conversationArchiveService{
        archiveRepo: archiveRepo,
        aiClient:    aiClient,
        config:      config,
    }
}

// CreateArchive 创建存档
func (s *conversationArchiveService) CreateArchive(
    ctx context.Context,
    userID uint,
    messages []models.ConversationMessage,
    archiveType models.ArchiveType,
) (*models.ConversationArchive, error) {
    if len(messages) == 0 {
        return nil, fmt.Errorf("messages cannot be empty")
    }

    archive := &models.ConversationArchive{
        UserID:      userID,
        ArchiveType: archiveType,
        MessageCount: len(messages),
    }

    // 设置时间范围
    startTime := messages[0].Timestamp
    endTime := messages[len(messages)-1].Timestamp
    archive.DateRangeStart = &startTime
    archive.DateRangeEnd = &endTime

    // 根据类型处理内容
    if archiveType == models.ArchiveTypeFull {
        // 完整内容
        content := s.formatMessages(messages)
        archive.Content = content
        archive.SetExpiry(s.config.ContextManagement.Archive.FullContentTTL)
    } else if archiveType == models.ArchiveTypeSummary {
        // 生成摘要
        if s.config.ContextManagement.Compression.AISummaryEnabled {
            summary, err := s.GenerateSummary(ctx, messages)
            if err != nil {
                return nil, fmt.Errorf("failed to generate summary: %w", err)
            }
            archive.Summary = summary
        }
        archive.SetExpiry(s.config.ContextManagement.Archive.SummaryTTL)
    }

    // 提取关键实体
    entities, err := s.ExtractKeyEntities(messages)
    if err != nil {
        return nil, fmt.Errorf("failed to extract entities: %w", err)
    }
    archive.SetKeyEntities(entities)

    // 保存到数据库
    if err := s.archiveRepo.Create(ctx, archive); err != nil {
        return nil, fmt.Errorf("failed to save archive: %w", err)
    }

    return archive, nil
}

// GenerateSummary 生成摘要
func (s *conversationArchiveService) GenerateSummary(ctx context.Context, messages []models.ConversationMessage) (string, error) {
    if !s.config.Enabled {
        return "", fmt.Errorf("AI service not enabled")
    }

    // 格式化对话
    conversation := s.formatMessages(messages)

    // 构建prompt
    prompt := fmt.Sprintf(`请将以下对话压缩成简短摘要(最多%d字),保留关键信息(书名、人名、主要观点):

对话内容:
%s

请直接返回摘要内容,不要添加其他说明。`, s.config.ContextManagement.Compression.MaxSummaryLength, conversation)

    // 调用AI生成摘要
    summary, err := s.aiClient.GenerateChatResponse(ctx, prompt, "")
    if err != nil {
        return "", fmt.Errorf("AI生成摘要失败: %w", err)
    }

    return strings.TrimSpace(summary), nil
}

// ExtractKeyEntities 提取关键实体
func (s *conversationArchiveService) ExtractKeyEntities(messages []models.ConversationMessage) (*models.KeyEntities, error) {
    entities := &models.KeyEntities{}

    for _, msg := range messages {
        // 从消息实体中提取
        for key, entity := range msg.Entities {
            value := strings.ToLower(entity.Value)

            switch key {
            case "book", "book_name":
                if entities.BookName == "" {
                    entities.BookName = entity.Value
                }
            case "topic":
                if entities.Topic == "" {
                    entities.Topic = entity.Value
                }
            }

            // 收集关键词
            if !contains(entities.Keywords, value) {
                entities.Keywords = append(entities.Keywords, value)
            }
        }

        // 从内容中提取书名模式
        if entities.BookName == "" {
            if bookName := extractBookName(msg.Content); bookName != "" {
                entities.BookName = bookName
            }
        }
    }

    return entities, nil
}

// GetUserArchives 获取用户存档
func (s *conversationArchiveService) GetUserArchives(ctx context.Context, userID uint, limit int) ([]*models.ConversationArchive, error) {
    if limit <= 0 {
        limit = 20 // 默认20条
    }
    return s.archiveRepo.GetByUserID(ctx, userID, limit, 0)
}

// CleanupExpiredArchives 清理过期存档
func (s *conversationArchiveService) CleanupExpiredArchives(ctx context.Context) (int64, error) {
    return s.archiveRepo.DeleteExpired(ctx)
}

// formatMessages 格式化消息为文本
func (s *conversationArchiveService) formatMessages(messages []models.ConversationMessage) string {
    var builder strings.Builder

    for _, msg := range messages {
        builder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
    }

    return builder.String()
}

// extractBookName 从文本中提取书名
func extractBookName(text string) string {
    // 简单模式匹配《书名》
    // TODO: 可以使用更复杂的正则或AI提取
    return ""
}

// contains 检查字符串是否在切片中
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

**Step 4: 运行测试**

```bash
go test ./internal/service -run TestConversationArchiveService -v
```

预期: 所有测试通过

**Step 5: 提交代码**

```bash
git add internal/service/conversation_archive_service.go internal/service/conversation_archive_service_test.go
git add internal/service/interfaces.go
git commit -m "feat: 实现存档服务,支持创建存档、生成摘要、提取关键实体"
```

---

## Task 5: 实现上下文Token管理器

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

func TestContextTokenManager_NeedsCleanup(t *testing.T) {
    config := &ai.AIConfig{
        Providers: []ai.AIProviderConfig{
            {MaxContextTokens: 1000},
        },
        ContextManagement: ai.ContextManagementConfig{
            WarningThreshold: 0.8,
            HardLimit:        0.95,
        },
    }

    manager := NewContextTokenManager(config, nil)

    // 测试token使用率
    t.Run("低于阈值", func(t *testing.T) {
        messages := []models.ConversationMessage{
            {Content: "短消息"}, // 约10 token
        }
        needs, ratio := manager.NeedsCleanup(messages)
        if needs {
            t.Errorf("不需要清理")
        }
        if ratio > 0.8 {
            t.Errorf("使用率应该低于0.8, got %v", ratio)
        }
    })

    t.Run("超过警告阈值", func(t *testing.T) {
        // 构造长消息
        longText := string(make([]byte, 500)) // 约1000 token
        messages := []models.ConversationMessage{
            {Content: longText},
        }
        needs, ratio := manager.NeedsCleanup(messages)
        if !needs {
            t.Errorf("应该需要清理")
        }
        if ratio < 0.8 {
            t.Errorf("使用率应该高于0.8, got %v", ratio)
        }
    })
}
```

**Step 2: 实现管理器**

创建 `internal/service/context_token_manager.go`:

```go
package service

import (
    "context"
    "fmt"

    "mmemory/internal/models"
    "mmemory/pkg/ai"
)

// ContextTokenManager 上下文Token管理器
type ContextTokenManager struct {
    config          *ai.AIConfig
    tokenEstimator  *TokenEstimator
    topicDetector   *TopicSwitchDetector
    archiveService  ConversationArchiveService
}

// NewContextTokenManager 创建管理器
func NewContextTokenManager(
    config *ai.AIConfig,
    archiveService ConversationArchiveService,
) *ContextTokenManager {
    return &ContextTokenManager{
        config:         config,
        tokenEstimator: NewTokenEstimator(),
        topicDetector:  NewTopicSwitchDetector(config.ContextManagement),
        archiveService: archiveService,
    }
}

// NeedsCleanup 检查是否需要清理
// 返回: (是否需要清理, token使用率)
func (m *ContextTokenManager) NeedsCleanup(messages []models.ConversationMessage) (bool, float64) {
    if len(messages) == 0 {
        return false, 0
    }

    // 获取模型的最大token数
    maxTokens := m.getMaxContextTokens()
    if maxTokens <= 0 {
        return false, 0
    }

    // 估算使用率
    ratio := m.tokenEstimator.EstimateUsageRatio(messages, maxTokens)

    // 超过警告阈值
    needsCleanup := ratio >= m.config.ContextManagement.WarningThreshold

    return needsCleanup, ratio
}

// CleanupMessages 清理消息
// 返回: (清理后的消息, 是否压缩, 是否需要询问用户, error)
func (m *ContextTokenManager) CleanupMessages(
    ctx context.Context,
    userID uint,
    messages []models.ConversationMessage,
) ([]models.ConversationMessage, bool, bool, error) {
    if len(messages) == 0 {
        return messages, false, false, nil
    }

    maxTokens := m.getMaxContextTokens()
    ratio := m.tokenEstimator.EstimateUsageRatio(messages, maxTokens)

    // 检测话题切换
    topicSwitched, frequentSwitch := m.topicDetector.DetectSwitch(messages)

    // 根据使用率选择策略
    var strategy CleanupStrategy
    if ratio >= m.config.ContextManagement.HardLimit {
        strategy = StrategyForceClean
    } else if ratio >= m.config.ContextManagement.WarningThreshold {
        if topicSwitched && !frequentSwitch {
            strategy = StrategyAskUser
        } else {
            strategy = StrategySmartClean
        }
    } else {
        return messages, false, false, nil
    }

    // 执行清理
    cleaned, compressed, askUser, err := m.executeCleanup(ctx, userID, messages, strategy)
    if err != nil {
        return nil, false, false, err
    }

    return cleaned, compressed, askUser, nil
}

// executeCleanup 执行清理策略
func (m *ContextTokenManager) executeCleanup(
    ctx context.Context,
    userID uint,
    messages []models.ConversationMessage,
    strategy CleanupStrategy,
) ([]models.ConversationMessage, bool, bool, error) {
    switch strategy {
    case StrategyForceClean:
        return m.forceClean(ctx, userID, messages)
    case StrategySmartClean:
        return m.smartClean(ctx, userID, messages)
    case StrategyAskUser:
        return messages, false, true, nil
    default:
        return messages, false, false, nil
    }
}

// forceClean 强制清理
func (m *ContextTokenManager) forceClean(
    ctx context.Context,
    userID uint,
    messages []models.ConversationMessage,
) ([]models.ConversationMessage, bool, bool, error) {
    // 1. 保留最近8条
    keepRecent := m.config.ContextManagement.KeepRecentMessages
    if keepRecent <= 0 {
        keepRecent = 8
    }

    if len(messages) <= keepRecent {
        return messages, false, false, nil
    }

    recent := messages[len(messages)-keepRecent:]
    oldMessages := messages[:len(messages)-keepRecent]

    // 2. 删除不重要的消息
    important := m.filterImportantMessages(oldMessages)

    // 3. 压缩旧消息
    if len(important) > 0 && m.config.ContextManagement.Compression.AISummaryEnabled {
        _, err := m.archiveService.CreateArchive(ctx, userID, important, models.ArchiveTypeSummary)
        if err != nil {
            // 失败则直接丢弃
            logger.Warnf("创建存档失败: %v", err)
        }
    }

    return recent, true, false, nil
}

// smartClean 智能清理
func (m *ContextTokenManager) smartClean(
    ctx context.Context,
    userID uint,
    messages []models.ConversationMessage,
) ([]models.ConversationMessage, bool, bool, error) {
    // 只删除不重要的消息
    filtered := m.filterImportantMessages(messages)

    if len(filtered) == len(messages) {
        // 没有删除任何消息
        return messages, false, false, nil
    }

    logger.Infof("智能清理: 从 %d 条消息减少到 %d 条", len(messages), len(filtered))
    return filtered, true, false, nil
}

// filterImportantMessages 过滤重要消息
func (m *ContextTokenManager) filterImportantMessages(messages []models.ConversationMessage) []models.ConversationMessage {
    var important []models.ConversationMessage

    for _, msg := range messages {
        // 判断消息是否重要
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
        "reminder": true,      // 已入库的提醒
        "query_activity": true, // 查询
    }

    // 如果是不重要的意图,检查是否已处理
    if unimportantIntents[msg.Intent] {
        // 可以根据实际情况判断
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
    StrategyForceClean                 // 强制清理: 压缩+保留最近
    StrategyAskUser                    // 询问用户
)
```

**Step 3: 运行测试**

```bash
go test ./internal/service -run TestContextTokenManager -v
```

预期: 所有测试通过

**Step 4: 提交代码**

```bash
git add internal/service/context_token_manager.go internal/service/context_token_manager_test.go
git commit -m "feat: 实现上下文Token管理器,支持智能清理和压缩策略"
```

---

## Task 6: 集成到消息处理器

**Files:**
- Modify: `internal/bot/handlers/message.go`
- Modify: `internal/service/interfaces.go` (添加ContextTokenManager接口)

**Step 1: 添加管理器接口**

在 `internal/service/interfaces.go` 中添加:

```go
// ContextTokenManagerService Token管理服务接口
type ContextTokenManagerService interface {
    // NeedsCleanup 检查是否需要清理
    NeedsCleanup(messages []models.ConversationMessage) (bool, float64)

    // CleanupMessages 清理消息
    CleanupMessages(ctx context.Context, userID uint, messages []models.ConversationMessage) ([]models.ConversationMessage, bool, bool, error)
}
```

**Step 2: 修改MessageHandler结构**

在 `internal/bot/handlers/message.go` 中修改MessageHandler结构(约第50-60行):

```go
type MessageHandler struct {
    // ... 现有字段 ...

    // 新增字段
    contextTokenManager service.ContextTokenManagerService
}
```

**Step 3: 修改构造函数**

在 `NewMessageHandler` 函数中添加参数(约第100-150行):

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
    contextTokenManager service.ContextTokenManagerService, // 新增
) *MessageHandler {
    return &MessageHandler{
        // ... 现有字段 ...
        contextTokenManager: contextTokenManager, // 新增
    }
}
```

**Step 4: 修改formatConversationHistory方法**

在 `internal/bot/handlers/message.go:1761` 修改:

```go
func (h *MessageHandler) formatConversationHistory(messages []models.ConversationMessage) string {
    if len(messages) == 0 {
        return ""
    }

    // 如果有Token管理器,先进行清理检查
    if h.contextTokenManager != nil {
        needsCleanup, ratio := h.contextTokenManager.NeedsCleanup(messages)
        if needsCleanup {
            logger.Infof("对话历史使用率: %.2f%%, 需要清理", ratio*100)

            // 执行清理(注意:这里是同步清理,不询问用户)
            // 询问用户需要另外的异步机制
            cleaned, compressed, _, err := h.contextTokenManager.CleanupMessages(
                context.Background(),
                0, // userID未知,需要传入
                messages,
            )
            if err != nil {
                logger.Warnf("清理对话历史失败: %v", err)
            } else {
                messages = cleaned
                if compressed {
                    logger.Infof("对话历史已压缩,从 %d 条消息", len(cleaned))
                }
            }
        }
    }

    // 最多保留最近20条消息(兜底���辑)
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

在 `cmd/bot/main.go` 或相关的DI容器中,创建并注入ContextTokenManager:

```go
// 创建Token管理器
contextTokenManager := service.NewContextTokenManager(
    aiConfig,
    archiveService,
)
```

**Step 6: 编译检查**

```bash
go build ./internal/bot/handlers
```

预期: 无编译错误

**Step 7: 提交代码**

```bash
git add internal/bot/handlers/message.go internal/service/interfaces.go
git add cmd/bot/main.go (如果修改了)
git commit -m "feat: 集成Token管理器到消息处理流程,自动清理超限对话"
```

---

## Task 7: 添加配置文件

**Files:**
- Modify: `pkg/config/ai_config.go`
- Modify: `configs/config.example.yaml`

**Step 1: 更新配置结构**

在 `pkg/config/ai_config.go` 中添加完整配置结构:

```go
type AIConfig struct {
    Enabled  bool                     `mapstructure:"enabled" yaml:"enabled"`
    Providers []AIProviderConfig      `mapstructure:"providers" yaml:"providers"`
    ContextManagement ContextManagementConfig `mapstructure:"context_management" yaml:"context_management"`
    // ... 其他字段
}

type AIProviderConfig struct {
    Name             string  `mapstructure:"name" yaml:"name"`
    Provider         string  `mapstructure:"provider" yaml:"provider"`
    APIKey           string  `mapstructure:"api_key" yaml:"api_key"`
    BaseURL          string  `mapstructure:"base_url" yaml:"base_url"`
    Model            string  `mapstructure:"model" yaml:"model"`
    BackupModel      string  `mapstructure:"backup_model" yaml:"backup_model"`
    MaxContextTokens int     `mapstructure:"max_context_tokens" yaml:"max_context_tokens"`
    Temperature      float64 `mapstructure:"temperature" yaml:"temperature"`
    MaxTokens        int     `mapstructure:"max_tokens" yaml:"max_tokens"`
}

type ContextManagementConfig struct {
    Enabled            bool               `mapstructure:"enabled" yaml:"enabled"`
    KeepRecentMessages int                `mapstructure:"keep_recent_messages" yaml:"keep_recent_messages"`
    WarningThreshold   float64            `mapstructure:"warning_threshold" yaml:"warning_threshold"`
    HardLimit          float64            `mapstructure:"hard_limit" yaml:"hard_limit"`
    TopicSwitch        TopicSwitchConfig  `mapstructure:"topic_switch" yaml:"topic_switch"`
    Archive            ArchiveConfig      `mapstructure:"archive" yaml:"archive"`
    Compression        CompressionConfig  `mapstructure:"compression" yaml:"compression"`
}

type TopicSwitchConfig struct {
    Enabled               bool `mapstructure:"enabled" yaml:"enabled"`
    TimeThresholdMinutes  int  `mapstructure:"time_threshold_minutes" yaml:"time_threshold_minutes"`
    IntentChangeThreshold int  `mapstructure:"intent_change_threshold" yaml:"intent_change_threshold"`
}

type ArchiveConfig struct {
    Enabled         bool          `mapstructure:"enabled" yaml:"enabled"`
    AskUser         bool          `mapstructure:"ask_user" yaml:"ask_user"`
    FullContentTTL  time.Duration `mapstructure:"full_content_ttl" yaml:"full_content_ttl"`
    SummaryTTL      time.Duration `mapstructure:"summary_ttl" yaml:"summary_ttl"`
}

type CompressionConfig struct {
    KeepEntities     bool `mapstructure:"keep_entities" yaml:"keep_entities"`
    AISummaryEnabled bool `mapstructure:"ai_summary_enabled" yaml:"ai_summary_enabled"`
    MaxSummaryLength int  `mapstructure:"max_summary_length" yaml:"max_summary_length"`
}
```

**Step 2: 更新配置示例**

在 `configs/config.example.yaml` 中添加:

```yaml
ai:
  enabled: false  # 默认关闭,需要手动启用

  providers:
    - name: "openai-gpt-4o-mini"
      provider: "openai"
      api_key: "${MMEMORY_AI_OPENAI_API_KEY}"
      base_url: "https://api.openai.com/v1"
      model: "gpt-4o-mini"
      backup_model: "gpt-3.5-turbo"
      max_context_tokens: 128000  # GPT-4o-mini的上下文长度
      temperature: 0.1
      max_tokens: 1000

    - name: "zai-glm-4.6"
      provider: "openai"
      api_key: "${MMEMORY_AI_ZAI_API_KEY}"
      base_url: "https://api.siliconflow.cn/v1"
      model: "zai-org/GLM-4.6"
      max_context_tokens: 128000  # GLM-4.6的上下文长度
      temperature: 0.1
      max_tokens: 1000

  # 上下文管理配置
  context_management:
    enabled: true
    keep_recent_messages: 8  # 保留最近8条对话

    # Token管理阈值
    warning_threshold: 0.8   # 80%触发警告
    hard_limit: 0.95         # 95%强制清理

    # 话题切换检测
    topic_switch:
      enabled: true
      time_threshold_minutes: 30       # 30分钟无消息视为话题切换
      intent_change_threshold: 2      # 意图改变2次视为频繁切换

    # 存档配置
    archive:
      enabled: true
      ask_user: true                  # 不确定时询问用户
      full_content_ttl: 720h          # 完整内容保留30天
      summary_ttl: 0                  # 摘要永久保存(0 = 不过期)

    # 压缩配置
    compression:
      keep_entities: true             # 保留关键实体
      ai_summary_enabled: true        # 使用AI生成摘要
      max_summary_length: 200         # 摘要最多200字
```

**Step 3: 提交配置**

```bash
git add pkg/config/ai_config.go configs/config.example.yaml
git commit -m "feat: 添加上下文管理配置,支持token限制和压缩策略"
```

---

## Task 8: 添加监控指标

**Files:**
- Modify: `pkg/metrics/metrics.go`
- Modify: `internal/service/context_token_manager.go`

**Step 1: 添加指标**

在 `pkg/metrics/metrics.go` 中添加:

```go
var (
    // ... 现有指标 ...

    // ContextManagementMetrics
    ContextCleanupTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "context_cleanup_total",
            Help: "Total number of context cleanup operations",
        },
        []string{"user_id", "strategy"}, // strategy: smart_clean, force_clean
    )

    ContextCompressionTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "context_compression_total",
            Help: "Total number of context compression operations",
        },
        []string{"user_id", "success"}, // success: true, false
    )

    ContextTokenUsageGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "context_token_usage_ratio",
            Help: "Context token usage ratio (0-1)",
        },
        []string{"user_id"},
    )
)

func init() {
    // ... 现有注册 ...

    prometheus.MustRegister(ContextCleanupTotal)
    prometheus.MustRegister(ContextCompressionTotal)
    prometheus.MustRegister(ContextTokenUsageGauge)
}
```

**Step 2: 在管理器中记录指标**

在 `internal/service/context_token_manager.go` 的方法中添加指标记录:

```go
func (m *ContextTokenManager) CleanupMessages(
    ctx context.Context,
    userID uint,
    messages []models.ConversationMessage,
) ([]models.ConversationMessage, bool, bool, error) {
    // ... 现有逻辑 ...

    // 执行清理后记录指标
    userStr := fmt.Sprintf("%d", userID)
    metrics.ContextTokenUsageGauge.WithLabelValues(userStr).Set(ratio)

    if compressed {
        metrics.ContextCompressionTotal.WithLabelValues(userStr, "true").Inc()
    }

    strategyName := "none"
    switch strategy {
    case StrategySmartClean:
        strategyName = "smart_clean"
    case StrategyForceClean:
        strategyName = "force_clean"
    case StrategyAskUser:
        strategyName = "ask_user"
    }
    metrics.ContextCleanupTotal.WithLabelValues(userStr, strategyName).Inc()

    return cleaned, compressed, askUser, nil
}
```

**Step 3: 测试指标**

```bash
# 启动应用
make run

# 查看指标
curl http://localhost:9090/metrics | grep context
```

预期输出:
```
context_cleanup_total{strategy="smart_clean",user_id="123"} 1.0
context_token_usage_ratio{user_id="123"} 0.75
context_compression_total{success="true",user_id="123"} 1.0
```

**Step 4: 提交代码**

```bash
git add pkg/metrics/metrics.go internal/service/context_token_manager.go
git commit -m "feat: 添加上下文管理的Prometheus监控指标"
```

---

## Task 9: 集成测试和文档

**Files:**
- Test: `internal/bot/handlers/message_integration_test.go` (更新)
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Step 1: 添加集成测试**

在 `internal/bot/handlers/message_integration_test.go` 中添加:

```go
func TestMessageHandler_ContextCleanup(t *testing.T) {
    // 创建mock服务
    mockTokenManager := new(MockContextTokenManagerService)

    handler := NewMessageHandler(
        nil, nil, nil, nil, nil, nil, nil, nil, nil,
        mockTokenManager, // 新增
    )

    // 测试上下文清理
    t.Run("Token超限触发清理", func(t *testing.T) {
        messages := []models.ConversationMessage{
            {Role: "user", Content: "消息1"},
            {Role: "assistant", Content: "回复1"},
            // ... 添加更多消息 ...
        }

        mockTokenManager.On("NeedsCleanup", messages).Return(true, 0.9)
        mockTokenManager.On("CleanupMessages", mock.Anything, uint(0), messages).Return(
            messages[:2], // 清理后只剩2条
            true,         // 已压缩
            false,        // 不询问
            nil,
        )

        result := handler.formatConversationHistory(messages)

        mockTokenManager.AssertExpectations(t)
        assert.Contains(t, result, "对话历史")
    })
}
```

**Step 2: 更新CLAUDE.md**

在 `CLAUDE.md` 中添加上下文管理说明:

```markdown
## Context Management

The system includes intelligent context management to prevent token limit issues:

**Features:**
- Token estimation with mixed language support (Chinese ×2 + English words)
- Automatic cleanup when usage exceeds 80% threshold
- Smart message filtering (removes unimportant conversations)
- AI-powered conversation compression
- User confirmation for important content archiving
- Per-model context length configuration

**Configuration:**
```yaml
ai:
  context_management:
    enabled: true
    keep_recent_messages: 8
    warning_threshold: 0.8
    hard_limit: 0.95
```

**Cleanup Strategies:**
1. **Smart Clean** (<95%): Removes unimportant messages only
2. **Force Clean** (>95%): Compresses old conversations + keeps recent 8
3. **Ask User**: Prompts user to confirm archiving

**Archive Types:**
- `full`: Complete conversation (expires in 30 days)
- `summary`: AI-generated summary (permanent)
```

**Step 3: 更新README.md**

添加使用说明:

```markdown
## Context Management

When conversation history grows large, the bot automatically:

1. Monitors token usage (80% warning, 95% hard limit)
2. Removes unimportant messages (confirmed reminders)
3. Compresses discussions using AI
4. Keeps recent 8 messages intact
5. Asks before archiving important content

### Configuration

Set per-model context limits in `config.yaml`:
```yaml
ai:
  providers:
    - name: "gpt-4o-mini"
      max_context_tokens: 128000
```

### Monitoring

Check context usage metrics:
```bash
curl http://localhost:9090/metrics | grep context_token_usage
```
```

**Step 4: 运行所有测试**

```bash
make test
```

预期: 所有测试通过

**Step 5: 提交文档**

```bash
git add CLAUDE.md README.md
git add internal/bot/handlers/message_integration_test.go
git commit -m "docs: 添加上下文管理功能文档和集成测试"
```

---

## 总结和验证清单

**已实现功能:**
1. ✅ Token估算器(支持中英文混合)
2. ✅ 话题切换检测
3. ✅ 对话存档服务
4. ✅ Token管理器(分级清理策略)
5. ✅ 集成到消息处理流程
6. ✅ 配置管理
7. ✅ Prometheus监控

**测试检查清单:**
- [ ] Token估算准确率>90%
- [ ] 话题切换检测正确识别时间间隔
- [ ] 话题切换检测正确识别意图变化
- [ ] 存档创建成功保存到数据库
- [ ] AI摘要生成正常工作
- [ ] 关键实体提取准确
- [ ] Token使用率<80%时不触发清理
- [ ] Token使用率>95%时强制清理
- [ ] 单一话题切换时询问用户
- [ ] 频繁切换时异步询问
- [ ] 不重要消息被正确过滤
- [ ] 最近8条消息始终保留
- [ ] Prometheus指标正常上报
- [ ] 日志记录清晰

**手动测试场景:**
1. **正常对话**: Token<80%,无清理
2. **提醒创建**: 已入库提醒被删除
3. **长对话**: Token>95%,触发强制清理
4. **话题切换**: 30分钟无消息后继续对话
5. **频繁切换**: 快速切换意图多次
6. **存档确认**: 询问用户是否存档
7. **查询存档**: 用户查看历史存档

**性能优化:**
- Token估算使用简单算法,无需外部依赖
- AI摘要仅在必要时调用
- 数据库查询添加索引
- 过期存档定期清理

**后续优化(YAGNI):**
- 使用tiktoken精确计算token
- 学习用户存档偏好
- 自动识别更复杂的模式
- 跨会话的上下文共享

**预计完成时间**: 每个任务 15-30 分钟,总计 3-5 小时
