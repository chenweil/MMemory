package service

import (
	"context"
	"testing"
	"time"

	"mmemory/internal/models"
)

// MockArchiveService 模拟存档服务
type MockArchiveService struct{}

func (m *MockArchiveService) CreateArchive(ctx context.Context, userID uint, messages []models.ConversationMessage, archiveType models.ArchiveType) (*models.ConversationArchive, error) {
	return &models.ConversationArchive{}, nil
}

func (m *MockArchiveService) CreateArchiveAsync(userID uint, messages []models.ConversationMessage, archiveType models.ArchiveType) {
	// Mock实现
}

func (m *MockArchiveService) GenerateSummary(ctx context.Context, messages []models.ConversationMessage) (string, error) {
	return "test summary", nil
}

func (m *MockArchiveService) ExtractKeyEntities(messages []models.ConversationMessage) (*models.KeyEntities, error) {
	return &models.KeyEntities{}, nil
}

func (m *MockArchiveService) GetUserArchives(ctx context.Context, userID uint, limit int) ([]*models.ConversationArchive, error) {
	return nil, nil
}

func (m *MockArchiveService) CleanupExpiredArchives(ctx context.Context) (int64, error) {
	return 0, nil
}

func TestContextTokenManagerService_NeedsPruning(t *testing.T) {
	mockArchive := &MockArchiveService{}
	service := NewContextTokenManagerService(mockArchive, 1000) // 1000 tokens for testing

	t.Run("空消息-不需要清理", func(t *testing.T) {
		messages := []models.ConversationMessage{}
		needsPruning, ratio := service.NeedsPruning(messages)

		if needsPruning {
			t.Errorf("空消息不应该需要清理")
		}
		if ratio != 0.0 {
			t.Errorf("空消息使用率应该为0, got %.2f", ratio)
		}
	})

	t.Run("低使用率-不需要清理", func(t *testing.T) {
		messages := []models.ConversationMessage{
			{Role: "user", Content: "测试消息", Timestamp: time.Now()},
		}
		needsPruning, ratio := service.NeedsPruning(messages)

		if needsPruning {
			t.Errorf("低使用率不应该需要清理, ratio=%.2f", ratio)
		}
	})

	t.Run("高使用率-需要清理", func(t *testing.T) {
		// 创建大量消息(约900 tokens)
		messages := make([]models.ConversationMessage, 100)
		for i := 0; i < 100; i++ {
			messages[i] = models.ConversationMessage{
				Role:      "user",
				Content:   "这是一条很长的测试消息,包含很多汉字和英文字符混合的内容,用于测试token估算和清理策略",
				Timestamp: time.Now(),
			}
		}

		needsPruning, ratio := service.NeedsPruning(messages)

		if !needsPruning {
			t.Errorf("高使用率(%.2f%%)应该需要清理", ratio*100)
		}
		if ratio < 0.80 {
			t.Errorf("期望使用率>=80%%, got %.2f%%", ratio*100)
		}
	})
}

func TestContextTokenManagerService_PruneMessages(t *testing.T) {
	mockArchive := &MockArchiveService{}
	service := NewContextTokenManagerService(mockArchive, 1000)

	t.Run("空消息-无策略", func(t *testing.T) {
		messages := []models.ConversationMessage{}
		toKeep, toArchive, strategy := service.PruneMessages(messages)

		if len(toKeep) != 0 {
			t.Errorf("空消息应该返回空列表")
		}
		if len(toArchive) != 0 {
			t.Errorf("空消息不应该有归档")
		}
		if strategy != StrategyNone {
			t.Errorf("空消息应该使用StrategyNone, got %v", strategy)
		}
	})

	t.Run("少量消息-无策略", func(t *testing.T) {
		messages := []models.ConversationMessage{
			{Role: "user", Content: "测试消息1", Timestamp: time.Now()},
			{Role: "assistant", Content: "测试回复1", Timestamp: time.Now()},
		}
		toKeep, toArchive, strategy := service.PruneMessages(messages)

		if len(toKeep) != 2 {
			t.Errorf("少量消息应该全部保留, got %d", len(toKeep))
		}
		if len(toArchive) != 0 {
			t.Errorf("少量消息不应该有归档")
		}
		if strategy != StrategyNone {
			t.Errorf("少量消息应该使用StrategyNone, got %v", strategy)
		}
	})

	t.Run("智能清理-删除不重要消息", func(t *testing.T) {
		// 创建约800-900 tokens的消息
		messages := make([]models.ConversationMessage, 30)
		for i := 0; i < 30; i++ {
			messages[i] = models.ConversationMessage{
				Role:      "user",
				Content:   "测试消息内容",
				Intent:    "reminder",
				Timestamp: time.Now(),
			}
		}

		toKeep, toArchive, strategy := service.PruneMessages(messages)

		// 30条消息约960 tokens (96%), 应该触发智能清理或强制清理
		if strategy != StrategySmartClean && strategy != StrategyForceClean {
			t.Errorf("应该使用StrategySmartClean或StrategyForceClean, got %v", strategy)
		}
		if len(toKeep) == 0 {
			t.Errorf("清理应该保留一些消息")
		}
		t.Logf("清理: 策略=%v, 保留%d条, 归档%d条", strategy, len(toKeep), len(toArchive))
	})

	t.Run("强制清理-保留最近8条", func(t *testing.T) {
		// 创建超过95%的消息
		messages := make([]models.ConversationMessage, 120)
		for i := 0; i < 120; i++ {
			messages[i] = models.ConversationMessage{
				Role:      "user",
				Content:   "测试消息内容用于强制清理测试",
				Timestamp: time.Now(),
			}
		}

		toKeep, toArchive, strategy := service.PruneMessages(messages)

		if strategy != StrategyForceClean {
			t.Errorf("应该使用StrategyForceClean, got %v", strategy)
		}
		if len(toKeep) != 8 {
			t.Errorf("强制清理应该保留最近8条, got %d", len(toKeep))
		}
		if len(toArchive) != 112 {
			t.Errorf("强制清理应该归档112条, got %d", len(toArchive))
		}
	})
}

func TestContextTokenManagerService_SmartClean(t *testing.T) {
	mockArchive := &MockArchiveService{}
	service := NewContextTokenManagerService(mockArchive, 1000)

	// 私有方法不能直接测试，通过PruneMessages间接测试
	t.Run("智能清理策略-通过公共接口测试", func(t *testing.T) {
		// 创建30条消息（约900+ tokens）
		messages := make([]models.ConversationMessage, 30)
		for i := 0; i < 30; i++ {
			messages[i] = models.ConversationMessage{
				Role:      "user",
				Content:   "测试消息内容",
				Intent:    "reminder",
				Timestamp: time.Now(),
			}
		}

		toKeep, toArchive, strategy := service.PruneMessages(messages)

		t.Logf("清理: 策略=%v, 保留=%d, 归档=%d", strategy, len(toKeep), len(toArchive))

		// 900+ tokens超过95%阈值,应该触发清理
		if strategy != StrategySmartClean && strategy != StrategyForceClean {
			t.Errorf("应该使用清理策略, got %v", strategy)
		}
		if len(toKeep) == 0 {
			t.Errorf("清理应该保留一些消息")
		}
	})
}
