package service

import (
	"context"
	"fmt"
	"time"

	"mmemory/internal/models"
	"mmemory/internal/repository/interfaces"
	"mmemory/pkg/logger"
)

type reminderService struct {
	reminderRepo interfaces.ReminderRepository
	parser       *parserService
	scheduler    SchedulerService
}

func NewReminderService(reminderRepo interfaces.ReminderRepository) ReminderService {
	return &reminderService{
		reminderRepo: reminderRepo,
		parser:       NewParserService(),
	}
}

// SetScheduler 设置调度器 (用于避免循环依赖)
func (s *reminderService) SetScheduler(scheduler SchedulerService) {
	s.scheduler = scheduler
}

func (s *reminderService) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	if reminder.UserID == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	if reminder.Title == "" {
		return fmt.Errorf("提醒标题不能为空")
	}
	if reminder.TargetTime == "" {
		return fmt.Errorf("提醒时间不能为空")
	}

	// 保存到数据库
	if err := s.reminderRepo.Create(ctx, reminder); err != nil {
		return err
	}

	// 添加到调度器
	if s.scheduler != nil && reminder.IsActive {
		if err := s.scheduler.AddReminder(reminder); err != nil {
			// 调度失败不影响数据库保存，只记录错误
			fmt.Printf("添加调度失败: %v", err)
		}
	}

	return nil
}

func (s *reminderService) ParseReminderFromText(ctx context.Context, text string, userID uint) (*models.Reminder, error) {
	return s.parser.ParseReminderFromText(ctx, text, userID)
}

func (s *reminderService) GetUserReminders(ctx context.Context, userID uint) ([]*models.Reminder, error) {
	return s.reminderRepo.GetByUserID(ctx, userID)
}

func (s *reminderService) GetReminderByID(ctx context.Context, id uint) (*models.Reminder, error) {
	if id == 0 {
		return nil, fmt.Errorf("提醒ID不能为空")
	}

	return s.reminderRepo.GetByID(ctx, id)
}

func (s *reminderService) UpdateReminder(ctx context.Context, reminder *models.Reminder) error {
	if reminder.ID == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	// 更新数据库
	if err := s.reminderRepo.Update(ctx, reminder); err != nil {
		return err
	}

	// 更新调度器
	if s.scheduler != nil {
		// 先移除旧的调度
		s.scheduler.RemoveReminder(reminder.ID)

		// 如果仍然活跃，添加新的调度
		if reminder.IsActive {
			if err := s.scheduler.AddReminder(reminder); err != nil {
				fmt.Printf("更新调度失败: %v", err)
			}
		}
	}

	return nil
}

func (s *reminderService) DeleteReminder(ctx context.Context, id uint) error {
	if id == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	// 从调度器移除
	if s.scheduler != nil {
		s.scheduler.RemoveReminder(id)
	}

	// 从数据库删除
	return s.reminderRepo.Delete(ctx, id)
}

func (s *reminderService) PauseReminder(ctx context.Context, id uint, duration time.Duration, reason string) error {
	if id == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	if duration <= 0 {
		return fmt.Errorf("暂停时长必须大于0")
	}

	reminder, err := s.reminderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if reminder == nil {
		return fmt.Errorf("提醒不存在")
	}

	pauseUntil := time.Now().Add(duration)
	reminder.PausedUntil = &pauseUntil
	reminder.PauseReason = reason

	if err := s.reminderRepo.Update(ctx, reminder); err != nil {
		return err
	}

	if s.scheduler != nil {
		if err := s.scheduler.RemoveReminder(id); err != nil {
			fmt.Printf("移除暂停提醒调度失败: %v", err)
		}
	}

	return nil
}

func (s *reminderService) ResumeReminder(ctx context.Context, id uint) error {
	if id == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	reminder, err := s.reminderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if reminder == nil {
		return fmt.Errorf("提醒不存在")
	}

	reminder.PausedUntil = nil
	reminder.PauseReason = ""

	if !reminder.IsActive {
		reminder.IsActive = true
	}

	if err := s.reminderRepo.Update(ctx, reminder); err != nil {
		return err
	}

	if s.scheduler != nil && reminder.IsActive {
		if err := s.scheduler.AddReminder(reminder); err != nil {
			fmt.Printf("恢复提醒调度失败: %v", err)
		}
	}

	return nil
}

// EditReminderParams 编辑提醒的参数
type EditReminderParams struct {
	ReminderID      uint
	NewTime         *string // 新的时间 (HH:MM:SS 格式)，可选
	NewPattern      *string // 新的重复模式，可选
	NewTitle        *string // 新的标题，可选
	NewDescription  *string // 新的描述，可选
	EditType        string  // 编辑类型：manual, ai, button（可选）
	EditReason      string  // 编辑原因（可选）
}

