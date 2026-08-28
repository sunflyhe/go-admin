// server 启动入口:加载配置 → 初始化日志 → 连接 MySQL → 执行迁移 → 启动 HTTP 服务,支持优雅关闭。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/config"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/database"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/auth"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/logger"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/platform/migrate"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/router"
	"github.com/hesunfly/hesunfly-admin-go/server/internal/system/audit"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(cfg.Log.Level)
	slog.Info("配置加载完成", "addr", cfg.Server.Addr, "mode", cfg.Server.Mode)

	db, err := database.New(
		cfg.MySQL.DSN,
		cfg.MySQL.MaxOpenConns,
		cfg.MySQL.MaxIdleConns,
		cfg.MySQL.ConnMaxLifetimeSec,
		cfg.MySQL.SlowThresholdMs,
	)
	if err != nil {
		slog.Error("数据库初始化失败", "error", err)
		os.Exit(1)
	}
	defer database.Close(db)

	// 启动时自动执行未应用的 up 迁移(空库可一键初始化)
	if err := migrate.Up(db); err != nil {
		slog.Error("执行迁移失败", "error", err)
		os.Exit(1)
	}

	jwtManager := auth.NewManager(
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.AccessTTLMinutes)*time.Minute,
		time.Duration(cfg.JWT.RefreshTTLHours)*time.Hour,
		cfg.JWT.Issuer,
	)
	recorder := audit.NewRecorder(db)
	defer recorder.Close()
	stopCleaner := audit.CleanupCleaner(db, cfg.Audit.RetentionDays, log)
	defer stopCleaner()

	engine := router.New(&router.Deps{
		Cfg:      cfg,
		DB:       db,
		Logger:   log,
		JWT:      jwtManager,
		Recorder: recorder,
	})

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	go func() {
		slog.Info("HTTP 服务启动", "addr", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP 服务异常退出", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("收到退出信号,开始优雅关闭")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("优雅关闭失败", "error", err)
	}
	slog.Info("服务已退出")
}
