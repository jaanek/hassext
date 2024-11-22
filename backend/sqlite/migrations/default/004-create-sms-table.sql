CREATE TABLE IF NOT EXISTS sms (
    id INTEGER PRIMARY KEY,
    uuid TEXT NOT NULL,
    created TIMESTAMP NOT NULL,
    cto TEXT NOT NULL,
    message TEXT NOT NULL,
    finished TIMESTAMP,
    retries INTEGER,
    cresult TEXT,
    pa_triggered_id INTEGER,
    FOREIGN KEY(pa_triggered_id) REFERENCES priceAlertTriggered(id) ON UPDATE CASCADE ON DELETE CASCADE
);
