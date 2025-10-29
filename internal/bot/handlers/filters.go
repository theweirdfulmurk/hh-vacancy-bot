package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"hh-vacancy-bot/internal/bot/utils"
	"hh-vacancy-bot/internal/models"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// User states for conversation flow
const (
	StateIdle             = ""
	StateAwaitingText     = "awaiting_text"
	StateAwaitingCity     = "awaiting_city"
	StateAwaitingSalary   = "awaiting_salary"
	StateAwaitingExp      = "awaiting_experience"
	StateAwaitingSchedule = "awaiting_schedule"
	StateAwaitingPeriod   = "awaiting_period"
	StateConfirmClear     = "confirm_clear_filters"
)

// /filters command
func HandleFilters(ctx *Context) tele.HandlerFunc {
	return func(c tele.Context) error {
		userID := c.Sender().ID

		// Clear any existing state
		if err := clearUserState(ctx, userID); err != nil {
			ctx.Logger.Warn("failed to clear user state", zap.Error(err))
		}

		message := "🔧 *Настройка фильтров*\n\n"
		message += "Выберите параметр для настройки:"

		return c.Send(
			message,
			utils.FiltersMenuKeyboard(),
			tele.ModeMarkdown,
		)
	}
}

// HandleText processes all text messages
func HandleText(ctx *Context) tele.HandlerFunc {
	return func(c tele.Context) error {
		text := strings.TrimSpace(c.Text())
		userID := c.Sender().ID

		// Check user state
		state, err := getUserState(ctx, userID)
		if err != nil {
			ctx.Logger.Warn("failed to get user state", zap.Error(err))
			state = StateIdle
		}

		// Handle state-based input
		if state != StateIdle {
			return handleStateInput(ctx, c, state)
		}

		// Handle menu buttons
		switch text {
		// Main menu
		case "🔧 Фильтры":
			return HandleFilters(ctx)(c)
		case "📋 Вакансии":
			return HandleVacancies(ctx)(c)
		case "⚙️ Настройки":
			return HandleSettings(ctx)(c)
		case "❓ Справка":
			return HandleHelp(ctx)(c)

		// Filters menu
		case "🔍 Текст поиска":
			return startTextFilter(ctx, c)
		case "📍 Город":
			return startCityFilter(ctx, c)
		case "💰 Зарплата":
			return startSalaryFilter(ctx, c)
		case "💼 Опыт":
			return startExperienceFilter(ctx, c)
		case "⏰ График":
			return startScheduleFilter(ctx, c)
		case "🗓 Период":
			return startPeriodFilter(ctx, c)
		case "📊 Показать фильтры":
			return showFilters(ctx, c)
		case "🗑 Очистить фильтры":
			return clearFilters(ctx, c)
		case "◀️ Назад":
			return c.Send("Главное меню", utils.MainMenuKeyboard())

		// Settings menu
		case "🔔 Включить уведомления", "🔕 Отключить уведомления":
			return toggleNotifications(ctx, c)
		case "⏰ Изменить интервал":
			return changeInterval(ctx, c)

		// Cancel
		case "❌ Отмена":
			return cancelConversation(ctx, c)

		default:
			// Handle interval selection
			if intervalMinutes := parseIntervalText(text); intervalMinutes > 0 {
				return saveInterval(ctx, c, intervalMinutes)
			}

			// Handle experience selection
			if models.IsValidExperience(text) {
				return saveExperience(ctx, c, text)
			}

			// Handle schedule selection
			if models.IsValidSchedule(text) {
				return saveSchedule(ctx, c, text)
			}

			return c.Reply("Используйте кнопки меню или команды")
		}
	}
}

// ==================== Text Filter ====================

func startTextFilter(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	if err := setUserState(ctx, userID, StateAwaitingText); err != nil {
		ctx.Logger.Error("failed to set user state", zap.Error(err))
	}

	return c.Send(
		"🔍 Введите текст для поиска (например: 'python разработчик'):",
		utils.CancelKeyboard(),
	)
}

func handleTextFilterInput(ctx *Context, c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	userID := c.Sender().ID

	if text == "" || text == "❌ Отмена" {
		return cancelConversation(ctx, c)
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := &models.UserFilter{
		UserID:      userID,
		FilterType:  models.FilterTypeText,
		FilterValue: text,
	}

	if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
		ctx.Logger.Error("failed to save text filter", zap.Error(err))
		return c.Send("😔 Ошибка при сохранении фильтра")
	}

	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	return c.Send(
		fmt.Sprintf("✅ Текст поиска установлен: *%s*", utils.EscapeMarkdown(text)),
		utils.FiltersMenuKeyboard(),
		tele.ModeMarkdownV2,
	)
}

