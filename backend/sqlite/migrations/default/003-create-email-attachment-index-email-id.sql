-- https://www.sqlite.org/foreignkeys.html#fk_indexes
CREATE INDEX IF NOT EXISTS index_email_attachment_email_id ON email_attachment(email_id);
