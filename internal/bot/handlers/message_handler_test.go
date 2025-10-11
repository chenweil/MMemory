package handlers

import (
	"testing"
	"time"

	"mmemory/internal/models"
)

// TestMatchReminders 测试关键词匹配算法
func TestMatchReminders(t *testing.T) {
	// 创建测试数据
	reminders := []*models.Reminder{
		{
			ID:          1,
			Title:       "每天健身",
			Description: "去健身房锻炼",
			IsActive:    true,
		},
		{
			ID:          2,
			Title:       "喝水提醒",
			Description: "每天喝8杯水",
			IsActive:    true,
		},
		{
			ID:          3,
			Title:       "健身打卡",
			Description: "健身房签到",
			IsActive:    true,
		},
		{
			ID:          4,
			Title:       "午休提醒",
			Description: "中午12点休息",
			IsActive:    false, // 非活跃提醒
		},
	}

	tests := []struct {
		name           string
		keywords       []string
		expectedCount  int
		expectedFirst  uint // 第一个匹配的ID
		checkScores    bool
		expectedScores []int
	}{
		{
			name:          "单个关键词匹配",
			keywords:      []string{"健身"},
			expectedCount: 2,
			expectedFirst: 1,
			checkScores:   true,
			expectedScores: []int{1, 1}, // 两个都包含"健身"
		},
		{
			name:          "多个关键词匹配",
			keywords:      []string{"健身", "打卡"},
			expectedCount: 2,
			expectedFirst: 3, // "健身打卡"得分更高(2分)
			checkScores:   true,
			expectedScores: []int{2, 1}, // 第一个2分，第二个1分
		},
		{
			name:          "没有匹配",
			keywords:      []string{"睡觉"},
			expectedCount: 0,
		},
		{
			name:          "匹配描述字段",
			keywords:      []string{"8杯水"},
			expectedCount: 1,
			expectedFirst: 2,
		},
		{
			name:          "空关键词",
			keywords:      []string{},
			expectedCount: 0,
		},
		{
			name:          "关键词包含空格",
			keywords:      []string{"健身", "  ", "打卡"},
			expectedCount: 2,
			expectedFirst: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matchReminders(reminders, tt.keywords)

			// 验证匹配数量
			if len(matches) != tt.expectedCount {
				t.Errorf("matchReminders() 返回 %d 个结果，期望 %d 个", len(matches), tt.expectedCount)
			}

			// 验证第一个匹配的ID
			if tt.expectedCount > 0 && len(matches) > 0 {
				if matches[0].reminder.ID != tt.expectedFirst {
					t.Errorf("第一个匹配ID = %d，期望 %d", matches[0].reminder.ID, tt.expectedFirst)
				}
			}

			// 验证得分
			if tt.checkScores && len(matches) > 0 {
				for i, expectedScore := range tt.expectedScores {
					if i >= len(matches) {
						break
					}
					if matches[i].score != expectedScore {
						t.Errorf("matches[%d].score = %d，期望 %d", i, matches[i].score, expectedScore)
					}
				}
			}

			// 验证非活跃提醒被过滤
			for _, match := range matches {
				if !match.reminder.IsActive {
					t.Errorf("非活跃提醒 ID=%d 不应该被匹配", match.reminder.ID)
				}
			}
		})
	}
}

// TestFilterKeywords 测试关键词过滤
func TestFilterKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "正常关键词",
			input:    []string{"健身", "打卡"},
			expected: []string{"健身", "打卡"},
		},
		{
			name:     "包含空字符串",
			input:    []string{"健身", "", "打卡"},
			expected: []string{"健身", "打卡"},
		},
		{
			name:     "包含空格",
			input:    []string{"健身", "  ", "打卡", " "},
			expected: []string{"健身", "打卡"},
		},
		{
			name:     "全是空字符串",
			input:    []string{"", "  ", " "},
			expected: []string{},
		},
		{
			name:     "空切片",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterKeywords(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("filterKeywords() 返回长度 = %d，期望 %d", len(result), len(tt.expected))
				return
			}

			for i, keyword := range result {
				if keyword != tt.expected[i] {
					t.Errorf("result[%d] = %s，期望 %s", i, keyword, tt.expected[i])
				}
			}
		})
	}
}

