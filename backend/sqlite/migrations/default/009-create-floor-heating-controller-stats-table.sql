CREATE TABLE IF NOT EXISTS floor_heating_controller_stats (
    id INTEGER PRIMARY KEY,
    controller_id TEXT NOT NULL,
    unix_days INTEGER NOT NULL,
    on_count INTEGER NOT NULL,
    UNIQUE(controller_id, unix_days)
);
