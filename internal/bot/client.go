package bot

import (
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// NewBotWithCustomClient 创建带有自定义HTTP客户端的Telegram Bot
func NewBotWithCustomClient(token string, debug bool) (*tgbotapi.BotAPI, error) {
	// 创建自定义HTTP客户端，优化连接配置
	client := &http.Client{
		Timeout: 120 * time.Second, // 总超时时间
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   false, // 保持连接复用
			ForceAttemptHTTP2:   true,  // 优先使用HTTP/2
		},
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = debug
	bot.Client = client // 设置自定义HTTP客户端

	return bot, nil
}