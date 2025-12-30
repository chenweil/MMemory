package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mmemory/internal/models"
)

// TestParserService_New 测试创建解析服务
func TestParserService_New(t *testing.T) {
	service := NewParserService()
	assert.NotNil(t, service)
}

// TestParserService_GetPatterns 测试获取解析模式
func TestParserService_GetPatterns(t *testing.T) {
	service := NewParserService()
	patterns := service.GetPatterns()

	// 应该返回多个解析模式
	assert.Greater(t, len(patterns), 10)

	// 验证第一个模式有有效的正则表达式
	assert.NotNil(t, patterns[0].Regex)
	assert.NotNil(t, patterns[0].ScheduleGen)
}

// TestParserService_ParseReminderFromText_Daily 测试每日提醒解析
func TestParserService_ParseReminderFromText_Daily(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name     string
		input    string
		userID   uint
		check    func(t *testing.T, reminder *models.Reminder, err error)
	}{
		{
			name:   "每天8点提醒",
			input:  "每天8点提醒我喝水",
			userID: 1,
			check: func(t *testing.T, reminder *models.Reminder, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, reminder)
				assert.Equal(t, "喝水", reminder.Title)
				assert.Equal(t, "08:00:00", reminder.TargetTime)
				assert.Equal(t, "daily", reminder.SchedulePattern)
				assert.Equal(t, models.ReminderTypeHabit, reminder.Type)
			},
		},
		{
			name:   "每天19:30提醒",
			input:  "每天19:30提醒我健身",
			userID: 1,
			check: func(t *testing.T, reminder *models.Reminder, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, reminder)
				assert.Equal(t, "健身", reminder.Title)
				assert.Equal(t, "19:30:00", reminder.TargetTime)
				assert.Equal(t, "daily", reminder.SchedulePattern)
			},
		},
		{
			name:   "每天上午9点提醒",
			input:  "每天上午9点提醒我吃早餐",
			userID: 1,
			check: func(t *testing.T, reminder *models.Reminder, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, reminder)
				assert.Equal(t, "吃早餐", reminder.Title)
				assert.Equal(t, "09:00:00", reminder.TargetTime)
			},
		},
		{
			name:   "每天下午2点提醒",
			input:  "每天下午2点提醒我休息",
			userID: 1,
			check: func(t *testing.T, reminder *models.Reminder, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, reminder)
				assert.Equal(t, "休息", reminder.Title)
				assert.Equal(t, "14:00:00", reminder.TargetTime)
			},
		},
		{
			name:   "每天晚上8点提醒",
			input:  "每天晚上8点提醒我复盘",
			userID: 1,
			check: func(t *testing.T, reminder *models.Reminder, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, reminder)
				assert.Equal(t, "复盘", reminder.Title)
				assert.Equal(t, "20:00:00", reminder.TargetTime)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminder, err := service.ParseReminderFromText(context.Background(), tt.input, tt.userID)
			tt.check(t, reminder, err)
		})
	}
}

// TestParserService_ParseReminderFromText_Weekly 测试每周提醒解析
func TestParserService_ParseReminderFromText_Weekly(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name   string
		input  string
		userID uint
	}{
		{
			name:   "每周一9点提醒",
			input:  "每周一9点提醒我开会",
			userID: 1,
		},
		{
			name:   "每周三五19点提醒",
			input:  "每周三五19点提醒我健身",
			userID: 1,
		},
		{
			name:   "工作日8点提醒",
			input:  "工作日8点提醒我起床",
			userID: 1,
		},
		{
			name:   "周末10点提醒",
			input:  "周末10点提醒我睡懒觉",
			userID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminder, err := service.ParseReminderFromText(context.Background(), tt.input, tt.userID)
			assert.NoError(t, err)
			assert.NotNil(t, reminder)
			assert.Contains(t, reminder.SchedulePattern, "weekly")
		})
	}
}

