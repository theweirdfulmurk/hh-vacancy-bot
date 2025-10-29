package middleware

import (
	"context"
	"fmt"
	"time"

	"hh-vacancy-bot/internal/storage/redis"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const (
	MaxRequestsPerMinute = 50
)

func RateLimit(cache *redis.Cache, logger *zap.Logger) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			user := c.Sender()
			if user == nil {
				return next(c)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			count, err := cache.IncrementUserRateLimit(ctx, user.ID)
			if err != nil {
				logger.Error("failed to check rate limit",
					zap.Int64("user_id", user.ID),
					zap.Error(err),
				)
				return next(c)
			}

			if count > MaxRequestsPerMinute {
				logger.Warn("rate limit exceeded",
					zap.Int64("user_id", user.ID),
					zap.Int64("count", count),
				)

				return c.Reply(fmt.Sprintf(
					"⚠️ Превышен лимит запросов. Пожалуйста, подождите минуту.\n"+
						"Максимум: %d запросов в минуту.",
					MaxRequestsPerMinute,
				))
			}

			return next(c)
		}
	}
}

func CheckHHAPIRateLimit(cache *redis.Cache, logger *zap.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	count, err := cache.GetHHAPIRateLimit(ctx)
	if err != nil {
		logger.Error("failed to check HH API rate limit", zap.Error(err))
		return nil
	}

	if count > 50 {
		return fmt.Errorf("HH API rate limit exceeded: %d requests", count)
	}

	_, err = cache.IncrementHHAPIRateLimit(ctx)
	if err != nil {
		logger.Error("failed to increment HH API rate limit", zap.Error(err))
	}

	return nil
}
