-- migrate:up
CREATE TABLE IF NOT EXISTS partial_parental_leave (
    id                          TEXT    NOT NULL PRIMARY KEY,
    salary_id                   TEXT    NOT NULL,
    start_date                  INTEGER NOT NULL,
    end_date                    INTEGER NOT NULL,
    sjuk_days_per_year          REAL    NOT NULL DEFAULT 0,
    lagsta_days_per_year        REAL    NOT NULL DEFAULT 0,
    skipped_work_days_per_year  REAL    NOT NULL DEFAULT 0,
    created_at                  INTEGER NOT NULL,
    updated_at                  INTEGER NOT NULL,
    FOREIGN KEY (salary_id) REFERENCES salary(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS full_parental_leave (
    id                TEXT    NOT NULL PRIMARY KEY,
    salary_id         TEXT    NOT NULL,
    start_date        INTEGER NOT NULL,
    end_date          INTEGER NOT NULL,
    sjuk_days_per_week REAL   NOT NULL DEFAULT 0,
    created_at        INTEGER NOT NULL,
    updated_at        INTEGER NOT NULL,
    FOREIGN KEY (salary_id) REFERENCES salary(id) ON DELETE CASCADE
);
