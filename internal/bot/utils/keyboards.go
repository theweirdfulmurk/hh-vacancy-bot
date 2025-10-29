package utils

import (
	"hh-vacancy-bot/internal/models"
	"strconv"

	tele "gopkg.in/telebot.v3"
)

func MainMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnFilters := menu.Text("🔧 Фильтры")
	btnVacancies := menu.Text("📋 Вакансии")
	btnSettings := menu.Text("⚙️ Настройки")
	btnHelp := menu.Text("❓ Справка")

	menu.Reply(
		menu.Row(btnFilters, btnVacancies),
		menu.Row(btnSettings, btnHelp),
	)

	return menu
}

func FiltersMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnText := menu.Text("🔍 Текст поиска")
	btnCity := menu.Text("📍 Город")
	btnSalary := menu.Text("💰 Зарплата")
	btnExperience := menu.Text("💼 Опыт")
	btnSchedule := menu.Text("⏰ График")
	btnShow := menu.Text("📊 Показать фильтры")
	btnClear := menu.Text("🗑 Очистить фильтры")
	btnBack := menu.Text("◀️ Назад")

	menu.Reply(
		menu.Row(btnText, btnCity),
		menu.Row(btnSalary, btnExperience),
		menu.Row(btnSchedule),
		menu.Row(btnShow, btnClear),
		menu.Row(btnBack),
	)

	return menu
}

func ExperienceKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}

	options := models.ExperienceOptions()
	var rows []tele.Row

	for _, option := range options {
		btn := menu.Text(option)
		rows = append(rows, menu.Row(btn))
	}

	btnCancel := menu.Text("❌ Отмена")
	rows = append(rows, menu.Row(btnCancel))

	menu.Reply(rows...)

	return menu
}

func ScheduleKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}

	options := models.ScheduleOptions()
	var rows []tele.Row

	for _, option := range options {
		btn := menu.Text(option)
		rows = append(rows, menu.Row(btn))
	}

	btnCancel := menu.Text("❌ Отмена")
	rows = append(rows, menu.Row(btnCancel))

	menu.Reply(rows...)

	return menu
}

func CancelKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	
	btnCancel := menu.Text("❌ Отмена")
	menu.Reply(menu.Row(btnCancel))

	return menu
}

func SettingsKeyboard(checkEnabled bool) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}

	var btnToggle tele.Btn
	if checkEnabled {
		btnToggle = menu.Text("🔕 Отключить уведомления")
	} else {
		btnToggle = menu.Text("🔔 Включить уведомления")
	}

	btnInterval := menu.Text("⏰ Изменить интервал")
	btnBack := menu.Text("◀️ Назад")

	menu.Reply(
		menu.Row(btnToggle),
		menu.Row(btnInterval),
		menu.Row(btnBack),
	)

	return menu
}

func IntervalKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}

	btn15 := menu.Text("15 минут")
	btn30 := menu.Text("30 минут")
	btn60 := menu.Text("1 час")
	btn120 := menu.Text("2 часа")
	btn360 := menu.Text("6 часов")
	btn720 := menu.Text("12 часов")
	btnCancel := menu.Text("❌ Отмена")

	menu.Reply(
		menu.Row(btn15, btn30),
		menu.Row(btn60, btn120),
		menu.Row(btn360, btn720),
		menu.Row(btnCancel),
	)

	return menu
}

func RemoveKeyboard() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{RemoveKeyboard: true}
}

func InlineVacancyKeyboard(vacancyURL string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnOpen := menu.URL("🔗 Открыть вакансию", vacancyURL)

	menu.Inline(
		menu.Row(btnOpen),
	)

	return menu
}

func InlinePaginationKeyboard(page, totalPages int, callbackPrefix string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	// no pagination needed
	if totalPages <= 1 {
		return menu
	}

	var buttons []tele.Btn

	if page > 0 {
		btnPrev := menu.Data("⬅️ Назад", callbackPrefix+"_prev", strconv.Itoa(page-1))
		buttons = append(buttons, btnPrev)
	}

	// show 1-based current page like "2/7"
	btnCurrent := menu.Data(strconv.Itoa(page+1)+"/"+strconv.Itoa(totalPages), callbackPrefix+"_current", "noop")
	buttons = append(buttons, btnCurrent)

	if page < totalPages-1 {
		btnNext := menu.Data("Вперёд ➡️", callbackPrefix+"_next", strconv.Itoa(page+1))
		buttons = append(buttons, btnNext)
	}

	menu.Inline(menu.Row(buttons...))
	return menu
}

func ConfirmKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}

	btnYes := menu.Text("✅ Да")
	btnNo := menu.Text("❌ Нет")

	menu.Reply(menu.Row(btnYes, btnNo))

	return menu
}