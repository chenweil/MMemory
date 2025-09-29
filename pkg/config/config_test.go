package config

import (
	"fmt"
	"testing"
	"time"
)

func TestConfigManager_Load(t *testing.T) {
	// 测试配置管理器的基本加载功能
	tests := []struct {
		name    string
		setup   func(*ConfigManager)
		wantErr bool
		check   func(*Config)
	}{
		{
			name: "使用默认值",
			setup: func(cm *ConfigManager) {
				// 设置一个有效的测试token以通过验证
				cm.Set("bot.token", "test_token_that_is_long_enough_for_validation_to_pass")
			},
			wantErr: false,
			check: func(cfg *Config) {
				if cfg.Database.Driver != "sqlite3" {
					t.Errorf("期望数据库驱动默认为 sqlite3，实际为 %s", cfg.Database.Driver)
				}
				if cfg.Scheduler.Timezone != "Asia/Shanghai" {
					t.Errorf("期望时区默认为 Asia/Shanghai，实际为 %s", cfg.Scheduler.Timezone)
				}
				if cfg.Logging.Level != "info" {
					t.Errorf("期望日志级别默认为 info，实际为 %s", cfg.Logging.Level)
				}
				if cfg.Monitoring.Enabled != true {
					t.Errorf("期望监控默认启用，实际为 %v", cfg.Monitoring.Enabled)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewConfigManager()
			if tt.setup != nil {
				tt.setup(cm)
			}
			
			cfg, err := cm.Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(cfg)
			}
		})
	}
}

