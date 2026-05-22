# Notifications

Async notifications for in-app feed (Next.js bell) and email delivery. SMS and push are stubbed for future use.

## Architecture

1. Domain code calls `Notifier` helpers (e.g. `MechanicVerified`).
2. `NotificationService.Notify` writes one row per channel (`in_app`, `email`, …) with an idempotency key.
3. In-app rows are marked `sent` immediately.
4. Email rows are pushed to Redis list `notifications:send`.
5. **`cmd/worker`** dequeues jobs, sends SMTP (or logs in dev), updates status.
6. If Redis is down, rows stay `pending`/`queued`; the worker polls the DB every 60s.

## Run locally

Terminal 1 — API:

```bash
make run
```

Terminal 2 — worker:

```bash
make run-worker
```

Requires Postgres and Redis (e.g. `docker-compose up -d postgres redis`).

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_FRONTEND_URL` | `http://localhost:3000` | Base URL in email bodies (Next.js) |
| `SMTP_*` | — | If `SMTP_HOST` empty, emails are logged only |
| `NOTIFICATIONS_MAX_RETRIES` | `5` | Email delivery retries |
| `NOTIFICATIONS_DEQUEUE_TIMEOUT_SEC` | `5` | Redis `BRPOP` timeout |

## API (Next.js)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/notifications` | In-app list (`?page&limit&unread_only=true`) |
| GET | `/api/v1/notifications/unread-count` | Badge count |
| PATCH | `/api/v1/notifications/:id/read` | Mark one read |
| PATCH | `/api/v1/notifications/read-all` | Mark all read |
| GET | `/api/v1/users/me/notification-preferences` | Preferences |
| PUT | `/api/v1/users/me/notification-preferences` | Update preferences |

Payload includes `href` for App Router links, e.g. `/mechanic/profile`.

## Event types

| Type | Trigger (current) |
|------|-------------------|
| `mechanic.apply_received` | `POST /mechanic/apply` |
| `mechanic.verified` | Admin `PUT /admin/mechanics/:userId/verify` |
| `quote.ready` | After `POST /installation/quotes` |
| `booking.confirmed` | After `POST /installation/bookings` (customer + mechanic) |
| `mechanic.en_route` | Mechanic sets booking status `en_route` |
| `qa.answer_posted` | `Notifier.QAAnswerPosted` (future Q&A) |

Sample payloads: [sample-payloads.md](./sample-payloads.md#notifications).
