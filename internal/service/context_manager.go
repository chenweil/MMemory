package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
	"mmemory/pkg/metrics"
)

const (
	defaultContextState       = "idle"
	defaultContextIntent      = "unknown"
	defaultContextSessionName = "default"
)

// ProcessMessageInput 上下文处理输入参数
type ProcessMessageInput struct {
	UserID    uint
	SessionID string
	Role      string
	Message   string
	Channel   string
	Locale    string
}

// UpdateContextStateInput 更新上下文状态的输入参数
type UpdateContextStateInput struct {
	UserID    uint
	SessionID string
	State     string
	Intent    string
	Entities  map[string]models.ConversationEntity
	Channel   string
	Locale    string
	TTL       time.Duration
}

// ContextManagerConfig 上下文管理配置
type ContextManagerConfig struct {
	MaxMessages int
	DefaultTTL  time.Duration
}

// ContextManager 上下文管理器
type ContextManager struct {
	repo              interfaces.ConversationContextRepository
	extractor         EntityExtractor
	intent            IntentTracker
	maxMessages       int
	defaultTTL        time.Duration
	nowFunc           func() time.Time
	tokenManager      ContextTokenManagerService
	archiveService    ConversationArchiveService
	maxTokens         int
	enableAutoCleanup bool // 是否启用自动清理
}

// EntityExtractor 实体提取器接口
type EntityExtractor interface {
	Extract(message string, ctx *models.ConversationContextState) map[string]models.ConversationEntity
}

// IntentTracker 意图识别接口
type IntentTracker interface {
	DetermineIntent(message string, ctx *models.ConversationContextState) string
}

// NewContextManager 创建上下文管理器
func NewContextManager(
	repo interfaces.ConversationContextRepository,
	extractor EntityExtractor,
	intent IntentTracker,
	config ContextManagerConfig,
	tokenManager ContextTokenManagerService,
	archiveService ConversationArchiveService,
	maxTokens int,
) *ContextManager {
	maxMessages := config.MaxMessages
	if maxMessages <= 0 {
		maxMessages = 20
	}

	defaultTTL := config.DefaultTTL
	if defaultTTL <= 0 {
		defaultTTL = 30 * time.Minute
	}

	if maxTokens <= 0 {
		maxTokens = 128000 // 默认128k
	}

	return &ContextManager{
		repo:              repo,
		extractor:         extractor,
		intent:            intent,
		maxMessages:       maxMessages,
		defaultTTL:        defaultTTL,
		nowFunc:           time.Now,
		tokenManager:      tokenManager,
		archiveService:    archiveService,
		maxTokens:         maxTokens,
		enableAutoCleanup: true, // 默认启用自动清理
	}
}

// ProcessMessage 处理消息并更新上下文
func (m *ContextManager) ProcessMessage(ctx context.Context, input ProcessMessageInput) (*models.ConversationContextState, error) {
	if input.UserID == 0 {
		return nil, fmt.Errorf("userID is required")
	}

	now := m.nowFunc()

	currentState, err := m.getOrCreateState(ctx, input.UserID, input.SessionID)
	if err != nil {
		return nil, err
	}

	if len(currentState.Messages) >= m.maxMessages {
		start := len(currentState.Messages) - (m.maxMessages - 1)
		currentState.Messages = currentState.Messages[start:]
	}

	if input.Channel != "" {
		currentState.Channel = input.Channel
	}
	if input.Locale != "" {
		currentState.Locale = input.Locale
	}

	intentValue := m.intent.DetermineIntent(input.Message, currentState)
	if intentValue == "" {
		intentValue = defaultContextIntent
	}
	currentState.Intent = intentValue

	entities := m.extractor.Extract(input.Message, currentState)
	if currentState.Entities == nil {
		currentState.Entities = make(map[string]models.ConversationEntity)
	}
	for key, value := range entities {
		value.UpdatedAt = now
		currentState.Entities[key] = value
	}

	messageEntities := make(map[string]models.ConversationEntityRef, len(entities))
	for key, value := range entities {
		messageEntities[key] = models.ConversationEntityRef{
			Name:       value.Name,
			Value:      value.Value,
			Confidence: value.Confidence,
			Source:     value.Source,
		}
	}

	currentState.Messages = append(currentState.Messages, models.ConversationMessage{
		Role:      input.Role,
		Content:   input.Message,
		Intent:    currentState.Intent,
		Entities:  messageEntities,
		Timestamp: now,
	})

	// 自动清理：检查Token使用率并触发清理
	if m.enableAutoCleanup && m.tokenManager != nil {
		if needsPruning, _ := m.tokenManager.NeedsPruning(currentState.Messages); needsPruning {
			// 清理消息
			toKeep, toArchive, strategy := m.tokenManager.PruneMessages(currentState.Messages)

			// 记录清理指标
			userStr := fmt.Sprintf("%d", input.UserID)
			metrics.RecordContextCleanup(userStr, strategy.String())

			// 异步归档被清理的消息
			if len(toArchive) > 0 && m.archiveService != nil {
				// 选择归档类型：强制清理使用摘要，智能清理使用完整内容
				var archiveType models.ArchiveType
				if strategy == StrategyForceClean {
					archiveType = models.ArchiveTypeSummary // 强制清理时尝试摘要
				} else {
					archiveType = models.ArchiveTypeFull // 智能清理时使用完整内容
				}

				m.archiveService.CreateArchiveAsync(input.UserID, toArchive, archiveType)

				logger.Infof("自动清理: userID=%d, strategy=%s, kept=%d, archived=%d",
					input.UserID, strategy.String(), len(toKeep), len(toArchive))
			}

			// 更新当前状态的消息列表
			currentState.Messages = toKeep

			// 记录压缩指标
			metrics.RecordContextCompression(userStr, true)
		}
	}

	currentState.LastActivity = now
	if currentState.CreatedAt.IsZero() {
		currentState.CreatedAt = now
	}
	currentState.TTLSeconds = int64(m.defaultTTL.Seconds())

	if err := m.persistState(ctx, currentState, m.defaultTTL); err != nil {
		return nil, err
	}

	return currentState, nil
}

