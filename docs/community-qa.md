# Community Q&A

Product- and vehicle-linked questions answered by **verified mechanics**, with public SEO-friendly pages.

Prerequisites: [mechanics.md](./mechanics.md) (verified mechanic profiles for answers).

---

## Flow

1. Authenticated user posts a question (`POST /questions`) linked to a product, category, or vehicle (make + model).
2. Verified mechanics reply (`POST /questions/:id/answers`).
3. Question author accepts the best answer (`PATCH /questions/:id/accept-answer/:answerId`).
4. Author or admin can close the thread (`PATCH /questions/:id/close`).
5. Question author receives `qa.answer_posted` notification when a mechanic answers.

---

## API

### Public

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/questions` | List questions (query: `q`, `product_id`, `category_id`, `make`, `model`, `year`, `status`, `page`, `limit`) |
| GET | `/api/v1/questions/:slug` | Question detail + answers (increments view count) |
| GET | `/api/v1/products/:id/questions` | Questions for a product |

### Customer (authenticated)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/questions` | Ask a question |
| PATCH | `/api/v1/questions/:id/accept-answer/:answerId` | Accept an answer (author only) |
| PATCH | `/api/v1/questions/:id/close` | Close thread (author) |

### Verified mechanic

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/questions/:id/answers` | Post an answer (one per mechanic per question) |

### Admin

| Method | Path | Description |
|--------|------|-------------|
| PATCH | `/api/v1/questions/:id/close` | Close any thread |

---

## Data model

| Table | Purpose |
|-------|---------|
| `questions` | Thread: title, body, slug, product/category/vehicle context, status |
| `answers` | Responses; `is_verified_mechanic`, `is_accepted` |

**Statuses:** `open`, `answered`, `closed`.

---

## Notifications

| Type | When |
|------|------|
| `qa.answer_posted` | Verified mechanic posts an answer (notifies question author) |

See [notifications.md](./notifications.md).

---

## SEO (frontend)

- Public pages: `/q/[slug]` from `slug` in API responses
- JSON-LD `QAPage` schema recommended on Next.js SSR pages
- Sitemap from `GET /questions`

**Frontend implementation guide:** [nextjs-community-qa-prompt.md](./nextjs-community-qa-prompt.md)

---

## Not yet implemented

- Voting / reputation
- Spam reporting and moderation queue
- Mechanic category certification gates for answers
