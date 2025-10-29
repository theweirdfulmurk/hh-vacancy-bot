package headhunter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// Client for requests to HeadHunter API
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	userAgent  string
}

func New(baseURL string, timeout time.Duration, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:    logger,
		userAgent: "HH-Vacancy-Bot/1.0",
	}
}

// doRequest for HTTP reqs with retries
func (c *Client) doRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	fullURL := c.baseURL + path
	if params != nil {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			c.logger.Debug("retrying request",
				zap.String("url", fullURL),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("read response body: %w", err)
			continue
		}

		// check response code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			c.logger.Debug("successful request",
				zap.String("url", fullURL),
				zap.Int("status", resp.StatusCode),
			)
			return body, nil
		}

		// log errors
		c.logger.Error("API error",
			zap.String("url", fullURL),
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)

		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			c.logger.Warn("rate limit hit, backing off")
			time.Sleep(5 * time.Second)
			lastErr = fmt.Errorf("rate limit exceeded")
			continue
		case http.StatusBadRequest:
			// Плохой запрос - не retry
			var apiErr ErrorResponse
			if err := json.Unmarshal(body, &apiErr); err == nil {
				return nil, fmt.Errorf("bad request: %s", apiErr.Description)
			}
			return nil, fmt.Errorf("bad request: %s", string(body))
		case http.StatusNotFound:
			return nil, fmt.Errorf("not found")
		case http.StatusForbidden:
			return nil, fmt.Errorf("forbidden")
		default:
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}

	return nil, fmt.Errorf("request failed after retries: %w", lastErr)
}

// get выполняет GET запрос
func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, params)
}

// parseResponse парсит JSON ответ
func (c *Client) parseResponse(data []byte, dest interface{}) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}