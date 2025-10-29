package headhunter

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"go.uber.org/zap"
)

type VacancySearchParams struct {
	Text       string 
	Area       string // area id
	Experience string 
	Salary     int    
	Schedule   string 
	Page       int    
	PerPage    int   
}

func (c *Client) SearchVacancies(ctx context.Context, params VacancySearchParams) (*VacancySearchResponse, error) {
	queryParams := url.Values{}

	if params.Text != "" {
		queryParams.Set("text", params.Text)
	}

	if params.Area != "" {
		queryParams.Set("area", params.Area)
	}

	if params.Experience != "" {
		queryParams.Set("experience", params.Experience)
	}

	if params.Salary > 0 {
		queryParams.Set("salary", strconv.Itoa(params.Salary))
		queryParams.Set("only_with_salary", "true")
	}

	if params.Schedule != "" {
		queryParams.Set("schedule", params.Schedule)
	}

	// pagination
	if params.Page > 0 {
		queryParams.Set("page", strconv.Itoa(params.Page))
	}

	if params.PerPage > 0 {
		queryParams.Set("per_page", strconv.Itoa(params.PerPage))
	} else {
		queryParams.Set("per_page", "20") 
	}

	data, err := c.get(ctx, "/vacancies", queryParams)
	if err != nil {
		c.logger.Error("failed to search vacancies",
			zap.String("text", params.Text),
			zap.String("area", params.Area),
			zap.Error(err),
		)
		return nil, fmt.Errorf("search vacancies: %w", err)
	}

	var response VacancySearchResponse
	if err := c.parseResponse(data, &response); err != nil {
		c.logger.Error("failed to parse search response", zap.Error(err))
		return nil, err
	}

	c.logger.Debug("vacancies found",
		zap.Int("found", response.Found),
		zap.Int("returned", len(response.Items)),
		zap.String("text", params.Text),
		zap.String("area", params.Area),
	)

	return &response, nil
}

func (c *Client) GetVacancy(ctx context.Context, vacancyID string) (*VacancyDetail, error) {
	path := fmt.Sprintf("/vacancies/%s", vacancyID)

	data, err := c.get(ctx, path, nil)
	if err != nil {
		c.logger.Error("failed to get vacancy",
			zap.String("vacancy_id", vacancyID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get vacancy: %w", err)
	}

	var vacancy VacancyDetail
	if err := c.parseResponse(data, &vacancy); err != nil {
		c.logger.Error("failed to parse vacancy detail", zap.Error(err))
		return nil, err
	}

	c.logger.Debug("vacancy retrieved",
		zap.String("vacancy_id", vacancyID),
		zap.String("name", vacancy.Name),
	)

	return &vacancy, nil
}

func (c *Client) SearchVacanciesSimilar(ctx context.Context, vacancyID string, page, perPage int) (*VacancySearchResponse, error) {
	path := fmt.Sprintf("/vacancies/%s/similar_vacancies", vacancyID)

	queryParams := url.Values{}
	if page > 0 {
		queryParams.Set("page", strconv.Itoa(page))
	}
	if perPage > 0 {
		queryParams.Set("per_page", strconv.Itoa(perPage))
	}

	data, err := c.get(ctx, path, queryParams)
	if err != nil {
		c.logger.Error("failed to search similar vacancies",
			zap.String("vacancy_id", vacancyID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("search similar vacancies: %w", err)
	}

	var response VacancySearchResponse
	if err := c.parseResponse(data, &response); err != nil {
		c.logger.Error("failed to parse similar vacancies response", zap.Error(err))
		return nil, err
	}

	c.logger.Debug("similar vacancies found",
		zap.String("vacancy_id", vacancyID),
		zap.Int("found", response.Found),
		zap.Int("returned", len(response.Items)),
	)

	return &response, nil
}

func ExtractVacancyIDs(response *VacancySearchResponse) []string {
	ids := make([]string, len(response.Items))
	for i, item := range response.Items {
		ids[i] = item.ID
	}
	return ids
}