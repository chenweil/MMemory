package service

import (
	"time"

	"mmemory/internal/models"
)

// TopicSwitchDetector 话题切换检测器
type TopicSwitchDetector struct {
	timeThreshold   time.Duration
	intentThreshold int
}

// NewTopicSwitchDetector 创建检测器
func NewTopicSwitchDetector(timeThresholdMinutes int, intentThreshold int) *TopicSwitchDetector {
	return &TopicSwitchDetector{
		timeThreshold:   time.Duration(timeThresholdMinutes) * time.Minute,
		intentThreshold: intentThreshold,
	}
}

// DetectSwitch 检测话题是否切换
// 返回: (是否切换, 是否频繁切换)
func (d *TopicSwitchDetector) DetectSwitch(messages []models.ConversationMessage) (bool, bool) {
	if len(messages) < 2 {
		return false, false
	}

	// 1. 检测时间间隔
	timeSwitch := d.detectTimeInterval(messages)

	// 2. 检测意图变化
	intentSwitch, frequentSwitch := d.detectIntentChange(messages)

	// 3. 综合判断
	switched := timeSwitch || intentSwitch

	return switched, frequentSwitch
}

// detectTimeInterval 检测时间间隔
func (d *TopicSwitchDetector) detectTimeInterval(messages []models.ConversationMessage) bool {
	if len(messages) < 2 {
		return false
	}

	// 检查最近两条消息的时间间隔
	last := messages[len(messages)-1]
	secondLast := messages[len(messages)-2]

	if last.Timestamp.Sub(secondLast.Timestamp) > d.timeThreshold {
		return true
	}

	return false
}

// detectIntentChange 检测意图变化
func (d *TopicSwitchDetector) detectIntentChange(messages []models.ConversationMessage) (bool, bool) {
	if len(messages) < d.intentThreshold+1 {
		return false, false
	}

	// 统计最近的意图变化次数
	changes := 0
	recent := messages[len(messages)-d.intentThreshold-1:]

	for i := 1; i < len(recent); i++ {
		if recent[i].Intent != recent[i-1].Intent {
			changes++
		}
	}

	// 至少有1次意图变化才算切换
	switched := changes >= 1

	// 达到阈值才算频繁切换
	frequent := changes >= d.intentThreshold

	return switched, frequent
}