// TestParsePauseDuration 测试暂停时长解析
func TestParsePauseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			name:     "1周",
			input:    "1week",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:     "2周",
			input:    "2week",
			expected: 14 * 24 * time.Hour,
		},
		{
			name:     "1天",
			input:    "1day",
			expected: 24 * time.Hour,
		},
		{
			name:     "3天",
			input:    "3day",
			expected: 3 * 24 * time.Hour,
		},
		{
			name:     "1月",
			input:    "1month",
			expected: 30 * 24 * time.Hour,
		},
		{
			name:     "中文周",
			input:    "1周",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:     "中文天",
			input:    "2天",
			expected: 2 * 24 * time.Hour,
		},
		{
			name:     "中文月",
			input:    "1月",
			expected: 30 * 24 * time.Hour,
		},
		{
			name:     "小时",
			input:    "24小时",
			expected: 24 * time.Hour,
		},
		{
			name:     "空字符串默认值",
			input:    "",
			expected: 7 * 24 * time.Hour, // 默认7天
		},
		{
			name:     "无效格式默认值",
			input:    "invalid",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:     "P格式 - 周",
			input:    "P1W",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:     "P格式 - 天",
			input:    "P3D",
			expected: 3 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePauseDuration(tt.input)

			if result != tt.expected {
				t.Errorf("parsePauseDuration(%q) = %v，期望 %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMatchReminders_Scoring 测试关键词匹配评分逻辑
func TestMatchReminders_Scoring(t *testing.T) {
	reminders := []*models.Reminder{
		{
			ID:          1,
			Title:       "健身",
			Description: "每天去健身房",
			IsActive:    true,
		},
		{
			ID:          2,
			Title:       "健身打卡",
			Description: "健身房签到",
			IsActive:    true,
		},
		{
			ID:          3,
			Title:       "跑步健身",
			Description: "晨跑和健身",
			IsActive:    true,
		},
	}

	// 使用多个关键词测试评分
	keywords := []string{"健身", "打卡"}
	matches := matchReminders(reminders, keywords)

	if len(matches) != 3 {
		t.Fatalf("期望3个匹配，实际得到 %d", len(matches))
	}

	// 验证按得分降序排列
	// ID=2 "健身打卡" 应该得分最高 (标题包含"健身"和"打卡" = 2分)
	if matches[0].reminder.ID != 2 {
		t.Errorf("最高分应该是ID=2，实际是ID=%d", matches[0].reminder.ID)
	}

	if matches[0].score != 2 {
		t.Errorf("最高分应该是2，实际是%d", matches[0].score)
	}

	// 其他两个提醒只匹配"健身"(1分)，应该按ID排序
	if matches[1].score != 1 || matches[2].score != 1 {
		t.Errorf("后两个得分应该都是1，实际是 %d 和 %d", matches[1].score, matches[2].score)
	}
}

// TestMatchReminders_EdgeCases 测试边界情况
func TestMatchReminders_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		reminders []*models.Reminder
		keywords  []string
		expected  int
	}{
		{
			name:      "空提醒列表",
			reminders: []*models.Reminder{},
			keywords:  []string{"健身"},
			expected:  0,
		},
		{
			name: "空关键词",
			reminders: []*models.Reminder{
				{ID: 1, Title: "健身", IsActive: true},
			},
			keywords: []string{},
			expected: 0,
		},
		{
			name: "所有提醒都不活跃",
			reminders: []*models.Reminder{
				{ID: 1, Title: "健身", IsActive: false},
				{ID: 2, Title: "跑步", IsActive: false},
			},
			keywords: []string{"健身"},
			expected: 0,
		},
		{
			name:      "nil提醒列表",
			reminders: nil,
			keywords:  []string{"健身"},
			expected:  0,
		},
		{
			name: "关键词大小写不敏感",
			reminders: []*models.Reminder{
				{ID: 1, Title: "健身打卡", IsActive: true},
			},
			keywords: []string{"健身"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matchReminders(tt.reminders, tt.keywords)

			if len(matches) != tt.expected {
				t.Errorf("matchReminders() 返回 %d 个结果，期望 %d 个", len(matches), tt.expected)
			}
		})
	}
}

// TestMatchReminders_Performance 测试性能（大数据量）
func TestMatchReminders_Performance(t *testing.T) {
	// 创建1000个提醒
	reminders := make([]*models.Reminder, 1000)
	for i := 0; i < 1000; i++ {
		reminders[i] = &models.Reminder{
			ID:          uint(i + 1),
			Title:       "提醒" + string(rune(i%100)),
			Description: "描述" + string(rune(i%50)),
			IsActive:    true,
		}
	}

	// 添加几个特定的测试提醒
	reminders[0].Title = "健身打卡"
	reminders[1].Title = "每天健身"
	reminders[2].Title = "健身房"

	keywords := []string{"健身"}

	start := time.Now()
	matches := matchReminders(reminders, keywords)
	duration := time.Since(start)

	// 性能要求：1000个提醒的匹配应该在10ms内完成
	if duration > 10*time.Millisecond {
		t.Errorf("匹配性能不佳：耗时 %v，期望 < 10ms", duration)
	}

	if len(matches) < 3 {
		t.Errorf("至少应该匹配3个提醒，实际匹配 %d 个", len(matches))
	}

	t.Logf("匹配1000个提醒耗时: %v, 匹配到: %d 个", duration, len(matches))
}
