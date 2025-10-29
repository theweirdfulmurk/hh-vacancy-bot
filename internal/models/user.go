package models

import "time"

type User struct {
	ID             int64      `db:"id"`
	Username       *string    `db:"username"`
	FirstName      *string    `db:"first_name"`
	LastName       *string    `db:"last_name"`
	CreatedAt      time.Time  `db:"created_at"`
	LastCheck      *time.Time `db:"last_check"`
	CheckEnabled   bool       `db:"check_enabled"`
	NotifyInterval int        `db:"notify_interval"` // in min
}

type UserFilter struct {
	ID          int64     `db:"id"`
	UserID      int64     `db:"user_id"`
	FilterType  string    `db:"filter_type"`  // text, area, salary, experience, schedule
	FilterValue string    `db:"filter_value"` // JSON or string
	CreatedAt   time.Time `db:"created_at"`
}

const (
	FilterTypeText       = "text"
	FilterTypeArea       = "area"
	FilterTypeSalary     = "salary"
	FilterTypeExperience = "experience"
	FilterTypeSchedule   = "schedule"
)