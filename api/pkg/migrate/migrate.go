// Package migrate 使用 golang-migrate 在启动时执行内嵌的 SQL 迁移。
// 默认只执行 up 迁移;down 迁移仅由人工在明确风险下执行,不提供自动路径。
package migrate

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"

	migrations "github.com/hesunfly/hesunfly-admin-go/api/migrations"
)

// Up 执行所有未应用的迁移。空库会依次执行到最新版本,并写入 schema_migrations 版本记录。
func Up(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层 sql.DB 失败: %w", err)
	}
	driver, err := mysql.WithInstance(sqlDB, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("初始化迁移驱动失败: %w", err)
	}
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("加载内嵌迁移文件失败: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		return fmt.Errorf("初始化 migrate 失败: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("执行迁移失败: %w", err)
	}
	version, dirty, _ := m.Version()
	fmt.Printf("migration done, version=%d dirty=%v\n", version, dirty)
	return nil
}
