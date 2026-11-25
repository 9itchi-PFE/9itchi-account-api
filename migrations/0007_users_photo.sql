-- +goose Up
-- The student's profile photo: one square JPEG, already reduced by the service (about 400 x 400 px). Never read in
-- list queries; the user view only says whether there is one.
ALTER TABLE users
  ADD COLUMN photo bytea CHECK (photo IS NULL OR length(photo) <= 1048576),
  ADD COLUMN photo_updated_at timestamptz;

-- +goose Down
ALTER TABLE users DROP COLUMN photo, DROP COLUMN photo_updated_at;
