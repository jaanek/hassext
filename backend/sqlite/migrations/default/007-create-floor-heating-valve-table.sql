CREATE TABLE IF NOT EXISTS floor_heating_valve (
    id INTEGER PRIMARY KEY,
    vname TEXT NOT NULL,
    vstate BOOLEAN NOT NULL,
    last_state_change TIMESTAMP NOT NULL,
    UNIQUE(vname)
);
