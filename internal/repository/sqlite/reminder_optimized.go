package sqlite

import (
	"context"
	"time"
	"fmt"

	"gorm.io/gorm"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
)

// OptimizedReminderRepository 优化的提醒仓储
type OptimizedReminderRepository struct {
	db *gorm.DB
}

// NewOptimizedReminderRepository 创建优化的提醒仓储
func NewOptimizedReminderRepository(db *gorm.DB) interfaces.ReminderRepository {
	return &OptimizedReminderRepository{db: db}
}

// Create 创建提醒（优化版）
func (r *OptimizedReminderRepository) Create(ctx context.Context, reminder *models.Reminder) error {
	// 使用事务确保数据一致性
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 验证数据
		if err := r.validateReminder(reminder); err != nil {
			return err
		}

		// 设置默认值
		if reminder.Timezone == "" {
			reminder.Timezone = "Asia/Shanghai"
		}
		if reminder.Type == "" {
			reminder.Type = models.ReminderTypeTask
		}

		// 创建记录
		if err := tx.Create(reminder).Error; err != nil {
			// 避免日志nil指针，使用fmt打印
			fmt.Printf("创建提醒失败: %v\n", err)
			return fmt.Errorf("创建提醒失败: %w", err)
		}

		// 避免日志nil指针，使用fmt打印
		fmt.Printf("✅ 提醒创建成功: ID=%d, Title=%s\n", reminder.ID, reminder.Title)
		return nil
	})
}

// GetByID 根据ID获取提醒（优化版）
func (r *OptimizedReminderRepository) GetByID(ctx context.Context, id uint) (*models.Reminder, error) {
	if id == 0 {
		return nil, fmt.Errorf("提醒ID不能为空")
	}

	var reminder models.Reminder
	// 使用预加载避免N+1查询问题
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("ReminderLogs", "status IN ?", []models.ReminderStatus{
			models.ReminderStatusPending,
			models.ReminderStatusSent,
		}).
		First(&reminder, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		fmt.Errorf("获取提醒失败 (ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取提醒失败: %w", err)
	}

	return &reminder, nil
}

// GetByUserID 根据用户ID获取提醒（优化版）
func (r *OptimizedReminderRepository) GetByUserID(ctx context.Context, userID uint) ([]*models.Reminder, error) {
	if userID == 0 {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	var reminders []*models.Reminder
	// 使用索引字段查询，只选择需要的字段
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").
		Find(&reminders).Error

	if err != nil {
		fmt.Errorf("获取用户提醒失败 (UserID: %d): %v", userID, err)
		return nil, fmt.Errorf("获取用户提醒失败: %w", err)
	}

	return reminders, nil
}

// GetActiveReminders 获取所有活跃的提醒（优化版）
func (r *OptimizedReminderRepository) GetActiveReminders(ctx context.Context) ([]*models.Reminder, error) {
	var reminders []*models.Reminder
	
	// 使用索引字段查询，避免全表扫描
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("user_id, target_time").
		Find(&reminders).Error

	if err != nil {
		fmt.Errorf("获取活跃提醒失败: %v", err)
		return nil, fmt.Errorf("获取活跃提醒失败: %w", err)
	}

	return reminders, nil
}

// Update 更新提醒（优化版）
func (r *OptimizedReminderRepository) Update(ctx context.Context, reminder *models.Reminder) error {
	if reminder.ID == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 验证数据
		if err := r.validateReminder(reminder); err != nil {
			return err
		}

		// 更新记录
	result := tx.Model(&models.Reminder{}).Where("id = ?", reminder.ID).Updates(map[string]interface{}{
		"title":            reminder.Title,
		"description":      reminder.Description,
		"type":             reminder.Type,
		"schedule_pattern": reminder.SchedulePattern,
		"target_time":      reminder.TargetTime,
		"timezone":         reminder.Timezone,
		"is_active":        reminder.IsActive,
		"updated_at":       time.Now(),
	})
		if result.Error != nil {
			fmt.Errorf("更新提醒失败 (ID: %d): %v", reminder.ID, result.Error)
			return fmt.Errorf("更新提醒失败: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("提醒不存在 (ID: %d)", reminder.ID)
		}

		fmt.Printf("✅ 提醒更新成功: ID=%d, Title=%s", reminder.ID, reminder.Title)
		return nil
	})
}

// Delete 删除提醒（优化版）
func (r *OptimizedReminderRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 首先删除相关的提醒记录
		if err := tx.Where("reminder_id = ?", id).Delete(&models.ReminderLog{}).Error; err != nil {
			fmt.Errorf("删除提醒记录失败 (ReminderID: %d): %v", id, err)
			return fmt.Errorf("删除提醒记录失败: %w", err)
		}

		// 然后删除提醒本身
		result := tx.Delete(&models.Reminder{}, id)
		if result.Error != nil {
			fmt.Errorf("删除提醒失败 (ID: %d): %v", id, result.Error)
			return fmt.Errorf("删除提醒失败: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("提醒不存在 (ID: %d)", id)
		}

		fmt.Printf("✅ 提醒删除成功: ID=%d", id)
		return nil
	})
}

