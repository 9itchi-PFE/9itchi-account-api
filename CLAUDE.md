# 9itchi Account API
Auth and identity hub: signup/login, JWT access (15 min) + refresh (7 d, hashed, rotated), password reset (Mailgun), Google and LinkedIn sign-in, profile, photo, language, admin users and notes, account erasure across services. Port 8081, DB `users`.

## Conventions (shared by every 9itchi Go service)
- `handlers/` thin (validate, authorize, call service, JSON) -> `services/` all logic, returns `(T, *models.ApiError)` -> `repositories/` only Postgres, plain errors (`ErrNotFound`, `ErrConflict`). Error body `{"error": "..."}`.
- DB calls use `repositories.QueryTimeout`; multi-table writes `Store.WithTx`; never select bytea (`users.photo`) in list queries.
- Schema: add `migrations/NNNN_name.sql` (goose, embedded, runs at startup). Never edit a merged migration.
- Auth: `Authorization: Bearer` AES-GCM-wrapped HS256 JWT (claims `id`, `email`, `role`, `typ`, `exp`; access only). `/admin/*` = `RequireRole("admin")`. `/internal/*` = `x-api-key` (`INTERNAL_API_KEY`), blocked by nginx.
- Events: exchange `9itchi.events` (topic, routing key = event name), names in `core/events.go`; empty `RABBITMQ_URL` = `LogPublisher`.
- Env only, validated in `utils/config.go`. Common: `PORT`, `ENVIRONMENT`, `DATABASE_URL`, `ITCHI_SECRET_KEY`, `ITCHI_ENCRYPTION_KEY`, `INTERNAL_API_KEY`, `INTERNAL_API_KEY_REQUIRED`. Never log tokens, passwords, PII.
- Tests: `{resource}_test.go`, testify, func-field fakes (no testify/mock), success + error paths. Repo tests: `//go:build integration`, embedded Postgres (`PG_BINARIES` for local PG 16 binaries).
- Check: `gofmt -l .; go build ./... && go vet ./... && go test ./...`; `go test -tags integration -count=1 ./repositories/...`; `golangci-lint run --build-tags integration`.

## Routes
- Public: `POST /auth/signup|login|refresh|forgot-password|reset-password`; `GET /auth/google|linkedin`, `/auth/{google,linkedin}/callback`.
- User: `POST /auth/logout`; `GET|PUT|DELETE /users/me`; `PUT /users/me/language`; `GET|PUT|DELETE /users/me/photo`.
- Admin: `GET /admin/users`, `GET|DELETE /admin/users/:id`, `PUT /admin/users/:id/suspend`, `GET|PUT /admin/users/:id/note`.
- Internal: `GET /internal/users/:id` (notification-api).

## Data, peers, events
- Tables: `users` (`photo` bytea, one square JPEG <= 1 MiB; upload cap 5 MB before resize), `refresh_tokens`, `password_resets`, `user_notes`. `users.email_verified` (`0008`). Latest `0008_users_email_verified.sql`.
- Eraser (`pkg/peers/eraser.go`): `DELETE /internal/users/:id` on email, resume, job, notification, chat, tracker, payments (in that order, 15 s each), then deletes the account. One failure keeps the account; retryable.
- Publishes `user.registered` (`{user_id}`). Consumes nothing.
- Providers: `core.OAuthProvider` Google (`pkg/google`) + LinkedIn OIDC (`pkg/linkedin`), each 503 until its ID/secret are set; `Mailer` Mailgun, else `NoopMailer`.
- Env: `RABBITMQ_URL`, `FRONTEND_URL`, `ADMIN_EMAIL` + `ADMIN_PASSWORD` (bootstrap first admin), `{NOTIFICATION,PAYMENTS,RESUME,EMAIL,JOB,CHAT,TRACKER}_API_URL`, `GOOGLE_CLIENT_ID|SECRET|REDIRECT_URL`, `LINKEDIN_CLIENT_ID|SECRET|REDIRECT_URL`, `MAILGUN_API_KEY|DOMAIN|FROM_EMAIL|API_BASE|RATE_LIMIT`.

## Domain rules
- Roles `candidate` | `admin`. Language `fr` | `en` | `ar` (default `fr`). Suspended = 403 on login and refresh.
- Unknown-email login burns a dummy bcrypt compare; forgot-password always answers generically; reset code TTL 15 min.
- OAuth: failure redirects to frontend with `error=<provider>_failed`; tokens in URL fragment; OAuth users get a password-less account (random hash); verified email required; sign-in sets `users.email_verified`, carried as JWT claim `email_verified` (payments-api student discount). Password signups stay unverified.
- Profile: LinkedIn/GitHub URLs reduced to canonical handles (other hosts refused); phone stored without spaces.
- Subscription data lives only in payments-api (no `users.plan_id`).

## Gotchas
- `pkg/notification` and `pkg/payments` clients are unused stubs.
- LinkedIn app keys still to create; check the 7 peer URLs in production before testing deletion.
