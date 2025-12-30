package config

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	
	"mmemory/pkg/logger"
)

type Config struct {
	Bot       BotConfig       `mapstructure:"bot"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Server    ServerConfig    `mapstructure:"server"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	App       AppConfig       `mapstructure:"app"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	AI        AIConfig        `mapstructure:"ai"`
	Weather   WeatherConfig   `mapstructure:"weather"`
}

type BotConfig struct {
	Token   string        `mapstructure:"token"`
	Debug   bool          `mapstructure:"debug"`
	Webhook WebhookConfig `mapstructure:"webhook"`
}

type WebhookConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
	Port    int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Driver         string `mapstructure:"driver"`
	DSN            string `mapstructure:"dsn"`
	MaxOpenConns   int           `mapstructure:"max_open_conns"`
	MaxIdleConns   int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"` // 连接最大生命周期
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"` // 连接最大空闲时间
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"` // 最大连接生命周期
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"` // 健康检查间隔
	PoolWarmupSize int           `mapstructure:"pool_warmup_size"` // 连接池预热大小
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type SchedulerConfig struct {
	Timezone           string        `mapstructure:"timezone"`
	MaxWorkers         int           `mapstructure:"max_workers"`
	MinWorkers         int           `mapstructure:"min_workers"`             // 最小工作线程数
	WorkQueueSize      int           `mapstructure:"work_queue_size"`         // 工作队列大小
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`  // 健康检查间隔
	TaskTimeout        time.Duration `mapstructure:"task_timeout"`            // 任务执行超时
}

type LoggingConfig struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	Output   string `mapstructure:"output"`
	FilePath string `mapstructure:"file_path"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
}

type MonitoringConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

type AIConfig struct {
	Enabled           bool                        `mapstructure:"enabled"`
	PrimaryProvider   string                      `mapstructure:"primary_provider"`
	FallbackProviders []string                    `mapstructure:"fallback_providers"`
	Providers        map[string]ProviderConfig     `mapstructure:"providers"`
	Cache           CacheConfig                  `mapstructure:"cache"`
	CircuitBreaker  CircuitBreakerConfig         `mapstructure:"circuit_breaker"`
	CostControl     CostControlConfig            `mapstructure:"cost_control"`
	OpenAI          OpenAIConfig                `mapstructure:"openai"`
	Prompts         PromptsConfig               `mapstructure:"prompts"`
}

// CostControlConfig 成本控制配置
type CostControlConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	MonthlyBudget     float64       `mapstructure:"monthly_budget"`
	DailyBudget       float64       `mapstructure:"daily_budget"`
	UserBudget        float64       `mapstructure:"user_budget"`
	AlertThreshold    float64       `mapstructure:"alert_threshold"`     // 告警阈值，如0.8表示80%
	WarningThreshold  float64       `mapstructure:"warning_threshold"`   // 警告阈值，如0.6表示60%
	AutoOptimization  bool          `mapstructure:"auto_optimization"`   // 是否自动优化
	PredictionEnabled bool          `mapstructure:"prediction_enabled"`  // 是否启用成本预测
}

type OpenAIConfig struct {
	APIKey       string        `mapstructure:"api_key"`
	BaseURL      string        `mapstructure:"base_url"`
	PrimaryModel string        `mapstructure:"primary_model"`
	BackupModel  string        `mapstructure:"backup_model"`
	Temperature  float32       `mapstructure:"temperature"`
	MaxTokens    int           `mapstructure:"max_tokens"`
	Timeout      time.Duration `mapstructure:"timeout"`
	MaxRetries   int           `mapstructure:"max_retries"`
}

type PromptsConfig struct {
	ReminderParse string `mapstructure:"reminder_parse"`
	ChatResponse  string `mapstructure:"chat_response"`
}

// ProviderConfig Provider配置（多Provider用）
type ProviderConfig struct {
	Name         string        `mapstructure:"name"`
	Endpoint     string        `mapstructure:"endpoint"`
	APIKey       string        `mapstructure:"api_key"`
	Model        string        `mapstructure:"model"`
	MaxTokens    int           `mapstructure:"max_tokens"`
	Temperature  float64       `mapstructure:"temperature"`
	Timeout      time.Duration `mapstructure:"timeout"`
	RateLimit    int           `mapstructure:"rate_limit"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	TTL      time.Duration `mapstructure:"ttl"`
	MaxSize  int           `mapstructure:"max_size"`
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	FailureThreshold int           `mapstructure:"failure_threshold"`
	SuccessThreshold int           `mapstructure:"success_threshold"`
	Timeout         time.Duration `mapstructure:"timeout"`
}

type WeatherConfig struct {
	Enabled  bool            `mapstructure:"enabled"`
	Provider WeatherProvider `mapstructure:"provider"`
	Timeout  time.Duration   `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
}

