package config

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// HotReloadManager 配置热更新管理器
type HotReloadManager struct {
	mu              sync.RWMutex
	manager         *ConfigManager
	ctx             context.Context
	cancel          context.CancelFunc
	reloadHandlers  map[string]func(*Config) error
	safeReloadFuncs map[string]func(*Config) error
	lastReloadTime  time.Time
	reloadCount     int64
	isReloading     bool
}

// NewHotReloadManager 创建热更新管理器
func NewHotReloadManager(manager *ConfigManager) *HotReloadManager {
	return &HotReloadManager{
		manager:         manager,
		reloadHandlers:  make(map[string]func(*Config) error),
		safeReloadFuncs: make(map[string]func(*Config) error),
		lastReloadTime:  time.Now(),
	}
}

// Start 启动热更新管理
func (h *HotReloadManager) Start(ctx context.Context) error {
	h.mu.Lock()
	h.ctx, h.cancel = context.WithCancel(ctx)
	h.mu.Unlock()

	// 注册配置管理器的重载回调
	h.manager.OnReload(func(newConfig *Config) {
		if err := h.handleConfigReload(newConfig); err != nil {
			log.Printf("配置热更新处理失败: %v", err)
		}
	})

	// 启动配置监听
	if err := h.manager.WatchConfig(ctx); err != nil {
		return fmt.Errorf("启动配置监听失败: %w", err)
	}

	log.Println("配置热更新管理器已启动")
	return nil
}

// Stop 停止热更新管理
func (h *HotReloadManager) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cancel != nil {
		h.cancel()
	}

	log.Println("配置热更新管理器已停止")
}

// RegisterReloadHandler 注册重载处理器
func (h *HotReloadManager) RegisterReloadHandler(name string, handler func(*Config) error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.reloadHandlers[name] = handler
	log.Printf("注册配置重载处理器: %s", name)
}

// RegisterSafeReloadFunc 注册安全重载函数
func (h *HotReloadManager) RegisterSafeReloadFunc(name string, reloadFunc func(*Config) error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.safeReloadFuncs[name] = reloadFunc
	log.Printf("注册安全配置重载函数: %s", name)
}

// handleConfigReload 处理配置重载
func (h *HotReloadManager) handleConfigReload(newConfig *Config) error {
	h.mu.Lock()
	if h.isReloading {
		h.mu.Unlock()
		return fmt.Errorf("配置重载正在进行中")
	}
	h.isReloading = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.isReloading = false
		h.lastReloadTime = time.Now()
		h.reloadCount++
		h.mu.Unlock()
	}()

	log.Println("开始处理配置热更新")

	// 执行安全重载函数
	if err := h.executeSafeReloadFuncs(newConfig); err != nil {
		return fmt.Errorf("安全配置重载失败: %w", err)
	}

	// 执行重载处理器
	if err := h.executeReloadHandlers(newConfig); err != nil {
		return fmt.Errorf("配置重载处理器执行失败: %w", err)
	}

	log.Println("配置热更新处理完成")
	return nil
}

// executeSafeReloadFuncs 执行安全重载函数
func (h *HotReloadManager) executeSafeReloadFuncs(newConfig *Config) error {
	h.mu.RLock()
	funcs := make(map[string]func(*Config) error)
	for name, fn := range h.safeReloadFuncs {
		funcs[name] = fn
	}
	h.mu.RUnlock()

	for name, reloadFunc := range funcs {
		log.Printf("执行安全配置重载函数: %s", name)
		if err := reloadFunc(newConfig); err != nil {
			return fmt.Errorf("安全配置重载函数 %s 执行失败: %w", name, err)
		}
	}

	return nil
}

// executeReloadHandlers 执行重载处理器
func (h *HotReloadManager) executeReloadHandlers(newConfig *Config) error {
	h.mu.RLock()
	handlers := make(map[string]func(*Config) error)
	for name, handler := range h.reloadHandlers {
		handlers[name] = handler
	}
	h.mu.RUnlock()

	for name, handler := range handlers {
		log.Printf("执行配置重载处理器: %s", name)
		if err := handler(newConfig); err != nil {
			log.Printf("配置重载处理器 %s 执行失败: %v", name, err)
			// 继续执行其他处理器，不中断整个重载过程
		}
	}

	return nil
}

// GetReloadStats 获取重载统计信息
func (h *HotReloadManager) GetReloadStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"reload_count":    h.reloadCount,
		"last_reload_time": h.lastReloadTime,
		"is_reloading":    h.isReloading,
		"handlers_count":  len(h.reloadHandlers),
		"safe_funcs_count": len(h.safeReloadFuncs),
	}
}

// IsReloading 检查是否正在重载
func (h *HotReloadManager) IsReloading() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isReloading
}

// GetLastReloadTime 获取最后重载时间
func (h *HotReloadManager) GetLastReloadTime() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastReloadTime
}

// GetReloadCount 获取重载次数
func (h *HotReloadManager) GetReloadCount() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.reloadCount
}