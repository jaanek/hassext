CREATE TABLE IF NOT EXISTS email_attachment (
    email_id INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_type TEXT NOT NULL,
    FOREIGN KEY(email_id) REFERENCES email(id) ON UPDATE CASCADE ON DELETE CASCADE
);
