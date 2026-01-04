package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConversationArchive_KeyEntities(t *testing.T) {
	archive := &ConversationArchive{}

	entities := &KeyEntities{
		BookName: "沉默的大多数",
		Topic:    "社会观察",
		Insights: []string{"关于社会现象的思考"},
	}

	// 测试设置
	err := archive.SetKeyEntities(entities)
	assert.NoError(t, err)
	assert.NotEmpty(t, archive.KeyEntities)

	// 测试获取
	retrieved, err := archive.GetKeyEntities()
	assert.NoError(t, err)
	assert.Equal(t, "沉默的大多数", retrieved.BookName)
	assert.Equal(t, "社会观察", retrieved.Topic)
	assert.Len(t, retrieved.Insights, 1)
}

func TestConversationArchive_ArchiveType_String(t *testing.T) {
	t.Run("Full类型", func(t *testing.T) {
		archiveType := ArchiveTypeFull
		assert.Equal(t, "full", archiveType.String())
	})

	t.Run("Summary类型", func(t *testing.T) {
		archiveType := ArchiveTypeSummary
		assert.Equal(t, "summary", archiveType.String())
	})
}

func TestConversationArchive_IsExpired(t *testing.T) {
	t.Run("已过期", func(t *testing.T) {
		archive := &ConversationArchive{}
		archive.SetExpiry(-1 * time.Hour) // 1小时前过期

		assert.True(t, archive.IsExpired())
	})

	t.Run("未过期", func(t *testing.T) {
		archive := &ConversationArchive{}
		archive.SetExpiry(1 * time.Hour) // 1小时后过期

		assert.False(t, archive.IsExpired())
	})

	t.Run("永不过期", func(t *testing.T) {
		archive := &ConversationArchive{}

		assert.False(t, archive.IsExpired())
	})
}

func TestConversationArchive_SetExpiry(t *testing.T) {
	t.Run("设置过期时间", func(t *testing.T) {
		archive := &ConversationArchive{}
		archive.SetExpiry(24 * time.Hour)

		assert.NotNil(t, archive.ExpiresAt)
	})

	t.Run("设置为永不过期", func(t *testing.T) {
		archive := &ConversationArchive{}
		archive.SetExpiry(0) // 0表示永不过期

		assert.Nil(t, archive.ExpiresAt)
	})
}
