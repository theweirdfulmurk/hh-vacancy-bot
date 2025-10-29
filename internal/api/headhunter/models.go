package headhunter

import "time"

type VacancySearchResponse struct {
	Items      []VacancyItem `json:"items"`
	Found      int           `json:"found"`
	Pages      int           `json:"pages"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
	Clusters   interface{}   `json:"clusters,omitempty"`
	Arguments  interface{}   `json:"arguments,omitempty"`
	AlternateURL string      `json:"alternate_url,omitempty"`
}

type VacancyItem struct {
	ID              string         `json:"id"`
	Premium         bool           `json:"premium"`
	Name            string         `json:"name"`
	Department      *Department    `json:"department"`
	HasTest         bool           `json:"has_test"`
	ResponseLetterRequired bool    `json:"response_letter_required"`
	Area            Area           `json:"area"`
	Salary          *Salary        `json:"salary"`
	Type            IDName         `json:"type"`
	Address         *Address       `json:"address"`
	ResponseURL     *string        `json:"response_url"`
	SortPointDistance *float64     `json:"sort_point_distance"`
	PublishedAt     time.Time      `json:"published_at"`
	CreatedAt       time.Time      `json:"created_at"`
	Archived        bool           `json:"archived"`
	ApplyAlternateURL string       `json:"apply_alternate_url"`
	ShowLogoInSearch *bool         `json:"show_logo_in_search"`
	InsiderInterview *InsiderInterview `json:"insider_interview"`
	URL             string         `json:"url"`
	AlternateURL    string         `json:"alternate_url"`
	Relations       []interface{}  `json:"relations"`
	Employer        Employer       `json:"employer"`
	Snippet         *Snippet       `json:"snippet"`
	Contacts        *Contacts      `json:"contacts"`
	Schedule        *IDName        `json:"schedule"`
	WorkingDays     []IDName       `json:"working_days"`
	WorkingTimeIntervals []IDName  `json:"working_time_intervals"`
	WorkingTimeModes []IDName      `json:"working_time_modes"`
	AcceptTemporary *bool          `json:"accept_temporary"`
	ProfessionalRoles []IDName     `json:"professional_roles"`
	AcceptIncompleteResumes bool   `json:"accept_incomplete_resumes"`
	Experience      *IDName        `json:"experience"`
	Employment      *IDName        `json:"employment"`
}

type Department struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Area struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Salary struct {
	From     *int   `json:"from"`
	To       *int   `json:"to"`
	Currency string `json:"currency"`
	Gross    bool   `json:"gross"`
}

type IDName struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Address struct {
	City       *string      `json:"city"`
	Street     *string      `json:"street"`
	Building   *string      `json:"building"`
	Description *string     `json:"description"`
	Lat        *float64     `json:"lat"`
	Lng        *float64     `json:"lng"`
	Raw        *string      `json:"raw"`
	Metro      *Metro       `json:"metro"`
	MetroStations []Metro   `json:"metro_stations"`
	ID         string       `json:"id"`
}

type Metro struct {
	StationName string  `json:"station_name"`
	LineName    string  `json:"line_name"`
	StationID   string  `json:"station_id"`
	LineID      string  `json:"line_id"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

type InsiderInterview struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Employer struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	URL          string      `json:"url"`
	AlternateURL string      `json:"alternate_url"`
	LogoURLs     *LogoURLs   `json:"logo_urls"`
	VacanciesURL string      `json:"vacancies_url"`
	Trusted      bool        `json:"trusted"`
	AccreditedITEmployer bool `json:"accredited_it_employer"`
}

type LogoURLs struct {
	Original string `json:"original"`
	Size90   string `json:"90"`
	Size240  string `json:"240"`
}

type Snippet struct {
	Requirement    *string `json:"requirement"`
	Responsibility *string `json:"responsibility"`
}

type Contacts struct {
	Name   string  `json:"name"`
	Email  *string `json:"email"`
	Phones []Phone `json:"phones"`
}

type Phone struct {
	Country string  `json:"country"`
	City    string  `json:"city"`
	Number  string  `json:"number"`
	Comment *string `json:"comment"`
}

type VacancyDetail struct {
	VacancyItem
	Description          string            `json:"description"`
	KeySkills           []IDName          `json:"key_skills"`
	AcceptHandicapped   bool              `json:"accept_handicapped"`
	AcceptKids          bool              `json:"accept_kids"`
	Test                *Test             `json:"test"`
	EmployerLogo        *EmployerLogo     `json:"employer_logo"`
	Languages           []Language        `json:"languages"`
	ProfessionalRoles   []IDName          `json:"professional_roles"`
	Code                *string           `json:"code"`
	Hidden              bool              `json:"hidden"`
	QuickResponsesAllowed bool            `json:"quick_responses_allowed"`
	BillingType         *IDName           `json:"billing_type"`
	AllowMessages       bool              `json:"allow_messages"`
	InitialCreatedAt    time.Time         `json:"initial_created_at"`
	Negotiations        interface{}       `json:"negotiations"`
	HasVacancyDescription bool            `json:"has_vacancy_description"`
}

type Test struct {
	Required bool `json:"required"`
}

type EmployerLogo struct {
	Original string `json:"original"`
	Size90   string `json:"90"`
	Size240  string `json:"240"`
}

type Language struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level IDName `json:"level"`
}

type AreaResponse struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Areas  []AreaResponse `json:"areas"`
	ParentID *string      `json:"parent_id"`
}

type City struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Region string `json:"region"`
	Path   string `json:"-"`
}

type ErrorResponse struct {
	Errors []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"errors"`
	Description string `json:"description"`
}

