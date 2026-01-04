package main

import (
	"context"
	"fmt"
	"log"
	"math"
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
	"mmemory/pkg/ai"
	"mmemory/pkg/config"
	"mmemory/pkg/logger"
	"mmemory/pkg/server"
	"mmemory/pkg/version"
)

func main() {
	// 创建配置管理器
	configManager := config.NewConfigManager()

	// 加载配置
	cfg, err := configManager.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	if err := logger.Init(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output, cfg.Logging.FilePath); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 打印版本信息
	versionInfo := version.GetInfo()
	logger.Infof("🚀 启动 %s %s", cfg.App.Name, version.GetVersionString())
	logger.Infof("📦 版本详情: Git=%s, Branch=%s, BuildTime=%s",
		versionInfo.GitCommit, versionInfo.GitBranch, version.FormatBuildTime())
	logger.Infof("🖥️  运行环境: %s (%s)", versionInfo.Platform, versionInfo.GoVersion)

	// 创建热更新管理器
	hotReloadManager := config.NewHotReloadManager(configManager)

	// 注册配置变更监听器
	setupConfigListeners(configManager, hotReloadManager)

	// 初始化数据库
	database, err := sqlite.NewDatabase(&cfg.Database)
	if err != nil {
		logger.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	logger.Info("✅ 数据库连接成功")

	// 初始化仓储层
	userRepo := sqlite.NewUserRepository(database.GetDB())
	reminderRepo := sqlite.NewReminderRepository(database.GetDB(), database.GetQueryOptimizer())
	reminderLogRepo := sqlite.NewReminderLogRepository(database.GetDB())
	conversationRepo := sqlite.NewConversationRepository(database.GetDB())
	conversationContextRepo := sqlite.NewConversationContextRepository(database.GetDB())
	conversationArchiveRepo := sqlite.NewConversationArchiveRepository(database.GetDB())
	dailyActivityRepo := sqlite.NewDailyActivityRepository(database.GetDB())

	// 初始化Telegram Bot（使用自定义HTTP客户端）
	botAPI, err := bot.NewBotWithCustomClient(cfg.Bot.Token, cfg.Bot.Debug)
	if err != nil {
		logger.Fatalf("创建Telegram Bot失败: %v", err)
	}

	logger.Infof("✅ Telegram Bot 授权成功: @%s", botAPI.Self.UserName)

	// 包装为接口实现（用于handlers测试）
	botInterface := bot.NewRealBotAPI(botAPI)

	// 初始化服务层
	userService := service.NewUserService(userRepo)
	reminderService := service.NewReminderService(reminderRepo)
	reminderLogService := service.NewReminderLogService(reminderLogRepo, reminderRepo)
	notificationService := service.NewNotificationService(botAPI) // NotificationService仍使用原始BotAPI
	schedulerService := service.NewSchedulerService(reminderRepo, reminderLogRepo, notificationService)
	monitoringService := service.NewMonitoringService(userRepo, reminderRepo, reminderLogRepo)
	conversationService := service.NewConversationService(conversationRepo)
	dailyActivityService := service.NewDailyActivityService(dailyActivityRepo)

	// 临时创建一个基础的ContextManager（稍后会替换为带Token管理的版本）
	contextManager := service.NewContextManager(
		conversationContextRepo,
		&service.DefaultEntityExtractor{},
		&service.DefaultIntentTracker{},
		service.ContextManagerConfig{},
		nil, // tokenManager - 先传nil
		nil, // archiveService - 先传nil
		0,   // maxTokens - 使用默认值
	)

	suggestionService := service.NewReminderSuggestionService(
		reminderRepo,
		reminderLogRepo,
		contextManager,
		service.SuggestionServiceConfig{},
	)

	// 初始化活动可视化服务
	activityVisualizationService := service.NewActivityVisualizationService(dailyActivityRepo)

	// 初始化智能分析服务
	activityAnalysisService := service.NewActivityAnalysisService(dailyActivityRepo)

	// 初始化AI服务（如果启用）
	var aiParserService service.AIParserService
	if cfg.AI.Enabled {
		logger.Info("🤖 AI功能已启用")

		// 获取默认配置
		defaultConfig := ai.GetDefaultAIConfig()

		// 转换配置格式，空值使用默认值
		aiConfig := &ai.AIConfig{
			Enabled: cfg.AI.Enabled,
			OpenAI: ai.OpenAIConfig{
				APIKey:       cfg.AI.OpenAI.APIKey,
				BaseURL:      cfg.AI.OpenAI.BaseURL,
				PrimaryModel: cfg.AI.OpenAI.PrimaryModel,
				BackupModel:  cfg.AI.OpenAI.BackupModel,
				Temperature:  cfg.AI.OpenAI.Temperature,
				MaxTokens:    cfg.AI.OpenAI.MaxTokens,
				Timeout:      cfg.AI.OpenAI.Timeout,
				MaxRetries:   cfg.AI.OpenAI.MaxRetries,
			},
			Prompts: ai.PromptsConfig{
				ReminderParse: cfg.AI.Prompts.ReminderParse,
				ChatResponse:  cfg.AI.Prompts.ChatResponse,
			},
		}

		// 确保Prompt包含天气功能（无论用户是否自定义）
		if aiConfig.Prompts.ReminderParse == "" {
			aiConfig.Prompts.ReminderParse = defaultConfig.Prompts.ReminderParse
			logger.Info("✅ 使用默认的ReminderParse Prompt模板（包含天气功能）")
		} else if !strings.Contains(aiConfig.Prompts.ReminderParse, "天气查询") {
			// 如果自定义Prompt中没有天气功能，则使用默认值
			logger.Warn("⚠️ 检测到自定义Prompt缺少天气功能，已替换为默认模板")
			aiConfig.Prompts.ReminderParse = defaultConfig.Prompts.ReminderParse
		} else {
			logger.Info("✅ 使用自定义ReminderParse Prompt模板（已包含天气功能）")
		}

		if aiConfig.Prompts.ChatResponse == "" {
			aiConfig.Prompts.ChatResponse = defaultConfig.Prompts.ChatResponse
			logger.Info("✅ 使用默认的ChatResponse Prompt模板")
		}

		// 验证AI配置
		if err := aiConfig.Validate(); err != nil {
			logger.Warnf("AI配置验证失败，将禁用AI功能: %v", err)
		} else {
			// 创建AIParserService
			aiParserService, err = service.NewAIParserService(aiConfig)
			if err != nil {
				logger.Warnf("初始化AI解析服务失败，将禁用AI功能: %v", err)
				aiParserService = nil
			} else {
				logger.Info("✅ AI解析服务初始化成功")
			}
		}
	} else {
		logger.Info("ℹ️ AI功能未启用，使用传统解析器")
	}

	// 重新创建带Token管理功能的ContextManager（Task 8: 集成新服务）
	var aiClientForArchive ai.AIClient
	if aiParserService != nil {
		// TODO: 从AIParserService获取AI客户端（需要进一步集成）
		// 目前先使用nil，ArchiveService会使用降级策略
		logger.Info("ℹ️ 存档服务的AI摘要功能暂未集成，将使用降级策略")
	}

	// 创建存档服务
	conversationArchiveService := service.NewConversationArchiveService(
		conversationArchiveRepo,
		aiClientForArchive,
		nil, // config - 将在Task 9中添加
	)

	// 创建Token管理器（默认128k tokens，适合GLM-4）
	contextTokenManager := service.NewContextTokenManagerService(
		conversationArchiveService,
		128000, // 128k tokens (GLM-4 max)
	)

	// 重新创建上下文管理器（集成Token管理和归档服务）
	contextManager = service.NewContextManager(
		conversationContextRepo,
		&service.DefaultEntityExtractor{},
		&service.DefaultIntentTracker{},
		service.ContextManagerConfig{},
		contextTokenManager,
		conversationArchiveService,
		128000, // maxTokens - 与Token管理器保持一致
	)

	// 重新创建SuggestionService（使用新的ContextManager）
	suggestionService = service.NewReminderSuggestionService(
		reminderRepo,
		reminderLogRepo,
		contextManager,
		service.SuggestionServiceConfig{},
	)

	logger.Info("✅ 上下文Token管理和归档服务已集成")

	// 建立服务之间的依赖关系
	if reminderServiceWithScheduler, ok := reminderService.(interface {
		SetScheduler(service.SchedulerService)
	}); ok {
		reminderServiceWithScheduler.SetScheduler(schedulerService)
	}

	// 启动监控服务
	var metricsServer *server.MetricsServer
	var monitoringCtx context.Context
	var monitoringCancel context.CancelFunc

	if cfg.Monitoring.Enabled {
		metricsServer = server.NewMetricsServer(cfg.Monitoring.Port)
		if err := metricsServer.Start(); err != nil {
			logger.Fatalf("启动指标服务器失败: %v", err)
		}

		// 启动监控服务
		monitoringCtx, monitoringCancel = context.WithCancel(context.Background())
		if err := monitoringService.Start(monitoringCtx); err != nil {
			logger.Fatalf("启动监控服务失败: %v", err)
		}
	}

	// 初始化天气服务（如果启用）
	if cfg.Weather.Enabled {
		logger.Info("🌤️ 天气服务已启用")
		_ = service.NewQWeatherService(&cfg.Weather)
	} else {
		logger.Info("ℹ️ 天气服务未启用，将使用模拟数据")
	}

	// 初始化消息处理器
	messageHandler := handlers.NewMessageHandler(reminderService, userService, reminderLogService, aiParserService, conversationService, contextManager, suggestionService, dailyActivityService, activityVisualizationService, activityAnalysisService)
	callbackHandler := handlers.NewCallbackHandler(reminderService, reminderLogService, schedulerService)

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

		// 停止热更新管理器
		if hotReloadManager != nil {
			hotReloadManager.Stop()
			logger.Info("配置热更新管理器已停止")
		}

		// 停止监控服务
		if cfg.Monitoring.Enabled {
			if monitoringCancel != nil {
				monitoringCancel()
			}
			if monitoringService != nil {
				monitoringService.Stop()
			}
			if metricsServer != nil {
				metricsServer.Stop(context.Background())
			}
		}

		cancel()
	}()

	// 启动消息处理循环（传入原始BotAPI用于GetUpdatesChan，传入接口用于handlers）
	if err := startBot(ctx, botAPI, botInterface, messageHandler, callbackHandler); err != nil {
		logger.Fatalf("Bot运行失败: %v", err)
	}

	logger.Info("👋 程序已退出")
}

