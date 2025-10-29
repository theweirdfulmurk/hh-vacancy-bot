package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Telegram
	TelegramToken string

	// Database
	PostgresDSN   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// HeadHunter API
	HHAPIBaseURL string
	HHAPITimeout time.Duration

	// Bot settings
	CheckInterval        time.Duration
	MaxVacanciesPerCheck int

	// Logging
	LogLevel string
}

func Load() (*Config, error) {
	cfg := &Config{
		// Defaults
		HHAPIBaseURL:         "https://api.hh.ru",
		HHAPITimeout:         30 * time.Second,
		CheckInterval:        5 * time.Minute,
		MaxVacanciesPerCheck: 10,
		LogLevel:             "info",
		RedisDB:              0,
	}

	cfg.TelegramToken = os.Getenv("TELEGRAM_TOKEN")
	if cfg.TelegramToken == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN is required")
	}

	cfg.PostgresDSN = os.Getenv("POSTGRES_DSN")
	if cfg.PostgresDSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}

	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		cfg.RedisAddr = addr
	} else {
		cfg.RedisAddr = "localhost:6379"
	}

	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")

	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		db, err := strconv.Atoi(redisDB)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
		}
		cfg.RedisDB = db
	}

	if baseURL := os.Getenv("HHAPI_BASE_URL"); baseURL != "" {
		cfg.HHAPIBaseURL = baseURL
	}

	if timeout := os.Getenv("HHAPI_TIMEOUT"); timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid HHAPI_TIMEOUT: %w", err)
		}
		cfg.HHAPITimeout = d
	}

	if interval := os.Getenv("CHECK_INTERVAL"); interval != "" {
		d, err := time.ParseDuration(interval)
		if err != nil {
			return nil, fmt.Errorf("invalid CHECK_INTERVAL: %w", err)
		}
		cfg.CheckInterval = d
	}

	if maxVacancies := os.Getenv("MAX_VACANCIES_PER_CHECK"); maxVacancies != "" {
		n, err := strconv.Atoi(maxVacancies)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_VACANCIES_PER_CHECK: %w", err)
		}
		cfg.MaxVacanciesPerCheck = n
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.TelegramToken == "" {
		return fmt.Errorf("telegram token is empty")
	}

	if c.PostgresDSN == "" {
		return fmt.Errorf("postgres DSN is empty")
	}

	if c.CheckInterval < time.Minute {
		return fmt.Errorf("check interval too small: %v", c.CheckInterval)
	}

	if c.MaxVacanciesPerCheck < 1 || c.MaxVacanciesPerCheck > 100 {
		return fmt.Errorf("max vacancies per check must be between 1 and 100")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	return nil
}