// GetContext 获取上下文
func (m *ContextManager) GetContext(ctx context.Context, userID uint) (*models.ConversationContextState, error) {
	if userID == 0 {
		return nil, fmt.Errorf("userID is required")
	}

	model, err := m.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, nil
	}

	if expired := isExpired(model, m.nowFunc()); expired {
		_ = m.repo.DeleteByUserID(ctx, userID)
		return nil, nil
	}

	return convertModelToState(model)
}

// UpdateContextState 更新上下文状态
func (m *ContextManager) UpdateContextState(ctx context.Context, input UpdateContextStateInput) error {
	if input.UserID == 0 {
		return fmt.Errorf("userID is required")
	}

	state, err := m.getOrCreateState(ctx, input.UserID, input.SessionID)
	if err != nil {
		return err
	}

	if input.State != "" {
		state.State = input.State
	}
	if input.Intent != "" {
		state.Intent = input.Intent
	}
	if input.Channel != "" {
		state.Channel = input.Channel
	}
	if input.Locale != "" {
		state.Locale = input.Locale
	}
	if input.Entities != nil {
		if state.Entities == nil {
			state.Entities = make(map[string]models.ConversationEntity)
		}
		now := m.nowFunc()
		for key, entity := range input.Entities {
			entity.UpdatedAt = now
			state.Entities[key] = entity
		}
	}

	ttl := m.defaultTTL
	if input.TTL > 0 {
		ttl = input.TTL
		state.TTLSeconds = int64(ttl.Seconds())
	}

	state.LastActivity = m.nowFunc()

	return m.persistState(ctx, state, ttl)
}

// ClearContext 清除上下文
func (m *ContextManager) ClearContext(ctx context.Context, userID uint) error {
	if userID == 0 {
		return fmt.Errorf("userID is required")
	}
	return m.repo.DeleteByUserID(ctx, userID)
}

// CleanupExpired 清理过期上下文
func (m *ContextManager) CleanupExpired(ctx context.Context) error {
	return m.repo.CleanupExpired(ctx, m.nowFunc())
}

func (m *ContextManager) getOrCreateState(ctx context.Context, userID uint, sessionID string) (*models.ConversationContextState, error) {
	if sessionID == "" {
		sessionID = defaultContextSessionName
	}

	model, err := m.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := m.nowFunc()

	if model == nil || isExpired(model, now) {
		if model != nil && isExpired(model, now) {
			_ = m.repo.DeleteByUserID(ctx, userID)
		}
		return &models.ConversationContextState{
			UserID:       userID,
			SessionID:    sessionID,
			State:        defaultContextState,
			Intent:       defaultContextIntent,
			Messages:     make([]models.ConversationMessage, 0),
			Entities:     make(map[string]models.ConversationEntity),
			CreatedAt:    now,
			LastActivity: now,
			TTLSeconds:   int64(m.defaultTTL.Seconds()),
		}, nil
	}

	state, err := convertModelToState(model)
	if err != nil {
		return nil, err
	}

	if sessionID != "" {
		state.SessionID = sessionID
	}

	return state, nil
}

