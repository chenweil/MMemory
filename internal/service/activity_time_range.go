package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var recentDaysPattern = regexp.MustCompile(`(\d{1,3})\s*天`)
var chineseDaysPattern = regexp.MustCompile(`([一二三四五六七八九十]{1,3})\s*天`)

// resolveActivityTimeRange 解析活动查询时间范围并返回统一的显示文案
func resolveActivityTimeRange(timeRange string, now time.Time) (time.Time, time.Time, string) {
	normalized := strings.TrimSpace(timeRange)
	if normalized == "" {
		normalized = "最近7天"
	}

	switch normalized {
	case "今天":
		startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return startTime, startTime.Add(24 * time.Hour), "今天"
	case "昨天":
		yesterday := now.AddDate(0, 0, -1)
		startTime := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
		return startTime, startTime.Add(24 * time.Hour), "昨天"
	case "这周", "本周":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startTime := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return startTime, startTime.AddDate(0, 0, 7), "这周"
	case "最近7天", "近7天":
		return now.AddDate(0, 0, -7), now, "最近7天"
	case "最近30天", "近30天":
		return now.AddDate(0, 0, -30), now, "最近30天"
	case "这月", "本月":
		startTime := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return startTime, startTime.AddDate(0, 1, 0), "本月"
	}

	if days, ok := parseRecentDays(normalized); ok {
		return now.AddDate(0, 0, -days), now, fmt.Sprintf("最近%d天", days)
	}

	return now.AddDate(0, 0, -7), now, "最近7天"
}

func parseRecentDays(timeRange string) (int, bool) {
	match := recentDaysPattern.FindStringSubmatch(timeRange)
	if len(match) > 1 {
		days, err := strconv.Atoi(match[1])
		if err == nil && days > 0 {
			return days, true
		}
	}

	match = chineseDaysPattern.FindStringSubmatch(timeRange)
	if len(match) > 1 {
		days, ok := parseChineseNumber(match[1])
		if ok && days > 0 {
			return days, true
		}
	}

	return 0, false
}

func parseChineseNumber(value string) (int, bool) {
	digits := map[rune]int{
		'一': 1,
		'二': 2,
		'三': 3,
		'四': 4,
		'五': 5,
		'六': 6,
		'七': 7,
		'八': 8,
		'九': 9,
	}

	if value == "十" {
		return 10, true
	}

	if strings.Contains(value, "十") {
		parts := strings.Split(value, "十")
		tens := 0
		if parts[0] == "" {
			tens = 1
		} else if len(parts[0]) == 1 {
			if digit, ok := digits[[]rune(parts[0])[0]]; ok {
				tens = digit
			} else {
				return 0, false
			}
		} else {
			return 0, false
		}

		ones := 0
		if len(parts) > 1 && parts[1] != "" {
			if len(parts[1]) != 1 {
				return 0, false
			}
			if digit, ok := digits[[]rune(parts[1])[0]]; ok {
				ones = digit
			} else {
				return 0, false
			}
		}

		return tens*10 + ones, true
	}

	if len([]rune(value)) == 1 {
		if digit, ok := digits[[]rune(value)[0]]; ok {
			return digit, true
		}
	}

	return 0, false
}
