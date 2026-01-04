package service

import (
	"mmemory/internal/models"
	"mmemory/pkg/logger"
)

type contextTokenManagerService struct {
	tokenEstimator     *TokenEstimator
	topicDetector      *TopicSwitchDetector
	archiveService     ConversationArchiveService
	warningThreshold   float64 // 80% 警告阈值
	hardLimitThreshold float64 // 95% 硬限制阈值
	maxTokens          int      // 最大token数
}

// NewContextTokenManagerService 创建Token管理器
func NewContextTokenManagerService(
	archiveService ConversationArchiveService,
	maxTokens int,
) ContextTokenManagerService {
	if maxTokens <= 0 {
		maxTokens = 128000 // 默认128k (GLM-4)
	}

	return &contextTokenManagerService{
		tokenEstimator:     NewTokenEstimator(),
		topicDetector:      NewTopicSwitchDetector(30, 3), // 30分钟, 3次意图变化
		archiveService:     archiveService,
		warningThreshold:   0.80,
		hardLimitThreshold: 0.95,
		maxTokens:          maxTokens,
	}
}

// NeedsPruning 检查是否需要清理
func (s *contextTokenManagerService) NeedsPruning(messages []models.ConversationMessage) (bool, float64) {
	if len(messages) == 0 {
		return false, 0.0
	}

	tokenCount := s.tokenEstimator.EstimateMessagesToken(messages)
	usageRatio := s.tokenEstimator.EstimateUsageRatio(messages, s.maxTokens)

	needsPruning := usageRatio >= s.warningThreshold

	if needsPruning {
		logger.Infof("Token使用率: %.2f%% (%d/%d), 需要清理",
			usageRatio*100, tokenCount, s.maxTokens)
	}

	return needsPruning, usageRatio
}

// PruneMessages 清理消息(状态无关)
// 返回: (保留的消息, 需要归档的消息, 使用的策略)
func (s *contextTokenManagerService) PruneMessages(messages []models.ConversationMessage) ([]models.ConversationMessage, []models.ConversationMessage, CleanupStrategy) {
	if len(messages) == 0 {
		return messages, nil, StrategyNone
	}

	_, usageRatio := s.NeedsPruning(messages)

	// 根据使用率选择策略
	var strategy CleanupStrategy
	var toKeep []models.ConversationMessage
	var toArchive []models.ConversationMessage

	if usageRatio >= s.hardLimitThreshold {
		// 强制清理: >95%, 保留最近8条, 其余归档
		strategy = StrategyForceClean
		toKeep, toArchive = s.forceCleanMessages(messages, 8)
		logger.Warnf("强制清理: 保留%d条, 归档%d条", len(toKeep), len(toArchive))
	} else if usageRatio >= s.warningThreshold {
		// 智能清理: 80-95%, 删除不重要消息
		strategy = StrategySmartClean
		toKeep, toArchive = s.smartCleanMessages(messages)
		logger.Infof("智能清理: 保留%d条, 归档%d条", len(toKeep), len(toArchive))
	} else {
		// 无需清理
		strategy = StrategyNone
		toKeep = messages
		toArchive = nil
	}

	return toKeep, toArchive, strategy
}

// smartCleanMessages 智能清理: 删除不重要的消息
// 策略: 删除已确认的提醒相关消息
func (s *contextTokenManagerService) smartCleanMessages(messages []models.ConversationMessage) ([]models.ConversationMessage, []models.ConversationMessage) {
	if len(messages) <= 20 {
		// 消息少于20条, 不需要清理
		return messages, nil
	}

	// 检测话题切换点
	hasTopicSwitch, _ := s.topicDetector.DetectSwitch(messages)

	// 如果没有话题切换,删除已确认的提醒消息
	if !hasTopicSwitch {
		return s.removeConfirmedReminders(messages)
	}

	// 如果有话题切换,保留最近的消息
	return s.keepRecentMessages(messages, 20)
}

// forceCleanMessages 强制清理: 保留最近N条, 其余归档
func (s *contextTokenManagerService) forceCleanMessages(messages []models.ConversationMessage, keepCount int) ([]models.ConversationMessage, []models.ConversationMessage) {
	if len(messages) <= keepCount {
		return messages, nil
	}

	// 保留最近的消息
	splitIndex := len(messages) - keepCount
	return messages[splitIndex:], messages[:splitIndex]
}

// removeConfirmedRemovers 删除已确认的提醒相关消息
func (s *contextTokenManagerService) removeConfirmedReminders(messages []models.ConversationMessage) ([]models.ConversationMessage, []models.ConversationMessage) {
	var toKeep []models.ConversationMessage
	var toArchive []models.ConversationMessage

	for _, msg := range messages {
		// 保留所有非提醒相关的消息
		if msg.Intent == "" || msg.Intent != "reminder" {
			toKeep = append(toKeep, msg)
		} else if msg.Content != "" {
			// 如果消息有内容,保留
			toKeep = append(toKeep, msg)
		} else {
			// 已确认的提醒消息,归档
			toArchive = append(toArchive, msg)
		}
	}

	// 如果没有需要归档的消息,全部保留
	if len(toArchive) == 0 {
		return messages, nil
	}

	return toKeep, toArchive
}

// keepRecentMessages 保留最近的消息
func (s *contextTokenManagerService) keepRecentMessages(messages []models.ConversationMessage, keepCount int) ([]models.ConversationMessage, []models.ConversationMessage) {
	if len(messages) <= keepCount {
		return messages, nil
	}

	splitIndex := len(messages) - keepCount
	return messages[splitIndex:], messages[:splitIndex]
}
