-- Write your migrate up statements here

ALTER TABLE users ADD COLUMN password VARCHAR(255) NOT NULL;

---- create above / drop below ----

ALTER TABLE users DROP COLUMN password;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
