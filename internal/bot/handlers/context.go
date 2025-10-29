package handlers

import (
	"hh-vacancy-bot/internal/api/headhunter"
	"hh-vacancy-bot/internal/config"
	"hh-vacancy-bot/internal/storage/postgres"
	"hh-vacancy-bot/internal/storage/redis"

	"go.uber.org/zap"
)

// Context contains deps for all handlers
type Context struct {
	Store    *postgres.Store
	Cache    *redis.Cache
	HHClient *headhunter.Client
	Config   *config.Config
	Logger   *zap.Logger
}