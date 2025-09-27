package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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
		// 每天提醒
		{
			Regex: regexp.MustCompile(`每天(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeHabit,
			ScheduleGen: func(matches []string) string {
				return "daily"
			},
		},
		// 每天上午/下午/晚上提醒
		{
			Regex: regexp.MustCompile(`每天(上午|下午|晚上)(\d{1,2})[点:]?(\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeHabit,
			ScheduleGen: func(matches []string) string {
				return "daily"
			},
		},
		// 每周提醒
		{
			Regex: regexp.MustCompile(`每周([一二三四五六日,，\s]+)(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeHabit,
			ScheduleGen: func(matches []string) string {
				weekdays := s.parseWeekdays(matches[1])
				return fmt.Sprintf("weekly:%s", strings.Join(weekdays, ","))
			},
		},
		// 工作日提醒
		{
			Regex: regexp.MustCompile(`(工作日|每个工作日)(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeHabit,
			ScheduleGen: func(matches []string) string {
				return "weekly:1,2,3,4,5" // 周一到周五
			},
		},
		// 周末提醒
		{
			Regex: regexp.MustCompile(`(周末|每个周末)(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeHabit,
			ScheduleGen: func(matches []string) string {
				return "weekly:6,7" // 周六周日
			},
		},
		// 具体日期提醒
		{
			Regex: regexp.MustCompile(`(\d{4})[年-](\d{1,2})[月-](\d{1,2})日?(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				return fmt.Sprintf("once:%s-%02s-%02s", matches[1], matches[2], matches[3])
			},
		},
		// 明天提醒
		{
			Regex: regexp.MustCompile(`明天(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				tomorrow := time.Now().AddDate(0, 0, 1)
				return fmt.Sprintf("once:%s", tomorrow.Format("2006-01-02"))
			},
		},
		// 明天上午/下午/晚上提醒
		{
			Regex: regexp.MustCompile(`明天(上午|下午|晚上)(\d{1,2})[点:]?(\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				tomorrow := time.Now().AddDate(0, 0, 1)
				return fmt.Sprintf("once:%s", tomorrow.Format("2006-01-02"))
			},
		},
		// 今晚/今天晚上提醒
		{
			Regex: regexp.MustCompile(`(今晚|今天晚上)(\d{1,2})[点:]?(\d{1,2})?\s*提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				today := time.Now()
				return fmt.Sprintf("once:%s", today.Format("2006-01-02"))
			},
		},
		// 今晚XX:XX:XX提醒（24小时制）
		{
			Regex: regexp.MustCompile(`今晚(\d{1,2}):(\d{1,2})\s*提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				today := time.Now()
				return fmt.Sprintf("once:%s", today.Format("2006-01-02"))
			},
		},
		// 今晚XX:XX提醒
		{
			Regex: regexp.MustCompile(`今晚(\d{1,2})[点:]?(\d{1,2})?\s*提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				today := time.Now()
				return fmt.Sprintf("once:%s", today.Format("2006-01-02"))
			},
		},
		// 每晚提醒（转化为每天提醒）
		{
			Regex: regexp.MustCompile(`(每晚|每天晚上)(\d{1,2})[点:]?(\d{1,2})?\s*提醒我(.+)`),
			Type:  models.ReminderTypeHabit,
			ScheduleGen: func(matches []string) string {
				return "daily"
			},
		},
		// 后天提醒
		{
			Regex: regexp.MustCompile(`后天(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				dayAfterTomorrow := time.Now().AddDate(0, 0, 2)
				return fmt.Sprintf("once:%s", dayAfterTomorrow.Format("2006-01-02"))
			},
		},
		// 下周几提醒
		{
			Regex: regexp.MustCompile(`下周([一二三四五六日])(\d{1,2})[点:](\d{1,2})?提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				weekday := s.chineseWeekdayToInt(matches[1])
				nextWeekDate := s.getNextWeekdayDate(weekday)
				return fmt.Sprintf("once:%s", nextWeekDate.Format("2006-01-02"))
			},
		},
		// X分钟后提醒
		{
			Regex: regexp.MustCompile(`(\d+)分钟后提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				minutes, _ := strconv.Atoi(matches[1])
				targetTime := time.Now().Add(time.Duration(minutes) * time.Minute)
				return fmt.Sprintf("once:%s", targetTime.Format("2006-01-02"))
			},
		},
		// X小时后提醒
		{
			Regex: regexp.MustCompile(`(\d+)小时后提醒我(.+)`),
			Type:  models.ReminderTypeTask,
			ScheduleGen: func(matches []string) string {
				hours, _ := strconv.Atoi(matches[1])
				targetTime := time.Now().Add(time.Duration(hours) * time.Hour)
				return fmt.Sprintf("once:%s", targetTime.Format("2006-01-02"))
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

// parseTime 解析时间，增强版支持上午下午晚上等
func (s *parserService) parseTime(matches []string) (hour, minute int, err error) {
	// 查找时间相关的匹配组
	for i := 1; i < len(matches); i++ {
		if matches[i] == "" {
			continue
		}
		
		// 处理上午下午晚上
		if matches[i] == "上午" || matches[i] == "下午" || matches[i] == "晚上" {
			// 下一个应该是小时数
			if i+1 < len(matches) && matches[i+1] != "" {
				if h, parseErr := strconv.Atoi(matches[i+1]); parseErr == nil {
					hour = s.adjustHourByPeriod(h, matches[i])
					// 检查是否有分钟数
					if i+2 < len(matches) && matches[i+2] != "" {
						if m, parseErr := strconv.Atoi(matches[i+2]); parseErr == nil && m >= 0 && m <= 59 {
							minute = m
						}
					}
					return hour, minute, nil
				}
			}
			continue
		}
		
		// 尝试解析小时
		if h, parseErr := strconv.Atoi(matches[i]); parseErr == nil && h >= 0 && h <= 24 {
			hour = h
			
			// 检查下一个匹配组是否为分钟
			if i+1 < len(matches) && matches[i+1] != "" {
				if m, parseErr := strconv.Atoi(matches[i+1]); parseErr == nil && m >= 0 && m <= 59 {
					minute = m
				}
			}
			// 如果hour>12且没有指定时间段，且下一个匹配组不是分钟，说明是24小时制
			if hour > 12 && i+1 < len(matches) && matches[i+1] != "" {
				if _, parseErr := strconv.Atoi(matches[i+1]); parseErr != nil {
					// 下一个不是数字，说明是24小时制，不需要调整
					return hour, minute, nil
				}
			}
			return hour, minute, nil
		}
	}
	
	return 0, 0, fmt.Errorf("无法解析时间")
}

// adjustHourByPeriod 根据时间段调整小时
func (s *parserService) adjustHourByPeriod(hour int, period string) int {
	switch period {
	case "上午":
		if hour >= 1 && hour <= 11 {
			return hour
		}
		if hour == 12 {
			return 0 // 上午12点是午夜0点
		}
	case "下午":
		if hour >= 1 && hour <= 11 {
			return hour + 12
		}
		if hour == 12 {
			return 12 // 下午12点是正午
		}
	case "晚上":
		if hour >= 1 && hour <= 11 {
			return hour + 12
		}
		if hour == 12 {
			return 0 // 晚上12点是午夜
		}
	}
	return hour
}

// parseTitle 解析标题
func (s *parserService) parseTitle(matches []string) string {
	// 标题通常是最后一个匹配组
	for i := len(matches) - 1; i >= 1; i-- {
		if matches[i] != "" && !s.isTimeString(matches[i]) && !s.isWeekdayString(matches[i]) && !s.isPeriodString(matches[i]) {
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

// chineseWeekdayToInt 中文星期转换为数字
func (s *parserService) chineseWeekdayToInt(weekday string) int {
	weekdayMap := map[string]int{
		"一": 1, "二": 2, "三": 3, "四": 4, 
		"五": 5, "六": 6, "日": 7,
	}
	return weekdayMap[weekday]
}

// getNextWeekdayDate 获取下周指定星期几的日期
func (s *parserService) getNextWeekdayDate(targetWeekday int) time.Time {
	now := time.Now()
	currentWeekday := int(now.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7 // 将周日从0改为7
	}
	
	daysUntilNext := (7 - currentWeekday + targetWeekday) % 7
	if daysUntilNext == 0 {
		daysUntilNext = 7 // 下周同一天
	}
	
	return now.AddDate(0, 0, daysUntilNext)
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

// isPeriodString 检查是否为时间段字符串
func (s *parserService) isPeriodString(str string) bool {
	periods := []string{"上午", "下午", "晚上", "今晚", "今天晚上", "每晚", "每天晚上", "工作日", "周末"}
	for _, period := range periods {
		if str == period {
			return true
		}
	}
	return false
}