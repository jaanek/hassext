CREATE TABLE IF NOT EXISTS heating_curve (
    id INTEGER PRIMARY KEY,
    external_temperature INTEGER NOT NULL,
    target_temperature INTEGER NOT NULL
);
