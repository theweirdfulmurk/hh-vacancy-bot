package utils

import (
	"fmt"
	"strconv"
	"strings"

	"hh-vacancy-bot/internal/api/headhunter"
	"hh-vacancy-bot/internal/models"
)

var snippetHighlightReplacer = strings.NewReplacer("<highlighttext>", "", "</highlighttext>", "")

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

	if len(vacancy.ProfessionalRoles) > 0 {
		roles := make([]string, 0, len(vacancy.ProfessionalRoles))
		for _, role := range vacancy.ProfessionalRoles {
			if role.Name != "" {
				roles = append(roles, role.Name)
			}
		}
		if len(roles) > 0 {
			sb.WriteString(fmt.Sprintf("🧭 *Профиль:* %s\n", EscapeMarkdown(strings.Join(roles, ", "))))
		}
	}

	if vacancy.Snippet != nil {
		if vacancy.Snippet.Requirement != nil {
			if requirement := formatSnippetField(*vacancy.Snippet.Requirement); requirement != "" {
				sb.WriteString(fmt.Sprintf("🗣️ *Требования:* %s\n", EscapeMarkdown(requirement)))
			}
		}
		if vacancy.Snippet.Responsibility != nil {
			if responsibility := formatSnippetField(*vacancy.Snippet.Responsibility); responsibility != "" {
				sb.WriteString(fmt.Sprintf("✍️ *Задачи:* %s\n", EscapeMarkdown(responsibility)))
			}
		}
	}

	// Published date
	publishedDate := vacancy.PublishedAt.Format("02.01.2006")
	sb.WriteString(fmt.Sprintf("📅 *Опубликовано:* %s\n", EscapeMarkdown(publishedDate)))

	// Link
	sb.WriteString(fmt.Sprintf("\n🔗 [Открыть вакансию](%s)", escapeMarkdownURL(vacancy.AlternateURL)))

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

	var sb strings.Builder
	sb.WriteString("👋 Привет, *")
	sb.WriteString(EscapeMarkdown(name))
	sb.WriteString("*\n")
	sb.WriteString(EscapeMarkdown("Я – Лингвокот!😼🔎 Я помогаю находить вакансии для лингвистов и переводчиков (и не только!) на HeadHunter."))
	sb.WriteString("\n\n")
	sb.WriteString("• ")
	sb.WriteString(EscapeMarkdown("Базовый поиск уже включает «лингвист», «переводчик», «NLP»"))
	sb.WriteString("\n")
	sb.WriteString("• ")
	sb.WriteString(EscapeMarkdown("Карточки выделяют языковые требования и профиль"))
	sb.WriteString("\n\n")
	sb.WriteString(EscapeMarkdown("Команды: /vacancies, /settings, /help"))
	sb.WriteString("\n\n")
	sb.WriteString(EscapeMarkdown("Добавьте свой язык, город и условия в /filters"))

	return sb.String()
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
	return `⚠️ *Фильтры не заданы*

Базовый поиск по словам «лингвист», «переводчик», «NLP» уже активен.
Добавьте язык, город и условия через /filters, чтобы получать более точные вакансии\.`
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
	case models.FilterTypePublishedWithin:
		return "Период публикации"
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
	case models.FilterTypePublishedWithin:
		if days, err := strconv.Atoi(value); err == nil {
			return "за " + FormatDays(days)
		}
		return "за " + value + " дней"
	default:
		return value
	}
}

func FormatDays(days int) string {
	if days <= 0 {
		return "0 дней"
	}

	remainder10 := days % 10
	remainder100 := days % 100

	var suffix string
	switch {
	case remainder10 == 1 && remainder100 != 11:
		suffix = "день"
	case remainder10 >= 2 && remainder10 <= 4 && (remainder100 < 12 || remainder100 > 14):
		suffix = "дня"
	default:
		suffix = "дней"
	}

	return fmt.Sprintf("%d %s", days, suffix)
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

func escapeMarkdownURL(url string) string {
	replacer := strings.NewReplacer(
		"(", "\\(",
		")", "\\)",
		"\\", "\\\\",
	)

	return replacer.Replace(url)
}

func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	const ellipsis = "..."
	ellipsisLen := len([]rune(ellipsis))

	if maxLen <= ellipsisLen {
		return string(runes[:maxLen])
	}

	return string(runes[:maxLen-ellipsisLen]) + ellipsis
}

func formatSnippetField(raw string) string {
	cleaned := snippetHighlightReplacer.Replace(raw)
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return ""
	}
	return TruncateString(cleaned, 180)
}
