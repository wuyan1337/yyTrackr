# yyTrackr Agent Guide

Go application with Gin, GORM/SQLite, HTMX and server-rendered templates.

- Entry point: `main.go`; server wiring and background jobs: `internal/app/server.go`.
- Keep subscription CRUD, categories, calendar views, analytics and per-account isolation.
- Notifications use Telegram and Webhook; shared message formatting lives in `internal/service/notification_text.go`.
- Settings include currencies, API keys and UI personalization. CSV/JSON exports are supported.
- Authentication requires registration/login; preserve logout and remembered sessions.
- Never delete production databases or user records during deployments.
- Format Go changes and run handlers/service unit tests, then `go build`.
- Keep deployment snapshots outside the source tree. Verify `/healthz` after restart.
