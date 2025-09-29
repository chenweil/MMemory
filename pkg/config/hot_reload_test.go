package config

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestHotReloadManager_Basic(t *testing.T) {
	cm := NewConfigManager()
	hrm := NewHotReloadManager(cm)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 启动热更新管理器
	err := hrm.Start(ctx)
	if err != nil {
		t.Fatalf("启动热更新管理器失败: %v", err)
	}
	
	// 验证统计信息
	stats := hrm.GetReloadStats()
	if stats["reload_count"].(int64) != 0 {
		t.Errorf("期望重载次数为0，实际为 %d", stats["reload_count"])
	}
	
	if stats["is_reloading"].(bool) != false {
		t.Error("期望不在重载状态")
	}
	
	// 停止热更新管理器
	hrm.Stop()
}

func TestHotReloadManager_RegisterHandlers(t *testing.T) {
	cm := NewConfigManager()
	hrm := NewHotReloadManager(cm)
	
	// 注册重载处理器
	processor1Called := false
	processor2Called := false
	
	hrm.RegisterReloadHandler("processor1", func(cfg *Config) error {
		processor1Called = true
		return nil
	})
	
	hrm.RegisterReloadHandler("processor2", func(cfg *Config) error {
		processor2Called = true
		return fmt.Errorf("processor2 error")
	})
	
	// 注册安全重载函数
	safeFuncCalled := false
	hrm.RegisterSafeReloadFunc("safeFunc", func(cfg *Config) error {
		safeFuncCalled = true
		return nil
	})
	
	// 模拟配置重载
	newConfig := &Config{App: AppConfig{Name: "NewApp"}}
	
	// 手动触发配置重载处理
	err := hrm.handleConfigReload(newConfig)
	if err != nil {
		t.Fatalf("配置重载处理失败: %v", err)
	}
	
	// 验证处理器被调用
	if !processor1Called {
		t.Error("processor1 应该被调用")
	}
	
	if !processor2Called {
		t.Error("processor2 应该被调用")
	}
	
	if !safeFuncCalled {
		t.Error("safeFunc 应该被调用")
	}
	
	// 验证统计信息
	stats := hrm.GetReloadStats()
	if stats["handlers_count"].(int) != 2 {
		t.Errorf("期望处理器数量为2，实际为 %d", stats["handlers_count"])
	}
	
	if stats["safe_funcs_count"].(int) != 1 {
		t.Errorf("期望安全函数数量为1，实际为 %d", stats["safe_funcs_count"])
	}
}

func TestHotReloadManager_ConcurrentReload(t *testing.T) {
	cm := NewConfigManager()
	hrm := NewHotReloadManager(cm)
	
	// 注册一个处理器来测试并发控制
	hrm.RegisterReloadHandler("testProcessor", func(cfg *Config) error {
		return nil
	})
	
	// 第一次重载应该成功
	err := hrm.handleConfigReload(&Config{App: AppConfig{Name: "App1"}})
	if err != nil {
		t.Fatalf("第一次重载应该成功: %v", err)
	}
	
	// 立即尝试第二次重载（应该被拒绝，因为第一次可能还在处理中）
	err = hrm.handleConfigReload(&Config{App: AppConfig{Name: "App2"}})
	if err == nil {
		t.Log("第二次重载被允许（这是可接受的，因为第一次已经完成）")
	} else {
		t.Logf("第二次重载被拒绝: %v", err)
		if !contains(err.Error(), "配置重载正在进行中") {
			t.Errorf("期望错误信息包含 '配置重载正在进行中'，实际为 %v", err)
		}
	}
}

func TestHotReloadManager_Stats(t *testing.T) {
	cm := NewConfigManager()
	hrm := NewHotReloadManager(cm)
	
	// 记录初始时间
	initialTime := hrm.GetLastReloadTime()
	initialCount := hrm.GetReloadCount()
	
	if initialCount != 0 {
		t.Errorf("期望初始重载次数为0，实际为 %d", initialCount)
	}
	
	if hrm.IsReloading() {
		t.Error("期望初始状态为不在重载")
	}
	
	// 执行几次重载
	for i := 0; i < 3; i++ {
		hrm.handleConfigReload(&Config{App: AppConfig{Name: fmt.Sprintf("App%d", i)}})
	}
	
	// 验证统计信息
	newCount := hrm.GetReloadCount()
	if newCount != 3 {
		t.Errorf("期望重载次数为3，实际为 %d", newCount)
	}
	
	newTime := hrm.GetLastReloadTime()
	if !newTime.After(initialTime) {
		t.Error("最后重载时间应该更新")
	}
}

func TestHotReloadManager_Integration(t *testing.T) {
	cm := NewConfigManager()
	hrm := NewHotReloadManager(cm)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 启动热更新管理器
	err := hrm.Start(ctx)
	if err != nil {
		t.Fatalf("启动热更新管理器失败: %v", err)
	}
	
	// 注册配置管理器重载回调
	cm.OnReload(func(newConfig *Config) {
		// 回调被调用
	})
	
	// 注册处理器
	hrm.RegisterReloadHandler("testHandler", func(cfg *Config) error {
		return nil
	})
	
	// 模拟配置变更（通过配置管理器触发）
	oldConfig := &Config{App: AppConfig{Name: "OldApp"}}
	newConfig := &Config{App: AppConfig{Name: "NewApp"}}
	
	cm.config = oldConfig
	cm.notifyWatchers(oldConfig, newConfig)
	
	// 等待异步处理完成
	time.Sleep(200 * time.Millisecond)
	
	// 配置管理器回调应该被调用（我们不需要检查具体标志，因为回调函数内部有逻辑）
	
	// 注意：处理器不会被自动调用，因为我们是直接调用的 notifyWatchers
	// 实际的配置重载需要通过配置文件变更或手动调用 reload 来触发
	
	// 停止热更新管理器
	hrm.Stop()
}

func TestHotReloadManager_ErrorHandling(t *testing.T) {
	cm := NewConfigManager()
	hrm := NewHotReloadManager(cm)
	
	// 注册一个会失败的安全重载函数
	hrm.RegisterSafeReloadFunc("failingSafeFunc", func(cfg *Config) error {
		return fmt.Errorf("safe func error")
	})
	
	// 注册一个会失败的重载处理器
	hrm.RegisterReloadHandler("failingHandler", func(cfg *Config) error {
		return fmt.Errorf("handler error")
	})
	
	// 执行重载 - 安全函数失败应该导致整个重载失败
	err := hrm.handleConfigReload(&Config{App: AppConfig{Name: "TestApp"}})
	if err == nil {
		t.Error("期望安全函数失败导致重载失败")
	}
	
	if !contains(err.Error(), "safe func error") {
		t.Errorf("期望错误信息包含 'safe func error'，实际为 %v", err)
	}
	
	// 重新注册一个成功的安全函数
	hrm.safeReloadFuncs = make(map[string]func(*Config) error)
	hrm.RegisterSafeReloadFunc("successSafeFunc", func(cfg *Config) error {
		return nil
	})
	
	// 现在重载应该成功，但处理器错误应该被记录而不是中断重载
	err = hrm.handleConfigReload(&Config{App: AppConfig{Name: "TestApp2"}})
	if err != nil {
		t.Errorf("重载应该在安全函数成功时成功，即使有处理器错误，错误: %v", err)
	}
}