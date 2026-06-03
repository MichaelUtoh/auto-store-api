# Support chat

Real-time **user ↔ admin** support chat. Supports **logged-in customers** and **guests**. Distinct from [Community Q&A](./community-qa.md) (public mechanic answers, SEO) and [Notifications](./notifications.md) (async bell + email).

**Status:** Implemented (REST + WebSocket). See [nextjs-support-chat-prompt.md](./nextjs-support-chat-prompt.md) for frontend.

**Sample payloads:** [sample-payloads.md](./sample-payloads.md#support-chat)

---

## Goals

| In scope (v1) | Out of scope (defer) |
|---------------|----------------------|
| Text messages, persisted history | File/image attachments |
| One open conversation per identity | Multi-topic threads per user |
| Guest + registered customer | Vendor/mechanic support queues |
| Real-time via WebSocket + Redis pub/sub | Typing indicators (optional v1.1) |
| Admin shared inbox | Assignment, SLA, canned replies |
| Guest email capture **after first message or admin reply** (option B) | Required email before chat |
| Link guest history on login/register | Full-text search, exports |

---

## Architecture

```text
Customer/Guest (Next.js widget) ──REST──► Go API ──► Postgres (conversations, messages)
Admin (Next.js /admin/support) ──REST──┘              │
       │                                               │
       └── WebSocket /ws/chat ◄── Redis PUB/SUB ──────┘
```

1. **REST** — history, send fallback, inbox, guest session, link-on-login.
2. **WebSocket** — subscribe to a conversation; receive `message.new` in real time.
3. **Redis** — `PUBLISH chat:conv:{conversation_id}` after each persisted message; all API instances subscribe and fan out to local connections.
4. **Postgres** — source of truth for conversations and messages.

When no admin is connected and the customer has provided `guest_email` (or is a registered user), enqueue **`support.admin_replied`** via existing `NotificationService` (in-app for users; email for guests with email).

---

## Identity model

Every conversation has **one owner identity** at creation:

| Owner | `user_id` | `guest_id` |
|-------|-----------|------------|
| Registered customer | set | `NULL` |
| Guest | `NULL` | set |

**Constraint:** exactly one of `user_id` or `guest_id` is non-null (until merged).

### Guest session

Guests do not use login JWT. Issue a **guest token** (signed JWT with `type: "guest"`, `guest_id`, expiry e.g. 30 days, refreshable).

| Token | Claims | Used for |
|-------|--------|----------|
| Access token (existing) | `user_id`, `role`, … | Registered customers + admins |
| Guest token (new) | `guest_id`, `type: "guest"` | Anonymous chat |

**FlexibleAuth middleware:** validate Bearer token — if user JWT, set `user_id`; else if guest JWT, set `guest_id`; else 401.

### Link guest → account

On login or register, client calls `POST /conversations/link-guest` with the stored guest token. Server attaches open (and optionally recent closed) guest conversations to `user_id`. Invalidate or mark guest session merged.

---

## Guest email (option B)

Email is **not required** to open chat or send the first message.

Prompt for email when **either**:

1. **After the guest sends their first message** — soft inline banner in the chat panel: “Add your email so we can reply if you leave this page.” Skippable; re-show on next visit until saved or dismissed for 7 days (client preference).
2. **When an admin sends the first reply** — if `guest_email` is still empty, show a more prominent prompt: “Support replied — add your email to get updates.” Same PATCH endpoint to save.

Saving email:

```http
PATCH /api/v1/conversations/:id
Authorization: Bearer <guest_token>
{ "guest_email": "ada@example.com", "guest_name": "Ada" }
```

Backend validates email format. Once saved, admin replies can trigger email via notification worker (reuse SMTP stack).

Registered users use profile email; no guest fields on their conversations.

---

## Data model

### `conversations`

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID | PK |
| `user_id` | UUID NULL | FK `users` |
| `guest_id` | UUID NULL | Indexed; stable guest identity |
| `guest_email` | string NULL | Optional until captured |
| `guest_name` | string NULL | Optional display name |
| `status` | enum | `open`, `closed` |
| `context_type` | enum NULL | `general`, `order`, `product` |
| `context_id` | UUID NULL | Order/product UUID when typed |
| `last_message_at` | timestamp | Inbox sort |
| `customer_last_read_at` | timestamp NULL | Unread for customer |
| `admin_last_read_at` | timestamp NULL | Unread for admin |
| `created_at`, `updated_at` | timestamps | |

**Indexes:** `(user_id, status)`, `(guest_id, status)`, `last_message_at DESC`.

**Business rule:** at most one `status = open` conversation per `user_id` or per `guest_id`.

### `messages`

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID | PK |
| `conversation_id` | UUID | FK |
| `sender_type` | enum | `customer`, `admin`, `system` |
| `sender_user_id` | UUID NULL | Set for admin; NULL for guest customer messages |
| `body` | text | Max 2000 chars |
| `created_at` | timestamp | |

**System messages** (optional v1): e.g. “Conversation closed”, “Please add your email…” — `sender_type: system`.

---

## WebSocket protocol

**Endpoint:** `GET /api/v1/ws/chat?token=<access_or_guest_token>`

Upgrade headers; validate token before accept. Ping/pong or application heartbeat every 30s.

### Client → server (JSON text frames)

```json
{ "type": "subscribe", "conversation_id": "<uuid>" }
```

```json
{ "type": "message", "conversation_id": "<uuid>", "body": "Hello" }
```

Optional v1.1:

```json
{ "type": "typing", "conversation_id": "<uuid>" }
```

### Server → client

```json
{
  "type": "message.new",
  "message": {
    "id": "<uuid>",
    "conversation_id": "<uuid>",
    "sender_type": "admin",
    "sender_user_id": "<uuid>",
    "body": "How can I help?",
    "created_at": "2026-06-03T12:00:00Z"
  }
}
```

```json
{ "type": "error", "code": "forbidden", "message": "..." }
```

**Authorization:** on subscribe and message, verify token owner matches conversation (`user_id` or `guest_id`). Admins may subscribe to any conversation.

**Send path:** persist to Postgres → publish Redis → fan out to subscribed connections (including sender, for id consistency).

**Fallback:** if WebSocket unavailable, `POST .../messages` + poll `GET .../messages?since=<iso8601>` every 3s while panel open.

---

## REST API

Base: `/api/v1`. Standard response envelope. Auth: **FlexibleAuth** (user or guest) unless noted.

### Guest session (no auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/chat/guest-session` | Create or return guest session (rate limited by IP) |
| POST | `/chat/guest-session/refresh` | Extend guest token (`Authorization: Bearer <guest_token>`) |

### Conversations (customer / guest)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/conversations/me` | Flexible | Get open conversation for current identity (404 if none) |
| POST | `/conversations` | Flexible | Get-or-create open conversation |
| GET | `/conversations/:id` | Owner or admin | Conversation metadata |
| GET | `/conversations/:id/messages` | Owner or admin | Paginated history (`page`, `limit`, optional `since`) |
| POST | `/conversations/:id/messages` | Owner or admin | Send message (REST fallback) |
| PATCH | `/conversations/:id` | Owner or admin | Close; update `guest_email` / `guest_name` |
| PATCH | `/conversations/:id/read` | Owner or admin | Update read cursor for caller side |
| POST | `/conversations/link-guest` | User JWT | Merge guest conversations into logged-in user |

### Admin

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/admin/conversations` | Admin | Inbox (`status`, `guest_only`, `unread_only`, `page`, `limit`) |
| GET | `/admin/conversations/unread-count` | Admin | Badge count |
| POST | `/admin/conversations/:id/messages` | Admin | Reply (same as customer POST; role checked) |

### WebSocket

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/ws/chat` | Flexible or admin | WebSocket upgrade (`token` query param) |

---

## POST /conversations body (optional)

```json
{
  "context_type": "order",
  "context_id": "550e8400-e29b-41d4-a716-446655440099",
  "guest_name": "Ada"
}
```

If an open conversation already exists for the identity, return it (ignore duplicate create).

---

## Notifications

| Type | When | Channels |
|------|------|----------|
| `support.admin_replied` | Admin sends message | In-app (registered user); email if guest has `guest_email` |
| `support.new_conversation` | Optional: guest/user opens first message | In-app for admins (future: admin notification preferences) |

Payload example:

```json
{
  "conversation_id": "<uuid>",
  "href": "/admin/support/<uuid>"
}
```

For registered customers, `href` on customer payload: `/support` or deep link to widget.

---

## Security & abuse

| Control | Implementation |
|---------|----------------|
| Rate limit guest session | e.g. 10/hour per IP |
| Rate limit messages | e.g. 20/min per identity + IP |
| Max body length | 2000 characters |
| Conversation access | Strict owner check on every REST/WS action |
| Guest token | Separate signing secret or `type` claim; short enough to limit blast radius |
| CAPTCHA | Optional env flag on `POST /chat/guest-session` (defer unless abuse) |
| Close conversation | Customer, guest, or admin |

---

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `GUEST_JWT_SECRET` | falls back to `JWT_SECRET` | Sign guest tokens |
| `GUEST_JWT_EXPIRY` | `720h` (30d) | Guest token TTL |
| `CHAT_MESSAGE_MAX_LEN` | `2000` | Max message body |
| `CHAT_RATE_LIMIT_PER_MIN` | `20` | Messages per identity |
| `APP_FRONTEND_URL` | (existing) | Email links for guest follow-up |

---

## Implementation phases

### Phase 1 — REST + dual auth

- Models, repositories, `ChatService`, `GuestTokenManager`, `FlexibleAuth`
- Guest session, conversations, messages, link-guest, admin inbox (REST only)
- Frontend widget with polling

### Phase 2 — WebSocket + Redis

- Hub, Redis pub/sub, WS handler on Gin
- Replace polling when connected; reconnect + history refetch

### Phase 3 — Polish ✅

- Unread counts and read cursors (`PATCH /conversations/:id/read`, admin badge)
- `support.admin_replied` — in-app + email for registered users; direct SMTP for guests with email
- `support.new_conversation` — in-app for all admins on first customer message
- Swagger annotations (`support-chat` tag); run `make swagger` to regenerate
- Docker Compose exposes API port directly (WebSocket works on `:8089`); see **Deployment** below

---

## Deployment

### Docker Compose (local)

`docker compose up` maps `8089:8089`. WebSocket clients connect to:

`ws://localhost:8089/api/v1/ws/chat?token=...`

Redis is required for cross-instance fan-out when running multiple API replicas.

### Reverse proxy (production)

If terminating TLS at nginx, forward WebSocket upgrade headers:

```nginx
location /api/v1/ws/chat {
    proxy_pass http://api:8089;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 3600s;
}
```

Do **not** gzip-compress WebSocket responses. The API applies gzip to other routes only; keep `/ws/chat` on a separate location without `gzip on` if you compress at the proxy layer.

## Related docs

- [notifications.md](./notifications.md) — async delivery
- [community-qa.md](./community-qa.md) — public Q&A (not chat)
- [endpoints.md](./endpoints.md) — planned routes summary
