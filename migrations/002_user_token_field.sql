-- Write your migrate up statements here

ALTER TABLE users ADD COLUMN token TEXT NOT NULL UNIQUE DEFAULT '';

---- create above / drop below ----

ALTER TABLE users DROP COLUMN token;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
