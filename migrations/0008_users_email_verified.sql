-- +goose Up
-- Whether the user proved they own their email. Only a Google or LinkedIn sign-in that vouches for the address sets
-- it (password signups are not checked). The access token carries it: payments-api gives the student discount only
-- to a verified email on a student domain.
ALTER TABLE users ADD COLUMN email_verified boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN email_verified;
