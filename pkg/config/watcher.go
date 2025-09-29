package config

// ConfigChangeListener 配置变更监听器接口
type ConfigChangeListener interface {
	// OnConfigChange 当配置发生变更时调用
	OnConfigChange(oldConfig, newConfig *Config)
}

// LoggingConfigListener 日志配置变更监听器
type LoggingConfigListener struct {
	reloadFunc func(level, format, output, filePath string)
}

// NewLoggingConfigListener 创建日志配置监听器
func NewLoggingConfigListener(reloadFunc func(level, format, output, filePath string)) *LoggingConfigListener {
	return &LoggingConfigListener{
		reloadFunc: reloadFunc,
	}
}

// OnConfigChange 处理日志配置变更
func (l *LoggingConfigListener) OnConfigChange(oldConfig, newConfig *Config) {
	if oldConfig.Logging.Level != newConfig.Logging.Level ||
		oldConfig.Logging.Format != newConfig.Logging.Format ||
		oldConfig.Logging.Output != newConfig.Logging.Output ||
		oldConfig.Logging.FilePath != newConfig.Logging.FilePath {
		
		if l.reloadFunc != nil {
			l.reloadFunc(
				newConfig.Logging.Level,
				newConfig.Logging.Format,
				newConfig.Logging.Output,
				newConfig.Logging.FilePath,
			)
		}
	}
}

// DatabaseConfigListener 数据库配置变更监听器
type DatabaseConfigListener struct {
	reloadFunc func(maxOpenConns, maxIdleConns int)
}

// NewDatabaseConfigListener 创建数据库配置监听器
func NewDatabaseConfigListener(reloadFunc func(maxOpenConns, maxIdleConns int)) *DatabaseConfigListener {
	return &DatabaseConfigListener{
		reloadFunc: reloadFunc,
	}
}

// OnConfigChange 处理数据库配置变更
func (d *DatabaseConfigListener) OnConfigChange(oldConfig, newConfig *Config) {
	if oldConfig.Database.MaxOpenConns != newConfig.Database.MaxOpenConns ||
		oldConfig.Database.MaxIdleConns != newConfig.Database.MaxIdleConns {
		
		if d.reloadFunc != nil {
			d.reloadFunc(
				newConfig.Database.MaxOpenConns,
				newConfig.Database.MaxIdleConns,
			)
		}
	}
}

// BotConfigListener Bot配置变更监听器
type BotConfigListener struct {
	reloadFunc func(debug bool)
}

// NewBotConfigListener 创建Bot配置监听器
func NewBotConfigListener(reloadFunc func(debug bool)) *BotConfigListener {
	return &BotConfigListener{
		reloadFunc: reloadFunc,
	}
}

// OnConfigChange 处理Bot配置变更
func (b *BotConfigListener) OnConfigChange(oldConfig, newConfig *Config) {
	if oldConfig.Bot.Debug != newConfig.Bot.Debug {
		if b.reloadFunc != nil {
			b.reloadFunc(newConfig.Bot.Debug)
		}
	}
}