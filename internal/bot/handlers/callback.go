package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"hh-vacancy-bot/internal/bot/middleware"
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
			ctx.Logger.Warn("callback is nil")
			return nil
		}

		// Log callback for debugging
		ctx.Logger.Info("received callback",
			zap.String("data", cb.Data),
			zap.String("unique", cb.Unique),
			zap.Int64("user_id", c.Sender().ID),
			zap.String("callback_id", cb.ID),
		)

		payload := strings.TrimPrefix(cb.Data, "\f")
		unique := cb.Unique
		if unique == "" {
			unique = payload
		}

		uniqueParts := strings.Split(unique, ":")
		payloadParts := []string{}
		if payload != "" {
			payloadParts = strings.Split(payload, ":")
		}

		action := uniqueParts[0]

		ctx.Logger.Info("parsed callback",
			zap.String("unique", unique),
			zap.String("payload", payload),
			zap.Strings("unique_parts", uniqueParts),
			zap.Strings("payload_parts", payloadParts),
		)

		// Route to appropriate handler
		switch action {
		case "filter_delete":
			return handleFilterDelete(ctx, c, uniqueParts)
		case "settings_toggle":
			return handleSettingsToggle(ctx, c)
		case "settings_interval":
			return handleSettingsInterval(ctx, c, uniqueParts)
		case "vacancy_page":
			return handleVacancyPage(ctx, c, payloadParts)
		case "confirm_yes":
			return handleConfirmYes(ctx, c)
		case "confirm_no":
			return handleConfirmNo(ctx, c)
		case "choose_area":
			return handleChooseArea(ctx, c, uniqueParts)
		default:
			ctx.Logger.Warn("unknown callback action",
				zap.String("action", action),
				zap.String("unique", unique),
				zap.String("payload", payload),
			)
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
		ctx.Logger.Error("failed to delete filter", zap.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка удаления"})
	}

	displayName := getFilterDisplayName(filterType)

	// Try to update the message
	if err := c.Edit(
		fmt.Sprintf("✅ Фильтр *%s* удалён", utils.EscapeMarkdown(displayName)),
		utils.FiltersMenuKeyboard(),
		tele.ModeMarkdownV2,
	); err != nil {
		ctx.Logger.Warn("failed to edit message", zap.Error(err))
		return c.Send(
			fmt.Sprintf("✅ Фильтр *%s* удалён", utils.EscapeMarkdown(displayName)),
			utils.FiltersMenuKeyboard(),
			tele.ModeMarkdownV2,
		)
	}

	return c.Respond(&tele.CallbackResponse{Text: "✅ Удалено"})
}

// ==================== Settings ====================

