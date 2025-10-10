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
	Driver       string `mapstructure:"driver"`
	DSN          string `mapstructure:"dsn"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type SchedulerConfig struct {
	Timezone   string `mapstructure:"timezone"`
	MaxWorkers int    `mapstructure:"max_workers"`
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
	Enabled bool           `mapstructure:"enabled"`
	OpenAI  OpenAIConfig   `mapstructure:"openai"`
	Prompts PromptsConfig  `mapstructure:"prompts"`
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
	
	cm.viper.SetDefault("server.port", "8080")
	cm.viper.SetDefault("server.host", "0.0.0.0")
	
	cm.viper.SetDefault("scheduler.timezone", "Asia/Shanghai")
	cm.viper.SetDefault("scheduler.max_workers", 10)
	
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
	cm.viper.SetDefault("ai.openai.base_url", "https://api.openai.com/v1")
	cm.viper.SetDefault("ai.openai.primary_model", "gpt-4o-mini")
	cm.viper.SetDefault("ai.openai.backup_model", "gpt-3.5-turbo")
	cm.viper.SetDefault("ai.openai.temperature", 0.1)
	cm.viper.SetDefault("ai.openai.max_tokens", 1000)
	cm.viper.SetDefault("ai.openai.timeout", "30s")
	cm.viper.SetDefault("ai.openai.max_retries", 3)
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
	