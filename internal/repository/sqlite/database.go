package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mmemory/internal/models"
	"mmemory/pkg/config"
	loggermm "mmemory/pkg/logger"
)

// Database 数据库连接池管理
type Database struct {
	db               *gorm.DB
	sqlDB            *sql.DB
	config           *config.DatabaseConfig
	queryOptimizer   *QueryOptimizer
	healthCheckStop  chan struct{}
	healthCheckWG    sync.WaitGroup
	mu               sync.RWMutex
	stats            DatabaseStats
}

// DatabaseStats 数据库连接池统计
type DatabaseStats struct {
	OpenConnections  int           // 当前打开的连接数
	InUseConnections int           // 使用中的连接数
	IdleConnections  int           // 空闲连接数
	WaitCount        int64         // 等待获取连接的请求数
	MaxOpenConns     int           // 最大打开连接数
	LastHealthCheck  time.Time     // 最后健康检查时间
	HealthStatus     string        // 健康状态: healthy, degraded, unhealthy
}

func NewDatabase(config *config.DatabaseConfig) (*Database, error) {
	// 配置GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(config.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层SQL数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)

	// 设置连接生命周期
	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	database := &Database{
		db:              db,
		sqlDB:           sqlDB,
		config:          config,
		queryOptimizer:  NewQueryOptimizer(50 * time.Millisecond), // 50ms 慢查询阈值
		healthCheckStop: make(chan struct{}),
		stats: DatabaseStats{
			MaxOpenConns: config.MaxOpenConns,
			HealthStatus: "healthy",
		},
	}

	// 连接池预热
	if config.PoolWarmupSize > 0 {
		database.warmupPool()
	}

	// 自动迁移数据库表
	if err := database.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 启动健康检查
	database.startHealthCheck()

	return database, nil
}

// warmupPool 连接池预热
func (d *Database) warmupPool() {
	loggermm.Infof("数据库连接池预热中，建立 %d 个初始连接...", d.config.PoolWarmupSize)
	for i := 0; i < d.config.PoolWarmupSize; i++ {
		if err := d.sqlDB.Ping(); err != nil {
			loggermm.Warnf("预热连接失败: %v", err)
			break
		}
	}
	loggermm.Info("数据库连接池预热完成")
}

// startHealthCheck 启动健康检查
func (d *Database) startHealthCheck() {
	if d.config.HealthCheckInterval <= 0 {
		return
	}

	d.healthCheckWG.Add(1)
	go func() {
		defer d.healthCheckWG.Done()
		ticker := time.NewTicker(d.config.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				d.checkHealth()
			case <-d.healthCheckStop:
				return
			}
		}
	}()

	loggermm.Infof("数据库健康检查已启动，间隔: %v", d.config.HealthCheckInterval)
}

// checkHealth 检查数据库连接健康状态
func (d *Database) checkHealth() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 执行 Ping 检查连接有效性
	if err := d.sqlDB.Ping(); err != nil {
		d.stats.HealthStatus = "unhealthy"
		loggermm.Warnf("数据库健康检查失败: %v", err)
		return
	}

	// 更新统计信息
	stats := d.sqlDB.Stats()
	d.stats.OpenConnections = stats.OpenConnections
	d.stats.InUseConnections = stats.InUse
	d.stats.IdleConnections = stats.Idle
	d.stats.WaitCount = stats.WaitCount
	d.stats.LastHealthCheck = time.Now()

	// 计算健康状态
	utilization := float64(stats.OpenConnections) / float64(d.config.MaxOpenConns)
	if utilization > 0.9 {
		d.stats.HealthStatus = "unhealthy"
		loggermm.Warnf("数据库连接池利用率过高: %.1f%%", utilization*100)
	} else if utilization > 0.7 {
		d.stats.HealthStatus = "degraded"
	} else {
		d.stats.HealthStatus = "healthy"
	}
}

// GetStats 获取数据库连接池统计信息
func (d *Database) GetStats() DatabaseStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 实时更新统计
	stats := d.sqlDB.Stats()
	return DatabaseStats{
		OpenConnections:  stats.OpenConnections,
		InUseConnections: stats.InUse,
		IdleConnections:  stats.Idle,
		WaitCount:        stats.WaitCount,
		MaxOpenConns:     d.config.MaxOpenConns,
		LastHealthCheck:  d.stats.LastHealthCheck,
		HealthStatus:     d.stats.HealthStatus,
	}
}

func (d *Database) AutoMigrate() error {
	return d.db.AutoMigrate(
		&models.User{},
		&models.Reminder{},
		&models.ReminderLog{},
		&models.Conversation{},
		&models.ConversationContext{},
		&models.DailyActivity{},
		&models.ConversationArchive{}, // 新增
	)
}

func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 防止重复关闭
	select {
	case <-d.healthCheckStop:
		// 通道已关闭，直接返回
		return nil
	default:
		close(d.healthCheckStop)
	}

	// 等待健康检查goroutine退出
	d.healthCheckWG.Wait()

	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) GetDB() *gorm.DB {
	return d.db
}

// 开始事务
func (d *Database) BeginTx(ctx context.Context) (*gorm.DB, error) {
	return d.db.WithContext(ctx).Begin(), nil
}

// GetQueryOptimizer 获取查询优化器
func (d *Database) GetQueryOptimizer() *QueryOptimizer {
	return d.queryOptimizer
}
