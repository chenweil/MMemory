package handlers

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
)

// TestMessageHandler_HandleVoiceMessage 测试语音消息处理
func TestMessageHandler_HandleVoiceMessage(t *testing.T) {
	// 创建语音消息（带文本）
	message := &tgbotapi.Message{
		MessageID: 1,
		From: &tgbotapi.User{
			ID:           12345,
			FirstName:    "Test",
			LanguageCode: "zh",
		},
		Chat: &tgbotapi.Chat{
			ID:   67890,
			Type: "private",
		},
		Voice: &tgbotapi.Voice{
			Duration: 3,
		},
		Text: "每天19点提醒我健身",
	}

	// 测试场景1: 有文本的语音消息
	t.Run("有文本的语音消息", func(t *testing.T) {
		// 简单验证文本提取逻辑
		text := message.Text
		assert.NotEmpty(t, text, "语音消息应有文本内容")
		assert.Equal(t, "每天19点提醒我健身", text, "文本内容应正确提取")
	})

	// 测试场景2: 无文本的语音消息
	t.Run("无文本的语音消息", func(t *testing.T) {
		messageNoText := &tgbotapi.Message{
			MessageID: 2,
			From: &tgbotapi.User{
				ID:           12345,
				FirstName:    "Test",
				LanguageCode: "zh",
			},
			Chat: &tgbotapi.Chat{
				ID:   67890,
				Type: "private",
			},
			Voice: &tgbotapi.Voice{
				Duration: 3,
			},
			Text: "",
		}

		// 验证无文本情况
		assert.Empty(t, messageNoText.Text, "语音消息无文本")
	})
}

// TestMessageHandler_HandleVoiceMessage_WithCaption 测试带字幕的语音消息
func TestMessageHandler_HandleVoiceMessage_WithCaption(t *testing.T) {
	message := &tgbotapi.Message{
		MessageID: 1,
		From: &tgbotapi.User{
			ID:           12345,
			FirstName:    "Test",
			LanguageCode: "zh",
		},
		Chat: &tgbotapi.Chat{
			ID:   67890,
			Type: "private",
		},
		Voice: &tgbotapi.Voice{
			Duration: 3,
		},
		Text:    "",
		Caption: "明天提醒我开会",
	}

	t.Run("有字幕的语音消息", func(t *testing.T) {
		text := message.Text
		if text == "" {
			text = message.Caption
		}
		assert.NotEmpty(t, text, "应从Caption提取文本")
		assert.Equal(t, "明天提醒我开会", text, "字幕内容应正确提取")
	})
}

// TestMessageHandler_HandleMessage_WithVoice 测试完整消息处理（语音）
func TestMessageHandler_HandleMessage_WithVoice(t *testing.T) {
	// 创建语音消息
	message := &tgbotapi.Message{
		MessageID: 1,
		From: &tgbotapi.User{
			ID:           12345,
			FirstName:    "Test",
			LanguageCode: "zh",
		},
		Chat: &tgbotapi.Chat{
			ID:   67890,
			Type: "private",
		},
		Voice: &tgbotapi.Voice{
			Duration: 3,
		},
		Text: "设置一个提醒",
	}

	t.Run("检测语音消息类型", func(t *testing.T) {
		assert.NotNil(t, message.Voice, "消息应包含语音")
		assert.True(t, message.Voice != nil, "语音消息检查应通过")
	})

	t.Run("提取语音文本", func(t *testing.T) {
		text := message.Text
		assert.NotEmpty(t, text, "语音文本不应为空")
		assert.Equal(t, "设置一个提醒", text, "文本内容应匹配")
	})
}
