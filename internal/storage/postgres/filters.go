package postgres

import (
	"context"
	"fmt"

	"hh-vacancy-bot/internal/models"

	"github.com/gocraft/dbr/v2"
	"go.uber.org/zap"
)

func (s *Store) SaveFilter(ctx context.Context, filter *models.UserFilter) error {
	query := `
		INSERT INTO user_filters (user_id, filter_type, filter_value, created_at)
		VALUES (?, ?, ?, NOW())
		ON CONFLICT (user_id, filter_type)
		DO UPDATE SET 
			filter_value = EXCLUDED.filter_value,
			created_at   = NOW()
		RETURNING id
	`

	var id int64
	err := s.sess.
		SelectBySql(query, filter.UserID, filter.FilterType, filter.FilterValue).
		LoadOneContext(ctx, &id)
	if err != nil {
		s.logger.Error("failed to save filter",
			zap.Int64("user_id", filter.UserID),
			zap.String("filter_type", filter.FilterType),
			zap.Error(err),
		)
		return fmt.Errorf("save filter: %w", err)
	}

	filter.ID = id

	s.logger.Info("filter saved",
		zap.Int64("user_id", filter.UserID),
		zap.String("filter_type", filter.FilterType),
		zap.String("filter_value", filter.FilterValue),
	)

	return nil
}

func (s *Store) GetUserFilters(ctx context.Context, userID int64) ([]models.UserFilter, error) {
	var filters []models.UserFilter

	_, err := s.sess.
		Select("*").
		From("user_filters").
		Where("user_id = ?", userID).
		OrderBy("filter_type").
		LoadContext(ctx, &filters)

	if err != nil {
		s.logger.Error("failed to get user filters",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get user filters: %w", err)
	}

	return filters, nil
}

func (s *Store) GetFilter(ctx context.Context, userID int64, filterType string) (*models.UserFilter, error) {
	var filter models.UserFilter

	err := s.sess.
		Select("*").
		From("user_filters").
		Where("user_id = ? AND filter_type = ?", userID, filterType).
		LoadOneContext(ctx, &filter)

	if err == dbr.ErrNotFound {
		return nil, nil
	}

	if err != nil {
		s.logger.Error("failed to get filter",
			zap.Int64("user_id", userID),
			zap.String("filter_type", filterType),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get filter: %w", err)
	}

	return &filter, nil
}

func (s *Store) DeleteFilter(ctx context.Context, userID int64, filterType string) error {
	result, err := s.sess.
		DeleteFrom("user_filters").
		Where("user_id = ? AND filter_type = ?", userID, filterType).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to delete filter",
			zap.Int64("user_id", userID),
			zap.String("filter_type", filterType),
			zap.Error(err),
		)
		return fmt.Errorf("delete filter: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("filter not found")
	}

	s.logger.Info("filter deleted",
		zap.Int64("user_id", userID),
		zap.String("filter_type", filterType),
	)

	return nil
}

func (s *Store) ClearUserFilters(ctx context.Context, userID int64) error {
	result, err := s.sess.
		DeleteFrom("user_filters").
		Where("user_id = ?", userID).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to clear user filters",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return fmt.Errorf("clear user filters: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	s.logger.Info("user filters cleared",
		zap.Int64("user_id", userID),
		zap.Int64("count", rowsAffected),
	)

	return nil
}

func (s *Store) HasFilters(ctx context.Context, userID int64) (bool, error) {
	var count int

	err := s.sess.
		Select("COUNT(*)").
		From("user_filters").
		Where("user_id = ?", userID).
		LoadOneContext(ctx, &count)

	if err != nil {
		s.logger.Error("failed to check filters existence",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return false, fmt.Errorf("has filters: %w", err)
	}

	return count > 0, nil
}

func (s *Store) GetFiltersMap(ctx context.Context, userID int64) (map[string]string, error) {
	filters, err := s.GetUserFilters(ctx, userID)
	if err != nil {
		return nil, err
	}

	filtersMap := make(map[string]string)
	for _, filter := range filters {
		filtersMap[filter.FilterType] = filter.FilterValue
	}

	return filtersMap, nil
}

func (s *Store) CountUserFilters(ctx context.Context, userID int64) (int, error) {
	var count int

	err := s.sess.
		Select("COUNT(*)").
		From("user_filters").
		Where("user_id = ?", userID).
		LoadOneContext(ctx, &count)

	if err != nil {
		s.logger.Error("failed to count user filters",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return 0, fmt.Errorf("count user filters: %w", err)
	}

	return count, nil
}
