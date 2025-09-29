package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mmemory/pkg/logger"
)

func TestIntegration_CompleteConfigWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}
	
	// 初始化logger以避免nil指针错误
	if err := logger.Init("info", "text", "stdout", ""); err != nil {
		t.Fatalf("初始化logger失败: %v", err)
	}
	
	// 创建临时目录
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	
	// 初始配置内容
	initialConfig := `
bot:
  token: "test_token_that_is_long_enough_for_validation_to_pass"
  debug: false
  webhook:
    enabled: false
    url: ""
    port: 8443

database:
  driver: "sqlite3"
  dsn: "./data/integration_test.db"
  max_open_conns: 25
  max_idle_conns: 10

server:
  port: "8080"
  host: "0.0.0.0"
  
scheduler:
  timezone: "Asia/Shanghai"
  max_workers: 10
  
logging:
  level: "info"
  format: "json"
  output: "stdout"
  file_path: "./data/integration_test.log"
  
app:
  name: "MMemoryIntegration"
  version: "v0.0.1"
  environment: "testing"

monitoring:
  enabled: true
  port: 9090
  path: "/metrics"
`
	
	// 写入初始配置文件
	if err := os.WriteFile(configFile, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("创建初始配置文件失败: %v", err)
	}
	
	// 创建配置管理器
	cm := NewConfigManager()
	
	// 加载初始配置
	cfg, err := cm.LoadFromFile(configFile)
	if err != nil {
		t.Fatalf("加载初始配置失败: %v", err)
	}
	
	// 验证初始配置
	if cfg.App.Name != "MMemoryIntegration" {
		t.Errorf("期望应用名称为 MMemoryIntegration，实际为 %s", cfg.App.Name)
	}
	
	if cfg.Logging.Level != "info" {
		t.Errorf("期望日志级别为 info，实际为 %s", cfg.Logging.Level)
	}
	
	// 创建热更新管理器
	hrm := NewHotReloadManager(cm)
	
	// 设置上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 启动热更新管理器
	if err := hrm.Start(ctx); err != nil {
		t.Fatalf("启动热更新管理器失败: %v", err)
	}
	
	// 注册各种监听器和处理器
	var (
		configReloaded    bool
	)
	
	// 日志配置监听器
	cm.AddWatcher(NewLoggingConfigListener(func(level, format, output, filePath string) {
		t.Logf("日志配置已更新: level=%s, format=%s, output=%s, file_path=%s", level, format, output, filePath)
	}))
	
	// 数据库配置安全重载
	hrm.RegisterSafeReloadFunc("database", func(newConfig *Config) error {
		t.Logf("数据库配置已更新: max_open_conns=%d, max_idle_conns=%d",
			newConfig.Database.MaxOpenConns, newConfig.Database.MaxIdleConns)
		return nil
	})
	
	// 注册一个普通的重载处理器
	hrm.RegisterReloadHandler("testHandler", func(newConfig *Config) error {
		t.Logf("普通重载处理器被调用")
		return nil
	})
	
	// Bot配置监听器
	cm.AddWatcher(NewBotConfigListener(func(debug bool) {
		t.Logf("Bot调试模式已更新: debug=%v", debug)
	}))
	
	// 通用重载回调
	cm.OnReload(func(newConfig *Config) {
		configReloaded = true
		t.Logf("配置重载完成: app=%s, version=%s, environment=%s",
			newConfig.App.Name, newConfig.App.Version, newConfig.App.Environment)
	})
	
	// 验证器
	validator := GetDefaultValidator()
	
	// 等待初始状态稳定
	time.Sleep(100 * time.Millisecond)
	
	// 测试1: 修改配置文件
	t.Run("配置文件热更新", func(t *testing.T) {
		updatedConfig := `
bot:
  token: "test_token_that_is_long_enough_for_validation_to_pass"
  debug: true  # 修改了这里
  webhook:
    enabled: false
    url: ""
    port: 8443

database:
  driver: "sqlite3"
  dsn: "./data/integration_test.db"
  max_open_conns: 50  # 修改了这里
  max_idle_conns: 15  # 修改了这里

server:
  port: "9090"  # 修改了这里
  host: "0.0.0.0"
  
scheduler:
  timezone: "Asia/Shanghai"
  max_workers: 10
  
logging:
  level: "debug"  # 修改了这里
  format: "text"  # 修改了这里
  output: "file"  # 修改了这里
  file_path: "./data/integration_test_updated.log"  # 修改了这里
  
app:
  name: "MMemoryIntegrationUpdated"  # 修改了这里
  version: "v0.0.2"  # 修改了这里
  environment: "testing"

monitoring:
  enabled: true
  port: 9090
  path: "/metrics"
`
		
		// 重置标志
		configReloaded = false
		
		// 写入更新后的配置
		if err := os.WriteFile(configFile, []byte(updatedConfig), 0644); err != nil {
			t.Fatalf("更新配置文件失败: %v", err)
		}
		
		// 等待文件系统事件和配置重载
		time.Sleep(500 * time.Millisecond)
		
		// 验证配置已更新
		currentCfg := cm.GetConfig()
		if currentCfg.App.Name != "MMemoryIntegrationUpdated" {
			t.Errorf("期望应用名称已更新为 MMemoryIntegrationUpdated，实际为 %s", currentCfg.App.Name)
		}
		
		if currentCfg.Logging.Level != "debug" {
			t.Errorf("期望日志级别已更新为 debug，实际为 %s", currentCfg.Logging.Level)
		}
		
		if currentCfg.Bot.Debug != true {
			t.Errorf("期望Bot调试模式已更新为 true，实际为 %v", currentCfg.Bot.Debug)
		}
		
		if currentCfg.Database.MaxOpenConns != 50 {
			t.Errorf("期望数据库最大连接数已更新为 50，实际为 %d", currentCfg.Database.MaxOpenConns)
		}
		
		// 验证监听器被调用
		if !configReloaded {
			t.Error("通用重载回调应该被调用")
		}
	})
	
	// 测试2: 验证配置验证
	t.Run("配置验证", func(t *testing.T) {
		currentCfg := cm.GetConfig()
		result := validator.Validate(currentCfg)
		
		if !result.IsValid {
			t.Errorf("当前配置应该通过验证，错误: %v", result.Errors)
			for i, err := range result.Errors {
				t.Logf("验证错误 %d: Field=%s, Message=%s", i, err.Field, err.Message)
			}
		}
		
		// 测试无效配置
		invalidConfig := *currentCfg
		invalidConfig.Bot.Token = "" // 使配置无效
		
		result = validator.Validate(&invalidConfig)
		if result.IsValid {
			t.Error("无效配置应该验证失败")
		}
		
		if len(result.Errors) == 0 {
			t.Error("无效配置应该有验证错误")
		}
	})
	
	// 测试3: 统计信息
	t.Run("统计信息", func(t *testing.T) {
		stats := hrm.GetReloadStats()
		
		if stats["reload_count"].(int64) == 0 {
			t.Error("应该至少有一次重载")
		}
		
		if stats["handlers_count"].(int) == 0 {
			t.Error("应该有注册的处理器")
		}
		
		if stats["safe_funcs_count"].(int) == 0 {
			t.Error("应该有注册的安全函数")
		}
		
		t.Logf("重载统计: %+v", stats)
	})
	
	// 测试4: 并发重载测试
	t.Run("并发重载", func(t *testing.T) {
		// 注册一个慢速处理器
		hrm.RegisterReloadHandler("slowProcessor", func(cfg *Config) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
		
		// 尝试并发重载
		done := make(chan bool, 2)
		
		go func() {
			hrm.handleConfigReload(cm.GetConfig())
			done <- true
		}()
		
		// 等待第一个重载开始
		time.Sleep(50 * time.Millisecond)
		
		go func() {
			err := hrm.handleConfigReload(cm.GetConfig())
			if err != nil {
				t.Logf("第二个重载被拒绝（期望的行为）: %v", err)
			}
			done <- true
		}()
		
		// 等待两个重载尝试完成
		<-done
		<-done
	})
	
	// 清理
	hrm.Stop()
	t.Log("集成测试完成")
}