type WeatherProvider struct {
	Provider string        `mapstructure:"provider"`
	QWeather QWeatherConfig `mapstructure:"qweather"`
}

type QWeatherConfig struct {
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// ConfigWatcher 配置监听器接口
type ConfigWatcher interface {
	OnConfigChange(oldConfig, newConfig *Config)
}

// ConfigManager 配置管理器
type ConfigManager struct {
	mu              sync.RWMutex
	config          *Config
	viper           *viper.Viper
	watchers        []ConfigWatcher
	watchCancel     context.CancelFunc
	reloadCallbacks []func(*Config)
}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		viper:           viper.New(),
		watchers:        make([]ConfigWatcher, 0),
		reloadCallbacks: make([]func(*Config), 0),
	}
}

// Load 加载配置（保持向后兼容）
func Load() (*Config, error) {
	manager := NewConfigManager()
	return manager.Load()
}

	// Load 加载配置
func (cm *ConfigManager) Load() (*Config, error) {
	// Only set default config paths if no config file is already specified
	if cm.viper.ConfigFileUsed() == "" && !cm.viper.IsSet("config_file") {
		cm.viper.SetConfigName("config")
		cm.viper.SetConfigType("yaml")
		cm.viper.AddConfigPath("./configs")
		cm.viper.AddConfigPath("./")
	}

	// 设置环境变量支持
	cm.setupEnvironment()

	// 设置默认值
	cm.setDefaults()

	// 读取配置文件
	if err := cm.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在，使用默认值和环境变量
	}

	var config Config
	if err := cm.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cm.validate(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	cm.mu.Lock()
	cm.config = &config
	cm.mu.Unlock()

	return &config, nil
}

