package service

import (
	"context"
	"testing"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockActivityRepository Mock活动仓储
type MockActivityRepository struct {
	mock.Mock
}

func (m *MockActivityRepository) Create(ctx context.Context, activity *models.DailyActivity) error {
	args := m.Called(ctx, activity)
	return args.Error(0)
}

func (m *MockActivityRepository) GetByID(ctx context.Context, id uint) (*models.DailyActivity, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DailyActivity), args.Error(1)
}

func (m *MockActivityRepository) GetByUserID(ctx context.Context, userID uint, limit, offset int) ([]*models.DailyActivity, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DailyActivity), args.Error(1)
}

func (m *MockActivityRepository) GetByType(ctx context.Context, userID uint, activityType models.ActivityType, limit, offset int) ([]*models.DailyActivity, error) {
	args := m.Called(ctx, userID, activityType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DailyActivity), args.Error(1)
}

func (m *MockActivityRepository) GetByDateRange(ctx context.Context, userID uint, startTime, endTime time.Time) ([]*models.DailyActivity, error) {
	args := m.Called(ctx, userID, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DailyActivity), args.Error(1)
}

func (m *MockActivityRepository) GetRecentActivities(ctx context.Context, userID uint, limit int) ([]*models.DailyActivity, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DailyActivity), args.Error(1)
}

func (m *MockActivityRepository) Update(ctx context.Context, activity *models.DailyActivity) error {
	args := m.Called(ctx, activity)
	return args.Error(0)
}

func (m *MockActivityRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockActivityRepository) GetStatistics(ctx context.Context, userID uint, startTime, endTime time.Time) (map[string]int64, error) {
	args := m.Called(ctx, userID, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int64), args.Error(1)
}

// TestNewActivityAnalysisService 测试创建智能分析服务
func TestNewActivityAnalysisService(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	assert.NotNil(t, service)
	assert.IsType(t, &ActivityAnalysisServiceImpl{}, service)
}

// TestActivityAnalysisService_DetectPatterns_Success 测试成功检测模式
func TestActivityAnalysisService_DetectPatterns_Success(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	// 准备测试数据
	now := time.Now()
	activities := []*models.DailyActivity{
		{
			ID:           1,
			UserID:       1,
			ActivityType: models.ActivityTypeExercise,
			OccurredAt:   now.Add(-24 * time.Hour),
			Source:       models.SourceManual,
		},
		{
			ID:           2,
			UserID:       1,
			ActivityType: models.ActivityTypeExercise,
			OccurredAt:   now.Add(-48 * time.Hour),
			Source:       models.SourceManual,
		},
	}

	mockRepo.On("GetByType", mock.Anything, uint(1), models.ActivityTypeExercise, 60, 0).Return(activities, nil)

	ctx := context.Background()
	patterns, err := service.DetectPatterns(ctx, 1, "exercise", 30)

	assert.NoError(t, err)
	assert.NotNil(t, patterns)
	assert.Greater(t, len(patterns), 0)
	mockRepo.AssertExpectations(t)
}

