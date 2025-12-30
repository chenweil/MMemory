package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/mock"

	"mmemory/internal/bot"
)

// MockBotAPI Mock实现的BotAPI接口
// 用于单元测试，可以精确控制和验证BotAPI的调用
//
// 使用testify/mock框架，提供：
// - 方法调用期望设置 (On)
// - 返回值控制 (Return)
// - 调用验证 (AssertExpectations)
// - 精确参数匹配 (MatchedBy)
//
// 使用示例：
//   mockBot := new(MockBotAPI)
//   mockBot.On("Send", mock.MatchedBy(func(c tgbotapi.Chattable) bool {
//       msg := c.(tgbotapi.MessageConfig)
//       return msg.ChatID == 123 && strings.Contains(msg.Text, "欢迎")
//   })).Return(tgbotapi.Message{}, nil)
//
//   handler.handleStartCommand(mockBot, message)
//
//   mockBot.AssertExpectations(t)
type MockBotAPI struct {
	mock.Mock
}

// Send Mock实现BotAPI的Send方法
//
// 测试中使用方式：
//
// 1. 简单Mock（不关心参数）:
//   mockBot.On("Send", mock.Anything).Return(tgbotapi.Message{}, nil)
//
// 2. 精确Mock（验证消息内容）:
//   mockBot.On("Send", mock.MatchedBy(func(c tgbotapi.Chattable) bool {
//       msg, ok := c.(tgbotapi.MessageConfig)
//       if !ok {
//           return false
//       }
//       return msg.ChatID == 123 &&
//              strings.Contains(msg.Text, "欢迎") &&
//              msg.ParseMode == tgbotapi.ModeHTML
//   })).Return(tgbotapi.Message{MessageID: 456}, nil)
//
// 3. 模拟错误：
//   mockBot.On("Send", mock.Anything).Return(tgbotapi.Message{}, errors.New("network error"))
func (m *MockBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	args := m.Called(c)
	return args.Get(0).(tgbotapi.Message), args.Error(1)
}

// Request Mock实现BotAPI的Request方法
//
// 测试中使用方式：
//
// 1. 成功响应：
//   mockBot.On("Request", mock.Anything).
//       Return(&tgbotapi.APIResponse{Ok: true}, nil)
//
// 2. 验证CallbackConfig：
//   mockBot.On("Request", mock.MatchedBy(func(c tgbotapi.Chattable) bool {
//       cb, ok := c.(tgbotapi.CallbackConfig)
//       if !ok {
//           return false
//       }
//       return cb.CallbackQueryID == "callback123" &&
//              strings.Contains(cb.Text, "已完成")
//   })).Return(&tgbotapi.APIResponse{Ok: true}, nil)
//
// 3. 模拟API错误：
//   mockBot.On("Request", mock.Anything).
//       Return(nil, errors.New("API error"))
func (m *MockBotAPI) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	args := m.Called(c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tgbotapi.APIResponse), args.Error(1)
}

// 确保MockBotAPI实现了bot.BotAPI接口
var _ bot.BotAPI = (*MockBotAPI)(nil)
