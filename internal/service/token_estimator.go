package service

import (
	"strings"
	"unicode"

	"mmemory/internal/models"
)

// TokenEstimator Token估算器
type TokenEstimator struct {
	// 中文: 1字符 ≈ 2 token
	// 英文: 1词 ≈ 1 token
	// 代码/数字: 1字符 ≈ 0.5 token
}

// NewTokenEstimator 创建估算器
func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{}
}

// EstimateToken 估算文本的token数量
func (e *TokenEstimator) EstimateToken(text string) int {
	if text == "" {
		return 0
	}

	totalTokens := 0

	// 按行分割
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		totalTokens += e.estimateLineTokens(line)
	}

	return totalTokens
}

// EstimateMessagesToken 估算多条消息的总token数
func (e *TokenEstimator) EstimateMessagesToken(messages []models.ConversationMessage) int {
	total := 0
	for _, msg := range messages {
		total += e.EstimateToken(msg.Content)
		// 加上元数据的token开销(约20 token)
		total += 20
	}
	return total
}

// estimateLineTokens 估算单行的token数
func (e *TokenEstimator) estimateLineTokens(line string) int {
	if len(line) == 0 {
		return 0
	}

	chineseChars := 0
	englishWords := 0
	otherChars := 0

	inWord := false
	for _, r := range line {
		if unicode.Is(unicode.Han, r) {
			// 中文字符
			chineseChars++
		} else if unicode.IsLetter(r) || r == '\'' || r == '-' {
			// 英文字母
			inWord = true
		} else {
			// 分隔符或数字
			if inWord {
				englishWords++
				inWord = false
			}
			if unicode.IsDigit(r) || unicode.IsPunct(r) {
				otherChars++
			}
		}
	}

	// 最后一个词
	if inWord {
		englishWords++
	}

	// 计算token
	// 中文: 1字符 = 2 token
	// 英文: 1词 = 1 token
	// 其他(数字、标点): 2字符 = 1 token
	tokens := chineseChars*2 + englishWords + otherChars/2

	if tokens == 0 && len(line) > 0 {
		return 1 // 至少1 token
	}

	return tokens
}

// EstimateUsageRatio 估算使用率
func (e *TokenEstimator) EstimateUsageRatio(messages []models.ConversationMessage, maxTokens int) float64 {
	if maxTokens <= 0 {
		return 0
	}

	used := e.EstimateMessagesToken(messages)
	return float64(used) / float64(maxTokens)
}
