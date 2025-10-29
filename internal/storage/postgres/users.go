package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"hh-vacancy-bot/internal/models"

	"github.com/gocraft/dbr/v2"
	"go.uber.org/zap"
)

func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	_, err := s.sess.
		InsertInto("users").
		Columns("id", "username", "first_name", "last_name", "created_at", "check_enabled", "notify_interval").
		Values(user.ID, user.Username, user.FirstName, user.LastName, time.Now(), user.CheckEnabled, user.NotifyInterval).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to create user",
			zap.Int64("user_id", user.ID),
			zap.Error(err),
		)
		return fmt.Errorf("create user: %w", err)
	}

	s.logger.Info("user created",
		zap.Int64("user_id", user.ID),
		zap.Stringp("username", user.Username),
	)

	return nil
}

func (s *Store) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	var user models.User

	err := s.sess.
		Select("*").
		From("users").
		Where("id = ?", userID).
		LoadOneContext(ctx, &user)

	if err == dbr.ErrNotFound {
		return nil, nil
	}

	if err != nil {
		s.logger.Error("failed to get user",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &user, nil
}

func (s *Store) GetOrCreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	existing, err := s.GetUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return existing, nil
	}

	if err := s.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Store) UpdateUser(ctx context.Context, user *models.User) error {
	_, err := s.sess.
		Update("users").
		Set("username", user.Username).
		Set("first_name", user.FirstName).
		Set("last_name", user.LastName).
		Set("check_enabled", user.CheckEnabled).
		Set("notify_interval", user.NotifyInterval).
		Where("id = ?", user.ID).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to update user",
			zap.Int64("user_id", user.ID),
			zap.Error(err),
		)
		return fmt.Errorf("update user: %w", err)
	}

	s.logger.Info("user updated", zap.Int64("user_id", user.ID))
	return nil
}

func (s *Store) UpdateLastCheck(ctx context.Context, userID int64) error {
	now := time.Now()

	_, err := s.sess.
		Update("users").
		Set("last_check", now).
		Where("id = ?", userID).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to update last check",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return fmt.Errorf("update last check: %w", err)
	}

	return nil
}

func (s *Store) SetCheckEnabled(ctx context.Context, userID int64, enabled bool) error {
	_, err := s.sess.
		Update("users").
		Set("check_enabled", enabled).
		Where("id = ?", userID).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to set check enabled",
			zap.Int64("user_id", userID),
			zap.Bool("enabled", enabled),
			zap.Error(err),
		)
		return fmt.Errorf("set check enabled: %w", err)
	}

	s.logger.Info("check enabled updated",
		zap.Int64("user_id", userID),
		zap.Bool("enabled", enabled),
	)

	return nil
}

func (s *Store) SetNotifyInterval(ctx context.Context, userID int64, intervalMinutes int) error {
	_, err := s.sess.
		Update("users").
		Set("notify_interval", intervalMinutes).
		Where("id = ?", userID).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to set notify interval",
			zap.Int64("user_id", userID),
			zap.Int("interval", intervalMinutes),
			zap.Error(err),
		)
		return fmt.Errorf("set notify interval: %w", err)
	}

	s.logger.Info("notify interval updated",
		zap.Int64("user_id", userID),
		zap.Int("interval", intervalMinutes),
	)

	return nil
}

func (s *Store) GetActiveUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User

	_, err := s.sess.
		Select("*").
		From("users").
		Where("check_enabled = ?", true).
		LoadContext(ctx, &users)

	if err != nil {
		s.logger.Error("failed to get active users", zap.Error(err))
		return nil, fmt.Errorf("get active users: %w", err)
	}

	return users, nil
}

func (s *Store) GetUsersToCheck(ctx context.Context) ([]models.User, error) {
	var users []models.User

	query := `
		SELECT * FROM users
		WHERE check_enabled = true
		AND (
			last_check IS NULL
			OR NOW() - last_check >= (notify_interval || ' minutes')::interval
		)
	`

	_, err := s.sess.
		SelectBySql(query).
		LoadContext(ctx, &users)

	if err != nil {
		s.logger.Error("failed to get users to check", zap.Error(err))
		return nil, fmt.Errorf("get users to check: %w", err)
	}

	s.logger.Debug("users to check",
		zap.Int("count", len(users)),
	)

	return users, nil
}

func (s *Store) DeleteUser(ctx context.Context, userID int64) error {
	_, err := s.sess.
		DeleteFrom("users").
		Where("id = ?", userID).
		ExecContext(ctx)

	if err != nil {
		s.logger.Error("failed to delete user",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return fmt.Errorf("delete user: %w", err)
	}

	s.logger.Info("user deleted", zap.Int64("user_id", userID))
	return nil
}

func (s *Store) GetUserStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var filterCount int
	err := s.sess.
		Select("COUNT(*)").
		From("user_filters").
		Where("user_id = ?", userID).
		LoadOneContext(ctx, &filterCount)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get filter count: %w", err)
	}

	stats["filter_count"] = filterCount

	var seenCount int
	err = s.sess.
		Select("COUNT(*)").
		From("user_seen_vacancies").
		Where("user_id = ?", userID).
		LoadOneContext(ctx, &seenCount)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("get seen count: %w", err)
	}

	stats["seen_vacancies_count"] = seenCount

	return stats, nil
}