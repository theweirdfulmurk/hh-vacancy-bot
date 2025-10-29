CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_check TIMESTAMP,
    check_enabled BOOLEAN NOT NULL DEFAULT true,
    notify_interval INT NOT NULL DEFAULT 60
);

CREATE TABLE IF NOT EXISTS user_filters (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filter_type VARCHAR(50) NOT NULL,
    filter_value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, filter_type)
);

CREATE TABLE IF NOT EXISTS vacancies_cache (
    id VARCHAR(50) PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    company VARCHAR(500),
    salary_from INT,
    salary_to INT,
    currency VARCHAR(10),
    area VARCHAR(255),
    area_id VARCHAR(50),
    url TEXT NOT NULL,
    published_at TIMESTAMP NOT NULL,
    experience VARCHAR(100),
    schedule VARCHAR(100),
    employment VARCHAR(100),
    raw_data JSONB,
    cached_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_seen_vacancies (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vacancy_id VARCHAR(50) NOT NULL REFERENCES vacancies_cache(id),
    seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, vacancy_id)
);