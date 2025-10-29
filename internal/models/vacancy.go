package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Vacancy struct {
	ID          string    `db:"id"`
	Title       string    `db:"title"`
	Company     *string   `db:"company"`
	SalaryFrom  *int      `db:"salary_from"`
	SalaryTo    *int      `db:"salary_to"`
	Currency    *string   `db:"currency"`
	Area        string    `db:"area"`
	AreaID      string    `db:"area_id"`
	URL         string    `db:"url"`
	PublishedAt time.Time `db:"published_at"`
	Experience  *string   `db:"experience"`
	Schedule    *string   `db:"schedule"`
	Employment  *string   `db:"employment"`
	RawData     RawJSON   `db:"raw_data"`
	CachedAt    time.Time `db:"cached_at"`
}

type UserSeenVacancy struct {
	UserID    int64     `db:"user_id"`
	VacancyID string    `db:"vacancy_id"`
	SeenAt    time.Time `db:"seen_at"`
}

type RawJSON json.RawMessage

func (r RawJSON) Value() (driver.Value, error) {
	if r == nil {
		return nil, nil
	}
	return json.RawMessage(r).MarshalJSON()
}

func (r *RawJSON) Scan(value interface{}) error {
	if value == nil {
		*r = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	*r = RawJSON(bytes)
	return nil
}