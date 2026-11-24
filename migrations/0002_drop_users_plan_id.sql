-- +goose Up
-- payments-api is the only source of truth for a user's plan and subscription (audit point 2 of the study).
-- users.plan_id was never written; ask payments-api (GET /subscriptions/me) instead.
ALTER TABLE users DROP COLUMN plan_id;

-- +goose Down
ALTER TABLE users ADD COLUMN plan_id uuid;
