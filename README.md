# 9itchi-account-api

Authentication and identity hub. Signup, login, JWT issue/refresh, Google sign-in, profiles, roles, account deletion, admin user management.

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
| GET | `/users/me` | user | GetMe |
| PUT | `/users/me` | user | UpdateMe |
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
