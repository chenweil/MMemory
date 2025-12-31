package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// BotAPI 定义Bot交互的最小接口
// 这个接口只包含handlers实际使用的方法，遵循接口隔离原则(ISP)
//
// 设计原则：
// - 最小化：仅包含handlers实际需要的2个方法
// - 易测试：可使用testify/mock轻松创建测试替身
// - 易扩展：需要新方法时直接在接口添加即可
type BotAPI interface {
	// Send 发送消息（支持所有实现了Chattable的消息类型）
	// 使用场景：
	// - 发送文本消息 (MessageConfig)
	// - 发送带按钮的消息 (MessageConfig + InlineKeyboardMarkup)
	// - 编辑消息 (EditMessageTextConfig)
	// 返回值：发送成功后的消息对象和可能的错误
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)

	// Request 发送请求并返回API响应
	// 使用场景：
	// - 回调查询响应 (CallbackConfig) - 需要APIResponse而不是Message
	// 返回值：Telegram API的原始响应和可能的错误
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

// RealBotAPI 将真实的tgbotapi.BotAPI包装为接口实现
// 这是一个适配器(Adapter Pattern)，让真实的BotAPI满足我们的接口定义
//
// 用途：
// - 生产环境：使用真实的Telegram Bot API
// - 保持向后兼容：对现有代码无影响
type RealBotAPI struct {
	bot *tgbotapi.BotAPI
}

// NewRealBotAPI 创建真实BotAPI的包装器
//
// 参数：
//   - bot: 真实的tgbotapi.BotAPI实例
//
// 返回：
//   - BotAPI接口实现
//
// 使用示例：
//   realBot := bot.NewRealBotAPI(tgbotapi.NewBotAPI(token))
//   handler.HandleMessage(ctx, realBot, message)
func NewRealBotAPI(bot *tgbotapi.BotAPI) BotAPI {
	return &RealBotAPI{bot: bot}
}

// Send 实现BotAPI接口的Send方法
// 直接委托给真实的BotAPI实例
func (r *RealBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return r.bot.Send(c)
}

// Request 实现BotAPI接口的Request方法
// 直接委托给真实的BotAPI实例
func (r *RealBotAPI) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return r.bot.Request(c)
}