// CountByStatus 按状态统计提醒数量
func (r *OptimizedReminderRepository) CountByStatus(ctx context.Context, status models.ReminderStatStatus) (int64, error) {
	var count int64
	
	switch status {
	case models.ReminderStatStatusActive:
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).Where("is_active = ?", true).Count(&count).Error
		return count, err
	case models.ReminderStatStatusCompleted:
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).Where("is_active = ?", false).Count(&count).Error
		return count, err
	case models.ReminderStatStatusExpired:
		// 这里需要根据业务逻辑定义过期的条件
		err := r.db.WithContext(ctx).Model(&models.Reminder{}).
			Where("is_active = ? AND schedule_pattern = ?", true, string(models.SchedulePatternOnce)).
			Count(&count).Error
		return count, err
	default:
		return 0, nil
	}
}

// validateReminder 验证提醒数据
func (r *OptimizedReminderRepository) validateReminder(reminder *models.Reminder) error {
	if reminder.UserID == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	if reminder.Title == "" {
		return fmt.Errorf("提醒标题不能为空")
	}
	if reminder.TargetTime == "" {
		return fmt.Errorf("目标时间不能为空")
	}
	if reminder.SchedulePattern == "" {
		return fmt.Errorf("调度模式不能为空")
	}

	// 验证时间格式
	if !r.isValidTimeFormat(reminder.TargetTime) {
		return fmt.Errorf("无效的时间格式，应为 HH:MM:SS 格式")
	}

	// 验证调度模式
	if !r.isValidSchedulePattern(reminder.SchedulePattern) {
		return fmt.Errorf("无效的调度模式")
	}

	return nil
}

// isValidTimeFormat 验证时间格式
func (r *OptimizedReminderRepository) isValidTimeFormat(timeStr string) bool {
	// 更严格的验证，检查小时、分钟、秒的范围
	if len(timeStr) != 8 || timeStr[2] != ':' || timeStr[5] != ':' {
		return false
	}
	
	// 解析各部分
	hour := timeStr[0:2]
	minute := timeStr[3:5]
	second := timeStr[6:8]
	
	// 验证数字范围和格式
	var h, m, s int
	if _, err := fmt.Sscanf(hour, "%d", &h); err != nil || h < 0 || h > 23 {
		return false
	}
	if _, err := fmt.Sscanf(minute, "%d", &m); err != nil || m < 0 || m > 59 {
		return false
	}
	if _, err := fmt.Sscanf(second, "%d", &s); err != nil || s < 0 || s > 59 {
		return false
	}
	
	return true
}

// isValidSchedulePattern 验证调度模式
func (r *OptimizedReminderRepository) isValidSchedulePattern(pattern string) bool {
	// 支持的模式：daily, weekly:1,3,5, monthly:1,15, once:2024-01-01
	if pattern == "daily" {
		return true
	}
	if len(pattern) > 7 && pattern[:7] == "weekly:" {
		return true
	}
	if len(pattern) > 8 && pattern[:8] == "monthly:" {
		return true
	}
	if len(pattern) > 5 && pattern[:5] == "once:" {
		return true
	}
	return false
}

// GetBySchedulePattern 根据调度模式获取提醒（新增方法）
func (r *OptimizedReminderRepository) GetBySchedulePattern(ctx context.Context, pattern string) ([]*models.Reminder, error) {
	var reminders []*models.Reminder
	
	err := r.db.WithContext(ctx).
		Where("schedule_pattern = ? AND is_active = ?", pattern, true).
		Order("target_time").
		Find(&reminders).Error

	if err != nil {
		fmt.Errorf("获取调度模式提醒失败 (Pattern: %s): %v", pattern, err)
		return nil, fmt.Errorf("获取调度模式提醒失败: %w", err)
	}

	return reminders, nil
}

// GetByTimeRange 根据时间范围获取提醒（新增方法）
func (r *OptimizedReminderRepository) GetByTimeRange(ctx context.Context, startTime, endTime string) ([]*models.Reminder, error) {
	var reminders []*models.Reminder
	
	err := r.db.WithContext(ctx).
		Where("target_time >= ? AND target_time <= ? AND is_active = ?", startTime, endTime, true).
		Order("target_time").
		Find(&reminders).Error

	if err != nil {
		fmt.Errorf("获取时间范围提醒失败 (%s - %s): %v", startTime, endTime, err)
		return nil, fmt.Errorf("获取时间范围提醒失败: %w", err)
	}

	return reminders, nil
}

// CountByUserID 统计用户的提醒数量（新增方法）
func (r *OptimizedReminderRepository) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	if userID == 0 {
		return 0, fmt.Errorf("用户ID不能为空")
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Reminder{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	if err != nil {
		fmt.Errorf("统计用户提醒数量失败 (UserID: %d): %v", userID, err)
		return 0, fmt.Errorf("统计用户提醒数量失败: %w", err)
	}

	return count, nil
}

// BatchUpdateStatus 批量更新提醒状态（新增方法）
func (r *OptimizedReminderRepository) BatchUpdateStatus(ctx context.Context, reminderIDs []uint, isActive bool) error {
	if len(reminderIDs) == 0 {
		return fmt.Errorf("提醒ID列表不能为空")
	}

	result := r.db.WithContext(ctx).
		Model(&models.Reminder{}).
		Where("id IN ?", reminderIDs).
		Update("is_active", isActive)

	if result.Error != nil {
		fmt.Errorf("批量更新提醒状态失败: %v", result.Error)
		return fmt.Errorf("批量更新提醒状态失败: %w", result.Error)
	}

	fmt.Printf("✅ 批量更新提醒状态成功: %d 条记录", result.RowsAffected)
	return nil
}