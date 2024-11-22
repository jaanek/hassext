CREATE TABLE IF NOT EXISTS thermostat (
    id INTEGER PRIMARY KEY,
    tname TEXT NOT NULL,
    target_temperature REAL NOT NULL,
    current_temperature REAL NOT NULL,
    last_update TIMESTAMP NOT NULL,
    UNIQUE(tname)
);
