package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"hh-vacancy-bot/internal/api/headhunter"
	"hh-vacancy-bot/internal/bot"
	"hh-vacancy-bot/internal/bot/scheduler"
	"hh-vacancy-bot/internal/config"
	"hh-vacancy-bot/internal/logger"
	"hh-vacancy-bot/internal/storage/postgres"
	"hh-vacancy-bot/internal/storage/redis"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("starting HH vacancy bot",
		zap.String("log_level", cfg.LogLevel),
		zap.Duration("check_interval", cfg.CheckInterval),
	)

	log.Info("connecting to PostgreSQL...")
	store, err := postgres.New(cfg.PostgresDSN, log)
	if err != nil {
		log.Fatal("failed to connect to PostgreSQL", zap.Error(err))
	}
	defer store.Close()

	log.Info("PostgreSQL connected successfully")

	log.Info("connecting to Redis...")
	cache, err := redis.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, log)
	if err != nil {
		log.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer cache.Close()

	log.Info("Redis connected successfully")

	hhClient := headhunter.New(cfg.HHAPIBaseURL, cfg.HHAPITimeout, log)
	log.Info("HeadHunter API client created")

	log.Info("initializing Telegram bot...")
	tgBot, err := bot.New(cfg, store, cache, hhClient, log)
	if err != nil {
		log.Fatal("failed to create bot", zap.Error(err))
	}

	log.Info("Telegram bot initialized successfully")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info("received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	log.Info("starting vacancy checker...")
	checker := scheduler.New(
		tgBot.GetBot(),
		store,
		cache,
		hhClient,
		cfg,
		log,
	)

	go checker.Start(ctx)

	log.Info("bot is running...")
	log.Info("press Ctrl+C to stop")

	if err := tgBot.Start(ctx); err != nil {
		log.Error("bot stopped with error", zap.Error(err))
	}

	log.Info("shutting down gracefully...")

	log.Info("bot stopped")
}