# 9itchi-account-api

Authentication and identity hub. Signup, login, JWT issue/refresh, Google sign-in, profiles, roles, account deletion, admin user management.

Port `8081`, health `/health`. Skeleton only: every endpoint returns `501 not implemented` until built.

## Endpoints

| Method | Path | Access | Handler |
|---|---|---|---|
| POST | `/auth/signup` | public | Signup |
| POST | `/auth/login` | public | Login |
| POST | `/auth/refresh` | public | RefreshToken |
| POST | `/auth/logout` | user | Logout |
| POST | `/auth/forgot-password` | public | ForgotPassword |
| POST | `/auth/reset-password` | public | ResetPassword |
| GET | `/auth/google` | public | GoogleSignIn |
| GET | `/auth/google/callback` | public | GoogleCallback |
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