// ==================== City Filter ====================

func startCityFilter(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	if err := setUserState(ctx, userID, StateAwaitingCity); err != nil {
		ctx.Logger.Error("failed to set user state", zap.Error(err))
	}

	message := "📍 Введите название города (например: Москва, Санкт-Петербург):"

	return c.Send(message, utils.CancelKeyboard())
}

func handleCityFilterInput(ctx *Context, c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	userID := c.Sender().ID

	if text == "" || text == "❌ Отмена" {
		return cancelConversation(ctx, c)
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areas, err := ctx.HHClient.SearchAreas(dbCtx, text)
	if err != nil {
		ctx.Logger.Error("failed to search areas", zap.Error(err))
		return c.Send("😔 Ошибка при поиске города. Попробуйте ещё раз.")
	}
	if len(areas) == 0 {
		return c.Send("🤷 Город не найден. Попробуйте уточнить.")
	}

	if len(areas) == 1 {
		area := areas[0]
		filter := &models.UserFilter{
			UserID:      userID,
			FilterType:  models.FilterTypeArea,
			FilterValue: area.ID,
		}
		if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
			ctx.Logger.Error("failed to save city filter", zap.Error(err))
			return c.Send("😔 Ошибка при сохранении фильтра")
		}
		_ = clearUserState(ctx, userID)
		return c.Send(
			fmt.Sprintf("✅ Город установлен: *%s*", utils.EscapeMarkdown(area.Path)),
			utils.FiltersMenuKeyboard(),
			tele.ModeMarkdownV2,
		)
	}

	// multiple matches → show inline list
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row

	for _, a := range areas {
		btnText := a.Name
		if a.Path != "" {
			parts := strings.Split(a.Path, " > ")
			if len(parts) > 1 {
				// parent = parts[len(parts)-2]
				btnText = fmt.Sprintf("%s (%s)", a.Name, parts[len(parts)-2])
			}
		}
		rows = append(rows, menu.Row(menu.Data(btnText, "choose_area:"+a.ID)))
	}

	menu.Inline(rows...)
	// keep state so we know we're waiting for selection
	return c.Send("Нашлось несколько вариантов — выберите точный:", menu)
}

// ==================== Salary Filter ====================

func startSalaryFilter(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	if err := setUserState(ctx, userID, StateAwaitingSalary); err != nil {
		ctx.Logger.Error("failed to set user state", zap.Error(err))
	}

	return c.Send(
		"💰 Введите минимальную желаемую зарплату в рублях (например: 100000):",
		utils.CancelKeyboard(),
	)
}

func handleSalaryFilterInput(ctx *Context, c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	userID := c.Sender().ID

	if text == "" || text == "❌ Отмена" {
		return cancelConversation(ctx, c)
	}

	salary, err := strconv.Atoi(text)
	if err != nil || salary <= 0 {
		return c.Send("❌ Неверный формат. Введите число (например: 100000):")
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := &models.UserFilter{
		UserID:      userID,
		FilterType:  models.FilterTypeSalary,
		FilterValue: text,
	}

	if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
		ctx.Logger.Error("failed to save salary filter", zap.Error(err))
		return c.Send("😔 Ошибка при сохранении фильтра")
	}

	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	return c.Send(
		fmt.Sprintf("✅ Минимальная зарплата установлена: *%s ₽*", utils.EscapeMarkdown(text)),
		utils.FiltersMenuKeyboard(),
		tele.ModeMarkdownV2,
	)
}

// ==================== Experience Filter ====================

func startExperienceFilter(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	if err := setUserState(ctx, userID, StateAwaitingExp); err != nil {
		ctx.Logger.Error("failed to set user state", zap.Error(err))
	}

	return c.Send(
		"💼 Выберите требуемый опыт работы:",
		utils.ExperienceKeyboard(),
	)
}

func saveExperience(ctx *Context, c tele.Context, experience string) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get experience ID from map
	expID := models.GetExperienceID(experience)

	filter := &models.UserFilter{
		UserID:      userID,
		FilterType:  models.FilterTypeExperience,
		FilterValue: expID,
	}

	if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
		ctx.Logger.Error("failed to save experience filter", zap.Error(err))
		return c.Send("😔 Ошибка при сохранении фильтра")
	}

	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	return c.Send(
		fmt.Sprintf("✅ Опыт работы установлен: *%s*", utils.EscapeMarkdown(experience)),
		utils.FiltersMenuKeyboard(),
		tele.ModeMarkdownV2,
	)
}

