-- +goose Up
-- The school chosen at sign-up (screen 1.1), a short code such as ENIS; the client shows the full name.
ALTER TABLE users ADD COLUMN school text CHECK (school IS NULL OR length(school) BETWEEN 1 AND 120);

-- +goose Down
ALTER TABLE users DROP COLUMN school;
