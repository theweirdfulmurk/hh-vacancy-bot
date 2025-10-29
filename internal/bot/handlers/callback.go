package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hh-vacancy-bot/internal/bot/utils"
	"hh-vacancy-bot/internal/models"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// HandleCallback processes all callback queries from inline buttons
func HandleCallback(ctx *Context) tele.HandlerFunc {
	return func(c tele.Context) error {
		cb := c.Callback()
		if cb == nil {
			return nil
		}

		// 1) handle unique-based callbacks (telebot style with payload)
		switch cb.Unique {
		case "choose_area":
			return handleChooseAreaPayload(ctx, c, cb.Data) // cb.Data = areaID
		}

		// 2) legacy colon-based callbacks you already use
		data := cb.Data
		parts := strings.Split(data, ":")
		if len(parts) < 1 {
			return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный формат"})
		}
		action := parts[0]

		switch action {
		case "filter_delete":
			return handleFilterDelete(ctx, c, parts)
		case "settings_toggle":
			return handleSettingsToggle(ctx, c)
		case "settings_interval":
			return handleSettingsInterval(ctx, c, parts)
		case "vacancy_page":
			return handleVacancyPage(ctx, c, parts)
		case "confirm_yes":
			return handleConfirmYes(ctx, c)
		case "confirm_no":
			return handleConfirmNo(ctx, c)
		case "choose_area": // if you ever send "choose_area:<id>"
			return handleChooseArea(ctx, c, parts)
		default:
			return c.Respond(&tele.CallbackResponse{Text: "❓ Неизвестное действие"})
		}
	}
}

// ==================== Filter Management ====================

func handleFilterDelete(ctx *Context, c tele.Context, parts []string) error {
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный формат"})
	}

	filterType := parts[1]
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ctx.Store.DeleteFilter(dbCtx, userID, filterType); err != nil {
		ctx.Logger.Error("failed to delete filter",
			zap.Int64("user_id", userID),
			zap.String("filter_type", filterType),
			zap.Error(err),
		)
		return c.Respond(&tele.CallbackResponse{
			Text: "😔 Ошибка при удалении фильтра",
		})
	}

	// Update message with remaining filters
	filters, err := ctx.Store.GetUserFilters(dbCtx, userID)
	if err != nil {
		ctx.Logger.Error("failed to get filters", zap.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "✅ Фильтр удален"})
	}

	var message string
	if len(filters) == 0 {
		message = "ℹ️ У вас нет установленных фильтров"
	} else {
		message = utils.FormatFiltersMessage(filters)
	}

	if err := c.Edit(message, utils.FiltersMenuKeyboard(), tele.ModeMarkdownV2); err != nil {
		ctx.Logger.Warn("failed to edit message", zap.Error(err))
	}

	return c.Respond(&tele.CallbackResponse{
		Text: "✅ Фильтр удален",
	})
}

// ==================== Settings ====================

func handleSettingsToggle(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := ctx.Store.GetUser(dbCtx, userID)
	if err != nil {
		ctx.Logger.Error("failed to get user", zap.Error(err))
		return c.Respond(&tele.CallbackResponse{
			Text: "😔 Ошибка при получении данных",
		})
	}

	newState := !user.CheckEnabled

	if err := ctx.Store.SetCheckEnabled(dbCtx, userID, newState); err != nil {
		ctx.Logger.Error("failed to toggle check",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return c.Respond(&tele.CallbackResponse{
			Text: "😔 Ошибка при изменении настроек",
		})
	}

	user.CheckEnabled = newState
	message := utils.FormatSettingsMessage(user)

	if err := c.Edit(message, utils.SettingsKeyboard(newState), tele.ModeMarkdownV2); err != nil {
		ctx.Logger.Warn("failed to edit message", zap.Error(err))
	}

	responseText := "✅ Уведомления включены"
	if !newState {
		responseText = "🔕 Уведомления отключены"
	}

	return c.Respond(&tele.CallbackResponse{Text: responseText})
}

func handleSettingsInterval(ctx *Context, c tele.Context, parts []string) error {
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный формат"})
	}

	// This callback is for showing interval selection
	return c.Send(
		"⏰ Выберите интервал проверки новых вакансий:",
		utils.IntervalKeyboard(),
	)
}

// ==================== Vacancy Pagination ====================

func handleVacancyPage(ctx *Context, c tele.Context, parts []string) error {
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный формат"})
	}

	// TODO: Implement pagination logic
	// For now, just acknowledge
	return c.Respond(&tele.CallbackResponse{
		Text: "📄 Переход на страницу...",
	})
}

