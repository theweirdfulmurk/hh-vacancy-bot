package postgres

import (
	"context"
	"fmt"
	"time"

	"hh-vacancy-bot/internal/models"

	"github.com/gocraft/dbr/v2"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func (s *Store) CacheVacancy(ctx context.Context, vacancy *models.Vacancy) error {
	// using plain SQL via InsertBySql for ON CONFLICT
	query := `
		INSERT INTO vacancies_cache (
			id, title, company, salary_from, salary_to, currency,
			area, area_id, url, published_at, experience, schedule,
			employment, raw_data, cached_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			company = EXCLUDED.company,
			salary_from = EXCLUDED.salary_from,
			salary_to = EXCLUDED.salary_to,
			currency = EXCLUDED.currency,
			area = EXCLUDED.area,
			area_id = EXCLUDED.area_id,
			url = EXCLUDED.url,
			published_at = EXCLUDED.published_at,
			experience = EXCLUDED.experience,
			schedule = EXCLUDED.schedule,
			employment = EXCLUDED.employment,
			raw_data = EXCLUDED.raw_data,
			cached_at = EXCLUDED.cached_at
	`

	_, err := s.sess.
		InsertBySql(query,
			vacancy.ID,
			vacancy.Title,
			vacancy.Company,
			vacancy.SalaryFrom,
			vacancy.SalaryTo,
			vacancy.Currency,
			vacancy.Area,
			vacancy.AreaID,
			vacancy.URL,
			vacancy.PublishedAt,
			vacancy.Experience,
			vacancy.Schedule,
			vacancy.Employment,
			vacancy.RawData,
			time.Now(),
		).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to cache vacancy",
			zap.String("vacancy_id", vacancy.ID),
			zap.Error(err),
		)
		return fmt.Errorf("cache vacancy: %w", err)
	}

	return nil
}

func (s *Store) GetVacancy(ctx context.Context, vacancyID string) (*models.Vacancy, error) {
	var vacancy models.Vacancy

	err := s.sess.
		Select("*").
		From("vacancies_cache").
		Where("id = ?", vacancyID).
		LoadOneContext(ctx, &vacancy)

	if err == dbr.ErrNotFound {
		return nil, nil
	}

	if err != nil {
		s.logger.Error("failed to get vacancy",
			zap.String("vacancy_id", vacancyID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get vacancy: %w", err)
	}

	return &vacancy, nil
}

func (s *Store) MarkVacancyAsSeen(ctx context.Context, userID int64, vacancyID string) error {
	query := `
		INSERT INTO user_seen_vacancies (user_id, vacancy_id, seen_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, vacancy_id) DO NOTHING
	`

	_, err := s.sess.
		InsertBySql(query, userID, vacancyID).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to mark vacancy as seen",
			zap.Int64("user_id", userID),
			zap.String("vacancy_id", vacancyID),
			zap.Error(err),
		)
		return fmt.Errorf("mark vacancy as seen: %w", err)
	}

	return nil
}

func (s *Store) IsVacancySeen(ctx context.Context, userID int64, vacancyID string) (bool, error) {
	var count int

	err := s.sess.
		Select("COUNT(*)").
		From("user_seen_vacancies").
		Where("user_id = ? AND vacancy_id = ?", userID, vacancyID).
		LoadOneContext(ctx, &count)

	if err != nil {
		s.logger.Error("failed to check if vacancy is seen",
			zap.Int64("user_id", userID),
			zap.String("vacancy_id", vacancyID),
			zap.Error(err),
		)
		return false, fmt.Errorf("is vacancy seen: %w", err)
	}

	return count > 0, nil
}

// GetUnseenVacancies returns vacancy ids via difference
func (s *Store) GetUnseenVacancies(ctx context.Context, userID int64, vacancyIDs []string) ([]string, error) {
	if len(vacancyIDs) == 0 {
		return []string{}, nil
	}

	query := `
		SELECT unnest($1::text[]) AS id
		EXCEPT
		SELECT vacancy_id FROM user_seen_vacancies WHERE user_id = $2
	`

	var unseenIDs []string

	rows, err := s.sess.
		SelectBySql(query, pq.Array(vacancyIDs), userID).
		Rows()

	if err != nil {
		s.logger.Error("failed to get unseen vacancies",
			zap.Int64("user_id", userID),
			zap.Int("total_vacancies", len(vacancyIDs)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get unseen vacancies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			s.logger.Error("failed to scan vacancy id",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			return nil, fmt.Errorf("scan vacancy id: %w", err)
		}
		unseenIDs = append(unseenIDs, id)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("failed during rows iteration",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	s.logger.Debug("unseen vacancies",
		zap.Int64("user_id", userID),
		zap.Int("total", len(vacancyIDs)),
		zap.Int("unseen", len(unseenIDs)),
	)

	return unseenIDs, nil
}

func (s *Store) GetUserSeenVacanciesCount(ctx context.Context, userID int64) (int, error) {
	var count int

	err := s.sess.
		Select("COUNT(*)").
		From("user_seen_vacancies").
		Where("user_id = ?", userID).
		LoadOneContext(ctx, &count)

	if err != nil {
		s.logger.Error("failed to get user seen vacancies count",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return 0, fmt.Errorf("get user seen vacancies count: %w", err)
	}

	return count, nil
}

func (s *Store) CleanOldVacanciesCache(ctx context.Context, daysOld int) (int64, error) {
	result, err := s.sess.
		DeleteFrom("vacancies_cache").
		Where("cached_at < NOW() - INTERVAL '? days'", daysOld).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to clean old vacancies cache",
			zap.Int("days_old", daysOld),
			zap.Error(err),
		)
		return 0, fmt.Errorf("clean old vacancies cache: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	s.logger.Info("old vacancies cleaned",
		zap.Int("days_old", daysOld),
		zap.Int64("count", rowsAffected),
	)

	return rowsAffected, nil
}

func (s *Store) CleanOldSeenVacancies(ctx context.Context, daysOld int) (int64, error) {
	result, err := s.sess.
		DeleteFrom("user_seen_vacancies").
		Where("seen_at < NOW() - INTERVAL '? days'", daysOld).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to clean old seen vacancies",
			zap.Int("days_old", daysOld),
			zap.Error(err),
		)
		return 0, fmt.Errorf("clean old seen vacancies: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	s.logger.Info("old seen vacancies cleaned",
		zap.Int("days_old", daysOld),
		zap.Int64("count", rowsAffected),
	)

	return rowsAffected, nil
}

func (s *Store) GetCachedVacanciesByIDs(ctx context.Context, vacancyIDs []string) ([]models.Vacancy, error) {
	if len(vacancyIDs) == 0 {
		return []models.Vacancy{}, nil
	}

	var vacancies []models.Vacancy

	_, err := s.sess.
		Select("*").
		From("vacancies_cache").
		Where("id = ANY(?)", pq.Array(vacancyIDs)).
		LoadContext(ctx, &vacancies)

	if err != nil {
		s.logger.Error("failed to get cached vacancies by IDs",
			zap.Int("count", len(vacancyIDs)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get cached vacancies by IDs: %w", err)
	}

	return vacancies, nil
}