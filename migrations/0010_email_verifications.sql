-- +goose Up
-- A password signup proves it owns its email with a 6-digit code sent there; until then it cannot sign in. Only the
-- hash of the code is kept, like password_resets. Admins are created by the platform and count as verified.
CREATE TABLE email_verifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  code_hash text NOT NULL,
  attempts integer NOT NULL DEFAULT 0,
  expires_at timestamptz NOT NULL,
  used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX email_verifications_user_id_idx ON email_verifications (user_id, created_at DESC);

UPDATE users SET email_verified = true WHERE role = 'admin';

-- +goose Down
DROP TABLE email_verifications;