// EditReminder 编辑提醒（支持部分更新和历史记录）
func (s *reminderService) EditReminder(ctx context.Context, params EditReminderParams) error {
	if params.ReminderID == 0 {
		return fmt.Errorf("提醒ID不能为空")
	}

	// 1. 获取现有提醒
	reminder, err := s.reminderRepo.GetByID(ctx, params.ReminderID)
	if err != nil {
		return fmt.Errorf("获取提醒失败: %w", err)
	}
	if reminder == nil {
		return fmt.Errorf("提醒不存在")
	}

	// 记录是否有修改
	modified := false

	// 2. 应用修改并记录历史
	if params.NewTime != nil && *params.NewTime != "" {
		// 记录编辑历史
		if err := s.recordEditHistory(ctx, reminder, "time", reminder.TargetTime, *params.NewTime, params.EditType, params.EditReason); err != nil {
			logger.Warnf("记录编辑历史失败: %v", err)
		}
		reminder.TargetTime = *params.NewTime
		modified = true
	}

	if params.NewPattern != nil && *params.NewPattern != "" {
		// 记录编辑历史
		if err := s.recordEditHistory(ctx, reminder, "pattern", reminder.SchedulePattern, *params.NewPattern, params.EditType, params.EditReason); err != nil {
			logger.Warnf("记录编辑历史失败: %v", err)
		}
		reminder.SchedulePattern = *params.NewPattern
		modified = true
	}

	if params.NewTitle != nil && *params.NewTitle != "" {
		// 记录编辑历史
		if err := s.recordEditHistory(ctx, reminder, "title", reminder.Title, *params.NewTitle, params.EditType, params.EditReason); err != nil {
			logger.Warnf("记录编辑历史失败: %v", err)
		}
		reminder.Title = *params.NewTitle
		modified = true
	}

	if params.NewDescription != nil {
		// 记录编辑历史
		if err := s.recordEditHistory(ctx, reminder, "description", reminder.Description, *params.NewDescription, params.EditType, params.EditReason); err != nil {
			logger.Warnf("记录编辑历史失败: %v", err)
		}
		reminder.Description = *params.NewDescription
		modified = true
	}

	// 如果没有任何修改，直接返回
	if !modified {
		return fmt.Errorf("没有提供任何修改参数")
	}

	// 3. 更新数据库
	if err := s.reminderRepo.Update(ctx, reminder); err != nil {
		return fmt.Errorf("更新数据库失败: %w", err)
	}

	// 4. 刷新调度器
	if s.scheduler != nil && reminder.IsActive {
		// 移除旧调度
		s.scheduler.RemoveReminder(params.ReminderID)

		// 添加新调度
		if err := s.scheduler.AddReminder(reminder); err != nil {
			fmt.Printf("重新调度失败: %v", err)
			// 调度失败不影响更新，只记录错误
		}
	}

	return nil
}

// recordEditHistory 记录编辑历史
func (s *reminderService) recordEditHistory(ctx context.Context, reminder *models.Reminder, fieldName, oldValue, newValue, editType, editReason string) error {
	// 暂时使用日志记录，后续可以通过数据库持久化
	logger.Infof("编辑历史: 提醒ID=%d, 字段=%s, 旧值=%s, 新值=%s, 类型=%s",
		reminder.ID, fieldName, oldValue, newValue, editType)

	// TODO: 实现数据库持久化
	// history := &models.ReminderEditHistory{
	// 	ReminderID: reminder.ID,
	// 	UserID:     reminder.UserID,
	// 	FieldName:  fieldName,
	// 	OldValue:   oldValue,
	// 	NewValue:   newValue,
	// 	EditType:   editType,
	// 	EditReason: editReason,
	// }
	// return s.editHistoryRepo.Create(ctx, history)
	return nil
}

// RollbackEdit 回滚编辑到指定历史记录
func (s *reminderService) RollbackEdit(ctx context.Context, reminderID uint, historyID uint) error {
	// TODO: 实现回滚功能
	// 1. 从数据库获取历史记录
	// 2. 恢复旧值
	// 3. 更新提醒
	// 4. 刷新调度器

	logger.Infof("回滚提醒 #%d 到历史记录 #%d", reminderID, historyID)
	return fmt.Errorf("回滚功能暂未实现")
}

// GetEditHistory 获取提醒的编辑历史
func (s *reminderService) GetEditHistory(ctx context.Context, reminderID uint, limit int) ([]*models.ReminderEditHistory, error) {
	// TODO: 实现获取编辑历史功能
	// 1. 从数据库查询编辑历史记录
	// 2. 按时间倒序排列
	// 3. 返回结果

	logger.Infof("获取提醒 #%d 的编辑历史，限制=%d", reminderID, limit)
	return nil, fmt.Errorf("获取编辑历史功能暂未实现")
}
