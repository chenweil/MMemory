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

	// 7. X分钟后提醒: "5分钟后提醒我喝水"
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`(\d+)分钟后提醒我(.+)`),
		Type:    models.ReminderTypeTask,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			minutes, _ := strconv.Atoi(matches[1])
			targetTime := time.Now().Add(time.Duration(minutes) * time.Minute)
			return models.SchedulePattern(fmt.Sprintf("once:%s", targetTime.Format("2006-01-02")))
		},
		TimeGen: func(matches []string) (int, int) {
			minutes, _ := strconv.Atoi(matches[1])
			targetTime := time.Now().Add(time.Duration(minutes) * time.Minute)
			return targetTime.Hour(), targetTime.Minute()
		},
	})

	// 8. X小时后提醒: "2小时后提醒我开会"
	p.patterns = append(p.patterns, &ReminderPattern{
		Pattern: regexp.MustCompile(`(\d+)小时后提醒我(.+)`),
		Type:    models.ReminderTypeTask,
		ScheduleGen: func(matches []string) models.SchedulePattern {
			hours, _ := strconv.Atoi(matches[1])
			targetTime := time.Now().Add(time.Duration(hours) * time.Hour)
			return models.SchedulePattern(fmt.Sprintf("once:%s", targetTime.Format("2006-01-02")))
		},
		TimeGen: func(matches []string) (int, int) {
			hours, _ := strconv.Atoi(matches[1])
			targetTime := time.Now().Add(time.Duration(hours) * time.Hour)
			return targetTime.Hour(), targetTime.Minute()
		},
	})
}

// Parse 实现Parser接口
func (p *RegexParser) Parse(ctx context.Context, userID string, message string) (*ai.ParseResult, error) {
	// 调用带上下文的解析，传入空的历史
	return p.ParseWithContext(ctx, userID, message, "")
}

// ParseWithContext 实现Parser接口（带上下文支持）
func (p *RegexParser) ParseWithContext(ctx context.Context, userID string, message string, conversationHistory string) (*ai.ParseResult, error) {
	message = strings.TrimSpace(message)

	// 首先检查活动记录模式
	if activityResult := p.checkActivityPatterns(message); activityResult != nil {
		return activityResult, nil
	}

	// 检���删除活动模式
	if deleteResult := p.checkDeleteActivityPatterns(message); deleteResult != nil {
		return deleteResult, nil
	}

	// 检查查询模式
	if queryResult := p.checkQueryPatterns(message); queryResult != nil {
		return queryResult, nil
	}

	// 遍历所有模式进行匹配
	for _, pattern := range p.patterns {
		matches := pattern.Pattern.FindStringSubmatch(message)
		if len(matches) > 0 {
			logger.Infof("Regex pattern matched: %s, with context: %v", pattern.Pattern.String(), conversationHistory != "")
			return p.buildParseResult(matches, pattern), nil
		}
	}

	// 没有匹配到任何模式
	return nil, ai.NewAIError(ai.ErrorTypeParsing, "no regex pattern matched", nil)
}

