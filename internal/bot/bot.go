package bot

import (
	"context"
	"fmt"
	"time"

	"hh-vacancy-bot/internal/api/headhunter"
	"hh-vacancy-bot/internal/bot/handlers"
	"hh-vacancy-bot/internal/bot/middleware"
	"hh-vacancy-bot/internal/config"
	"hh-vacancy-bot/internal/storage/postgres"
	"hh-vacancy-bot/internal/storage/redis"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// Bot represents Telegram bot
type Bot struct {
	bot      *tele.Bot
	store    *postgres.Store
	cache    *redis.Cache
	hhClient *headhunter.Client
	config   *config.Config
	logger   *zap.Logger
}

func New(
	cfg *config.Config,
	store *postgres.Store,
	cache *redis.Cache,
	hhClient *headhunter.Client,
	logger *zap.Logger,
) (*Bot, error) {
	pref := tele.Settings{
		Token:  cfg.TelegramToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	bot := &Bot{
		bot:      b,
		store:    store,
		cache:    cache,
		hhClient: hhClient,
		config:   cfg,
		logger:   logger,
	}

	bot.setupMiddleware()

	bot.registerHandlers()

	logger.Info("bot initialized successfully")

	return bot, nil
}

func (b *Bot) setupMiddleware() {
	b.bot.Use(middleware.Recovery(b.logger))

	b.bot.Use(middleware.Logger(b.logger))

	b.bot.Use(middleware.RateLimit(b.cache, b.logger))
}

func (b *Bot) registerHandlers() {
	ctx := &handlers.Context{
		Store:    b.store,
		Cache:    b.cache,
		HHClient: b.hhClient,
		Config:   b.config,
		Logger:   b.logger,
	}

	b.bot.Handle("/start", handlers.HandleStart(ctx))
	b.bot.Handle("/help", handlers.HandleHelp(ctx))
	b.bot.Handle("/filters", handlers.HandleFilters(ctx))
	b.bot.Handle("/vacancies", handlers.HandleVacancies(ctx))
	b.bot.Handle("/settings", handlers.HandleSettings(ctx))

	b.bot.Handle(tele.OnText, handlers.HandleText(ctx))

	b.bot.Handle(tele.OnCallback, handlers.HandleCallback(ctx))

	b.logger.Info("handlers registered")
}

func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info("starting bot...")

	go b.bot.Start()

	<-ctx.Done()

	b.logger.Info("stopping bot...")
	b.bot.Stop()

	return nil
}

func (b *Bot) Stop() {
	b.logger.Info("bot stopped")
	b.bot.Stop()
}

func (b *Bot) GetBot() *tele.Bot {
	return b.bot
}