func TestConfigManager_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效配置",
			config: &Config{
				Bot: BotConfig{
					Token: "test_token_that_is_long_enough_for_validation_to_pass",
				},
				Database: DatabaseConfig{
					Driver:       "sqlite3",
					DSN:          "./data/test.db",
					MaxOpenConns: 25,
					MaxIdleConns: 10,
				},
				Server: ServerConfig{
					Port: "8080",
					Host: "0.0.0.0",
				},
				Scheduler: SchedulerConfig{
					Timezone:   "Asia/Shanghai",
					MaxWorkers: 10,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				App: AppConfig{
					Name:        "MMemory",
					Version:     "v0.0.1",
					Environment: "development",
				},
				Monitoring: MonitoringConfig{
					Enabled: true,
					Port:    9090,
					Path:    "/metrics",
				},
			},
			wantErr: false,
		},
		{
			name: "空Bot Token",
			config: &Config{
				Bot: BotConfig{
					Token: "",
				},
				Database: DatabaseConfig{
					Driver:       "sqlite3",
					DSN:          "./data/test.db",
					MaxOpenConns: 25,
					MaxIdleConns: 10,
				},
				Server: ServerConfig{
					Port: "8080",
					Host: "0.0.0.0",
				},
				Scheduler: SchedulerConfig{
					Timezone:   "Asia/Shanghai",
					MaxWorkers: 10,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				App: AppConfig{
					Name:        "MMemory",
					Version:     "v0.0.1",
					Environment: "development",
				},
			},
			wantErr: true,
			errMsg:  "字段不能为空",
		},
		{
			name: "无效日志级别",
			config: &Config{
				Bot: BotConfig{
					Token: "test_token_that_is_long_enough_for_validation_to_pass",
				},
				Database: DatabaseConfig{
					Driver:       "sqlite3",
					DSN:          "./data/test.db",
					MaxOpenConns: 25,
					MaxIdleConns: 10,
				},
				Server: ServerConfig{
					Port: "8080",
					Host: "0.0.0.0",
				},
				Scheduler: SchedulerConfig{
					Timezone:   "Asia/Shanghai",
					MaxWorkers: 10,
				},
				Logging: LoggingConfig{
					Level:  "invalid",
					Format: "json",
					Output: "stdout",
				},
				App: AppConfig{
					Name:        "MMemory",
					Version:     "v0.0.1",
					Environment: "development",
				},
			},
			wantErr: true,
			errMsg:  "无效的日志级别",
		},
		{
			name: "无效端口",
			config: &Config{
				Bot: BotConfig{
					Token: "test_token_that_is_long_enough_for_validation_to_pass",
				},
				Database: DatabaseConfig{
					Driver:       "sqlite3",
					DSN:          "./data/test.db",
					MaxOpenConns: 25,
					MaxIdleConns: 10,
				},
				Server: ServerConfig{
					Port: "99999",
					Host: "0.0.0.0",
				},
				Scheduler: SchedulerConfig{
					Timezone:   "Asia/Shanghai",
					MaxWorkers: 10,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				App: AppConfig{
					Name:        "MMemory",
					Version:     "v0.0.1",
					Environment: "development",
				},
			},
			wantErr: true,
			errMsg:  "端口必须在1-65535范围内",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := GetDefaultValidator()
			result := validator.Validate(tt.config)
			var err error
			if !result.IsValid {
				err = fmt.Errorf("validation failed: %v", result.Errors)
			}
			
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("期望错误信息包含 %q，实际错误为 %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestConfigManager_Getters(t *testing.T) {
	cm := NewConfigManager()
	cm.Set("test.string", "hello")
	cm.Set("test.int", 42)
	cm.Set("test.bool", true)
	cm.Set("test.duration", "5s")
	
	tests := []struct {
		name     string
		key      string
		expected interface{}
		getter   func() interface{}
	}{
		{
			name:     "获取字符串",
			key:      "test.string",
			expected: "hello",
			getter:   func() interface{} { return cm.GetString("test.string") },
		},
		{
			name:     "获取整数",
			key:      "test.int",
			expected: 42,
			getter:   func() interface{} { return cm.GetInt("test.int") },
		},
		{
			name:     "获取布尔值",
			key:      "test.bool",
			expected: true,
			getter:   func() interface{} { return cm.GetBool("test.bool") },
		},
		{
			name:     "获取时长",
			key:      "test.duration",
			expected: 5 * time.Second,
			getter:   func() interface{} { return cm.GetDuration("test.duration") },
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result != tt.expected {
				t.Errorf("期望 %v，实际 %v", tt.expected, result)
			}
		})
	}
}

func TestConfigManager_ConfigChange(t *testing.T) {
	cm := NewConfigManager()
	
	// 初始配置
	initialConfig := &Config{
		App: AppConfig{
			Name:    "InitialApp",
			Version: "v1.0.0",
		},
	}
	
	cm.config = initialConfig
	
	// 新配置
	newConfig := &Config{
		App: AppConfig{
			Name:    "UpdatedApp",
			Version: "v2.0.0",
		},
	}
	
	// 添加监听器
	listenerCalled := false
	var oldCfg, newCfg *Config
	
	cm.AddWatcher(&testConfigWatcher{
		onChange: func(old, new *Config) {
			listenerCalled = true
			oldCfg = old
			newCfg = new
		},
	})
	
	// 触发重载
	cm.notifyWatchers(initialConfig, newConfig)
	
	// 等待异步调用完成
	time.Sleep(100 * time.Millisecond)
	
	if !listenerCalled {
		t.Error("监听器未被调用")
	}
	
	if oldCfg.App.Name != "InitialApp" {
		t.Errorf("期望旧配置名称为 InitialApp，实际为 %s", oldCfg.App.Name)
	}
	
	if newCfg.App.Name != "UpdatedApp" {
		t.Errorf("期望新配置名称为 UpdatedApp，实际为 %s", newCfg.App.Name)
	}
}

// 辅助类型和函数

type testConfigWatcher struct {
	onChange func(oldConfig, newConfig *Config)
}

func (t *testConfigWatcher) OnConfigChange(oldConfig, newConfig *Config) {
	if t.onChange != nil {
		t.onChange(oldConfig, newConfig)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}