// setupConfigListeners 设置配置变更监听器
func setupConfigListeners(configManager *config.ConfigManager, hotReloadManager *config.HotReloadManager) {
	ctx := context.Background()

	// 启动热更新管理器
	if err := hotReloadManager.Start(ctx); err != nil {
		logger.Warnf("启动配置热更新失败: %v", err)
	} else {
		logger.Info("配置热更新管理器已启动")
	}

	// 注册日志配置监听器
	loggingListener := config.NewLoggingConfigListener(func(level, format, output, filePath string) {
		logger.Infof("检测到日志配置变更，重新初始化日志系统")
		if err := logger.Init(level, format, output, filePath); err != nil {
			logger.Errorf("日志配置热更新失败: %v", err)
		} else {
			logger.Info("日志配置热更新成功")
		}
	})
	configManager.AddWatcher(loggingListener)

	// 注册数据库配置监听器（安全重载）
	hotReloadManager.RegisterSafeReloadFunc("database", func(newConfig *config.Config) error {
		logger.Infof("检测到数据库配置变更，连接池参数更新: max_open_conns=%d, max_idle_conns=%d",
			newConfig.Database.MaxOpenConns, newConfig.Database.MaxIdleConns)
		// 这里可以添加数据库连接池的动态调整逻辑
		return nil
	})

	// 注册Bot配置监听器
	botListener := config.NewBotConfigListener(func(debug bool) {
		logger.Infof("检测到Bot配置变更，调试模式: %v", debug)
		// 这里可以添加Bot调试模式的动态调整逻辑
	})
	configManager.AddWatcher(botListener)

	// 注册通用的重载回调
	configManager.OnReload(func(newConfig *config.Config) {
		logger.Infof("配置重载完成，当前版本: %s, 环境: %s",
			newConfig.App.Version, newConfig.App.Environment)
	})
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           // 最大重试次数
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	BackoffFactor   float64       // 退避因子
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      5,
		InitialDelay:    1 * time.Second,
		MaxDelay:        60 * time.Second,
		BackoffFactor:   2.0,
	}
}

