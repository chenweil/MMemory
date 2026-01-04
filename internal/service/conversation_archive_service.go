package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
	"mmemory/pkg/metrics"
)

type conversationArchiveService struct {
	archiveRepo interfaces.ConversationArchiveRepository
	aiClient    ai.AIClient
	config      *ai.AIConfig

	// 用户级锁,防止并发归档同一用户的数据
	userLocks map[uint]*sync.Mutex
	locksLock sync.RWMutex
}

// NewConversationArchiveService 创建存档服务
func NewConversationArchiveService(
	archiveRepo interfaces.ConversationArchiveRepository,
	aiClient ai.AIClient,
	config *ai.AIConfig,
) ConversationArchiveService {
	return &conversationArchiveService{
		archiveRepo: archiveRepo,
		aiClient:    aiClient,
		config:      config,
		userLocks:   make(map[uint]*sync.Mutex),
	}
}

// getUserLock 获取用户级锁
func (s *conversationArchiveService) getUserLock(userID uint) *sync.Mutex {
	s.locksLock.Lock()
	defer s.locksLock.Unlock()

	if _, exists := s.userLocks[userID]; !exists {
		s.userLocks[userID] = &sync.Mutex{}
	}

	return s.userLocks[userID]
}

// CreateArchive 创建存档(同步)
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
		UserID:       userID,
		ArchiveType:  archiveType,
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

		// 默认TTL: 30天
		archive.SetExpiry(720 * time.Hour)
	} else if archiveType == models.ArchiveTypeSummary {
		// 生成摘要(默认启用AI摘要)
		if s.config != nil && s.config.Enabled {
			summary, err := s.GenerateSummary(ctx, messages)
			if err != nil {
				// 降级策略1: 摘要失败,使用完整内容
				logger.Warnf("AI生成摘要失败,降级为完整内容: %v", err)
				archiveType = models.ArchiveTypeFull
				archive.Content = s.formatMessages(messages)
				archive.ArchiveType = archiveType
				archive.SetExpiry(720 * time.Hour) // 30天
			} else {
				archive.Summary = summary
				archive.SetExpiry(0) // 摘要永不过期
			}
		} else {
			// AI未启用，降级为完整内容
			archiveType = models.ArchiveTypeFull
			archive.Content = s.formatMessages(messages)
			archive.ArchiveType = archiveType
			archive.SetExpiry(720 * time.Hour)
		}
	}

	// 提取关键实体
	entities, err := s.ExtractKeyEntities(messages)
	if err != nil {
		logger.Warnf("提取关键实体失败: %v", err)
	} else {
		archive.SetKeyEntities(entities)
	}

	// 保存到数据库
	if err := s.archiveRepo.Create(ctx, archive); err != nil {
		return nil, fmt.Errorf("failed to save archive: %w", err)
	}

	return archive, nil
}

// CreateArchiveAsync 异步创建存档(fire-and-forget,带降级策略)
func (s *conversationArchiveService) CreateArchiveAsync(
	userID uint,
	messages []models.ConversationMessage,
	archiveType models.ArchiveType,
) {
	// 在后台goroutine中执行
	go func() {
		// 获取用户级锁,防止并发归档
		lock := s.getUserLock(userID)
		lock.Lock()
		defer lock.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logger.Infof("开始异步存档: userID=%d, messages=%d, type=%s", userID, len(messages), archiveType)

		startTime := time.Now()

		archive, err := s.CreateArchive(ctx, userID, messages, archiveType)
		if err != nil {
			// 降级策略2: 完全失败,至少记录日志
			logger.Errorf("异步存档失败: userID=%d, messages=%d, error=%v", userID, len(messages), err)

			// 记录失败指标
			userStr := fmt.Sprintf("%d", userID)
			metrics.ContextArchiveFailureTotal.WithLabelValues(userStr, archiveType.String()).Inc()

			// TODO: 可以将失败的消息存到文件或队列,稍后重试
			return
		}

		duration := time.Since(startTime).Seconds()
		logger.Infof("异步存档成功: userID=%d, archiveID=%d, duration=%.2fs", userID, archive.ID, duration)

		// 记录成功指标
		metrics.ContextArchiveDuration.WithLabelValues(archiveType.String()).Observe(duration)
	}()
}

// GenerateSummary 生成摘要
func (s *conversationArchiveService) GenerateSummary(ctx context.Context, messages []models.ConversationMessage) (string, error) {
	if s.config == nil || !s.config.Enabled {
		return "", fmt.Errorf("AI service not enabled")
	}

	// 格式化对话
	conversation := s.formatMessages(messages)

	// 构建prompt (默认最大200字摘要)
	prompt := fmt.Sprintf(`请将以下对话压缩成简短摘要(最多200字),保留关键信息(书名、人名、主要观点):

对话内容:
%s

请直接返回摘要内容,不要添加其他说明。`, conversation)

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
			// 类型断言：entity.Value是interface{}类型，需要断言为string
			valueStr, ok := entity.Value.(string)
			if !ok {
				continue // 跳过非字符串值
			}

			value := strings.ToLower(valueStr)

			switch key {
			case "book", "book_name":
				if entities.BookName == "" {
					entities.BookName = valueStr // 使用原始值（非小写）
				}
			case "topic":
				if entities.Topic == "" {
					entities.Topic = valueStr
				}
			}

			// 收集关键词
			if !stringSliceContains(entities.Keywords, value) {
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
		limit = 20
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

// stringSliceContains 检查字符串是否在切片中
func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
