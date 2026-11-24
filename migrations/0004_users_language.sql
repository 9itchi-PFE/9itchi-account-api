-- +goose Up
-- The language 9itchi writes to the user in (emails, notifications): the one of the interface they use.
ALTER TABLE users ADD COLUMN language text NOT NULL DEFAULT 'fr' CHECK (language IN ('fr', 'en', 'ar'));

-- +goose Down
ALTER TABLE users DROP COLUMN language;