// calculateBackoff 计算退避延迟
func calculateBackoff(attempt int, config RetryConfig) time.Duration {
	if attempt == 0 {
		return 0
	}
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	return time.Duration(delay)
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

// isTransientError 检查是否为临时错误（应该重试）
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "deadline exceeded")
}

// isFatalError 检查是否为致命错误（不应该重试）
func isFatalError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Unauthorized") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "conflict") ||
		strings.Contains(errStr, "invalid token") ||
		strings.Contains(errStr, "bot was blocked")
}

// logTelegramError 记录Telegram相关错误，区分错误类型
func logTelegramError(err error, operation string) {
	if isEOFError(err) {
		logger.Warnf("Telegram API 连接错误 [%s]: %v (类型: EOF错误/网络中断)", operation, err)
	} else {
		logger.Errorf("Telegram API 错误 [%s]: %v (类型: %T)", operation, err, err)
	}
}

func startBot(ctx context.Context, botAPI *tgbotapi.BotAPI, botInterface bot.BotAPI, messageHandler *handlers.MessageHandler, callbackHandler *handlers.CallbackHandler) error {
	logger.Info("🤖 Bot开始接收消息...")

	// 使用默认重试配置（0 表示使用函数内部的默认值）
	maxRetries := 0
	var retryDelay time.Duration = 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("停止接收消息")
			botAPI.StopReceivingUpdates()
			return nil

		default:
			if err := runUpdatesWithRetry(ctx, botAPI, botInterface, messageHandler, callbackHandler, maxRetries, retryDelay); err != nil {
				logger.Errorf("Bot运行失败，等待后重试: %v", err)
				// 使用指数退避的外层延迟
				time.Sleep(10 * time.Second)
				continue
			}
		}
	}
}

