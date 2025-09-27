package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	reminderLogRepo := sqlite.NewReminderLogRepository(database.GetDB())

	// 初始化Telegram Bot
	bot, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
	if err != nil {
		logger.Fatalf("创建Telegram Bot失败: %v", err)
	}

	bot.Debug = cfg.Bot.Debug
	logger.Infof("✅ Telegram Bot 授权成功: @%s", bot.Self.UserName)

	// 初始化服务层
	userService := service.NewUserService(userRepo)
	reminderService := service.NewReminderService(reminderRepo)
	reminderLogService := service.NewReminderLogService(reminderLogRepo, reminderRepo)
	notificationService := service.NewNotificationService(bot)
	schedulerService := service.NewSchedulerService(reminderRepo, reminderLogRepo, notificationService)

	// 建立服务之间的依赖关系
	reminderService.SetScheduler(schedulerService)

	// 初始化消息处理器
	messageHandler := handlers.NewMessageHandler(reminderService, userService, reminderLogService)
	callbackHandler := handlers.NewCallbackHandler(reminderLogService, schedulerService)

	// 启动调度器
	if err := schedulerService.Start(); err != nil {
		logger.Fatalf("启动调度器失败: %v", err)
	}
	defer schedulerService.Stop()

	// 启动超时处理器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startOvertimeProcessor(ctx, reminderLogService, notificationService)

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("🔄 收到停止信号，正在关闭...")
		cancel()
	}()

	// 启动消息处理循环
	if err := startBot(ctx, bot, messageHandler, callbackHandler); err != nil {
		logger.Fatalf("Bot运行失败: %v", err)
	}

	logger.Info("👋 程序已退出")
}

func startBot(ctx context.Context, bot *tgbotapi.BotAPI, messageHandler *handlers.MessageHandler, callbackHandler *handlers.CallbackHandler) error {
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
			// 处理消息
			if update.Message != nil {
				go func(msg *tgbotapi.Message) {
					if err := messageHandler.HandleMessage(ctx, bot, msg); err != nil {
						logger.Errorf("处理消息失败: %v", err)
					}
				}(update.Message)
			}

			// 处理回调查询
			if update.CallbackQuery != nil {
				go func(callback *tgbotapi.CallbackQuery) {
					if err := callbackHandler.HandleCallback(ctx, bot, callback); err != nil {
						logger.Errorf("处理回调失败: %v", err)
					}
				}(update.CallbackQuery)
			}
		}
	}
}

// startOvertimeProcessor 启动超时处理器
func startOvertimeProcessor(ctx context.Context, reminderLogService service.ReminderLogService, notificationService service.NotificationService) {
	logger.Info("⏰ 超时处理器启动")
	
	ticker := time.NewTicker(30 * time.Minute) // 每30分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("超时处理器停止")
			return
		case <-ticker.C:
			// 检查超时的提醒
			overdueLogs, err := reminderLogService.GetOverdueReminders(ctx)
			if err != nil {
				logger.Errorf("获取超时提醒失败: %v", err)
				continue
			}

			for _, log := range overdueLogs {
				// 发送关怀消息
				if err := notificationService.SendFollowUp(ctx, log); err != nil {
					logger.Errorf("发送关怀消息失败 (LogID: %d): %v", log.ID, err)
					continue
				}

				// 更新关怀次数
				if err := reminderLogService.UpdateFollowUpCount(ctx, log.ID); err != nil {
					logger.Errorf("更新关怀次数失败 (LogID: %d): %v", log.ID, err)
				}
				
				logger.Debugf("💌 已发送关怀消息: LogID=%d, 次数=%d", log.ID, log.FollowUpCount+1)
			}

			if len(overdueLogs) > 0 {
				logger.Infof("📤 处理了 %d 个超时提醒", len(overdueLogs))
			}
		}
	}
}
