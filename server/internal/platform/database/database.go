// Package database 建立 GORM MySQL 连接并配置连接池。
package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New 打开 MySQL 连接。仅将 GORM 用于查询映射,库结构由 SQL migration 管理,不使用 AutoMigrate。
func New(dsn string, maxOpen, maxIdle, connMaxLifetimeSec, slowThresholdMs int) (*gorm.DB, error) {
	lvl := gormlogger.Warn
	if slowThresholdMs > 0 {
		lvl = gormlogger.Warn
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(lvl),
		// 禁用默认事务包装,写操作在 service 层显式使用事务。
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetimeSec) * time.Second)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL ping 失败: %w", err)
	}
	slog.Info("mysql connected", "maxOpenConns", maxOpen, "maxIdleConns", maxIdle)
	return db, nil
}

// Close 优雅关闭底层连接池。
func Close(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
