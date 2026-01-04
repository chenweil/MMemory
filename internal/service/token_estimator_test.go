package service

import (
	"testing"

	"mmemory/internal/models"
)

func TestTokenEstimator_EstimateToken(t *testing.T) {
	estimator := NewTokenEstimator()

	tests := []struct {
		name     string
		text     string
		min      int // 最小期望值
		max      int // 最大期望值
	}{
		{
			name: "纯中文",
			text: "这是一段中文文本",
			min:  15, // 8个中文字符 × 2 ≈ 16
			max:  20,
		},
		{
			name: "纯英文",
			text: "This is English text",
			min:  3, // 4个词
			max:  5,
		},
		{
			name: "中英混合",
			text: "Hello 世界",
			min:  5, // 1英文词 + 2中文×2
			max:  7,
		},
		{
			name: "空字符串",
			text: "",
			min:  0,
			max:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimator.EstimateToken(tt.text)
			if result < tt.min || result > tt.max {
				t.Errorf("EstimateToken() = %v, expected range [%v, %v]", result, tt.min, tt.max)
			}
		})
	}
}

func TestTokenEstimator_EstimateMessagesToken(t *testing.T) {
	estimator := NewTokenEstimator()

	messages := []models.ConversationMessage{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "Hello"},
	}

	result := estimator.EstimateMessagesToken(messages)
	expected := 2*2 + 1 + 20*2 // 中文4 + 英文1 + 元数据40
	if result < expected*9/10 || result > expected*11/10 {
		t.Errorf("EstimateMessagesToken() = %v, expected around %v", result, expected)
	}
}

func TestTokenEstimator_EstimateUsageRatio(t *testing.T) {
	estimator := NewTokenEstimator()

	messages := []models.ConversationMessage{
		{Role: "user", Content: "这是一段测试文本"},
	}

	ratio := estimator.EstimateUsageRatio(messages, 1000)
	// "这是一段测试文本" = 8个中文字符 × 2 = 16 tokens
	// 加上20 token元数据 = 36 tokens
	// 36 / 1000 = 0.036
	expected := 36.0 / 1000.0
	if ratio < expected*0.9 || ratio > expected*1.1 {
		t.Errorf("EstimateUsageRatio() = %v, expected around %v", ratio, expected)
	}
}
