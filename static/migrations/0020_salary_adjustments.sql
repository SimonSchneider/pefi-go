-- migrate:up
ALTER TABLE inkomstbasbelopp ADD COLUMN prisbasbelopp REAL NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS salary_adjustment (
    id                      TEXT    NOT NULL PRIMARY KEY,
    salary_id               TEXT    NOT NULL,
    valid_from              INTEGER NOT NULL,
    vacation_days_per_year  REAL    NOT NULL DEFAULT 0,
    sick_days_per_occasion  REAL    NOT NULL DEFAULT 0,
    sick_occasions_per_year REAL    NOT NULL DEFAULT 0,
    vab_days_per_year       REAL    NOT NULL DEFAULT 0,
    created_at              INTEGER NOT NULL,
    updated_at              INTEGER NOT NULL,
    FOREIGN KEY (salary_id) REFERENCES salary(id) ON DELETE CASCADE
);
