package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mmemory/internal/bot/handlers"
	"mmemory/internal/repository/sqlite"
	"mmemory/internal/service"
	"mmemory/pkg/config"
	"mmemory/pkg/logger"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	if err := logger.Init(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output, cfg.Logging.FilePath); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	logger.Infof("🚀 启动 %s %s", cfg.App.Name, cfg.App.Version)

	// 初始化数据库
	database, err := sqlite.NewDatabase(&cfg.Database)
	if err != nil {
		logger.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	logger.Info("✅ 数据库连接成功")

	// 初始化仓储层
	userRepo := sqlite.NewUserRepository(database.GetDB())
	reminderRepo := sqlite.NewReminderRepository(database.GetDB())

	// 初始化服务层
	userService := service.NewUserService(userRepo)
	reminderService := service.NewReminderService(reminderRepo)

	// 初始化Telegram Bot
	bot, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
	if err != nil {
		logger.Fatalf("创建Telegram Bot失败: %v", err)
	}

	bot.Debug = cfg.Bot.Debug
	logger.Infof("✅ Telegram Bot 授权成功: @%s", bot.Self.UserName)

	// 初始化消息处理器
	messageHandler := handlers.NewMessageHandler(reminderService, userService)

	// 启动Bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("🔄 收到停止信号，正在关闭...")
		cancel()
	}()

	// 启动消息处理循环
	if err := startBot(ctx, bot, messageHandler); err != nil {
		logger.Fatalf("Bot运行失败: %v", err)
	}

	logger.Info("👋 程序已退出")
}

func startBot(ctx context.Context, bot *tgbotapi.BotAPI, messageHandler *handlers.MessageHandler) error {
	logger.Info("🤖 Bot开始接收消息...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			logger.Info("停止接收消息")
			bot.StopReceivingUpdates()
			return nil

		case update := <-updates:
			if update.Message == nil {
				continue
			}

			// 在goroutine中处理消息，避免阻塞
			go func(msg *tgbotapi.Message) {
				if err := messageHandler.HandleMessage(ctx, bot, msg); err != nil {
					logger.Errorf("处理消息失败: %v", err)
				}
			}(update.Message)
		}
	}
}