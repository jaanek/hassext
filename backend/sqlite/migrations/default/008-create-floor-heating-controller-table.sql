CREATE TABLE IF NOT EXISTS floor_heating_controller (
    id INTEGER PRIMARY KEY,
    cname TEXT NOT NULL,
    last_state_change TIMESTAMP,
    UNIQUE(cname)
);
