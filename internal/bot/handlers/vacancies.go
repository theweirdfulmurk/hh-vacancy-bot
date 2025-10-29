package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"hh-vacancy-bot/internal/api/headhunter"
	"hh-vacancy-bot/internal/bot/middleware"
	"hh-vacancy-bot/internal/bot/utils"
	"hh-vacancy-bot/internal/models"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// /vacancies
func HandleVacancies(ctx *Context) tele.HandlerFunc {
	return func(c tele.Context) error {
		userID := c.Sender().ID

		dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		filtersMap, err := ctx.Store.GetFiltersMap(dbCtx, userID)
		if err != nil {
			ctx.Logger.Error("failed to get user filters", zap.Error(err))
			return c.Reply("😔 Ошибка при получении фильтров")
		}

		if len(filtersMap) == 0 {
			message := utils.FormatNoFiltersMessage()
			return c.Send(message, tele.ModeMarkdownV2)
		}

		searchMsg, _ := c.Bot().Send(c.Recipient(), "🔍 Ищу вакансии...")

		if err := middleware.CheckHHAPIRateLimit(ctx.Cache, ctx.Logger); err != nil {
			ctx.Logger.Warn("HH API rate limit", zap.Error(err))
			c.Bot().Edit(searchMsg, "⚠️ Слишком много запросов. Попробуйте через минуту.")
			return nil
		}

		searchParams := buildSearchParams(filtersMap)
		if ctx.Config.MaxVacanciesPerCheck > 0 {
			searchParams.PerPage = ctx.Config.MaxVacanciesPerCheck
		}

		response, err := ctx.HHClient.SearchVacancies(dbCtx, searchParams)
		if err != nil {
			ctx.Logger.Error("failed to search vacancies",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			c.Bot().Edit(searchMsg, "😔 Ошибка при поиске вакансий")
			return nil
		}

		c.Bot().Delete(searchMsg)

		if len(response.Items) == 0 {
			message := utils.FormatNoVacanciesMessage()
			return c.Send(message, tele.ModeMarkdownV2)
		}

		go cacheVacancies(ctx, response.Items)

		vacancyIDs := headhunter.ExtractVacancyIDs(response)
		unseenIDs, err := ctx.Store.GetUnseenVacancies(dbCtx, userID, vacancyIDs)
		if err != nil {
			ctx.Logger.Error("failed to get unseen vacancies", zap.Error(err))
			unseenIDs = vacancyIDs
		}

		var unseenVacancies []headhunter.VacancyItem
		unseenMap := make(map[string]bool)
		for _, id := range unseenIDs {
			unseenMap[id] = true
		}

		for _, vacancy := range response.Items {
			if unseenMap[vacancy.ID] {
				unseenVacancies = append(unseenVacancies, vacancy)
			}
		}

		var delivered []headhunter.VacancyItem

		cleanupPaginationMessages(ctx, c, userID)

		if len(unseenVacancies) == 0 {
			infoMessage := fmt.Sprintf(
				"ℹ️ *Новых вакансий нет*\n\nПоказываю вакансии за %s.",
				utils.EscapeMarkdown(utils.FormatDays(searchParams.PublishedWithinDays)),
			)

			if err := c.Send(infoMessage, tele.ModeMarkdownV2); err != nil {
				ctx.Logger.Error("failed to send no-new-vacancies message", zap.Error(err))
				return c.Reply("😔 Ошибка при отправке вакансий")
			}

			messageIDs, err := deliverVacancyCards(ctx, c, response.Items, userID)
			if err != nil {
				ctx.Logger.Error("failed to send historical vacancies", zap.Error(err))
				return c.Reply("😔 Ошибка при отправке вакансий")
			}

			rememberPaginationMessages(ctx, userID, messageIDs)

			delivered = response.Items
		} else {
			maxVacancies := ctx.Config.MaxVacanciesPerCheck
			if maxVacancies > 0 && len(unseenVacancies) > maxVacancies {
				unseenVacancies = unseenVacancies[:maxVacancies]
			}

			messageIDs, err := sendVacanciesToUser(ctx, c, unseenVacancies, userID)
			if err != nil {
				ctx.Logger.Error("failed to send vacancies", zap.Error(err))
				return c.Reply("😔 Ошибка при отправке вакансий")
			}

			rememberPaginationMessages(ctx, userID, messageIDs)

			delivered = unseenVacancies
		}

		if len(delivered) > 0 {
			go markVacanciesAsSeen(ctx, userID, delivered)
		}

		sendPaginationControls(ctx, c, response.Page, response.Pages, searchParams.PublishedWithinDays)

		return nil
	}
}

func buildSearchParams(filters map[string]string) headhunter.VacancySearchParams {
	params := headhunter.VacancySearchParams{
		Page:                0,
		PerPage:             20,
		PublishedWithinDays: models.DefaultPublishedWithinDays,
	}

	if text, ok := filters[models.FilterTypeText]; ok {
		params.Text = text
	}

	if area, ok := filters[models.FilterTypeArea]; ok {
		params.Area = area
	}

	if salary, ok := filters[models.FilterTypeSalary]; ok {
		if s, err := strconv.Atoi(salary); err == nil {
			params.Salary = s
		}
	}

	if exp, ok := filters[models.FilterTypeExperience]; ok {
		params.Experience = exp
	}

	if schedule, ok := filters[models.FilterTypeSchedule]; ok {
		params.Schedule = schedule
	}

	days := params.PublishedWithinDays
	if raw, ok := filters[models.FilterTypePublishedWithin]; ok {
		if parsed, err := strconv.Atoi(raw); err == nil {
			if parsed < models.MinPublishedWithinDays {
				parsed = models.MinPublishedWithinDays
			}
			if parsed > models.MaxPublishedWithinDays {
				parsed = models.MaxPublishedWithinDays
			}
			days = parsed
		}
	}

	params.PublishedWithinDays = days

	now := time.Now()
	dateTo := now
	from := now.Add(-time.Duration(days) * 24 * time.Hour)
	params.DateTo = &dateTo
	params.DateFrom = &from

	return params
}

func sendVacanciesToUser(ctx *Context, c tele.Context, vacancies []headhunter.VacancyItem, userID int64) ([]int, error) {
	summaryMsg := fmt.Sprintf(
		"📋 *Найдено новых вакансий:* %d\n\n",
		len(vacancies),
	)

	sent, err := c.Bot().Send(
		c.Chat(),
		summaryMsg,
		&tele.SendOptions{ParseMode: tele.ModeMarkdownV2},
	)
	if err != nil {
		return nil, err
	}

	messageIDs, err := deliverVacancyCards(ctx, c, vacancies, userID)
	if err != nil {
		return nil, err
	}

	messageIDs = append([]int{sent.ID}, messageIDs...)

	return messageIDs, nil
}

func deliverVacancyCards(ctx *Context, c tele.Context, vacancies []headhunter.VacancyItem, userID int64) ([]int, error) {
	var messageIDs []int

	for i, vacancy := range vacancies {
		message := utils.FormatVacancy(&vacancy)

		keyboard := utils.InlineVacancyKeyboard(vacancy.AlternateURL)

		sent, err := c.Bot().Send(
			c.Chat(),
			message,
			&tele.SendOptions{ParseMode: tele.ModeMarkdownV2, ReplyMarkup: keyboard},
		)
		if err != nil {
			ctx.Logger.Error("failed to send vacancy",
				zap.Int("index", i),
				zap.Int64("user_id", userID),
				zap.String("vacancy_id", vacancy.ID),
				zap.Error(err),
			)
			continue
		}

		messageIDs = append(messageIDs, sent.ID)

		if i < len(vacancies)-1 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	return messageIDs, nil
}

func sendPaginationControls(ctx *Context, c tele.Context, page, totalPages, days int) {
	if totalPages <= 1 {
		return
	}

	text := fmt.Sprintf("📄 Страница %d из %d", page+1, totalPages)
	if days > 0 {
		text = fmt.Sprintf("%s • за %s", text, utils.FormatDays(days))
	}

	if err := c.Send(text, utils.InlinePaginationKeyboard(page, totalPages, "vacancy_page")); err != nil {
		ctx.Logger.Warn("failed to send pagination controls", zap.Error(err))
	}
}

func sendPaginationControls(ctx *Context, c tele.Context, page, totalPages, days int) {
	if totalPages <= 1 {
		return
	}

	text := fmt.Sprintf("📄 Страница %d из %d", page+1, totalPages)
	if days > 0 {
		text = fmt.Sprintf("%s • за %s", text, utils.FormatDays(days))
	}

	if _, err := c.Send(text, utils.InlinePaginationKeyboard(page, totalPages, "vacancy_page")); err != nil {
		ctx.Logger.Warn("failed to send pagination controls", zap.Error(err))
	}
}

func cacheVacancies(ctx *Context, vacancies []headhunter.VacancyItem) {
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, item := range vacancies {
		vacancy := &models.Vacancy{
			ID:          item.ID,
			Title:       item.Name,
			Area:        item.Area.Name,
			AreaID:      item.Area.ID,
			URL:         item.AlternateURL,
			PublishedAt: item.PublishedAt.Time,
		}

		if item.Employer.Name != "" {
			vacancy.Company = &item.Employer.Name
		}

		if item.Salary != nil {
			vacancy.SalaryFrom = item.Salary.From
			vacancy.SalaryTo = item.Salary.To
			vacancy.Currency = &item.Salary.Currency
		}

		if item.Experience != nil {
			vacancy.Experience = &item.Experience.Name
		}

		if item.Schedule != nil {
			vacancy.Schedule = &item.Schedule.Name
		}

		if item.Employment != nil {
			vacancy.Employment = &item.Employment.Name
		}

		if err := ctx.Store.CacheVacancy(dbCtx, vacancy); err != nil {
			ctx.Logger.Error("failed to cache vacancy",
				zap.String("vacancy_id", vacancy.ID),
				zap.Error(err),
			)
		}
	}
}

func markVacanciesAsSeen(ctx *Context, userID int64, vacancies []headhunter.VacancyItem) {
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, vacancy := range vacancies {
		if err := ctx.Store.MarkVacancyAsSeen(dbCtx, userID, vacancy.ID); err != nil {
			ctx.Logger.Error("failed to mark vacancy as seen",
				zap.Int64("user_id", userID),
				zap.String("vacancy_id", vacancy.ID),
				zap.Error(err),
			)
		}
	}

	ctx.Logger.Info("marked vacancies as seen",
		zap.Int64("user_id", userID),
		zap.Int("count", len(vacancies)),
	)
}

const (
	paginationMessagesKey = "pagination_messages"
	paginationMessagesTTL = 15 * time.Minute
)

func rememberPaginationMessages(ctx *Context, userID int64, messageIDs []int) {
	if ctx.Cache == nil || len(messageIDs) == 0 {
		return
	}

	cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ctx.Cache.SetTempData(cacheCtx, userID, paginationMessagesKey, messageIDs, paginationMessagesTTL); err != nil {
		ctx.Logger.Warn("failed to store pagination messages",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
	}
}

func cleanupPaginationMessages(ctx *Context, c tele.Context, userID int64) {
	if ctx.Cache == nil {
		return
	}

	cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var messageIDs []int
	if err := ctx.Cache.GetTempData(cacheCtx, userID, paginationMessagesKey, &messageIDs); err != nil {
		ctx.Logger.Debug("no cached pagination messages",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return
	}

	chat := c.Chat()
	if chat == nil {
		ctx.Logger.Warn("unable to cleanup pagination messages: chat is nil",
			zap.Int64("user_id", userID),
		)
		return
	}

	for _, id := range messageIDs {
		msg := &tele.Message{ID: id, Chat: chat}
		if err := c.Bot().Delete(msg); err != nil {
			ctx.Logger.Warn("failed to delete old vacancy message",
				zap.Int("message_id", id),
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
		}
	}

	if err := ctx.Cache.DeleteTempData(cacheCtx, userID, paginationMessagesKey); err != nil {
		ctx.Logger.Warn("failed to clear pagination cache",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
	}
}
