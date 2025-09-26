package service

import (
	"context"
	"testing"

	"mmemory/internal/models"
)

func TestParserService_ParseReminderFromText(t *testing.T) {
	parser := NewParserService()
	ctx := context.Background()
	userID := uint(1)

	tests := []struct {
		name     string
		text     string
		wantType models.ReminderType
		wantErr  bool
	}{
		{
			name:     "每天提醒解析",
			text:     "每天19点提醒我复盘工作",
			wantType: models.ReminderTypeHabit,
			wantErr:  false,
		},
		{
			name:     "每天提醒带分钟",
			text:     "每天19:30提醒我复盘工作",
			wantType: models.ReminderTypeHabit,
			wantErr:  false,
		},
		{
			name:     "每周提醒解析",
			text:     "每周一三五19点提醒我健身",
			wantType: models.ReminderTypeHabit,
			wantErr:  false,
		},
		{
			name:     "一次性提醒解析",
			text:     "2024年10月1日19点提醒我交房租",
			wantType: models.ReminderTypeTask,
			wantErr:  false,
		},
		{
			name:     "明天提醒解析",
			text:     "明天10点提醒我开会",
			wantType: models.ReminderTypeTask,
			wantErr:  false,
		},
		{
			name:     "无法解析的文本",
			text:     "这是一个无法解析的文本",
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "空文本",
			text:     "",
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reminder, err := parser.ParseReminderFromText(ctx, tt.text, userID)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReminderFromText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				if reminder != nil {
					t.Errorf("ParseReminderFromText() 期望返回nil，但得到了 %v", reminder)
				}
				return
			}
			
			if reminder == nil {
				t.Errorf("ParseReminderFromText() 期望返回reminder，但得到了nil")
				return
			}
			
			if reminder.Type != tt.wantType {
				t.Errorf("ParseReminderFromText() type = %v, want %v", reminder.Type, tt.wantType)
			}
			
			if reminder.UserID != userID {
				t.Errorf("ParseReminderFromText() userID = %v, want %v", reminder.UserID, userID)
			}
			
			if reminder.Title == "" {
				t.Errorf("ParseReminderFromText() title 不应该为空")
			}
			
			if reminder.TargetTime == "" {
				t.Errorf("ParseReminderFromText() targetTime 不应该为空")
			}
		})
	}
}

func TestParserService_parseTime(t *testing.T) {
	parser := NewParserService()

	tests := []struct {
		name        string
		matches     []string
		wantHour    int
		wantMinute  int
		wantErr     bool
	}{
		{
			name:        "解析小时",
			matches:     []string{"", "19", "", "工作"},
			wantHour:    19,
			wantMinute:  0,
			wantErr:     false,
		},
		{
			name:        "解析小时和分钟",
			matches:     []string{"", "19", "30", "工作"},
			wantHour:    19,
			wantMinute:  30,
			wantErr:     false,
		},
		{
			name:        "无效时间",
			matches:     []string{"", "工作", "", ""},
			wantHour:    0,
			wantMinute:  0,
			wantErr:     true,
		},
		{
			name:        "超出范围的小时",
			matches:     []string{"", "25", "", "工作"},
			wantHour:    0,
			wantMinute:  0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hour, minute, err := parser.parseTime(tt.matches)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if hour != tt.wantHour {
					t.Errorf("parseTime() hour = %v, want %v", hour, tt.wantHour)
				}
				if minute != tt.wantMinute {
					t.Errorf("parseTime() minute = %v, want %v", minute, tt.wantMinute)
				}
			}
		})
	}
}

func TestParserService_parseWeekdays(t *testing.T) {
	parser := NewParserService()

	tests := []struct {
		name        string
		weekdayStr  string
		wantLen     int
		wantContain []string
	}{
		{
			name:        "解析一三五",
			weekdayStr:  "一三五",
			wantLen:     3,
			wantContain: []string{"1", "3", "5"},
		},
		{
			name:        "解析周日",
			weekdayStr:  "日",
			wantLen:     1,
			wantContain: []string{"7"},
		},
		{
			name:        "解析混合格式",
			weekdayStr:  "一，三，五",
			wantLen:     3,
			wantContain: []string{"1", "3", "5"},
		},
		{
			name:        "无效字符串",
			weekdayStr:  "abc",
			wantLen:     0,
			wantContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weekdays := parser.parseWeekdays(tt.weekdayStr)
			
			if len(weekdays) != tt.wantLen {
				t.Errorf("parseWeekdays() length = %v, want %v", len(weekdays), tt.wantLen)
			}
			
			for _, want := range tt.wantContain {
				found := false
				for _, got := range weekdays {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("parseWeekdays() 缺少期望的值 %v，实际得到 %v", want, weekdays)
				}
			}
		})
	}
}