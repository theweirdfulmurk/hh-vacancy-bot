package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"hh-vacancy-bot/internal/api/headhunter"
	"hh-vacancy-bot/internal/bot/middleware"
	"hh-vacancy-bot/internal/bot/utils"
	"hh-vacancy-bot/internal/config"
	"hh-vacancy-bot/internal/models"
	"hh-vacancy-bot/internal/storage/postgres"
	"hh-vacancy-bot/internal/storage/redis"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type VacancyChecker struct {
	bot      *tele.Bot
	store    *postgres.Store
	cache    *redis.Cache
	hhClient *headhunter.Client
	config   *config.Config
	logger   *zap.Logger
}

func New(
	bot *tele.Bot,
	store *postgres.Store,
	cache *redis.Cache,
	hhClient *headhunter.Client,
	cfg *config.Config,
	logger *zap.Logger,
) *VacancyChecker {
	return &VacancyChecker{
		bot:      bot,
		store:    store,
		cache:    cache,
		hhClient: hhClient,
		config:   cfg,
		logger:   logger,
	}
}

func (vc *VacancyChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(vc.config.CheckInterval)
	defer ticker.Stop()

	vc.logger.Info("vacancy checker started",
		zap.Duration("interval", vc.config.CheckInterval),
	)

	time.Sleep(30 * time.Second)
	vc.checkVacanciesForAllUsers(ctx)

	for {
		select {
		case <-ctx.Done():
			vc.logger.Info("vacancy checker stopped")
			return
		case <-ticker.C:
			vc.checkVacanciesForAllUsers(ctx)
		}
	}
}

func (vc *VacancyChecker) checkVacanciesForAllUsers(ctx context.Context) {
	vc.logger.Info("starting vacancy check for all users")

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	users, err := vc.store.GetUsersToCheck(dbCtx)
	if err != nil {
		vc.logger.Error("failed to get users to check", zap.Error(err))
		return
	}

	if len(users) == 0 {
		vc.logger.Debug("no users to check")
		return
	}

	vc.logger.Info("checking vacancies for users", zap.Int("count", len(users)))

	for _, user := range users {
		if err := vc.checkVacanciesForUser(dbCtx, &user); err != nil {
			vc.logger.Error("failed to check vacancies for user",
				zap.Int64("user_id", user.ID),
				zap.Error(err),
			)
			continue
		}

		if err := vc.store.UpdateLastCheck(dbCtx, user.ID); err != nil {
			vc.logger.Error("failed to update last check",
				zap.Int64("user_id", user.ID),
				zap.Error(err),
			)
		}

		time.Sleep(2 * time.Second)
	}

	vc.logger.Info("finished vacancy check for all users")
}

func (vc *VacancyChecker) checkVacanciesForUser(ctx context.Context, user *models.User) error {
	vc.logger.Debug("checking vacancies for user", zap.Int64("user_id", user.ID))

	filtersMap, err := vc.store.GetFiltersMap(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get filters: %w", err)
	}

	if len(filtersMap) == 0 {
		vc.logger.Debug("user has no filters", zap.Int64("user_id", user.ID))
		return nil
	}

	if err := middleware.CheckHHAPIRateLimit(vc.cache, vc.logger); err != nil {
		vc.logger.Warn("HH API rate limit, skipping user", zap.Int64("user_id", user.ID))
		return nil
	}

	searchParams := buildSearchParams(filtersMap)

	searchParams.PerPage = vc.config.MaxVacanciesPerCheck

	response, err := vc.hhClient.SearchVacancies(ctx, searchParams)
	if err != nil {
		return fmt.Errorf("search vacancies: %w", err)
	}

	if len(response.Items) == 0 {
		vc.logger.Debug("no vacancies found", zap.Int64("user_id", user.ID))
		return nil
	}

	vacancyIDs := headhunter.ExtractVacancyIDs(response)
	unseenIDs, err := vc.store.GetUnseenVacancies(ctx, user.ID, vacancyIDs)
	if err != nil {
		return fmt.Errorf("get unseen vacancies: %w", err)
	}

	if len(unseenIDs) == 0 {
		vc.logger.Debug("no new vacancies", zap.Int64("user_id", user.ID))
		return nil
	}

	var newVacancies []headhunter.VacancyItem
	unseenMap := make(map[string]bool)
	for _, id := range unseenIDs {
		unseenMap[id] = true
	}

	for _, vacancy := range response.Items {
		if unseenMap[vacancy.ID] {
			newVacancies = append(newVacancies, vacancy)
		}
	}

	if err := vc.sendNotifications(ctx, user.ID, newVacancies); err != nil {
		return fmt.Errorf("send notifications: %w", err)
	}

	go vc.cacheVacancies(newVacancies)

	go vc.markVacanciesAsSeen(user.ID, newVacancies)

	vc.logger.Info("sent new vacancies to user",
		zap.Int64("user_id", user.ID),
		zap.Int("count", len(newVacancies)),
	)

	return nil
}

func (vc *VacancyChecker) sendNotifications(ctx context.Context, userID int64, vacancies []headhunter.VacancyItem) error {
	recipient := &tele.User{ID: userID}

	summaryMsg := fmt.Sprintf(
		"🔔 *Новые вакансии\\!*\n\nНайдено новых вакансий: %d\n\n",
		len(vacancies),
	)

	if _, err := vc.bot.Send(recipient, summaryMsg, tele.ModeMarkdownV2); err != nil {
		return fmt.Errorf("send summary: %w", err)
	}

	for i, vacancy := range vacancies {
		message := utils.FormatVacancy(&vacancy)
		keyboard := utils.InlineVacancyKeyboard(vacancy.AlternateURL)

		if _, err := vc.bot.Send(recipient, message, keyboard, tele.ModeMarkdownV2); err != nil {
			vc.logger.Error("failed to send vacancy notification",
				zap.Int64("user_id", userID),
				zap.String("vacancy_id", vacancy.ID),
				zap.Error(err),
			)
			continue
		}

		if i < len(vacancies)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

func (vc *VacancyChecker) cacheVacancies(vacancies []headhunter.VacancyItem) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, item := range vacancies {
		vacancy := convertToDBVacancy(&item)

		if err := vc.store.CacheVacancy(ctx, vacancy); err != nil {
			vc.logger.Error("failed to cache vacancy",
				zap.String("vacancy_id", vacancy.ID),
				zap.Error(err),
			)
		}
	}
}

func (vc *VacancyChecker) markVacanciesAsSeen(userID int64, vacancies []headhunter.VacancyItem) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, vacancy := range vacancies {
		if err := vc.store.MarkVacancyAsSeen(ctx, userID, vacancy.ID); err != nil {
			vc.logger.Error("failed to mark vacancy as seen",
				zap.Int64("user_id", userID),
				zap.String("vacancy_id", vacancy.ID),
				zap.Error(err),
			)
		}
	}
}

func buildSearchParams(filters map[string]string) headhunter.VacancySearchParams {
	params := headhunter.VacancySearchParams{
		Page:    0,
		PerPage: 20,
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

	return params
}

func convertToDBVacancy(item *headhunter.VacancyItem) *models.Vacancy {
	vacancy := &models.Vacancy{
		ID:          item.ID,
		Title:       item.Name,
		Area:        item.Area.Name,
		AreaID:      item.Area.ID,
		URL:         item.AlternateURL,
		PublishedAt: item.PublishedAt,
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

	return vacancy
}