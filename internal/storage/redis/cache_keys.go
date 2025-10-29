package redis

import (
	"context"
	"fmt"
	"time"
)

const (
	CitiesCacheTTL         = 24 * time.Hour 
	VacancySearchCacheTTL  = 5 * time.Minute 
	RateLimitWindowTTL     = 1 * time.Minute  
	UserStateCacheTTL      = 30 * time.Minute 
)


func CitiesKey() string {
	return "cities:russia"
}

func VacancySearchKey(userID int64) string {
	return fmt.Sprintf("search:user:%d", userID)
}

func RateLimitKey(userID int64) string {
	return fmt.Sprintf("ratelimit:user:%d", userID)
}

func HHAPIRateLimitKey() string {
	return "ratelimit:hhapi"
}

func UserStateKey(userID int64) string {
	return fmt.Sprintf("state:user:%d", userID)
}

func (c *Cache) GetCities(ctx context.Context) (interface{}, error) {
	var cities interface{}
	err := c.Get(ctx, CitiesKey(), &cities)
	if err != nil {
		return nil, err
	}
	return cities, nil
}

func (c *Cache) SetCities(ctx context.Context, cities interface{}) error {
	return c.Set(ctx, CitiesKey(), cities, CitiesCacheTTL)
}

func (c *Cache) GetVacancySearchResults(ctx context.Context, userID int64) (interface{}, error) {
	var results interface{}
	err := c.Get(ctx, VacancySearchKey(userID), &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (c *Cache) SetVacancySearchResults(ctx context.Context, userID int64, results interface{}) error {
	return c.Set(ctx, VacancySearchKey(userID), results, VacancySearchCacheTTL)
}

func (c *Cache) InvalidateVacancySearchCache(ctx context.Context, userID int64) error {
	return c.Delete(ctx, VacancySearchKey(userID))
}

func (c *Cache) IncrementUserRateLimit(ctx context.Context, userID int64) (int64, error) {
	key := RateLimitKey(userID)
	return c.IncrementWithExpiry(ctx, key, RateLimitWindowTTL)
}

func (c *Cache) GetUserRateLimit(ctx context.Context, userID int64) (int64, error) {
	key := RateLimitKey(userID)
	return c.GetInt(ctx, key)
}

func (c *Cache) IncrementHHAPIRateLimit(ctx context.Context) (int64, error) {
	key := HHAPIRateLimitKey()
	return c.IncrementWithExpiry(ctx, key, RateLimitWindowTTL)
}

func (c *Cache) GetHHAPIRateLimit(ctx context.Context) (int64, error) {
	key := HHAPIRateLimitKey()
	return c.GetInt(ctx, key)
}

func (c *Cache) SetUserState(ctx context.Context, userID int64, state string) error {
	key := UserStateKey(userID)
	return c.SetString(ctx, key, state, UserStateCacheTTL)
}

func (c *Cache) GetUserState(ctx context.Context, userID int64) (string, error) {
	key := UserStateKey(userID)
	return c.GetString(ctx, key)
}

func (c *Cache) DeleteUserState(ctx context.Context, userID int64) error {
	key := UserStateKey(userID)
	return c.Delete(ctx, key)
}

func (c *Cache) SetTempData(ctx context.Context, userID int64, key string, value interface{}, ttl time.Duration) error {
	fullKey := fmt.Sprintf("temp:user:%d:%s", userID, key)
	return c.Set(ctx, fullKey, value, ttl)
}

func (c *Cache) GetTempData(ctx context.Context, userID int64, key string, dest interface{}) error {
	fullKey := fmt.Sprintf("temp:user:%d:%s", userID, key)
	return c.Get(ctx, fullKey, dest)
}

func (c *Cache) DeleteTempData(ctx context.Context, userID int64, key string) error {
	fullKey := fmt.Sprintf("temp:user:%d:%s", userID, key)
	return c.Delete(ctx, fullKey)
}