// TestParserService_ParseReminderFromText_Date 测试特定日期提醒解析
func TestParserService_ParseReminderFromText_Date(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name   string
		input  string
		userID uint
	}{
		{
			name:   "2025年10月1日9点提醒",
			input:  "2025年10月1日9点提醒我交房租",
			userID: 1,
		},
		{
			name:   "明天9点提醒",
			input:  "明天9点提醒我开会",
			userID: 1,
		},
		{
			name:   "今晚8点提醒",
			input:  "今晚8点提醒我跑步",
			userID: 1,
		},
		{
			name:   "后天10点提醒",
			input:  "后天10点提醒我考试",
			userID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminder, err := service.ParseReminderFromText(context.Background(), tt.input, tt.userID)
			assert.NoError(t, err)
			assert.NotNil(t, reminder)
			assert.Contains(t, reminder.SchedulePattern, "once")
		})
	}
}

// TestParserService_ParseReminderFromText_Relative 测试相对时间提醒解析
// 注意：当前解析器的parseTime函数支持绝对时间，也支持小时的相对时间（如"2小时后"）
// 但不支持分钟级相对时间（如"30分钟后"），因为30 > 24超出了小时范围
func TestParserService_ParseReminderFromText_Relative(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name          string
		input         string
		userID        uint
		expectSuccess bool
	}{
		{
			name:          "30分钟后提醒",
			input:         "30分钟后提醒我喝水",
			userID:        1,
			expectSuccess: false, // 30 > 24, parseTime会失败
		},
		{
			name:          "2小时后提醒",
			input:         "2小时后提醒我休息",
			userID:        1,
			expectSuccess: true, // 2 <= 24, parseTime会成功
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminder, err := service.ParseReminderFromText(context.Background(), tt.input, tt.userID)
			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, reminder)
				assert.Equal(t, "休息", reminder.Title)
			} else {
				assert.Error(t, err)
				assert.Nil(t, reminder)
			}
		})
	}
}

// TestParserService_ParseReminderFromText_Invalid 测试无效输入
func TestParserService_ParseReminderFromText_Invalid(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "空字符串",
			input: "",
		},
		{
			name:  "无效格式",
			input: "这是一个随机消息",
		},
		{
			name:  "无法解析",
			input: "帮我做点事",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminder, err := service.ParseReminderFromText(context.Background(), tt.input, 1)
			assert.Error(t, err)
			assert.Nil(t, reminder)
		})
	}
}

// TestParserService_ParseTime 测试时间解析
func TestParserService_ParseTime(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name        string
		matches     []string
		expectedHour int
		expectedMin  int
		expectError  bool
	}{
		{
			name:         "普通时间",
			matches:      []string{"", "8", "30"},
			expectedHour: 8,
			expectedMin:  30,
			expectError:  false,
		},
		{
			name:         "只有小时",
			matches:      []string{"", "9"},
			expectedHour: 9,
			expectedMin:  0,
			expectError:  false,
		},
		{
			name:        "上午时间",
			matches:     []string{"", "上午", "9"},
			expectedHour: 9,
			expectedMin:  0,
			expectError:  false,
		},
		{
			name:        "下午时间",
			matches:     []string{"", "下午", "3"},
			expectedHour: 15,
			expectedMin:  0,
			expectError:  false,
		},
		{
			name:        "晚上时间",
			matches:     []string{"", "晚上", "8"},
			expectedHour: 20,
			expectedMin:  0,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hour, minute, err := service.parseTime(tt.matches)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHour, hour)
				assert.Equal(t, tt.expectedMin, minute)
			}
		})
	}
}