func runUpdatesWithRetry(ctx context.Context, botAPI *tgbotapi.BotAPI, botInterface bot.BotAPI, messageHandler *handlers.MessageHandler, callbackHandler *handlers.CallbackHandler, maxRetries int, retryDelay time.Duration) error {
	config := DefaultRetryConfig()
	// 如果调用者提供了参数，使用调用者的参数
	if maxRetries > 0 {
		config.MaxRetries = maxRetries
	}
	if retryDelay > 0 {
		config.InitialDelay = retryDelay
	}

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60 // 设置超时为60秒，与HTTP客户端的120秒总超时保持合理比例

		// 获取更新通道 (GetUpdatesChan 不返回错误，只返回通道)
		updates := botAPI.GetUpdatesChan(u)

		// 处理更新（传入接口供handlers使用）
		err := processUpdates(ctx, updates, botInterface, messageHandler, callbackHandler)

		if err == nil {
			// 成功，无需重试
			return nil
		}

		// 检查是否为致命错误
		if isFatalError(err) {
			logger.Errorf("Fatal error encountered, giving up: %v", err)
			return fmt.Errorf("fatal error: %w", err)
		}

		// 检查是否还有重试机会
		if attempt < config.MaxRetries {
			delay := calculateBackoff(attempt+1, config)
			// 只在特定重试次数时记录详细日志，避免日志刷屏
			if attempt == 0 || attempt == 2 || attempt == 4 || attempt == config.MaxRetries-1 {
				logger.Warnf("Retry attempt %d/%d failed, retrying in %v: %v", attempt+1, config.MaxRetries, delay, err)
			} else {
				logger.Debugf("Retry attempt %d/%d failed, retrying in %v: %v", attempt+1, config.MaxRetries, delay, err)
			}
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("max retries (%d) exceeded", config.MaxRetries)
}

func processUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel, botInterface bot.BotAPI, messageHandler *handlers.MessageHandler, callbackHandler *handlers.CallbackHandler) error {
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

			// 处理消息（使用接口）
			if update.Message != nil {
				go func(msg *tgbotapi.Message) {
					if err := messageHandler.HandleMessage(ctx, botInterface, msg); err != nil {
						logTelegramError(err, "处理消息")
					}
				}(update.Message)
			}

			// 处理回调查询（使用接口）
			if update.CallbackQuery != nil {
				go func(callback *tgbotapi.CallbackQuery) {
					if err := callbackHandler.HandleCallback(ctx, botInterface, callback); err != nil {
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
