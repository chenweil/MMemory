package ai

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mmemory/internal/models"
	"mmemory/pkg/ai"
	"mmemory/pkg/logger"
)

// RegexParser 传统正则表达式解析器
// 用于在AI服务不可用时提供基础的提醒解析能力
type RegexParser struct {
	patterns []*ReminderPattern
}

// ReminderPattern 提醒解析模式
type ReminderPattern struct {
	Pattern     *regexp.Regexp
	Type        models.ReminderType
	ScheduleGen func(matches []string) models.SchedulePattern
	TimeGen     func(matches []string) (int, int) // hour, minute
}

// NewRegexParser 创建正则解析器
func NewRegexParser() *RegexParser {
	parser := &RegexParser{
		patterns: make([]*ReminderPattern, 0),
	}
	parser.initPatterns()
	return parser
}

// initPatterns 初始化解析模式
func (p *RegexParser) initPatterns() {
	// 注意: 模式顺序很重要！更具体的模式应该放在前面

	// 0. 今日具体时间: "今天15:10提醒我开会"、"今天下午3点提醒我开会"
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`今天\s*(上午|中午|下午|晚上|早上|早晨|午后)?\s*(\d{1,2})(?:[:：点时](\d{1,2}))?\s*(?:分)?\s*提醒我\s*(.+)`),
		Type:    models.ReminderTypeTask,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			today := time.Now().Format("2006-01-02")
			return models.SchedulePattern(fmt.Sprintf("once:%s", today))
		},
		TimeGen: func(matches []string) (int, int) {
			period := matches[1]
			hour, _ := strconv.Atoi(matches[2])
			minute := 0
			if len(matches) > 3 && matches[3] != "" {
				minute, _ = strconv.Atoi(matches[3])
			}
			return normalizeHourForPeriod(hour, period), minute
		},
	})

	// 1. 每天带分钟: "每天9点30分提醒我锻炼"（更具体，放在前面）
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`每天.*?(\d+)点(\d+)分.*?提醒我(.+)`),
		Type:    models.ReminderTypeHabit,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			return models.SchedulePatternDaily
		},
		TimeGen: func(matches []string) (int, int) {
			hour, _ := strconv.Atoi(matches[1])
			minute, _ := strconv.Atoi(matches[2])
			return hour, minute
		},
	})

	// 2. 每天提醒: "每天早上8点提醒我喝水"
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`每天.*?(\d+)点.*?提醒我(.+)`),
		Type:    models.ReminderTypeHabit,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			return models.SchedulePatternDaily
		},
		TimeGen: func(matches []string) (int, int) {
			hour, _ := strconv.Atoi(matches[1])
			return hour, 0
		},
	})

	// 3. 每周提醒: "每周一下午3点提醒我开会"
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`每周([一二三四五六日天]).*?(\d+)点.*?提醒我(.+)`),
		Type:    models.ReminderTypeHabit,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			weekday := parseWeekday(matches[1])
			return models.SchedulePattern(fmt.Sprintf("weekly:%d", weekday))
		},
		TimeGen: func(matches []string) (int, int) {
			hour, _ := strconv.Atoi(matches[2])
			return hour, 0
		},
	})

	// 4. 工作日提醒: "工作日早上9点提醒我上班"
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`工作日.*?(\d+)点.*?提醒我(.+)`),
		Type:    models.ReminderTypeHabit,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			return "weekly:1,2,3,4,5" // 周一到周五
		},
		TimeGen: func(matches []string) (int, int) {
			hour, _ := strconv.Atoi(matches[1])
			return hour, 0
		},
	})

	// 5. 明天提醒: "明天下午2点提醒我取快递"
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`明天.*?(\d+)点.*?提醒我(.+)`),
		Type:    models.ReminderTypeTask,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
			return models.SchedulePattern(fmt.Sprintf("once:%s", tomorrow))
		},
		TimeGen: func(matches []string) (int, int) {
			hour, _ := strconv.Atoi(matches[1])
			return hour, 0
		},
	})

	// 6. 具体日期: "2025年10月15日上午10点提醒我体检"
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日.*?(\d+)点.*?提醒我(.+)`),
		Type:    models.ReminderTypeTask,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			year := matches[1]
			month := fmt.Sprintf("%02s", matches[2])
			day := fmt.Sprintf("%02s", matches[3])
			return models.SchedulePattern(fmt.Sprintf("once:%s-%s-%s", year, month, day))
		},
		TimeGen: func(matches []string) (int, int) {
			hour, _ := strconv.Atoi(matches[4])
			return hour, 0
		},
	})
}

// Parse 实现Parser接口
func (p *RegexParser) Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	message = strings.TrimSpace(message)

	// 遍历所有模式进行匹配
	for _, pattern := range p.patterns {
		matches := pattern.Pattern.FindStringSubmatch(message)
		if len(matches) > 0 {
			logger.Infof("Regex pattern matched: %s", pattern.Pattern.String())
			return p.buildParseResult(matches, pattern), nil
		}
	}

	// 没有匹配到任何模式
	return nil, ai.NewAIError(ai.ErrorTypeParsing, "no regex pattern matched", nil)
}

// buildParseResult 构建解析结果
func (p *RegexParser) buildParseResult(matches []string, pattern *ReminderPattern) *ai.ParseResult {
	// 提取标题（最后一个捕获组通常是提醒内容）
	title := strings.TrimSpace(matches[len(matches)-1])
	title = strings.Trim(title, "。.!！?？ ")

	// 生成时间信息
	hour, minute := pattern.TimeGen(matches)

	// 生成调度模式
	schedulePattern := pattern.ScheduleGen(matches)

	return &ai.ParseResult{
		Intent:     ai.IntentReminder,
		Confidence: 0.75, // 正则解析置信度设为0.75
		Reminder: &ai.ReminderInfo{
			Title:           title,
			Type:            pattern.Type,
			SchedulePattern: schedulePattern,
			Time: ai.TimeInfo{
				Hour:            hour,
				Minute:          minute,
				Timezone:        "Asia/Shanghai",
				IsRelativeTime:  false,
				ScheduleDetails: string(schedulePattern),
			},
		},
		ParsedBy:    p.GetName(),
		ProcessTime: 0, // 正则解析几乎无延迟
		Timestamp:   time.Now(),
	}
}

// GetName 实现Parser接口
func (p *RegexParser) GetName() string {
	return "regex-parser"
}

// GetPriority 实现Parser接口
func (p *RegexParser) GetPriority() int {
	return ai.ParserTypeRegex.Priority()
}

// IsHealthy 实现Parser接口
func (p *RegexParser) IsHealthy() bool {
	return true // 正则解析器总是健康的
}

// parseWeekday 解析中文星期为数字
func parseWeekday(weekday string) int {
	weekdayMap := map[string]int{
		"一": 1, "二": 2, "三": 3, "四": 4,
		"五": 5, "六": 6, "日": 0, "天": 0,
	}
	if day, ok := weekdayMap[weekday]; ok {
		return day
	}
	return 1 // 默认周一
}

func normalizeHourForPeriod(hour int, period string) int {
	if hour < 0 || hour > 23 {
		return hour
	}

	p := strings.TrimSpace(period)
	if p == "" {
		return hour
	}

	switch {
	case strings.Contains(p, "下午"), strings.Contains(p, "午后"):
		if hour < 12 {
			return hour + 12
		}
	case strings.Contains(p, "晚上"):
		if hour < 12 {
			return hour + 12
		}
	case strings.Contains(p, "中午"):
		if hour == 0 {
			return 12
		}
		if hour < 11 {
			return hour + 12
		}
	case strings.Contains(p, "上午"):
		if hour == 12 {
			return 0
		}
	case strings.Contains(p, "早上"), strings.Contains(p, "早晨"):
		if hour == 12 {
			return 0
		}
	}
	return hour
}
