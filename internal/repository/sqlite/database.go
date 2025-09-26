package sqlite

import (
	"context"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mmemory/internal/models"
	"mmemory/pkg/config"
)

type Database struct {
	db *gorm.DB
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

	// 设置连接池
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)

	database := &Database{db: db}

	// 自动迁移数据库表
	if err := database.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return database, nil
}

func (d *Database) AutoMigrate() error {
	return d.db.AutoMigrate(
		&models.User{},
		&models.Reminder{},
		&models.ReminderLog{},
		&models.Conversation{},
	)
}

func (d *Database) Close() error {
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