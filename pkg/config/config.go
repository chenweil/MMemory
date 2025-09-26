package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Bot       BotConfig       `mapstructure:"bot"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Server    ServerConfig    `mapstructure:"server"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	App       AppConfig       `mapstructure:"app"`
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

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("./")

	// 设置环境变量前缀
	viper.SetEnvPrefix("MMEMORY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证必需的配置
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("bot.debug", false)
	viper.SetDefault("bot.webhook.enabled", false)
	viper.SetDefault("bot.webhook.port", 8443)
	
	viper.SetDefault("database.driver", "sqlite3")
	viper.SetDefault("database.dsn", "./data/mmemory.db")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	
	viper.SetDefault("scheduler.timezone", "Asia/Shanghai")
	viper.SetDefault("scheduler.max_workers", 10)
	
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.file_path", "./data/mmemory.log")
	
	viper.SetDefault("app.name", "MMemory")
	viper.SetDefault("app.version", "v0.0.1")
	viper.SetDefault("app.environment", "development")
}

func validate(config *Config) error {
	if config.Bot.Token == "" {
		return fmt.Errorf("Telegram Bot Token不能为空")
	}
	
	if config.Database.DSN == "" {
		return fmt.Errorf("数据库DSN不能为空")
	}
	
	return nil
}