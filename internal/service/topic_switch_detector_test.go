package service

import (
	"testing"
	"time"

	"mmemory/internal/models"
)

func TestTopicSwitchDetector_DetectSwitch(t *testing.T) {
	// 创建检测器: 30分钟时间阈值, 2次意图变化阈值
	detector := NewTopicSwitchDetector(30, 2)

	t.Run("时间间隔检测", func(t *testing.T) {
		now := time.Now()
		messages := []models.ConversationMessage{
			{Intent: "chat", Timestamp: now.Add(-40 * time.Minute)},
			{Intent: "chat", Timestamp: now},
		}

		switched, frequent := detector.DetectSwitch(messages)
		if !switched {
			t.Errorf("期望检测到话题切换(时间间隔)")
		}
		if frequent {
			t.Errorf("不应该是频繁切换")
		}
	})

	t.Run("意图变化检测", func(t *testing.T) {
		now := time.Now()
		messages := []models.ConversationMessage{
			{Intent: "reminder", Timestamp: now.Add(-5 * time.Minute)},
			{Intent: "chat", Timestamp: now.Add(-3 * time.Minute)},
			{Intent: "reminder", Timestamp: now},
		}

		switched, frequent := detector.DetectSwitch(messages)
		if !switched {
			t.Errorf("期望检测到话题切换(意图变化)")
		}
		if !frequent {
			t.Errorf("应该检测到频繁切换")
		}
	})

	t.Run("无切换", func(t *testing.T) {
		now := time.Now()
		messages := []models.ConversationMessage{
			{Intent: "chat", Timestamp: now.Add(-5 * time.Minute)},
			{Intent: "chat", Timestamp: now},
		}

		switched, _ := detector.DetectSwitch(messages)
		if switched {
			t.Errorf("不应该检测到话题切换")
		}
	})
}