func TestIntegration_EnvironmentVariableOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}
	
	// 设置环境变量
	os.Setenv("MMEMORY_APP_NAME", "EnvOverrideApp")
	os.Setenv("MMEMORY_LOGGING_LEVEL", "warn")
	os.Setenv("MMEMORY_SERVER_PORT", "3000")
	defer func() {
		os.Unsetenv("MMEMORY_APP_NAME")
		os.Unsetenv("MMEMORY_LOGGING_LEVEL")
		os.Unsetenv("MMEMORY_SERVER_PORT")
	}()
	
	// 创建配置管理器
	cm := NewConfigManager()
	
	// 设置一个有效的测试token以通过验证（环境变量不会覆盖token，所以需要设置）
	cm.Set("bot.token", "test_token_that_is_long_enough_for_validation_to_pass")
	
	// 加载配置（应该使用环境变量覆盖）
	cfg, err := cm.Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	
	// 验证环境变量覆盖
	if cfg.App.Name != "EnvOverrideApp" {
		t.Errorf("期望应用名称被环境变量覆盖为 EnvOverrideApp，实际为 %s", cfg.App.Name)
	}
	
	if cfg.Logging.Level != "warn" {
		t.Errorf("期望日志级别被环境变量覆盖为 warn，实际为 %s", cfg.Logging.Level)
	}
	
	if cfg.Server.Port != "3000" {
		t.Errorf("期望服务器端口被环境变量覆盖为 3000，实际为 %s", cfg.Server.Port)
	}
}

func TestIntegration_DefaultValues(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}
	
	// 创建配置管理器，不设置配置文件
	cm := NewConfigManager()
	
	// 设置一个有效的测试token以通过验证
	cm.Set("bot.token", "test_token_that_is_long_enough_for_validation_to_pass")
	
	// 加载配置（应该使用默认值）
	cfg, err := cm.Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	
	// 验证默认值
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
}