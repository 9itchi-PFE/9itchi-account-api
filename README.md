# 9itchi-account-api

Authentication and identity hub. Signup, login, JWT issue/refresh, Google and LinkedIn sign-in, profiles, roles, account deletion, admin user management.

Port `8081`, health `/health`. All endpoints are implemented.

Access token 15 min, refresh token 7 days (stored hashed, rotated on refresh). Password reset codes go through `core.Mailer`: Mailgun when `MAILGUN_API_KEY` and `MAILGUN_DOMAIN` are set, otherwise a no-op (nothing is sent). Send failures are logged generically and never change the `forgot-password` response. Set `ADMIN_EMAIL` and `ADMIN_PASSWORD` to create the first admin at startup when none exists.

## Mailgun (all optional)

| Env | Default | Notes |
|---|---|---|
| `MAILGUN_API_KEY` | empty | with `MAILGUN_DOMAIN`, enables sending |
| `MAILGUN_DOMAIN` | empty | sending domain |
| `MAILGUN_FROM_EMAIL` | `9itchi <no-reply@MAILGUN_DOMAIN>` | sender |
| `MAILGUN_API_BASE` | US region | `https://api.eu.mailgun.net` for EU |
| `MAILGUN_RATE_LIMIT` | `10` | sends per second |

3 attempts (retry only on 5xx/network errors), 15s cap per call.

## Events (RabbitMQ, optional)

With `RABBITMQ_URL` set, `user.registered` (`user_id` only: no name, no email) is published to the topic exchange `9itchi.events` when a student account is created, by signup or by a first sign-in with Google or LinkedIn. The message is persistent JSON, confirmed by the broker, and carries a message id. Without the variable the event name is only logged.

A publish failure is logged and never fails the signup. A broker that cannot be reached is left alone for 15 s; registrations made meanwhile are not counted (there is no outbox). Broker tests: `RABBITMQ_TEST_URL=amqp://guest:guest@localhost:5672/ go test -tags integration ./core/` (skipped without the variable).

## Endpoints

| Method | Path | Access | Handler |
|---|---|---|---|
| POST | `/auth/signup` | public | Signup |
| POST | `/auth/login` | public | Login |
| POST | `/auth/refresh` | public | RefreshToken |
| POST | `/auth/logout` | user | Logout |
| POST | `/auth/forgot-password` | public | ForgotPassword |
| POST | `/auth/reset-password` | public | ResetPassword |
| GET | `/auth/google` | public | GoogleSignIn (302 to Google, 503 when unconfigured) |
| GET | `/auth/google/callback` | public | GoogleCallback (302 to `FRONTEND_URL/auth/callback#access_token=...&refresh_token=...`, failures to `FRONTEND_URL/auth/signin?error=google_failed`) |
| GET | `/auth/linkedin` | public | LinkedInSignIn (302 to LinkedIn, OpenID Connect; 503 when unconfigured) |
| GET | `/auth/linkedin/callback` | public | LinkedInCallback (same redirects as Google, failures to `...?error=linkedin_failed`) |
| GET | `/users/me` | user | GetMe |
| PUT | `/users/me` | user | UpdateMe |
| PUT | `/users/me/language` | user | SetLanguage (`fr`, `en` or `ar`: the language 9itchi writes to the user in; also accepted at signup, `fr` by default) |
| DELETE | `/users/me` | user | DeleteMe |
| GET | `/admin/users` | admin | ListUsers |
| GET | `/admin/users/:id` | admin | GetUser |
| PUT | `/admin/users/:id/suspend` | admin | SuspendUser |
| DELETE | `/admin/users/:id` | admin | DeleteUser |
| GET | `/internal/users/:id` | internal | InternalGetUser |

## Run

```
cp .env.example .env
go run main.go
```
- Users carry an optional `school` (a short code such as `ENIS`, at most 120 characters): set at sign-up, changed with `PUT /users/me` (`"school": ""` clears it, no key keeps it), returned in the user view. Migration `0003`.
