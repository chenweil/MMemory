package config

import (
	"testing"
)

func TestLoggingConfigListener(t *testing.T) {
	listenerCalled := false
	var receivedLevel, receivedFormat, receivedOutput, receivedFilePath string
	
	// 创建监听器
	listener := NewLoggingConfigListener(func(level, format, output, filePath string) {
		listenerCalled = true
		receivedLevel = level
		receivedFormat = format
		receivedOutput = output
		receivedFilePath = filePath
	})
	
	// 测试配置变更
	oldConfig := &Config{
		Logging: LoggingConfig{
			Level:    "info",
			Format:   "json",
			Output:   "stdout",
			FilePath: "/old/path.log",
		},
	}
	
	newConfig := &Config{
		Logging: LoggingConfig{
			Level:    "debug",
			Format:   "text",
			Output:   "file",
			FilePath: "/new/path.log",
		},
	}
	
	// 触发配置变更
	listener.OnConfigChange(oldConfig, newConfig)
	
	if !listenerCalled {
		t.Error("监听器应该被调用")
	}
	
	if receivedLevel != "debug" {
		t.Errorf("期望日志级别为 debug，实际为 %s", receivedLevel)
	}
	
	if receivedFormat != "text" {
		t.Errorf("期望日志格式为 text，实际为 %s", receivedFormat)
	}
	
	if receivedOutput != "file" {
		t.Errorf("期望日志输出为 file，实际为 %s", receivedOutput)
	}
	
	if receivedFilePath != "/new/path.log" {
		t.Errorf("期望日志文件路径为 /new/path.log，实际为 %s", receivedFilePath)
	}
}

func TestLoggingConfigListener_NoChange(t *testing.T) {
	listenerCalled := false
	
	// 创建监听器
	listener := NewLoggingConfigListener(func(level, format, output, filePath string) {
		listenerCalled = true
	})
	
	// 测试相同配置（不应触发变更）
	config := &Config{
		Logging: LoggingConfig{
			Level:    "info",
			Format:   "json",
			Output:   "stdout",
			FilePath: "/path.log",
		},
	}
	
	// 触发配置变更（相同配置）
	listener.OnConfigChange(config, config)
	
	if listenerCalled {
		t.Error("监听器不应该在配置没有变更时被调用")
	}
}