// TestParserService_AdjustHourByPeriod 测试时间段调整
func TestParserService_AdjustHourByPeriod(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name     string
		hour     int
		period   string
		expected int
	}{
		{"上午9点", 9, "上午", 9},
		{"上午12点", 12, "上午", 0},
		{"下午3点", 3, "下午", 15},
		{"下午12点", 12, "下午", 12},
		{"晚上8点", 8, "晚上", 20},
		{"晚上12点", 12, "晚上", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.adjustHourByPeriod(tt.hour, tt.period)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParserService_ParseWeekdays 测试星期解析
func TestParserService_ParseWeekdays(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"周一", "周一", []string{"1"}},
		{"周五", "周五", []string{"5"}},
		{"周日", "周日", []string{"7"}},
		{"周一三五", "周一三五", []string{"1", "3", "5"}},
		{"周二四六", "周二四六", []string{"2", "4", "6"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.parseWeekdays(tt.input)
			// 使用ElementsMatch因为map遍历顺序不确定
			assert.ElementsMatch(t, tt.expected, result)
			assert.Len(t, result, len(tt.expected))
		})
	}
}

// TestParserService_ChineseWeekdayToInt 测试中文星期转换
func TestParserService_ChineseWeekdayToInt(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		input    string
		expected int
	}{
		{"一", 1},
		{"二", 2},
		{"三", 3},
		{"四", 4},
		{"五", 5},
		{"六", 6},
		{"日", 7},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.chineseWeekdayToInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParserService_IsTimeString 测试时间字符串检查
func TestParserService_IsTimeString(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		input    string
		expected bool
	}{
		{"0", true},
		{"30", true},
		{"59", true},
		{"60", false},
		{"abc", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.isTimeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParserService_IsWeekdayString 测试星期字符串检查
func TestParserService_IsWeekdayString(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		input    string
		expected bool
	}{
		{"周一", true},
		{"周五", true},
		{"周日", true},
		{"天", true},
		{"周一三五", true},
		{"今天", true}, // contains "天"
		{"明天", true}, // contains "天"
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.isWeekdayString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParserService_IsPeriodString 测试时间段字符串检查
func TestParserService_IsPeriodString(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		input    string
		expected bool
	}{
		{"上午", true},
		{"下午", true},
		{"晚上", true},
		{"今晚", true},
		{"工作日", true},
		{"周末", true},
		{"上午好", false},
		{"下午茶", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := service.isPeriodString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParserService_GetNextWeekdayDate 测试获取下周日期
func TestParserService_GetNextWeekdayDate(t *testing.T) {
	service := NewParserService()

	today := time.Now()
	currentWeekday := int(today.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7
	}

	// 测试获取下周一的日期
	nextMonday := service.getNextWeekdayDate(1)
	assert.NotNil(t, nextMonday)

	// 下周一应该在今天之后的1-7天内
	daysToMonday := int(nextMonday.Sub(today).Hours() / 24)
	assert.GreaterOrEqual(t, daysToMonday, 1)
	assert.LessOrEqual(t, daysToMonday, 7)

	// 测试获取下周日的日期
	nextSunday := service.getNextWeekdayDate(7)
	assert.NotNil(t, nextSunday)

	// 下周日也应该是1-7天内
	daysToSunday := int(nextSunday.Sub(today).Hours() / 24)
	assert.GreaterOrEqual(t, daysToSunday, 1)
	assert.LessOrEqual(t, daysToSunday, 7)

	// 两个日期都应该在今天之后
	assert.True(t, nextMonday.After(today), "下周一应该在今天之后")
	assert.True(t, nextSunday.After(today), "下周日应该在今天之后")
}

// TestParserService_ParseTitle 测试标题解析
func TestParserService_ParseTitle(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name     string
		matches  []string
		expected string
	}{
		{
			name:     "普通标题",
			matches:  []string{"每天8点提醒我", "8", "", "喝水"},
			expected: "喝水",
		},
		{
			name:     "带时间",
			matches:  []string{"明天9点提醒我", "9", "", "开会"},
			expected: "开会",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.parseTitle(tt.matches)
			assert.Equal(t, tt.expected, result)
		})
	}
}