// TestActivityAnalysisService_DetectPatterns_EmptyActivities 测试空活动列表
func TestActivityAnalysisService_DetectPatterns_EmptyActivities(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	mockRepo.On("GetByType", mock.Anything, uint(1), models.ActivityTypeExercise, 60, 0).Return([]*models.DailyActivity{}, nil)

	ctx := context.Background()
	_, err := service.DetectPatterns(ctx, 1, "exercise", 30)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestActivityAnalysisService_DetectPatterns_InvalidDays 测试无效天数
func TestActivityAnalysisService_DetectPatterns_InvalidDays(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	now := time.Now()
	activities := []*models.DailyActivity{
		{
			ID:           1,
			UserID:       1,
			ActivityType: models.ActivityTypeExercise,
			OccurredAt:   now,
			Source:       models.SourceManual,
		},
	}

	mockRepo.On("GetByType", mock.Anything, uint(1), models.ActivityTypeExercise, 60, 0).Return(activities, nil)

	ctx := context.Background()
	patterns, err := service.DetectPatterns(ctx, 1, "exercise", -1) // 无效天数

	assert.NoError(t, err)
	assert.NotNil(t, patterns)
	mockRepo.AssertExpectations(t)
}

// TestActivityAnalysisService_DetectPatterns_RepositoryError 测试仓储错误
func TestActivityAnalysisService_DetectPatterns_RepositoryError(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	mockRepo.On("GetByType", mock.Anything, uint(1), models.ActivityTypeExercise, 60, 0).Return(nil, assert.AnError)

	ctx := context.Background()
	patterns, err := service.DetectPatterns(ctx, 1, "exercise", 30)

	assert.Error(t, err)
	assert.Nil(t, patterns)
	mockRepo.AssertExpectations(t)
}

// TestActivityAnalysisService_AnalyzeTimeDistribution 测试时间分布分析
func TestActivityAnalysisService_AnalyzeTimeDistribution(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	now := time.Now()
	activities := []*models.DailyActivity{
		{
			ID:          1,
			UserID:      1,
			ActivityType: models.ActivityTypeExercise,
			OccurredAt:  time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location()),
		},
		{
			ID:          2,
			UserID:      1,
			ActivityType: models.ActivityTypeExercise,
			OccurredAt:  time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location()),
		},
	}

	mockRepo.On("GetByType", mock.Anything, uint(1), models.ActivityTypeExercise, 60, 0).Return(activities, nil)

	ctx := context.Background()
	result, err := service.AnalyzeTimeDistribution(ctx, 1, "exercise", 30)

	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	mockRepo.AssertExpectations(t)
}

// TestActivityAnalysisService_DetectAnomalies 测试异常检测
func TestActivityAnalysisService_DetectAnomalies(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	now := time.Now()
	activities := []*models.DailyActivity{
		{
			ID:           1,
			UserID:       1,
			ActivityType: models.ActivityTypeExercise,
			OccurredAt:   now.Add(-24 * time.Hour),
			Source:       models.SourceManual,
		},
	}

	mockRepo.On("GetByType", mock.Anything, uint(1), models.ActivityTypeExercise, 60, 0).Return(activities, nil)

	ctx := context.Background()
	_, err := service.DetectAnomalies(ctx, 1, "exercise", 30)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestActivityAnalysisService_GenerateSuggestions 测试生成建议
func TestActivityAnalysisService_GenerateSuggestions(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	// Mock GetByDateRange for all activity types
	mockRepo.On("GetByDateRange", mock.Anything, uint(1), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*models.DailyActivity{}, nil)

	ctx := context.Background()
	suggestions, err := service.GenerateSuggestions(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, suggestions)
}

// TestActivityAnalysisService_AnalyzeHabitFormation 测试习惯形成分析
func TestActivityAnalysisService_AnalyzeHabitFormation(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	now := time.Now()
	activities := []*models.DailyActivity{
		{
			ID:          1,
			UserID:      1,
			ActivityType: models.ActivityTypeExercise,
			OccurredAt:  now.Add(-24 * time.Hour),
		},
	}

	mockRepo.On("GetByType", mock.Anything, uint(1), models.ActivityTypeExercise, 60, 0).Return(activities, nil)

	ctx := context.Background()
	report, err := service.AnalyzeHabitFormation(ctx, 1, "exercise")

	assert.NoError(t, err)
	assert.NotNil(t, report)
	mockRepo.AssertExpectations(t)
}

// TestActivityAnalysisService_GetActivityInsights 测试获取活动洞察
func TestActivityAnalysisService_GetActivityInsights(t *testing.T) {
	mockRepo := new(MockActivityRepository)
	service := NewActivityAnalysisService(mockRepo)

	// Mock GetByDateRange for all activity types
	mockRepo.On("GetByDateRange", mock.Anything, uint(1), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*models.DailyActivity{}, nil)

	ctx := context.Background()
	insights, err := service.GetActivityInsights(ctx, 1, 30)

	assert.NoError(t, err)
	assert.NotNil(t, insights)
}

// 初始化日志
func init() {
	logger.Init("info", "text", "stdout", "")
}