// ==================== Confirmation ====================

func handleConfirmYes(ctx *Context, c tele.Context) error {
	// Check what we're confirming based on the message
	if c.Message() != nil && strings.Contains(c.Message().Text, "очистить все фильтры") {
		return confirmClearFilters(ctx, c)
	}

	return c.Respond(&tele.CallbackResponse{Text: "✅ Подтверждено"})
}

func handleConfirmNo(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	// Clear any pending state
	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	if err := c.Edit("❌ Операция отменена", utils.FiltersMenuKeyboard()); err != nil {
		ctx.Logger.Warn("failed to edit message", zap.Error(err))
		return c.Send("❌ Операция отменена", utils.FiltersMenuKeyboard())
	}

	return c.Respond(&tele.CallbackResponse{Text: "❌ Отменено"})
}

func handleChooseArea(ctx *Context, c tele.Context, parts []string) error {
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный формат"})
	}
	areaID := parts[1]
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// persist chosen area
	filter := &models.UserFilter{
		UserID:      userID,
		FilterType:  models.FilterTypeArea,
		FilterValue: areaID,
	}
	if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
		ctx.Logger.Error("failed to save city filter (cb)", zap.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка при сохранении"})
	}

	name := "город выбран"
	if ar, err := ctx.HHClient.GetArea(dbCtx, areaID); err == nil && ar != nil {
		name = ar.Name
	}

	_ = clearUserState(ctx, userID)

	// try to edit previous message; if fails, just ack
	if c.Message() != nil {
		_ = c.Edit(
			fmt.Sprintf("✅ Город установлен: *%s*", utils.EscapeMarkdown(name)),
			utils.FiltersMenuKeyboard(),
			tele.ModeMarkdownV2,
		)
	}

	return c.Respond(&tele.CallbackResponse{Text: "✅ Выбрано"})
}

// ==================== Inline Keyboards Generators ====================

func InlineFiltersKeyboard(ctx *Context, userID int64) (*tele.ReplyMarkup, error) {
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filters, err := ctx.Store.GetUserFilters(dbCtx, userID)
	if err != nil {
		return nil, err
	}

	menu := &tele.ReplyMarkup{}
	var rows []tele.Row

	for _, filter := range filters {
		filterName := getFilterDisplayName(filter.FilterType)
		btnDelete := menu.Data(
			fmt.Sprintf("🗑 %s", filterName),
			fmt.Sprintf("filter_delete:%s", filter.FilterType),
		)
		rows = append(rows, menu.Row(btnDelete))
	}

	if len(rows) > 0 {
		menu.Inline(rows...)
	}

	return menu, nil
}

func InlineSettingsKeyboard(enabled bool) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	var btnToggle tele.Btn
	if enabled {
		btnToggle = menu.Data("🔕 Отключить", "settings_toggle")
	} else {
		btnToggle = menu.Data("🔔 Включить", "settings_toggle")
	}

	btnInterval := menu.Data("⏰ Интервал", "settings_interval")

	menu.Inline(
		menu.Row(btnToggle),
		menu.Row(btnInterval),
	)

	return menu
}

func InlineConfirmKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnYes := menu.Data("✅ Да", "confirm_yes")
	btnNo := menu.Data("❌ Нет", "confirm_no")

	menu.Inline(menu.Row(btnYes, btnNo))

	return menu
}

// ==================== Helpers ====================

func handleChooseAreaPayload(ctx *Context, c tele.Context, areaID string) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := &models.UserFilter{
		UserID:      userID,
		FilterType:  models.FilterTypeArea,
		FilterValue: areaID,
	}
	if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
		ctx.Logger.Error("failed to save city filter (payload)", zap.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка сохранения"})
	}

	name := "Город выбран"
	if ar, err := ctx.HHClient.GetArea(dbCtx, areaID); err == nil && ar != nil {
		name = ar.Name
	}
	_ = clearUserState(ctx, userID)

	if c.Message() != nil {
		_ = c.Edit(
			fmt.Sprintf("✅ Город установлен: *%s*", utils.EscapeMarkdown(name)),
			utils.FiltersMenuKeyboard(),
			tele.ModeMarkdownV2,
		)
	}
	return c.Respond(&tele.CallbackResponse{Text: "✅ Выбрано"})
}

func getFilterDisplayName(filterType string) string {
	switch filterType {
	case "text":
		return "Текст"
	case "area":
		return "Город"
	case "salary":
		return "Зарплата"
	case "experience":
		return "Опыт"
	case "schedule":
		return "График"
	default:
		return filterType
	}
}
