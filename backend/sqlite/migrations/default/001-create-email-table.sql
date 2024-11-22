CREATE TABLE IF NOT EXISTS email (
    id INTEGER PRIMARY KEY,
    email_type TEXT,
    email_from TEXT,
    email_to TEXT,
    subject TEXT,
    body_html TEXT,
    created TIMESTAMP,
    sent TIMESTAMP,
    retries INTEGER,
    last_error TEXT
);
