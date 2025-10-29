package handlers

import (
	"context"
	"fmt"
	"time"

	"hh-vacancy-bot/internal/bot/utils"
	"hh-vacancy-bot/internal/models"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// /settings command
func HandleSettings(ctx *Context) tele.HandlerFunc {
	return func(c tele.Context) error {
		userID := c.Sender().ID

		dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		user, err := ctx.Store.GetUser(dbCtx, userID)
		if err != nil {
			ctx.Logger.Error("failed to get user",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			return c.Send("😔 Ошибка при получении настроек")
		}

		// Check if user has filters
		hasFilters, err := ctx.Store.HasFilters(dbCtx, userID)
		if err != nil {
			ctx.Logger.Error("failed to check filters", zap.Error(err))
			hasFilters = false
		}

		if !hasFilters {
			return c.Send(
				"⚠️ *Настройка уведомлений*\n\n"+
					"Сначала настройте фильтры поиска вакансий\\.\n\n"+
					"Используйте команду /filters",
				utils.MainMenuKeyboard(),
				tele.ModeMarkdownV2,
			)
		}

		message := utils.FormatSettingsMessage(user)

		return c.Send(
			message,
			utils.SettingsKeyboard(user.CheckEnabled),
			tele.ModeMarkdownV2,
		)
	}
}

// Handle settings text buttons (legacy support)
func HandleSettingsText(ctx *Context, c tele.Context, text string) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := ctx.Store.GetUser(dbCtx, userID)
	if err != nil {
		ctx.Logger.Error("failed to get user", zap.Error(err))
		return c.Send("😔 Ошибка при получении данных")
	}

	switch text {
	case "🔔 Включить уведомления":
		return enableNotifications(ctx, c, user)
	case "🔕 Отключить уведомления":
		return disableNotifications(ctx, c, user)
	case "⏰ Изменить интервал":
		return changeInterval(ctx, c)
	default:
		return nil
	}
}

func enableNotifications(ctx *Context, c tele.Context, user *models.User) error {
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if user has filters
	hasFilters, err := ctx.Store.HasFilters(dbCtx, user.ID)
	if err != nil {
		ctx.Logger.Error("failed to check filters", zap.Error(err))
		return c.Send("😔 Ошибка при проверке фильтров")
	}

	if !hasFilters {
		return c.Send(
			"⚠️ Сначала настройте фильтры поиска\\.\n\n"+
				"Используйте команду /filters",
			utils.SettingsKeyboard(false),
			tele.ModeMarkdownV2,
		)
	}

	if err := ctx.Store.SetCheckEnabled(dbCtx, user.ID, true); err != nil {
		ctx.Logger.Error("failed to enable notifications",
			zap.Int64("user_id", user.ID),
			zap.Error(err),
		)
		return c.Send("😔 Ошибка при включении уведомлений")
	}

	user.CheckEnabled = true
	message := utils.FormatSettingsMessage(user)

	return c.Send(
		"✅ Уведомления включены\\!\n\n"+message,
		utils.SettingsKeyboard(true),
		tele.ModeMarkdownV2,
	)
}

func disableNotifications(ctx *Context, c tele.Context, user *models.User) error {
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ctx.Store.SetCheckEnabled(dbCtx, user.ID, false); err != nil {
		ctx.Logger.Error("failed to disable notifications",
			zap.Int64("user_id", user.ID),
			zap.Error(err),
		)
		return c.Send("😔 Ошибка при отключении уведомлений")
	}

	user.CheckEnabled = false
	message := utils.FormatSettingsMessage(user)

	return c.Send(
		"🔕 Уведомления отключены\n\n"+message,
		utils.SettingsKeyboard(false),
		tele.ModeMarkdownV2,
	)
}

// Display user statistics
func DisplayUserStats(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := ctx.Store.GetUserStats(dbCtx, userID)
	if err != nil {
		ctx.Logger.Error("failed to get user stats", zap.Error(err))
		return c.Send("😔 Ошибка при получении статистики")
	}

	message := fmt.Sprintf(
		"*📊 Статистика*\n\n"+
			"Фильтров: %d\n"+
			"Просмотрено вакансий: %d",
		stats["filter_count"],
		stats["seen_vacancies_count"],
	)

	return c.Send(message, tele.ModeMarkdown)
}