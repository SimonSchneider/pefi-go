-- migrate:up
ALTER TABLE inkomstbasbelopp RENAME TO swe_yearly_params;
ALTER TABLE swe_yearly_params ADD COLUMN schablon_ranta REAL NOT NULL DEFAULT 0.0125;
ALTER TABLE swe_yearly_params ADD COLUMN isk_fribelopp REAL NOT NULL DEFAULT 0;
ALTER TABLE account ADD COLUMN is_isk INTEGER NOT NULL DEFAULT 0;
