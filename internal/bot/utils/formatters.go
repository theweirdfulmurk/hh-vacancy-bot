package utils

import (
	"fmt"
	"strings"

	"hh-vacancy-bot/internal/api/headhunter"
	"hh-vacancy-bot/internal/models"
)

// Format vacancy for Telegram
func FormatVacancy(vacancy *headhunter.VacancyItem) string {
	var sb strings.Builder

	// Vacancy name in bold
	sb.WriteString(fmt.Sprintf("*%s*\n\n", EscapeMarkdown(vacancy.Name)))

	// Company
	if vacancy.Employer.Name != "" {
		sb.WriteString(fmt.Sprintf("🏢 *Компания:* %s\n", EscapeMarkdown(vacancy.Employer.Name)))
	}

	// Paycheck
	if vacancy.Salary != nil {
		salaryStr := EscapeMarkdown(FormatSalary(vacancy.Salary))
		sb.WriteString(fmt.Sprintf("💰 *Зарплата:* %s\n", salaryStr))
	} else {
		sb.WriteString("💰 *Зарплата:* не указана\n")
	}

	// City
	sb.WriteString(fmt.Sprintf("📍 *Город:* %s\n", EscapeMarkdown(vacancy.Area.Name)))

	// Experience
	if vacancy.Experience != nil {
		sb.WriteString(fmt.Sprintf("💼 *Опыт:* %s\n", EscapeMarkdown(vacancy.Experience.Name)))
	}

	// Hours
	if vacancy.Schedule != nil {
		sb.WriteString(fmt.Sprintf("⏰ *График:* %s\n", EscapeMarkdown(vacancy.Schedule.Name)))
	}

	// Employment type
	if vacancy.Employment != nil {
		sb.WriteString(fmt.Sprintf("📋 *Занятость:* %s\n", EscapeMarkdown(vacancy.Employment.Name)))
	}

	// Published date
	publishedDate := vacancy.PublishedAt.Format("02.01.2006")
	sb.WriteString(fmt.Sprintf("📅 *Опубликовано:* %s\n", EscapeMarkdown(publishedDate)))

	// Link
	sb.WriteString(fmt.Sprintf("\n🔗 [Открыть вакансию](%s)", vacancy.AlternateURL))

	return sb.String()
}

func FormatSalary(salary *headhunter.Salary) string {
	currency := salary.Currency
	if currency == "RUR" || currency == "RUB" {
		currency = "₽"
	} else if currency == "USD" {
		currency = "$"
	} else if currency == "EUR" {
		currency = "€"
	}

	gross := ""
	if salary.Gross {
		gross = " (до вычета налогов)"
	}

	if salary.From != nil && salary.To != nil {
		// Changed order: currency symbol after amount (Russian style)
		return fmt.Sprintf("%d - %d %s%s", *salary.From, *salary.To, currency, gross)
	} else if salary.From != nil {
		return fmt.Sprintf("от %d %s%s", *salary.From, currency, gross)
	} else if salary.To != nil {
		return fmt.Sprintf("до %d %s%s", *salary.To, currency, gross)
	}

	return "не указана"
}

func FormatVacancyList(vacancies []headhunter.VacancyItem, total int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📋 *Найдено вакансий:* %d\n", total))
	sb.WriteString(fmt.Sprintf("*Показано:* %d\n\n", len(vacancies)))

	for i, vacancy := range vacancies {
		sb.WriteString(fmt.Sprintf("*%d\\. %s*\n", i+1, EscapeMarkdown(vacancy.Name)))
		
		if vacancy.Employer.Name != "" {
			sb.WriteString(fmt.Sprintf("   🏢 %s\n", EscapeMarkdown(vacancy.Employer.Name)))
		}
		
		if vacancy.Salary != nil {
			sb.WriteString(fmt.Sprintf("   💰 %s\n", EscapeMarkdown(FormatSalary(vacancy.Salary))))
		}
		
		sb.WriteString(fmt.Sprintf("   📍 %s\n", EscapeMarkdown(vacancy.Area.Name)))
		sb.WriteString("\n")
	}

	return sb.String()
}