// checkActivityPatterns 检查活动记录模式
func (p *RegexParser) checkActivityPatterns(message string) *ai.ParseResult {
	// 记录看书活动
	// 匹配：看/读/阅读 + 书名（可选书名号）+ 可选的章节
	bookPattern := regexp.MustCompile(`(看|读|阅读|在看书|在读书).*?(?:《(.+?)》|"(.+?)"|「(.+?)」|『(.+?)』|书名[是为][:是](.+?)|(?:书|文章|本)(?:名|叫|是)?([^\s第章]+?))(?:.*?(第[一两二三四五六七八九十百千万0-9]+章|第[一两二三四五六七八九十百千万0-9]+节|Chapter\s*\d+|\d+章|Part\s*\d+))?`)
	if matches := bookPattern.FindStringSubmatch(message); len(matches) > 0 {
		// 提取书名
		var bookName string
		// 优先从带书名号的捕获组中提取
		for i := 3; i <= 8; i++ {
			if i < len(matches) && matches[i] != "" {
				bookName = strings.TrimSpace(matches[i])
				// 清理可能的冒号等标点
				bookName = strings.TrimSuffix(bookName, "：")
				bookName = strings.TrimSuffix(bookName, ":")
				bookName = strings.TrimSuffix(bookName, "。")
				if bookName != "" {
					break
				}
			}
		}

		if bookName == "" {
			// 如果没有匹配到书名，尝试从整个消息中提取
			// 查找"看/读"和"第X章"之间的内容
			afterRead := regexp.MustCompile(`(?:看|读|阅读|在看书|在读书)\s*(.+?)(?:\s+第[^\s]*?章|$)`)
			if nameMatches := afterRead.FindStringSubmatch(message); len(nameMatches) > 1 {
				bookName = strings.TrimSpace(nameMatches[1])
				bookName = strings.TrimSuffix(bookName, "：")
				bookName = strings.TrimSuffix(bookName, ":")
				bookName = strings.TrimSuffix(bookName, "。")
				// 提取第一个词或短语（避免包含"第二章"等）
				if idx := strings.Index(bookName, "第"); idx > 0 {
					bookName = strings.TrimSpace(bookName[:idx])
				}
			}
		}

		if bookName == "" {
			bookName = "未知书名"
		}

		// 提取章节（从原始消息中）
		var chapter string
		chapterPattern := regexp.MustCompile(`(第[一两二三四五六七八九十百千万0-9]+章|第[一两二三四五六七八九十百千万0-9]+节|Chapter\s*\d+|\d+章|Part\s*\d+|第二章|第二章)`)
		if chapterMatches := chapterPattern.FindStringSubmatch(message); len(chapterMatches) > 0 {
			chapter = strings.TrimSpace(chapterMatches[0])
		} else {
			chapter = "" // 如果没有章节信息，留空
		}

		details := map[string]interface{}{
			"book_name": bookName,
		}
		if chapter != "" {
			details["chapter"] = chapter
		}

		logger.Infof("Regex matched record_activity for book: %s, chapter: %s", bookName, chapter)
		return &ai.ParseResult{
			Intent:     ai.IntentRecordActivity,
			Confidence: 0.80,
			RecordActivity: &ai.ActivityRecordInfo{
				ActivityType: "read_book",
				Details:      details,
			},
			ParsedBy:    p.GetName(),
			ProcessTime: 0,
			Timestamp:   time.Now(),
		}
	}

	// 记录喝水活动
	if matched, _ := regexp.MatchString(`(喝了|喝过|饮水).*(水|杯|毫升|ml)`, message); matched {
		logger.Infof("Regex matched record_activity for water: %s", message)
		return &ai.ParseResult{
			Intent:     ai.IntentRecordActivity,
			Confidence: 0.80,
			RecordActivity: &ai.ActivityRecordInfo{
				ActivityType: "drink_water",
				Details: map[string]interface{}{
					"amount": "1杯",
				},
			},
			ParsedBy:    p.GetName(),
			ProcessTime: 0,
			Timestamp:   time.Now(),
		}
	}

	// 记录运动活动（排除删除相关词汇）
	if matched, _ := regexp.MatchString(`^(?!.*(删除|清除|去掉|移除)).*(跑步|健身|运动|锻炼|跳绳|游泳|骑车)`, message); matched {
		logger.Infof("Regex matched record_activity for exercise: %s", message)
		return &ai.ParseResult{
			Intent:     ai.IntentRecordActivity,
			Confidence: 0.80,
			RecordActivity: &ai.ActivityRecordInfo{
				ActivityType: "exercise",
				Details: map[string]interface{}{
					"type": "运动",
				},
			},
			ParsedBy:    p.GetName(),
			ProcessTime: 0,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// checkQueryPatterns 检查查询模式
func (p *RegexParser) checkQueryPatterns(message string) *ai.ParseResult {
	// 查询书籍
	if matched, _ := regexp.MatchString(`(看过|读过|阅读|书).*(哪些|多少|哪章|哪个章节|什么|进度)`, message); matched {
		logger.Infof("Regex matched query_activity for books: %s", message)
		return &ai.ParseResult{
			Intent:     ai.IntentQueryActivity,
			Confidence: 0.85,
			QueryActivity: &ai.ActivityQueryInfo{
				QueryType:    "by_type",
				ActivityType: "read_book",
			},
			ParsedBy:    p.GetName(),
			ProcessTime: 0,
			Timestamp:   time.Now(),
		}
	}

	// 查询喝水
	if matched, _ := regexp.MatchString(`(喝过|喝水|水).*(吗|多少|几次)`, message); matched {
		logger.Infof("Regex matched query_activity for water: %s", message)
		return &ai.ParseResult{
			Intent:     ai.IntentQueryActivity,
			Confidence: 0.85,
			QueryActivity: &ai.ActivityQueryInfo{
				QueryType:    "by_type",
				ActivityType: "drink_water",
			},
			ParsedBy:    p.GetName(),
			ProcessTime: 0,
			Timestamp:   time.Now(),
		}
	}

	// 查询运动
	if matched, _ := regexp.MatchString(`(运动|健身|跑步|锻炼).*(多少|几次|吗)`, message); matched {
		logger.Infof("Regex matched query_activity for exercise: %s", message)
		return &ai.ParseResult{
			Intent:     ai.IntentQueryActivity,
			Confidence: 0.85,
			QueryActivity: &ai.ActivityQueryInfo{
				QueryType:    "by_type",
				ActivityType: "exercise",
			},
			ParsedBy:    p.GetName(),
			ProcessTime: 0,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// checkDeleteActivityPatterns 检查删除活动模式
func (p *RegexParser) checkDeleteActivityPatterns(message string) *ai.ParseResult {
	// 删除书籍记录
	deleteBookPattern := regexp.MustCompile(`(?:删除|去掉|移除|清除|不要).*?(?:《(.+?)》|"(.+?)"|「(.+?)」|『(.+?)』|书名[是为][:是](.+?)).*?(?:记录|这条|那条)`)
	if matches := deleteBookPattern.FindStringSubmatch(message); len(matches) > 0 {
		var bookName string
		// 优先从带书名号的捕获组中提取
		for i := 1; i < len(matches); i++ {
			if matches[i] != "" {
				bookName = strings.TrimSpace(matches[i])
				// 清理可能的冒号等标点
				bookName = strings.TrimSuffix(bookName, "：")
				bookName = strings.TrimSuffix(bookName, ":")
				bookName = strings.TrimSuffix(bookName, "。")
				if bookName != "" {
					break
				}
			}
		}

		if bookName != "" {
			logger.Infof("Regex matched delete_activity for book: %s", bookName)
			return &ai.ParseResult{
				Intent:     ai.IntentDeleteActivity,
				Confidence: 0.85,
				DeleteActivity: &ai.DeleteActivityInfo{
					ActivityType: "read_book",
					Criteria: map[string]interface{}{
						"book_name": bookName,
					},
				},
				ParsedBy:    p.GetName(),
				ProcessTime: 0,
				Timestamp:   time.Now(),
			}
		}
	}

	// 删除喝水记录（支持时间范围）
	deleteWaterPattern := regexp.MustCompile(`(?:删除|清除|去掉|移除).*(?:昨|前|今|明|天).*?(水|喝水).*记录`)
	if matches := deleteWaterPattern.FindStringSubmatch(message); len(matches) > 0 {
		// 尝试提取时间范围
		timeRange := ""
		if strings.Contains(message, "昨天") {
			timeRange = "昨天"
		} else if strings.Contains(message, "前天") {
			timeRange = "前天"
		} else if strings.Contains(message, "今天") {
			timeRange = "今天"
		}

		criteria := map[string]interface{}{}
		if timeRange != "" {
			criteria["time_range"] = timeRange
		}

		logger.Infof("Regex matched delete_activity for water: time_range=%s", timeRange)
		return &ai.ParseResult{
			Intent:     ai.IntentDeleteActivity,
			Confidence: 0.85,
			DeleteActivity: &ai.DeleteActivityInfo{
				ActivityType: "drink_water",
				Criteria:     criteria,
			},
			ParsedBy:    p.GetName(),
			ProcessTime: 0,
			Timestamp:   time.Now(),
		}
	}

	// 删除运动记录
	if matched, _ := regexp.MatchString(`(?:删除|清除|去掉|移除).*(运动|健身|跑步|锻炼).*记录`, message); matched {
		logger.Infof("Regex matched delete_activity for exercise")
		return &ai.ParseResult{
			Intent:     ai.IntentDeleteActivity,
			Confidence: 0.85,
			DeleteActivity: &ai.DeleteActivityInfo{
				ActivityType: "exercise",
				Criteria:     map[string]interface{}{},
			},
			ParsedBy:    p.GetName(),
			ProcessTime: 0,
			Timestamp:   time.Now(),
		}
	}

	return nil
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