func TestDatabaseConfigListener(t *testing.T) {
	listenerCalled := false
	var receivedMaxOpenConns, receivedMaxIdleConns int
	
	// 创建监听器
	listener := NewDatabaseConfigListener(func(maxOpenConns, maxIdleConns int) {
		listenerCalled = true
		receivedMaxOpenConns = maxOpenConns
		receivedMaxIdleConns = maxIdleConns
	})
	
	// 测试配置变更
	oldConfig := &Config{
		Database: DatabaseConfig{
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
	}
	
	newConfig := &Config{
		Database: DatabaseConfig{
			MaxOpenConns: 25,
			MaxIdleConns: 10,
		},
	}
	
	// 触发配置变更
	listener.OnConfigChange(oldConfig, newConfig)
	
	if !listenerCalled {
		t.Error("监听器应该被调用")
	}
	
	if receivedMaxOpenConns != 25 {
		t.Errorf("期望最大连接数为 25，实际为 %d", receivedMaxOpenConns)
	}
	
	if receivedMaxIdleConns != 10 {
		t.Errorf("期望空闲连接数为 10，实际为 %d", receivedMaxIdleConns)
	}
}

func TestDatabaseConfigListener_PartialChange(t *testing.T) {
	listenerCalled := false
	var receivedMaxOpenConns, receivedMaxIdleConns int
	
	// 创建监听器
	listener := NewDatabaseConfigListener(func(maxOpenConns, maxIdleConns int) {
		listenerCalled = true
		receivedMaxOpenConns = maxOpenConns
		receivedMaxIdleConns = maxIdleConns
	})
	
	// 测试只有部分配置变更
	oldConfig := &Config{
		Database: DatabaseConfig{
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
	}
	
	newConfig := &Config{
		Database: DatabaseConfig{
			MaxOpenConns: 20, // 只变更了这个字段
			MaxIdleConns: 5,
		},
	}
	
	// 触发配置变更
	listener.OnConfigChange(oldConfig, newConfig)
	
	if !listenerCalled {
		t.Error("监听器应该被调用")
	}
	
	if receivedMaxOpenConns != 20 {
		t.Errorf("期望最大连接数为 20，实际为 %d", receivedMaxOpenConns)
	}
	
	if receivedMaxIdleConns != 5 {
		t.Errorf("期望空闲连接数为 5，实际为 %d", receivedMaxIdleConns)
	}
}

func TestDatabaseConfigListener_NoChange(t *testing.T) {
	listenerCalled := false
	
	// 创建监听器
	listener := NewDatabaseConfigListener(func(maxOpenConns, maxIdleConns int) {
		listenerCalled = true
	})
	
	// 测试相同配置（不应触发变更）
	config := &Config{
		Database: DatabaseConfig{
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
	}
	
	// 触发配置变更（相同配置）
	listener.OnConfigChange(config, config)
	
	if listenerCalled {
		t.Error("监听器不应该在配置没有变更时被调用")
	}
}

func TestBotConfigListener(t *testing.T) {
	listenerCalled := false
	var receivedDebug bool
	
	// 创建监听器
	listener := NewBotConfigListener(func(debug bool) {
		listenerCalled = true
		receivedDebug = debug
	})
	
	// 测试配置变更
	oldConfig := &Config{
		Bot: BotConfig{
			Debug: false,
		},
	}
	
	newConfig := &Config{
		Bot: BotConfig{
			Debug: true,
		},
	}
	
	// 触发配置变更
	listener.OnConfigChange(oldConfig, newConfig)
	
	if !listenerCalled {
		t.Error("监听器应该被调用")
	}
	
	if receivedDebug != true {
		t.Errorf("期望调试模式为 true，实际为 %v", receivedDebug)
	}
}

func TestBotConfigListener_NoChange(t *testing.T) {
	listenerCalled := false
	
	// 创建监听器
	listener := NewBotConfigListener(func(debug bool) {
		listenerCalled = true
	})
	
	// 测试相同配置（不应触发变更）
	config := &Config{
		Bot: BotConfig{
			Debug: false,
		},
	}
	
	// 触发配置变更（相同配置）
	listener.OnConfigChange(config, config)
	
	if listenerCalled {
		t.Error("监听器不应该在配置没有变更时被调用")
	}
}

func TestConfigListeners_NilCallback(t *testing.T) {
	// 测试nil回调函数的处理
	
	// LoggingConfigListener with nil callback
	loggingListener := NewLoggingConfigListener(nil)
	oldConfig := &Config{
		Logging: LoggingConfig{Level: "info"},
	}
	newConfig := &Config{
		Logging: LoggingConfig{Level: "debug"},
	}
	
	// 不应该panic
	loggingListener.OnConfigChange(oldConfig, newConfig)
	
	// DatabaseConfigListener with nil callback
	databaseListener := NewDatabaseConfigListener(nil)
	oldDbConfig := &Config{
		Database: DatabaseConfig{MaxOpenConns: 10},
	}
	newDbConfig := &Config{
		Database: DatabaseConfig{MaxOpenConns: 20},
	}
	
	// 不应该panic
	databaseListener.OnConfigChange(oldDbConfig, newDbConfig)
	
	// BotConfigListener with nil callback
	botListener := NewBotConfigListener(nil)
	oldBotConfig := &Config{
		Bot: BotConfig{Debug: false},
	}
	newBotConfig := &Config{
		Bot: BotConfig{Debug: true},
	}
	
	// 不应该panic
	botListener.OnConfigChange(oldBotConfig, newBotConfig)
}

func TestConfigListeners_InterfaceImplementation(t *testing.T) {
	// 验证所有监听器都实现了 ConfigWatcher 接口
	
	loggingListener := NewLoggingConfigListener(func(level, format, output, filePath string) {})
	if _, ok := interface{}(loggingListener).(ConfigWatcher); !ok {
		t.Error("LoggingConfigListener 应该实现 ConfigWatcher 接口")
	}
	
	databaseListener := NewDatabaseConfigListener(func(maxOpenConns, maxIdleConns int) {})
	if _, ok := interface{}(databaseListener).(ConfigWatcher); !ok {
		t.Error("DatabaseConfigListener 应该实现 ConfigWatcher 接口")
	}
	
	botListener := NewBotConfigListener(func(debug bool) {})
	if _, ok := interface{}(botListener).(ConfigWatcher); !ok {
		t.Error("BotConfigListener 应该实现 ConfigWatcher 接口")
	}
}