// ==================== Schedule Filter ====================

func startScheduleFilter(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	if err := setUserState(ctx, userID, StateAwaitingSchedule); err != nil {
		ctx.Logger.Error("failed to set user state", zap.Error(err))
	}

	return c.Send(
		"⏰ Выберите желаемый график работы:",
		utils.ScheduleKeyboard(),
	)
}

func saveSchedule(ctx *Context, c tele.Context, schedule string) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get schedule ID from map
	scheduleID := models.GetScheduleID(schedule)

	filter := &models.UserFilter{
		UserID:      userID,
		FilterType:  models.FilterTypeSchedule,
		FilterValue: scheduleID,
	}

	if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
		ctx.Logger.Error("failed to save schedule filter", zap.Error(err))
		return c.Send("😔 Ошибка при сохранении фильтра")
	}

	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	return c.Send(
		fmt.Sprintf("✅ График работы установлен: *%s*", utils.EscapeMarkdown(schedule)),
		utils.FiltersMenuKeyboard(),
		tele.ModeMarkdownV2,
	)
}

// ==================== Period Filter ====================

func startPeriodFilter(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	if err := setUserState(ctx, userID, StateAwaitingPeriod); err != nil {
		ctx.Logger.Error("failed to set user state", zap.Error(err))
	}

	message := "🗓 За какой период показывать вакансии?\n" +
		fmt.Sprintf("Введите количество дней от %d до %d или выберите на клавиатуре.",
			models.MinPublishedWithinDays, models.MaxPublishedWithinDays)

	return c.Send(message, utils.PeriodKeyboard())
}