func FormatUserFilters(filters map[string]string, cities []headhunter.City) string {
	var sb strings.Builder

	sb.WriteString("*Ваши текущие фильтры:*\n\n")

	if len(filters) == 0 {
		sb.WriteString("_Фильтры не установлены_\n")
		return sb.String()
	}

	if text, ok := filters[models.FilterTypeText]; ok && text != "" {
		sb.WriteString(fmt.Sprintf("🔍 *Текст:* %s\n", EscapeMarkdown(text)))
	}

	if areaID, ok := filters[models.FilterTypeArea]; ok && areaID != "" {
		cityName := areaID
		for _, city := range cities {
			if city.ID == areaID {
				cityName = city.Name
				break
			}
		}
		sb.WriteString(fmt.Sprintf("📍 *Город:* %s\n", EscapeMarkdown(cityName)))
	}

	if salary, ok := filters[models.FilterTypeSalary]; ok && salary != "" {
		sb.WriteString(fmt.Sprintf("💰 *Зарплата от:* %s ₽\n", salary))
	}

	if exp, ok := filters[models.FilterTypeExperience]; ok && exp != "" {
		expName := models.ExperienceDisplayNames[exp]
		if expName == "" {
			expName = exp
		}
		sb.WriteString(fmt.Sprintf("💼 *Опыт:* %s\n", EscapeMarkdown(expName)))
	}

	if schedule, ok := filters[models.FilterTypeSchedule]; ok && schedule != "" {
		scheduleName := models.ScheduleDisplayNames[schedule]
		if scheduleName == "" {
			scheduleName = schedule
		}
		sb.WriteString(fmt.Sprintf("⏰ *График:* %s\n", EscapeMarkdown(scheduleName)))
	}

	return sb.String()
}

func FormatWelcomeMessage(firstName string) string {
	name := firstName
	if name == "" {
		name = "друг"
	}

	return fmt.Sprintf(`👋 Привет, *%s*\!

Я бот для поиска вакансий на HeadHunter\.

*Что я умею:*
• Искать вакансии по вашим фильтрам
• Автоматически уведомлять о новых вакансиях
• Фильтровать по городу, зарплате, опыту и графику

*Команды:*
/filters \- настроить фильтры поиска
/vacancies \- получить вакансии
/settings \- настройки уведомлений
/help \- справка

Начните с настройки фильтров \- /filters`, EscapeMarkdown(name))
}

func FormatHelpMessage() string {
	return `*📖 Справка*

*Основные команды:*

/start \- начать работу с ботом
/filters \- настроить фильтры поиска
/vacancies \- получить вакансии по фильтрам
/settings \- настройки уведомлений
/help \- справка

*Как работать с ботом:*

1️⃣ Настройте фильтры командой /filters
   \- Укажите город
   \- Минимальную зарплату
   \- Опыт работы
   \- График работы
   \- Ключевые слова

2️⃣ Получите вакансии командой /vacancies

3️⃣ Включите автоматические уведомления в /settings

*По вопросам:* @dinabyebye & @theweirdfulmurk`
}

func FormatNoFiltersMessage() string {
	return `⚠️ *У вас нет установленных фильтров*

Настройте фильтры командой /filters, чтобы начать поиск вакансий\.`
}

func FormatNoVacanciesMessage() string {
	return `😔 *Вакансии не найдены*

Попробуйте изменить фильтры командой /filters`
}

func FormatSettingsMessage(user *models.User) string {
	var sb strings.Builder

	sb.WriteString("*⚙️ Настройки уведомлений*\n\n")

	status := "❌ Отключены"
	if user.CheckEnabled {
		status = "✅ Включены"
	}
	sb.WriteString(fmt.Sprintf("*Статус:* %s\n", status))

	sb.WriteString(fmt.Sprintf("*Интервал:* каждые %d минут\n", user.NotifyInterval))

	return sb.String()
}

func FormatFiltersMessage(filters []models.UserFilter) string {
	if len(filters) == 0 {
		return "ℹ️ У вас нет установленных фильтров"
	}

	var sb strings.Builder
	sb.WriteString("*📋 Ваши фильтры:*\n\n")

	for _, filter := range filters {
		filterName := getFilterDisplayName(filter.FilterType)
		filterValue := formatFilterValue(filter.FilterType, filter.FilterValue)
		
		sb.WriteString(fmt.Sprintf("• *%s:* %s\n", 
			EscapeMarkdown(filterName),
			EscapeMarkdown(filterValue),
		))
	}

	return sb.String()
}

func getFilterDisplayName(filterType string) string {
	switch filterType {
	case models.FilterTypeText:
		return "Текст поиска"
	case models.FilterTypeArea:
		return "Город"
	case models.FilterTypeSalary:
		return "Минимальная зарплата"
	case models.FilterTypeExperience:
		return "Опыт"
	case models.FilterTypeSchedule:
		return "График"
	default:
		return filterType
	}
}

func formatFilterValue(filterType, value string) string {
	switch filterType {
	case models.FilterTypeSalary:
		return value + " ₽"
	case models.FilterTypeExperience:
		return models.GetExperienceDisplayName(value)
	case models.FilterTypeSchedule:
		return models.GetScheduleDisplayName(value)
	default:
		return value
	}
}

// EscapeMarkdown escapes special characters for Telegram MarkdownV2
func EscapeMarkdown(text string) string {
	// _ * [ ] ( ) ~ ` > # + - = | { } . !
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)

	return replacer.Replace(text)
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}