func handleSettingsToggle(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := ctx.Store.GetUser(dbCtx, userID)
	if err != nil {
		ctx.Logger.Error("failed to get user", zap.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка"})
	}

	newState := !user.CheckEnabled
	user.CheckEnabled = newState

	if err := ctx.Store.UpdateUser(dbCtx, user); err != nil {
		ctx.Logger.Error("failed to update user", zap.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка сохранения"})
	}

	// Update the keyboard
	if err := c.Edit(
		"⚙️ Настройки уведомлений:",
		utils.SettingsKeyboard(newState),
	); err != nil {
		ctx.Logger.Warn("failed to edit message", zap.Error(err))
	}

	responseText := "✅ Уведомления включены"
	if !newState {
		responseText = "🔕 Уведомления отключены"
	}

	return c.Respond(&tele.CallbackResponse{Text: responseText})
}

func handleSettingsInterval(ctx *Context, c tele.Context, parts []string) error {
	// This callback is for showing interval selection
	return c.Send(
		"⏰ Выберите интервал проверки новых вакансий:",
		utils.IntervalKeyboard(),
	)
}

// ==================== Vacancy Pagination ====================

func handleVacancyPage(ctx *Context, c tele.Context, parts []string) error {
	if len(parts) == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный формат"})
	}

	action := parts[0]

	switch action {
	case "noop":
		return c.Respond(&tele.CallbackResponse{Text: "📄 Уже на этой странице"})
	case "goto":
		if len(parts) < 2 {
			return c.Respond(&tele.CallbackResponse{Text: "❌ Нет номера страницы"})
		}

		targetPage, err := strconv.Atoi(parts[1])
		if err != nil || targetPage < 0 {
			return c.Respond(&tele.CallbackResponse{Text: "❌ Неверная страница"})
		}

		userID := c.Sender().ID

		dbCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		filtersMap, err := ctx.Store.GetFiltersMap(dbCtx, userID)
		if err != nil {
			ctx.Logger.Error("failed to load filters for pagination", zap.Error(err))
			return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка фильтров"})
		}
		if len(filtersMap) == 0 {
			return c.Respond(&tele.CallbackResponse{Text: "ℹ️ Настройте фильтры через /filters"})
		}

		if err := middleware.CheckHHAPIRateLimit(ctx.Cache, ctx.Logger); err != nil {
			ctx.Logger.Warn("HH API rate limit (pagination)", zap.Error(err))
			return c.Respond(&tele.CallbackResponse{Text: "⚠️ Попробуйте позже"})
		}

		params := buildSearchParams(filtersMap)
		if ctx.Config.MaxVacanciesPerCheck > 0 {
			params.PerPage = ctx.Config.MaxVacanciesPerCheck
		}
		params.Page = targetPage

		response, err := ctx.HHClient.SearchVacancies(dbCtx, params)
		if err != nil {
			ctx.Logger.Error("failed to fetch vacancy page", zap.Error(err))
			return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка запроса"})
		}

		totalPages := response.Pages
		if totalPages == 0 {
			totalPages = 1
		}

		if targetPage >= totalPages {
			return c.Respond(&tele.CallbackResponse{Text: "⚠️ Страница недоступна"})
		}

		indicator := fmt.Sprintf("📄 Страница %d из %d", targetPage+1, totalPages)
		if params.PublishedWithinDays > 0 {
			indicator = fmt.Sprintf("%s • за %s", indicator, utils.FormatDays(params.PublishedWithinDays))
		}

		if err := c.Edit(indicator, utils.InlinePaginationKeyboard(targetPage, totalPages, "vacancy_page")); err != nil {
			ctx.Logger.Warn("failed to edit pagination message", zap.Error(err))
		}

		go cacheVacancies(ctx, response.Items)

		if len(response.Items) == 0 {
			if err := c.Send("🤷 На этой странице вакансий нет"); err != nil {
				ctx.Logger.Warn("failed to send empty page message", zap.Error(err))
			}
			return c.Respond(&tele.CallbackResponse{Text: "ℹ️ Нет вакансий"})
		}

		header := fmt.Sprintf("📄 *Вакансии — страница %d/%d*", targetPage+1, totalPages)
		if err := c.Send(header, tele.ModeMarkdownV2); err != nil {
			ctx.Logger.Error("failed to send pagination header", zap.Error(err))
		}

		if err := deliverVacancyCards(ctx, c, response.Items, userID); err != nil {
			ctx.Logger.Error("failed to send vacancies page", zap.Error(err))
			return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка отправки"})
		}

		go markVacanciesAsSeen(ctx, userID, response.Items)

		return c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("📄 Стр. %d", targetPage+1)})
	default:
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный формат"})
	}
}

// ==================== Confirmation ====================

func handleConfirmYes(ctx *Context, c tele.Context) error {
	// This is just a pass-through - actual confirmation logic is in filters.go
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

// ==================== Area Selection ====================

func handleChooseArea(ctx *Context, c tele.Context, parts []string) error {
	if len(parts) < 2 {
		ctx.Logger.Warn("choose_area callback without area ID", zap.Strings("parts", parts))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Неверный формат"})
	}

	areaID := parts[1]
	userID := c.Sender().ID

	ctx.Logger.Info("handling area selection",
		zap.Int64("user_id", userID),
		zap.String("area_id", areaID),
	)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Save the selected area as a filter
	filter := &models.UserFilter{
		UserID:      userID,
		FilterType:  models.FilterTypeArea,
		FilterValue: areaID,
	}

	if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
		ctx.Logger.Error("failed to save city filter", zap.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "😔 Ошибка при сохранении"})
	}

	// Get area name for display
	areaName := "Город выбран"
	if area, err := ctx.HHClient.GetArea(dbCtx, areaID); err == nil && area != nil {
		areaName = area.Name
		ctx.Logger.Info("area name resolved", zap.String("name", areaName))
	} else {
		ctx.Logger.Warn("failed to get area name", zap.Error(err))
	}

	// Clear conversation state
	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	// Update the message with the result
	editErr := c.Edit(
		fmt.Sprintf("✅ Город установлен: *%s*", utils.EscapeMarkdown(areaName)),
		utils.FiltersMenuKeyboard(),
		tele.ModeMarkdownV2,
	)

	if editErr != nil {
		ctx.Logger.Warn("failed to edit message", zap.Error(editErr))
		// Fallback: send new message if edit fails
		return c.Send(
			fmt.Sprintf("✅ Город установлен: *%s*", utils.EscapeMarkdown(areaName)),
			utils.FiltersMenuKeyboard(),
			tele.ModeMarkdownV2,
		)
	}

	return c.Respond(&tele.CallbackResponse{Text: "✅ Выбрано"})
}

// ==================== Helpers ====================

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
