package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"mmemory/internal/models"
)

type parserService struct{}

func NewParserService() *parserService {
	return &parserService{}
}

// ParsePattern 解析模式结构
type ParsePattern struct {
	Regex       *regexp.Regexp
	Type        models.ReminderType
	ScheduleGen func(matches []string) string
}

// GetPatterns 获取所有解析模式
func (s *parserService) GetPatterns() []ParsePattern {
	return []ParsePattern{
		{
			Regex: regexp.MustCompile(`每天(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeHabit,
			ScheduleGen: func(matches []string) string {
				return "daily"
			},
		},
		{
			Regex: regexp.MustCompile(`每周([一二三四五六日,，\s]+)(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeHabit,
			ScheduleGen: func(matches []string) string {
				weekdays := s.parseWeekdays(matches[1])
				return fmt.Sprintf("weekly:%s", strings.Join(weekdays, ","))
			},
		},
		{
			Regex: regexp.MustCompile(`(\d{4})[年-](\d{1,2})[月-](\d{1,2})日?(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				return fmt.Sprintf("once:%s-%02s-%02s", matches[1], matches[2], matches[3])
			},
		},
		{
			Regex: regexp.MustCompile(`明天(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				// TODO: 计算明天的日期
				return "once:tomorrow"
			},
		},
	}
}

// ParseReminderFromText 从文本解析提醒
func (s *parserService) ParseReminderFromText(ctx context.Context, text string, userID uint) (*models.Reminder, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("文本不能为空")
	}

	patterns := s.GetPatterns()
	
	for _, pattern := range patterns {
		matches := pattern.Regex.FindStringSubmatch(text)
		if len(matches) == 0 {
			continue
		}

		// 解析时间
		hour, minute, err := s.parseTime(matches)
		if err != nil {
			continue
		}

		// 解析标题
		title := s.parseTitle(matches)
		if title == "" {
			continue
		}

		// 生成调度模式
		schedulePattern := pattern.ScheduleGen(matches)

		reminder := &models.Reminder{
			UserID:          userID,
			Title:           title,
			Type:            pattern.Type,
			SchedulePattern: schedulePattern,
			TargetTime:      fmt.Sprintf("%02d:%02d:00", hour, minute),
			IsActive:        true,
		}

		return reminder, nil
	}

	return nil, fmt.Errorf("无法解析提醒格式")
}

// parseTime 解析时间
func (s *parserService) parseTime(matches []string) (hour, minute int, err error) {
	// 查找时间相关的匹配组
	for i := 1; i < len(matches); i++ {
		if matches[i] == "" {
			continue
		}
		
		// 尝试解析小时
		if h, parseErr := strconv.Atoi(matches[i]); parseErr == nil && h >= 0 && h <= 23 {
			hour = h
			
			// 检查下一个匹配组是否为分钟
			if i+1 < len(matches) && matches[i+1] != "" {
				if m, parseErr := strconv.Atoi(matches[i+1]); parseErr == nil && m >= 0 && m <= 59 {
					minute = m
				}
			}
			return hour, minute, nil
		}
	}
	
	return 0, 0, fmt.Errorf("无法解析时间")
}

// parseTitle 解析标题
func (s *parserService) parseTitle(matches []string) string {
	// 标题通常是最后一个匹配组
	for i := len(matches) - 1; i >= 1; i-- {
		if matches[i] != "" && !s.isTimeString(matches[i]) && !s.isWeekdayString(matches[i]) {
			return strings.TrimSpace(matches[i])
		}
	}
	return ""
}

// parseWeekdays 解析星期
func (s *parserService) parseWeekdays(weekdayStr string) []string {
	weekdayMap := map[string]string{
		"一": "1", "二": "2", "三": "3", "四": "4", 
		"五": "5", "六": "6", "日": "7", "天": "7",
	}
	
	var weekdays []string
	for chinese, number := range weekdayMap {
		if strings.Contains(weekdayStr, chinese) {
			weekdays = append(weekdays, number)
		}
	}
	
	return weekdays
}

// isTimeString 检查是否为时间字符串
func (s *parserService) isTimeString(str string) bool {
	if num, err := strconv.Atoi(str); err == nil && num >= 0 && num <= 59 {
		return true
	}
	return false
}

// isWeekdayString 检查是否为星期字符串
func (s *parserService) isWeekdayString(str string) bool {
	weekdays := []string{"一", "二", "三", "四", "五", "六", "日", "天"}
	for _, weekday := range weekdays {
		if strings.Contains(str, weekday) {
			return true
		}
	}
	return false
}