-- Write your migrate up statements here

ALTER TABLE projects ADD COLUMN fts_column tsvector GENERATED ALWAYS AS (to_tsvector('english', title || ' ' || slug || ' ' || description)) STORED;

CREATE INDEX projects_fts_search_idx ON projects USING GIN (fts_column);

---- create above / drop below ----

DROP INDEX projects_fts_search_idx ON projects;

ALTER TABLE projects DROP COLUMN fts_column;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
