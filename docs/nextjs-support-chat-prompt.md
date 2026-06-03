# Next.js Frontend Prompt — Support Chat

Use this document to implement **real-time support chat** (customer/guest ↔ admin) in the Auto-Store web frontend. Extends [nextjs-frontend-prompt.md](./nextjs-frontend-prompt.md). Backend spec: [support-chat.md](./support-chat.md). Payloads: [sample-payloads.md](./sample-payloads.md#support-chat).

**Status:** Backend implemented (REST + WebSocket + notifications). Implement frontend against the contract below.

---

## 1. What to build

| Persona | Capability |
|---------|------------|
| **Guest** | Chat without login; guest token in storage; email prompt per option B |
| **Logged-in customer** | Same widget; uses access token; history merges on login |
| **Admin** | Inbox at `/admin/support`; reply in real time; guest vs registered labels |

**UX:** Floating support button (bottom-right) → slide-over panel. Minimal, matches site accent — same design language as the rest of the storefront.

**Not in scope:** Attachments, typing indicators (unless backend adds them), multi-admin assignment UI.

---

## 2. Auth: two tokens

| Identity | Token | Storage |
|----------|-------|---------|
| Registered | `access_token` from login | Existing auth storage |
| Guest | `guest_token` from `POST /chat/guest-session` | `localStorage` key e.g. `guest_chat_token` + `guest_id` |

**API client:** for chat routes, send `Authorization: Bearer <access_token>` if logged in, else `Bearer <guest_token>`.

### Bootstrap (app layout or chat provider)

```typescript
async function ensureChatIdentity(user: User | null) {
  if (user) return { kind: "user" as const, token: getAccessToken() };
  let guestToken = localStorage.getItem("guest_chat_token");
  if (!guestToken) {
    const res = await api.post("/chat/guest-session", {}); // no auth
    guestToken = res.data.guest_token;
    localStorage.setItem("guest_chat_token", guestToken);
    localStorage.setItem("guest_id", res.data.guest_id);
  }
  return { kind: "guest" as const, token: guestToken };
}
```

Refresh guest token before expiry via `POST /chat/guest-session/refresh`.

### Link on login/register

After successful login or register:

```typescript
const guestToken = localStorage.getItem("guest_chat_token");
if (guestToken) {
  await api.post("/conversations/link-guest", { guest_token: guestToken }, userAccessToken);
  localStorage.removeItem("guest_chat_token");
  localStorage.removeItem("guest_id");
}
```

Then reconnect WebSocket with user access token.

---

## 3. Guest email — option B

**Do not block** the first message on email.

### Trigger 1 — after guest sends first message

Show a **dismissible inline banner** below the header (not a modal):

- Copy: “Add your email so we can reply if you leave this page.”
- Fields: email (required if submitting), name (optional)
- Actions: **Save** → `PATCH /conversations/:id` with `{ guest_email, guest_name }`; **Not now** → hide for 7 days (`localStorage` `chat_email_prompt_dismissed_at`)

Only show when: guest identity, `conversation.guest_email` is empty, and guest has sent ≥ 1 message.

### Trigger 2 — when admin replies and email still empty

Upgrade to a **soft sticky prompt** above the composer:

- Copy: “Support replied — add your email to get updates if you close this tab.”
- Same PATCH on save
- Do not block reading admin messages or replying without email

### Registered users

Never show guest email prompts; profile email is used for offline notifications.

---

## 4. API contract

Base: `NEXT_PUBLIC_API_URL/api/v1`. Envelope: `success`, `data`, `error`, `errors`, `meta`.

### Endpoints

| Method | Path | Auth | Notes |
|--------|------|------|--------|
| POST | `/chat/guest-session` | No | Issue guest token |
| POST | `/chat/guest-session/refresh` | Guest | Refresh TTL |
| GET | `/conversations/me` | Flexible | Current open thread |
| POST | `/conversations` | Flexible | Get-or-create open thread |
| GET | `/conversations/:id/messages` | Flexible / admin | `page`, `limit`, `since` |
| POST | `/conversations/:id/messages` | Flexible / admin | REST send fallback |
| PATCH | `/conversations/:id` | Flexible / admin | Close; guest email/name |
| PATCH | `/conversations/:id/read` | Flexible / admin | Mark read |
| POST | `/conversations/link-guest` | User | Body: `{ guest_token }` |
| GET | `/admin/conversations` | Admin | Inbox |
| GET | `/admin/conversations/unread-count` | Admin | Header badge |
| WS | `/ws/chat?token=...` | Flexible / admin | Realtime |

### Conversation shape

```typescript
type ConversationStatus = "open" | "closed";
type ContextType = "general" | "order" | "product";

interface Conversation {
  id: string;
  user_id: string | null;
  guest_id: string | null;
  guest_email: string | null;
  guest_name: string | null;
  status: ConversationStatus;
  context_type: ContextType | null;
  context_id: string | null;
  last_message_at: string;
  unread_count?: number; // server-computed for current viewer
  created_at: string;
}

interface ChatMessage {
  id: string;
  conversation_id: string;
  sender_type: "customer" | "admin" | "system";
  sender_user_id: string | null;
  body: string;
  created_at: string;
}
```

### Create / open conversation

```http
POST /conversations
Authorization: Bearer <token>
Content-Type: application/json

{
  "context_type": "product",
  "context_id": "550e8400-e29b-41d4-a716-446655440010"
}
```

Optional `guest_name` on first create for guests.

Deep link: order confirmation page opens widget with `context_type: "order"` and order id.

---

## 5. WebSocket client

**URL:** `${WS_BASE}/api/v1/ws/chat?token=${encodeURIComponent(token)}`

Use `wss://` in production. Derive from `NEXT_PUBLIC_API_URL` (replace `http` → `ws`).

### Lifecycle

1. Open panel → `ensureChatIdentity` → `POST /conversations` if no open thread → load messages → connect WS.
2. On open → send `{ type: "subscribe", conversation_id }`.
3. On `message.new` → append if not duplicate id → scroll to bottom → if admin message and guest without email, show trigger 2 banner.
4. Send: `POST .../messages` or WebSocket `{ "type": "message", "conversation_id", "body" }` (both persist and broadcast `message.new`).
5. Close panel → unsubscribe, close WS (save bandwidth).
6. Reconnect: exponential backoff (1s, 2s, 4s, max 30s); on reconnect refetch `GET .../messages?since=<lastMessageAt>`.

### Frame types

See [support-chat.md § WebSocket protocol](./support-chat.md#websocket-protocol).

---

## 6. Customer UI — chat widget

### Layout

- **Closed:** circular button, message icon, optional unread dot from `GET /conversations/me` poll every 60s when logged in (guest: poll when token exists).
- **Open:** `fixed` panel ~380×520px mobile full-screen sheet; header “Support”, close button, message list, composer.

### Message list

- Customer messages: align right, accent bubble.
- Admin messages: align left, neutral bubble.
- System messages: centered, muted text, small.
- Show time on hover or every 5+ min gap.
- Empty state: “Ask us anything about orders, parts, or fitment.”

### Composer

- Textarea + Send; Enter sends, Shift+Enter newline.
- Disable send when empty or over 2000 chars.
- After guest first send → show email banner (trigger 1).

### Close conversation

Footer link “Mark as resolved” → `PATCH /conversations/:id` `{ status: "closed" }` → clear local thread; next open creates new conversation.

---

## 7. Admin UI — `/admin/support`

**Route:** `(admin)/admin/support` — require `role === ADMIN`.

### Inbox (master)

- List from `GET /admin/conversations?status=open&page&limit`
- Sort by `last_message_at` desc (server default)
- Row: display name (`guest_name` or user first/last), email snippet, last message preview, unread badge, **Guest** chip if `user_id` null
- Filters: All | Guests only | Unread only
- Header badge: `GET /admin/conversations/unread-count`

### Thread (detail)

- Select row → load messages → WS subscribe as admin
- Reply composer same as customer
- Actions: Close conversation
- Sidebar optional: `context_type` / `context_id` link to order or product admin page

### Realtime

- New customer message → update inbox preview + unread without full reload
- Play subtle sound optional (off by default)

---

## 8. Contextual entry points

| Page | On “Help” / widget open |
|------|-------------------------|
| Product detail | `POST /conversations` with `context_type: "product"`, `context_id` |
| Order detail | `context_type: "order"`, `context_id` |
| Global footer | `context_type: "general"` or omit |

Show small context chip in widget header: “About order #1234” linking to order page.

---

## 9. Error handling

| HTTP | UX |
|------|-----|
| 401 | Guest: new guest session; User: refresh token or login |
| 403 | “You don’t have access to this conversation” |
| 404 on `/conversations/me` | Call `POST /conversations` |
| 429 | “Too many messages — wait a moment” |
| WS disconnect | Banner “Reconnecting…” + fallback poll |

---

## 10. File structure (suggested)

```text
src/
  components/support/
    SupportChatProvider.tsx   # identity, conversation, WS
    SupportChatButton.tsx
    SupportChatPanel.tsx
    MessageList.tsx
    MessageComposer.tsx
    GuestEmailPrompt.tsx    # option B triggers
  app/admin/support/
    page.tsx                  # inbox + thread split view
  lib/chat/
    api.ts
    websocket.ts
    types.ts
```

---

## 11. Notifications

When the customer is offline, the API sends async alerts:

| Type | Recipient | Action in UI |
|------|-----------|--------------|
| `support.admin_replied` | Registered customer | Bell → open `/support` widget (`payload.href`) |
| `support.new_conversation` | Admin | Bell → `/admin/support/:conversationId` |

Guests with saved email receive admin replies via SMTP (no bell entry).

After reading a thread, call `PATCH /conversations/:id/read` to clear unread counts.

---

## 12. Acceptance checklist

- [ ] Guest can send first message without email
- [ ] Email banner appears after guest’s first message (skippable)
- [ ] Stronger email prompt when admin replies and email empty
- [ ] Logged-in user uses same widget; no guest prompts
- [ ] Login merges guest history via `link-guest`
- [ ] Admin inbox shows guest vs registered
- [ ] Messages appear in real time both directions with WS
- [ ] REST fallback works when WS blocked
- [ ] Unread badge updates for admin and customer
- [ ] Contextual open from product/order pages

---

## 13. One-shot build prompt

**“Build a minimalist Next.js support chat widget and admin inbox for the Auto-Store API: guest session + logged-in users, WebSocket realtime with REST fallback, guest email capture after first message or when admin replies (option B), link guest on login, floating customer widget and `/admin/support` inbox. Follow docs/nextjs-support-chat-prompt.md and support-chat.md.”**
