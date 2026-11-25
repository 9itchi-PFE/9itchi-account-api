-- +goose Up
-- The rest of the student's profile (screen « Mon profil »): all optional, all changed by the student.
-- year is a short code (l3, m1, m2, ing1, ing2, ing3, other); the client shows the full wording.
ALTER TABLE users
  ADD COLUMN phone text CHECK (phone IS NULL OR length(phone) BETWEEN 8 AND 20),
  ADD COLUMN city text CHECK (city IS NULL OR length(city) BETWEEN 1 AND 80),
  ADD COLUMN speciality text CHECK (speciality IS NULL OR length(speciality) BETWEEN 1 AND 120),
  ADD COLUMN year text CHECK (year IS NULL OR year IN ('l3', 'm1', 'm2', 'ing1', 'ing2', 'ing3', 'other')),
  ADD COLUMN linkedin text CHECK (linkedin IS NULL OR length(linkedin) BETWEEN 1 AND 200),
  ADD COLUMN github text CHECK (github IS NULL OR length(github) BETWEEN 1 AND 200);

-- +goose Down
ALTER TABLE users
  DROP COLUMN phone, DROP COLUMN city, DROP COLUMN speciality, DROP COLUMN year, DROP COLUMN linkedin, DROP COLUMN github;
