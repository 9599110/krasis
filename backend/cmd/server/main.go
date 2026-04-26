package main

import (
	"context"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/krasis/krasis/internal/config"
	"github.com/krasis/krasis/internal/server"
	"github.com/krasis/krasis/pkg/database"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		panic("load config: " + err.Error())
	}

	logger, err := initLogger(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		panic("init logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("starting krasis server",
		zap.String("mode", cfg.Server.Mode),
		zap.Int("port", cfg.Server.Port),
	)

	pgPool, err := database.NewPostgresPool(ctx, database.PostgresConfig{
		Host:            cfg.Database.Postgres.Host,
		Port:            cfg.Database.Postgres.Port,
		User:            cfg.Database.Postgres.User,
		Password:        cfg.Database.Postgres.Password,
		DBName:          cfg.Database.Postgres.DBName,
		SSLMode:         cfg.Database.Postgres.SSLMode,
		MaxOpenConns:    cfg.Database.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Database.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.Postgres.ConnMaxLifetime,
	})
	if err != nil {
		logger.Fatal("failed to connect postgres", zap.Error(err))
	}
	defer pgPool.Close()
	logger.Info("connected to postgres")

	rdb, err := database.NewRedisClient(database.RedisConfig{
		Addr:     cfg.Database.Redis.Addr,
		Password: cfg.Database.Redis.Password,
		DB:       cfg.Database.Redis.DB,
		PoolSize: cfg.Database.Redis.PoolSize,
	})
	if err != nil {
		logger.Fatal("failed to connect redis", zap.Error(err))
	}
	defer rdb.Close()
	logger.Info("connected to redis")

	srv := server.New(cfg, pgPool, rdb, logger)

	if err := srv.Start(ctx); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}

func initLogger(level, format string) (*zap.Logger, error) {
	var cfg zap.Config

	switch format {
	case "json":
		cfg = zap.NewProductionConfig()
	default:
		cfg = zap.NewDevelopmentConfig()
	}

	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	return cfg.Build()
}