func handlePeriodFilterInput(ctx *Context, c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	userID := c.Sender().ID

	if text == "" || text == "❌ Отмена" {
		return cancelConversation(ctx, c)
	}

	days := extractDays(text)
	if days == 0 {
		return c.Send(
			fmt.Sprintf("Не получилось распознать число. Укажите от %d до %d дней.",
				models.MinPublishedWithinDays, models.MaxPublishedWithinDays),
			utils.PeriodKeyboard(),
		)
	}

	if days < models.MinPublishedWithinDays {
		days = models.MinPublishedWithinDays
	}
	if days > models.MaxPublishedWithinDays {
		days = models.MaxPublishedWithinDays
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := &models.UserFilter{
		UserID:      userID,
		FilterType:  models.FilterTypePublishedWithin,
		FilterValue: strconv.Itoa(days),
	}

	if err := ctx.Store.SaveFilter(dbCtx, filter); err != nil {
		ctx.Logger.Error("failed to save period filter", zap.Error(err))
		return c.Send("😔 Ошибка при сохранении периода")
	}

	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	return c.Send(
		fmt.Sprintf("✅ Период установлен: *за %s*", utils.EscapeMarkdown(utils.FormatDays(days))),
		utils.FiltersMenuKeyboard(),
		tele.ModeMarkdownV2,
	)
}

// ==================== Show & Clear Filters ====================

func showFilters(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filters, err := ctx.Store.GetUserFilters(dbCtx, userID)
	if err != nil {
		ctx.Logger.Error("failed to get user filters", zap.Error(err))
		return c.Send("😔 Ошибка при получении фильтров")
	}

	if len(filters) == 0 {
		return c.Send(
			"ℹ️ У вас нет установленных фильтров",
			utils.FiltersMenuKeyboard(),
		)
	}

	message := utils.FormatFiltersMessage(filters)

	return c.Send(
		message,
		utils.FiltersMenuKeyboard(),
		tele.ModeMarkdownV2,
	)
}

func clearFilters(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	if err := setUserState(ctx, userID, StateConfirmClear); err != nil {
		ctx.Logger.Warn("failed to set confirm clear state", zap.Error(err))
	}

	return c.Send(
		"🗑 Вы уверены, что хотите очистить все фильтры?",
		utils.ConfirmKeyboard(),
	)
}

func confirmClearFilters(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ctx.Store.ClearUserFilters(dbCtx, userID); err != nil {
		ctx.Logger.Error("failed to clear filters", zap.Error(err))
		return c.Send("😔 Ошибка при очистке фильтров")
	}

	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	return c.Send(
		"✅ Все фильтры удалены",
		utils.FiltersMenuKeyboard(),
	)
}

// ==================== State Management ====================

func handleStateInput(ctx *Context, c tele.Context, state string) error {
	switch state {
	case StateAwaitingText:
		return handleTextFilterInput(ctx, c)
	case StateAwaitingCity:
		return handleCityFilterInput(ctx, c)
	case StateAwaitingSalary:
		return handleSalaryFilterInput(ctx, c)
	case StateAwaitingExp:
		txt := strings.TrimSpace(c.Text())
		if !models.IsValidExperience(txt) {
			return c.Send("Выберите один из вариантов кнопками ниже", utils.ExperienceKeyboard())
		}
		return saveExperience(ctx, c, txt)
	case StateAwaitingSchedule:
		txt := strings.TrimSpace(c.Text())
		if !models.IsValidSchedule(txt) {
			return c.Send("Выберите один из вариантов кнопками ниже", utils.ScheduleKeyboard())
		}
		return saveSchedule(ctx, c, txt)
	case StateAwaitingPeriod:
		return handlePeriodFilterInput(ctx, c)
	case StateConfirmClear:
		return handleClearFiltersConfirm(ctx, c)
	default:
		_ = clearUserState(ctx, c.Sender().ID)
		return c.Reply("Используйте кнопки меню или команды")
	}
}

func handleClearFiltersConfirm(ctx *Context, c tele.Context) error {
	text := strings.TrimSpace(c.Text())

	switch text {
	case "✅ Да", "Да":
		return confirmClearFilters(ctx, c)
	case "❌ Нет", "Нет", "❌ Отмена":
		return cancelConversation(ctx, c)
	default:
		return c.Send(
			"Пожалуйста, выберите один из вариантов на клавиатуре",
			utils.ConfirmKeyboard(),
		)
	}
}

func setUserState(ctx *Context, userID int64, state string) error {
	key := fmt.Sprintf("user:%d:state", userID)
	return ctx.Cache.SetString(context.Background(), key, state, 30*time.Minute)
}

func getUserState(ctx *Context, userID int64) (string, error) {
	key := fmt.Sprintf("user:%d:state", userID)
	return ctx.Cache.GetString(context.Background(), key)
}

func clearUserState(ctx *Context, userID int64) error {
	key := fmt.Sprintf("user:%d:state", userID)
	return ctx.Cache.Delete(context.Background(), key)
}

func cancelConversation(ctx *Context, c tele.Context) error {
	userID := c.Sender().ID

	if err := clearUserState(ctx, userID); err != nil {
		ctx.Logger.Warn("failed to clear state", zap.Error(err))
	}

	return c.Send(
		"❌ Операция отменена",
		utils.FiltersMenuKeyboard(),
	)
}

// ==================== Helpers ====================

var digitsRegexp = regexp.MustCompile(`\d+`)

func extractDays(text string) int {
	match := digitsRegexp.FindString(text)
	if match == "" {
		return 0
	}

	days, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}

	return days
}

func parseIntervalText(text string) int {
	switch text {
	case "15 минут":
		return 15
	case "30 минут":
		return 30
	case "1 час":
		return 60
	case "2 часа":
		return 120
	case "6 часов":
		return 360
	case "12 часов":
		return 720
	default:
		return 0
	}
}

func toggleNotifications(ctx *Context, c tele.Context) error {
	text := strings.TrimSpace(c.Text())
	return HandleSettingsText(ctx, c, text)
}

func changeInterval(ctx *Context, c tele.Context) error {
	return c.Send(
		"⏰ Выберите интервал проверки новых вакансий:",
		utils.IntervalKeyboard(),
	)
}

func saveInterval(ctx *Context, c tele.Context, intervalMinutes int) error {
	userID := c.Sender().ID

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ctx.Store.SetNotifyInterval(dbCtx, userID, intervalMinutes); err != nil {
		ctx.Logger.Error("failed to set notify interval", zap.Error(err))
		return c.Send("😔 Ошибка при сохранении интервала")
	}

	user, err := ctx.Store.GetUser(dbCtx, userID)
	if err != nil {
		ctx.Logger.Error("failed to get user", zap.Error(err))
		return c.Send("😔 Ошибка при получении данных")
	}

	message := utils.FormatSettingsMessage(user)

	return c.Send(
		"✅ Интервал проверки обновлен\n\n"+message,
		utils.SettingsKeyboard(user.CheckEnabled),
		tele.ModeMarkdownV2,
	)
}
