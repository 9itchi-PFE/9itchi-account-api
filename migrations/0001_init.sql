-- +goose Up
-- Draft schema generated from models/db. Refine (constraints, indexes, FKs) as features land;
-- nothing is deployed yet, so editing this file is fine until the first release.
CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL DEFAULT '',
  email text NOT NULL DEFAULT '',
  password_hash text NOT NULL DEFAULT '',
  role text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT '',
  plan_id uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