func (m *ContextManager) persistState(ctx context.Context, state *models.ConversationContextState, ttl time.Duration) error {
	if state == nil {
		return fmt.Errorf("context state is nil")
	}

	messagesData, err := json.Marshal(state.Messages)
	if err != nil {
		return fmt.Errorf("marshal messages failed: %w", err)
	}

	entitiesData, err := json.Marshal(state.Entities)
	if err != nil {
		return fmt.Errorf("marshal entities failed: %w", err)
	}

	model, err := m.repo.GetByUserID(ctx, state.UserID)
	if err != nil {
		return err
	}

	expiresAt := m.nowFunc().Add(ttl)

	if model == nil {
		model = &models.ConversationContext{
			UserID:       state.UserID,
			SessionID:    state.SessionID,
			State:        state.State,
			Intent:       state.Intent,
			Channel:      state.Channel,
			Locale:       state.Locale,
			MessagesJSON: string(messagesData),
			EntitiesJSON: string(entitiesData),
			TTLSeconds:   int64(ttl.Seconds()),
			LastActivity: state.LastActivity,
			CreatedAt:    state.CreatedAt,
			UpdatedAt:    m.nowFunc(),
			ExpiresAt:    &expiresAt,
		}
	} else {
		model.SessionID = state.SessionID
		model.State = state.State
		model.Intent = state.Intent
		model.Channel = state.Channel
		model.Locale = state.Locale
		model.MessagesJSON = string(messagesData)
		model.EntitiesJSON = string(entitiesData)
		model.TTLSeconds = int64(ttl.Seconds())
		model.LastActivity = state.LastActivity
		model.UpdatedAt = m.nowFunc()
		model.ExpiresAt = &expiresAt
	}

	return m.repo.CreateOrUpdate(ctx, model)
}

func convertModelToState(model *models.ConversationContext) (*models.ConversationContextState, error) {
	state := &models.ConversationContextState{
		UserID:       model.UserID,
		SessionID:    model.SessionID,
		State:        model.State,
		Intent:       model.Intent,
		Channel:      model.Channel,
		Locale:       model.Locale,
		Messages:     make([]models.ConversationMessage, 0),
		Entities:     make(map[string]models.ConversationEntity),
		CreatedAt:    model.CreatedAt,
		LastActivity: model.LastActivity,
		TTLSeconds:   model.TTLSeconds,
	}

	if model.MessagesJSON != "" {
		if err := json.Unmarshal([]byte(model.MessagesJSON), &state.Messages); err != nil {
			return nil, fmt.Errorf("unmarshal messages failed: %w", err)
		}
	}

	if model.EntitiesJSON != "" {
		if err := json.Unmarshal([]byte(model.EntitiesJSON), &state.Entities); err != nil {
			return nil, fmt.Errorf("unmarshal entities failed: %w", err)
		}
	}

	return state, nil
}

func isExpired(model *models.ConversationContext, now time.Time) bool {
	if model.ExpiresAt == nil {
		return false
	}
	return model.ExpiresAt.Before(now)
}

// DefaultEntityExtractor 简单实体提取实现
type DefaultEntityExtractor struct{}

// Extract 实体提取逻辑
func (e *DefaultEntityExtractor) Extract(message string, ctx *models.ConversationContextState) map[string]models.ConversationEntity {
	result := make(map[string]models.ConversationEntity)
	lower := strings.ToLower(message)

	if strings.Contains(lower, "明天") {
		result["datetime"] = models.ConversationEntity{
			Name:       "datetime",
			Value:      "tomorrow",
			Confidence: 0.6,
			Source:     "heuristic",
		}
	}

	if strings.Contains(lower, "每天") {
		result["recurrence"] = models.ConversationEntity{
			Name:       "recurrence",
			Value:      "daily",
			Confidence: 0.7,
			Source:     "heuristic",
		}
	}

	if containsReminderKeyword(lower) {
		result["task"] = models.ConversationEntity{
			Name:       "task",
			Value:      strings.TrimSpace(message),
			Confidence: 0.5,
			Source:     "message",
		}
	}

	return result
}

func containsReminderKeyword(lower string) bool {
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "建议") || strings.Contains(lower, "推荐") {
		return false
	}
	keywords := []string{"提醒", "安排", "计划", "打卡", "复盘", "学习", "完成"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// DefaultIntentTracker 简单意图识别实现
type DefaultIntentTracker struct{}

// DetermineIntent 识别意图
func (t *DefaultIntentTracker) DetermineIntent(message string, ctx *models.ConversationContextState) string {
	lower := strings.ToLower(strings.TrimSpace(message))

	switch {
	case lower == "":
		return defaultContextIntent
	case strings.Contains(lower, "提醒我") || (strings.Contains(lower, "提醒") && (strings.Contains(lower, "创建") || strings.Contains(lower, "新增") || strings.Contains(lower, "设置"))):
		return "create_reminder"
	case strings.Contains(lower, "修改") || strings.Contains(lower, "编辑"):
		return "modify_reminder"
	case strings.Contains(lower, "列表") || strings.Contains(lower, "有哪些"):
		return "list_reminder"
	case strings.Contains(lower, "还有建议") || strings.Contains(lower, "推荐") || strings.Contains(lower, "建议"):
		return "suggestion_request"
	default:
		if ctx != nil && ctx.Intent != "" {
			return ctx.Intent
		}
		return defaultContextIntent
	}
}
