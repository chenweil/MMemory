package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mmemory/internal/bot"
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

	// 初始化Telegram Bot（使用自定义HTTP客户端）
	bot, err := bot.NewBotWithCustomClient(cfg.Bot.Token, cfg.Bot.Debug)
	if err != nil {
		logger.Fatalf("创建Telegram Bot失败: %v", err)
	}

	logger.Infof("✅ Telegram Bot 授权成功: @%s", bot.Self.UserName)

	// 初始化服务层
	userService := service.NewUserService(userRepo)
	reminderService := service.NewReminderService(reminderRepo)
	reminderLogService := service.NewReminderLogService(reminderLogRepo, reminderRepo)
	notificationService := service.NewNotificationService(bot)
	schedulerService := service.NewSchedulerService(reminderRepo, reminderLogRepo, notificationService)

	// 建立服务之间的依赖关系
	if reminderServiceWithScheduler, ok := reminderService.(interface{ SetScheduler(service.SchedulerService) }); ok {
		reminderServiceWithScheduler.SetScheduler(schedulerService)
	}

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

// isEOFError 检查是否为EOF相关错误
func isEOFError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "EOF") || 
		strings.Contains(errStr, "unexpected EOF") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe")
}

// logTelegramError 记录Telegram相关错误，区分错误类型
func logTelegramError(err error, operation string) {
	if isEOFError(err) {
		logger.Warnf("Telegram API 连接错误 [%s]: %v (类型: EOF错误/网络中断)", operation, err)
	} else {
		logger.Errorf("Telegram API 错误 [%s]: %v (类型: %T)", operation, err, err)
	}
}

func startBot(ctx context.Context, bot *tgbotapi.BotAPI, messageHandler *handlers.MessageHandler, callbackHandler *handlers.CallbackHandler) error {
	logger.Info("🤖 Bot开始接收消息...")

	maxRetries := 3
	retryDelay := 5 * time.Second
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("停止接收消息")
			bot.StopReceivingUpdates()
			return nil
			
		default:
			if err := runUpdatesWithRetry(ctx, bot, messageHandler, callbackHandler, maxRetries, retryDelay); err != nil {
				logger.Errorf("Bot运行失败，即将重试: %v", err)
				time.Sleep(retryDelay)
				continue
			}
		}
	}
}

func runUpdatesWithRetry(ctx context.Context, bot *tgbotapi.BotAPI, messageHandler *handlers.MessageHandler, callbackHandler *handlers.CallbackHandler, maxRetries int, retryDelay time.Duration) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30 // 减少超时时间到30秒，降低网络中断风险

	// 获取更新通道 (GetUpdatesChan 不返回错误，只返回通道)
	updates := bot.GetUpdatesChan(u)

	// 处理更新
	return processUpdates(ctx, updates, bot, messageHandler, callbackHandler)
}

func processUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, messageHandler *handlers.MessageHandler, callbackHandler *handlers.CallbackHandler) error {
	consecutiveErrors := 0
	maxConsecutiveErrors := 10
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("停止接收消息")
			return nil

		case update, ok := <-updates:
			if !ok {
				return fmt.Errorf("更新通道已关闭")
			}
			
			// 重置连续错误计数
			consecutiveErrors = 0
			
			// 处理消息
			if update.Message != nil {
				go func(msg *tgbotapi.Message) {
					if err := messageHandler.HandleMessage(ctx, bot, msg); err != nil {
						logTelegramError(err, "处理消息")
					}
				}(update.Message)
			}

			// 处理回调查询
			if update.CallbackQuery != nil {
				go func(callback *tgbotapi.CallbackQuery) {
					if err := callbackHandler.HandleCallback(ctx, bot, callback); err != nil {
						logTelegramError(err, "处理回调")
					}
				}(update.CallbackQuery)
			}
			
		case <-time.After(5 * time.Minute):
			// 5分钟内没有收到任何更新，记录心跳日志
			logger.Debug("🫀 Bot心跳检测：运行正常，暂无新消息")
			consecutiveErrors++
			
			if consecutiveErrors > maxConsecutiveErrors {
				logger.Warn("连续多次没有收到更新，可能存在连接问题")
				return fmt.Errorf("连接可能存在问题，需要重新初始化")
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
