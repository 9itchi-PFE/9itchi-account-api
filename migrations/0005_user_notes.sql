-- +goose Up
-- What the team writes about a student for itself. Never returned to the student; deleted with the account.
CREATE TABLE user_notes (
  user_id uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  note text NOT NULL CHECK (char_length(note) <= 2000),
  updated_by uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_notes;