// LoadFromFile 从指定文件加载配置
func (cm *ConfigManager) LoadFromFile(filePath string) (*Config, error) {
	cm.viper.SetConfigFile(filePath)
	
	// 设置环境变量支持
	cm.setupEnvironment()

	// 设置默认值
	cm.setDefaults()

	// 读取配置文件
	if err := cm.viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := cm.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cm.validate(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	cm.mu.Lock()
	cm.config = &config
	cm.mu.Unlock()

	return &config, nil
}

// setupEnvironment 设置环境变量支持
func (cm *ConfigManager) setupEnvironment() {
	cm.viper.SetEnvPrefix("MMEMORY")
	cm.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cm.viper.AutomaticEnv()
}

// setDefaults 设置配置默认值
func (cm *ConfigManager) setDefaults() {
	cm.viper.SetDefault("bot.debug", false)
	cm.viper.SetDefault("bot.webhook.enabled", false)
	cm.viper.SetDefault("bot.webhook.port", 8443)
	
	cm.viper.SetDefault("database.driver", "sqlite3")
	cm.viper.SetDefault("database.dsn", "./data/mmemory.db")
	cm.viper.SetDefault("database.max_open_conns", 25)
	cm.viper.SetDefault("database.max_idle_conns", 10)
	cm.viper.SetDefault("database.conn_max_lifetime", "5m")
	cm.viper.SetDefault("database.conn_max_idle_time", "1m")
	cm.viper.SetDefault("database.max_conn_lifetime", "30m")
	cm.viper.SetDefault("database.health_check_interval", "1m")
	cm.viper.SetDefault("database.pool_warmup_size", 5)
	
	cm.viper.SetDefault("server.port", "8080")
	cm.viper.SetDefault("server.host", "0.0.0.0")
	
	cm.viper.SetDefault("scheduler.timezone", "Asia/Shanghai")
	cm.viper.SetDefault("scheduler.max_workers", 10)
	cm.viper.SetDefault("scheduler.min_workers", 2)
	cm.viper.SetDefault("scheduler.work_queue_size", 100)
	cm.viper.SetDefault("scheduler.health_check_interval", "1m")
	cm.viper.SetDefault("scheduler.task_timeout", "30s")
	
	cm.viper.SetDefault("logging.level", "info")
	cm.viper.SetDefault("logging.format", "json")
	cm.viper.SetDefault("logging.output", "stdout")
	cm.viper.SetDefault("logging.file_path", "./data/mmemory.log")
	
	cm.viper.SetDefault("app.name", "MMemory")
	cm.viper.SetDefault("app.version", "v0.0.1")
	cm.viper.SetDefault("app.environment", "development")
	
	cm.viper.SetDefault("monitoring.enabled", true)
	cm.viper.SetDefault("monitoring.port", 9090)
	cm.viper.SetDefault("monitoring.path", "/metrics")
	
	// AI配置默认值
	cm.viper.SetDefault("ai.enabled", false)

	// 成本控制配置默认值
	cm.viper.SetDefault("ai.cost_control.enabled", true)
	cm.viper.SetDefault("ai.cost_control.monthly_budget", 100.0)
	cm.viper.SetDefault("ai.cost_control.daily_budget", 3.33) // 月预算/30
	cm.viper.SetDefault("ai.cost_control.user_budget", 10.0)
	cm.viper.SetDefault("ai.cost_control.alert_threshold", 0.9)
	cm.viper.SetDefault("ai.cost_control.warning_threshold", 0.6)
	cm.viper.SetDefault("ai.cost_control.auto_optimization", true)
	cm.viper.SetDefault("ai.cost_control.prediction_enabled", true)
	cm.viper.SetDefault("ai.openai.base_url", "https://api.openai.com/v1")
	cm.viper.SetDefault("ai.openai.primary_model", "gpt-4o-mini")
	cm.viper.SetDefault("ai.openai.backup_model", "gpt-3.5-turbo")
	cm.viper.SetDefault("ai.openai.temperature", 0.1)
	cm.viper.SetDefault("ai.openai.max_tokens", 1000)
	cm.viper.SetDefault("ai.openai.timeout", "30s")
	cm.viper.SetDefault("ai.openai.max_retries", 3)

	// 天气服务配置默认值
	cm.viper.SetDefault("weather.enabled", false)
	cm.viper.SetDefault("weather.provider.provider", "qweather")
	cm.viper.SetDefault("weather.timeout", "10s")
	cm.viper.SetDefault("weather.max_retries", 3)
	cm.viper.SetDefault("weather.provider.qweather.base_url", "https://devapi.qweather.com/v7")
	cm.viper.SetDefault("weather.provider.qweather.timeout", "5s")
}

// GetConfig 获取当前配置
func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// WatchConfig 启用配置热更新监听
func (cm *ConfigManager) WatchConfig(ctx context.Context) error {
	cm.viper.WatchConfig()
	
	cm.viper.OnConfigChange(func(e fsnotify.Event) {
		logger.Infof("配置文件发生变更: %s", e.Name)
		if err := cm.reload(); err != nil {
			logger.Errorf("配置重载失败: %v", err)
		} else {
			logger.Info("配置重载成功")
		}
	})

	// 启动上下文监听
	go func() {
		<-ctx.Done()
		cm.StopWatching()
	}()

	return nil
}

// StopWatching 停止配置监听
func (cm *ConfigManager) StopWatching() {
	if cm.watchCancel != nil {
		cm.watchCancel()
	}
}

// reload 重载配置
func (cm *ConfigManager) reload() error {
	var newConfig Config
	if err := cm.viper.Unmarshal(&newConfig); err != nil {
		return fmt.Errorf("解析新配置失败: %w", err)
	}

	if err := cm.validate(&newConfig); err != nil {
		return fmt.Errorf("新配置验证失败: %w", err)
	}

	cm.mu.Lock()
	oldConfig := cm.config
	cm.config = &newConfig
	cm.mu.Unlock()

	// 通知监听器
	cm.notifyWatchers(oldConfig, &newConfig)
	
	// 执行重载回调
	cm.executeReloadCallbacks(&newConfig)

	return nil
}

// AddWatcher 添加配置变更监听器
func (cm *ConfigManager) AddWatcher(watcher ConfigWatcher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.watchers = append(cm.watchers, watcher)
}

// OnReload 添加重载回调函数
func (cm *ConfigManager) OnReload(callback func(*Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.reloadCallbacks = append(cm.reloadCallbacks, callback)
}

// notifyWatchers 通知所有监听器
func (cm *ConfigManager) notifyWatchers(oldConfig, newConfig *Config) {
	for _, watcher := range cm.watchers {
		go func(w ConfigWatcher) {
			w.OnConfigChange(oldConfig, newConfig)
		}(watcher)
	}
}

// executeReloadCallbacks 执行重载回调
func (cm *ConfigManager) executeReloadCallbacks(newConfig *Config) {
	for _, callback := range cm.reloadCallbacks {
		go callback(newConfig)
	}
}

// validate 验证配置
func (cm *ConfigManager) validate(config *Config) error {
	var errors []string

	// 验证Bot配置
	if config.Bot.Token == "" {
		errors = append(errors, "Telegram Bot Token不能为空")
	}
	
	// 仅在Token不为空时验证格式（允许测试使用短Token）
	if config.Bot.Token != "" && len(config.Bot.Token) < 40 {
		errors = append(errors, "Telegram Bot Token格式不正确")
	}

	// 验证数据库配置
	if config.Database.DSN == "" {
		errors = append(errors, "数据库DSN不能为空")
	}

	if config.Database.MaxOpenConns <= 0 {
		errors = append(errors, "数据库最大连接数必须大于0")
	}

	if config.Database.MaxIdleConns < 0 {
		errors = append(errors, "数据库空闲连接数不能为负数")
	}

	// 验证服务器配置
	if config.Server.Port == "" {
		errors = append(errors, "服务器端口不能为空")
	}

	if config.Server.Host == "" {
		errors = append(errors, "服务器主机地址不能为空")
	}

	// 验证调度器配置
	if config.Scheduler.Timezone == "" {
		errors = append(errors, "调度器时区不能为空")
	}

	if config.Scheduler.MaxWorkers <= 0 {
		errors = append(errors, "调度器最大工作线程数必须大于0")
	}

	// 验证日志配置
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[config.Logging.Level] {
		errors = append(errors, "日志级别必须是 debug、info、warn 或 error")
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[config.Logging.Format] {
		errors = append(errors, "日志格式必须是 json 或 text")
	}

	validLogOutputs := map[string]bool{"stdout": true, "file": true, "both": true}
	if !validLogOutputs[config.Logging.Output] {
		errors = append(errors, "日志输出必须是 stdout、file 或 both")
	}

	// 验证监控配置
	if config.Monitoring.Enabled {
		if config.Monitoring.Port <= 0 || config.Monitoring.Port > 65535 {
			errors = append(errors, "监控端口必须在1-65535范围内")
		}

		if config.Monitoring.Path == "" {
			errors = append(errors, "监控路径不能为空")
		}
	}

	// 验证天气服务配置
	if config.Weather.Enabled {
		if config.Weather.Provider.Provider == "qweather" {
			// 开发环境跳过API Key验证，允许使用模拟数据
			if config.App.Environment != "development" && config.Weather.Provider.QWeather.APIKey == "" {
				errors = append(errors, "和风天气API Key不能为空")
			}
			if config.Weather.Provider.QWeather.BaseURL == "" {
				errors = append(errors, "和风天气BaseURL不能为空")
			}
		}

		if config.Weather.Timeout <= 0 {
			errors = append(errors, "天气服务超时时间必须大于0")
		}

		if config.Weather.MaxRetries < 0 {
			errors = append(errors, "天气服务重试次数不能为负数")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("配置验证失败:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// GetString 获取字符串配置值
func (cm *ConfigManager) GetString(key string) string {
	return cm.viper.GetString(key)
}

// GetInt 获取整数配置值
func (cm *ConfigManager) GetInt(key string) int {
	return cm.viper.GetInt(key)
}

// GetBool 获取布尔配置值
func (cm *ConfigManager) GetBool(key string) bool {
	return cm.viper.GetBool(key)
}

// GetDuration 获取时长配置值
func (cm *ConfigManager) GetDuration(key string) time.Duration {
	return cm.viper.GetDuration(key)
}

// IsSet 检查配置项是否已设置
func (cm *ConfigManager) IsSet(key string) bool {
	return cm.viper.IsSet(key)
}

// Set 设置配置值（用于测试）
func (cm *ConfigManager) Set(key string, value interface{}) {
	cm.viper.Set(key, value)
}
	