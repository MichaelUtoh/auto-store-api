# Next.js Frontend Prompt — Community Q&A

Use this document to implement **Community Q&A** in the Auto-Store web frontend. It extends the base contract in [nextjs-frontend-prompt.md](./nextjs-frontend-prompt.md) (auth envelope, Tailwind minimalist design, API client). Backend reference: [community-qa.md](./community-qa.md), [sample-payloads.md](./sample-payloads.md#community-qa).

---

## 1. What to build

**Community Q&A** is a public knowledge base where shoppers ask part-fit and install questions, and **verified mechanics** answer. It is separate from product **reviews** (star ratings after purchase).

| Persona | Capability |
|---------|------------|
| **Guest** | Browse questions, read threads (SEO pages), search/filter |
| **Logged-in customer** | Ask questions, accept best answer, close own threads |
| **Verified mechanic** (`role: MECHANIC`, `mechanic_profile.is_verified`) | Post one answer per question |
| **Admin** | Close any thread |

**Trust signal:** Show a badge on mechanic answers: “Verified mechanic · {business_name}” using `author.mechanic_profile`.

---

## 2. API contract

Base URL: `NEXT_PUBLIC_API_URL/api/v1`. All JSON uses the standard envelope (`success`, `data`, `error`, `errors`, `meta`). See [nextjs-frontend-prompt.md §3](./nextjs-frontend-prompt.md#3-backend-api-contract).

### Endpoints

| Method | Path | Auth | Notes |
|--------|------|------|--------|
| GET | `/questions` | No | List; query below |
| GET | `/questions/:slug` | No | Detail + answers; **increments `view_count`** |
| GET | `/products/:id/questions` | No | Product-scoped list (same list item shape) |
| POST | `/questions` | Yes | Create question |
| POST | `/questions/:id/answers` | Verified mechanic | `:id` = question **UUID**, not slug |
| PATCH | `/questions/:id/accept-answer/:answerId` | Yes (author) | Accept one answer |
| PATCH | `/questions/:id/close` | Yes (author) or Admin | Close thread |

### List query parameters (`GET /questions`)

| Param | Type | Description |
|-------|------|-------------|
| `q` | string | Search title/body |
| `product_id` | UUID | Filter by product |
| `category_id` | UUID | Filter by category |
| `make`, `model` | string | Vehicle filter |
| `year` | number | Vehicle year |
| `status` | `open` \| `answered` \| `closed` | Optional; default public list excludes `closed` |
| `page`, `limit` | number | Pagination (default limit 20) |

### Create question body (`POST /questions`)

At least one context is **required**:

- `product_id` (UUID), or
- `category_id` (UUID), or
- `make` + `model` (both strings; `year` optional)

```json
{
  "title": "Will these pads fit a 2018 Camry LE?",
  "body": "I see compatibility for 2015-2020 but want to confirm LE trim.",
  "product_id": "550e8400-e29b-41d4-a716-446655440010"
}
```

Validation: `title` 5–200 chars, `body` 10–5000 chars.

### Create answer body (`POST /questions/:id/answers`)

```json
{ "body": "Yes — fits 2018 Camry LE with standard brake package." }
```

Errors to handle:

| HTTP | Meaning |
|------|---------|
| 403 | Not a verified mechanic |
| 400 | `you have already answered this question` |
| 400 | `question is closed` |

### Notifications

When a mechanic answers, the question author gets type `qa.answer_posted` with payload:

```json
{
  "question_id": "<uuid>",
  "href": "/q/<slug>"
}
```

Wire the notification bell to navigate to `payload.href` (see [notifications.md](./notifications.md)).

---

## 3. TypeScript types

Match API `snake_case` or map to `camelCase` in a thin mapper. Suggested types:

```typescript
type QuestionStatus = "open" | "answered" | "closed";

interface QuestionAuthor {
  id: string;
  first_name: string;
  last_name: string;
}

interface MechanicProfileSummary {
  id: string;
  status: string;
  business_name: string;
  is_verified: boolean;
}

interface AnswerAuthor {
  id: string;
  first_name: string;
  last_name: string;
  mechanic_profile?: MechanicProfileSummary;
}

interface Answer {
  id: string;
  question_id: string;
  body: string;
  is_accepted: boolean;
  is_verified_mechanic: boolean;
  author: AnswerAuthor;
  created_at: string;
}

interface QuestionListItem {
  id: string;
  title: string;
  slug: string;
  status: QuestionStatus;
  view_count: number;
  product_id?: string;
  category_id?: string;
  make?: string;
  model?: string;
  year?: number;
  author: QuestionAuthor;
  answer_count: number;
  accepted_answer?: Answer;
  created_at: string;
}

interface QuestionDetail extends QuestionListItem {
  body: string;
  answers: Answer[];
  updated_at: string;
}

interface CreateQuestionInput {
  title: string;
  body: string;
  product_id?: string;
  category_id?: string;
  make?: string;
  model?: string;
  year?: number;
}
```

---

## 4. Routes (App Router)

| Route | Rendering | Data |
|-------|-----------|------|
| `/q` | Server or client | `GET /questions?page&limit` — browse hub |
| `/q/[slug]` | **Server Component (SSR)** | `GET /questions/:slug` — thread detail, SEO |
| `/q/ask` | Client | `POST /questions` — form (protected) |
| `/products/[id]` (existing) | Embed section | `GET /products/:id/questions` — widget |
| `/mechanic/q` (optional) | Client | Open questions for mechanics to answer |

**Canonical URLs:** Use `/q/[slug]` only (not UUID in public URLs). Slug comes from API on create.

**Redirects:** After `POST /questions`, redirect to `/q/{slug}`.

---

## 5. Pages and UI

Follow the minimalist design in [nextjs-frontend-prompt.md §2](./nextjs-frontend-prompt.md#2-design-direction-minimalist--modern): whitespace, one accent, thin borders, no component library.

### 5.1 Product detail — Q&A tab/section

On `GET /products/:id`:

- Fetch `GET /products/:id/questions?limit=5`.
- Show list: title, status badge, answer count, relative date.
- Link each row → `/q/[slug]`.
- CTA: **“Ask about this product”** → `/q/ask?product_id={id}` (pre-fill `product_id`, hide vehicle fields).
- If no questions: empty state + same CTA.

Do **not** confuse with reviews block (stars); keep Q&A visually distinct (e.g. “Questions & answers” vs “Customer reviews”).

### 5.2 Browse hub — `/q`

- Search input → `q` param.
- Optional filters: vehicle make/model/year (reuse garage picker if you have one), status chips.
- Cards: title, excerpt (truncate accepted answer body or “No answers yet”), author, answer count, status, views.
- Pagination from `meta`.

### 5.3 Thread detail — `/q/[slug]`

Server-render for SEO:

- **Header:** title, status badge (`open` / `answered` / `closed`), view count, author, created date.
- **Context chips:** product link (if `product_id`), or “{year} {make} {model}”.
- **Body:** full question text.
- **Answers list:** sort accepted first (API already orders); each card:
  - Author name
  - If `is_verified_mechanic` && `author.mechanic_profile`: badge **Verified mechanic · {business_name}**
  - If `is_accepted`: **Accepted answer** label
  - Body, timestamp

**Actions (client islands):**

| Who | UI |
|-----|-----|
| Verified mechanic, thread not closed, no existing answer from self | Answer form → `POST /questions/:id/answers` |
| Question author, not closed | “Accept” on each non-accepted answer → `PATCH .../accept-answer/:answerId` |
| Question author or admin | “Close thread” → `PATCH .../close` with confirm dialog |
| Guest | “Log in to ask” / “Apply as mechanic to answer” links |

After accept/answer/close: revalidate path or refetch detail.

### 5.4 Ask question — `/q/ask`

Protected route (redirect to login if no token).

**Form fields:**

- Title (required)
- Body (required, textarea)
- Context (one of):
  - **Product** — hidden or select if `?product_id=` in query
  - **Category** — dropdown from `GET /categories`
  - **Vehicle** — make, model, year (required make+model if no product/category)

Client validation mirrors API mins before submit. On success → `router.push(/q/${slug})`.

### 5.5 Mechanic dashboard (optional)

`/mechanic/q` — list `GET /questions?status=open` with filters; link to thread; answer inline or on detail page.

Show only if `user.role === "MECHANIC"` && `user.mechanic_profile?.is_verified`. Otherwise link to `/mechanic/apply` or profile status.

---

## 6. SEO

Implement on `/q/[slug]` (Server Component):

### Metadata

```typescript
export async function generateMetadata({ params }: { params: { slug: string } }) {
  const q = await fetchQuestion(params.slug);
  return {
    title: `${q.title} | Auto-Store Q&A`,
    description: q.body.slice(0, 160),
    alternates: { canonical: `/q/${q.slug}` },
  };
}
```

### JSON-LD (`QAPage`)

Emit in `<script type="application/ld+json">` when at least one answer exists:

- `mainEntity` = Question (`name`, `text`, `dateCreated`, `author`)
- `acceptedAnswer` or `suggestedAnswer` array from `answers` (`is_accepted` → accepted)
- Use mechanic `business_name` in answer author if verified

### Sitemap

Add `/q/[slug]` entries from paginated `GET /questions` (exclude `closed` or include with lower priority — your choice; API hides closed from default list).

### Indexing

- `noindex` optional for `status === "closed"` threads.
- Avoid duplicate content: one canonical slug per thread.

---

## 7. Auth and permissions (client)

Derive from `GET /users/me` (or login user object):

```typescript
const canAnswer =
  user?.role === "MECHANIC" &&
  user?.mechanic_profile?.is_verified === true;

const isAuthor = user?.id === question.author.id;

const canAccept = isAuthor && question.status !== "closed";

const canClose =
  isAuthor || user?.role === "ADMIN";
```

**Do not** rely on UI-only checks — API enforces 403. Show friendly messages when POST answer returns 403.

---

## 8. API client helpers

Extend your shared `api.ts`:

```typescript
export function listQuestions(params: Record<string, string | number | undefined>) {
  const qs = new URLSearchParams(/* omit undefined */);
  return api.get<QuestionListItem[]>(`/questions?${qs}`, { auth: false });
}

export function getQuestionBySlug(slug: string) {
  return api.get<QuestionDetail>(`/questions/${slug}`, { auth: false });
}

export function listProductQuestions(productId: string, page = 1) {
  return api.get<QuestionListItem[]>(`/products/${productId}/questions?page=${page}`, { auth: false });
}

export function createQuestion(body: CreateQuestionInput) {
  return api.post<QuestionDetail>("/questions", body, { auth: true });
}

export function createAnswer(questionId: string, body: string) {
  return api.post<Answer>(`/questions/${questionId}/answers`, { body }, { auth: true });
}

export function acceptAnswer(questionId: string, answerId: string) {
  return api.patch<QuestionDetail>(`/questions/${questionId}/accept-answer/${answerId}`, {}, { auth: true });
}

export function closeQuestion(questionId: string) {
  return api.patch<QuestionDetail>(`/questions/${questionId}/close`, {}, { auth: true });
}
```

Use **question `id` (UUID)** for mutations; use **`slug`** for public routes and fetches.

---

## 9. State and UX details

- **Optimistic UI:** Optional for accept; prefer refetch on success for simplicity.
- **Loading:** Skeleton cards on list/detail; disable submit while posting.
- **Errors:** Map `errors[]` to fields on ask form; toast for 403/409 on answer.
- **View count:** Detail fetch increments count — avoid double-fetch in Strict Mode (single server fetch on SSR is enough).
- **Status badges:** Subtle pills — `open` (neutral), `answered` (accent), `closed` (muted).
- **Empty states:** “No questions yet — be the first to ask” with CTA.

---

## 10. Integration checklist

- [ ] Types for list item, detail, answer, create input
- [ ] `/q` browse + search + pagination
- [ ] `/q/[slug]` SSR + metadata + JSON-LD
- [ ] `/q/ask` with product/category/vehicle context
- [ ] Product page Q&A section + deep link to ask
- [ ] Mechanic answer form + verified badge
- [ ] Author accept + close actions
- [ ] Notification `qa.answer_posted` → link to `/q/[slug]`
- [ ] Sitemap entries for public questions
- [ ] Distinct from reviews UI on product page

---

## 11. References

| Doc | Purpose |
|-----|---------|
| [nextjs-frontend-prompt.md](./nextjs-frontend-prompt.md) | Base stack, auth, design |
| [community-qa.md](./community-qa.md) | Backend feature overview |
| [endpoints.md](./endpoints.md) | Full route table |
| [sample-payloads.md](./sample-payloads.md#community-qa) | Example JSON |
| [mechanics.md](./mechanics.md) | Verified mechanic rules |
| [notifications.md](./notifications.md) | Bell + `qa.answer_posted` |

---

## 12. One-line prompt for an AI or developer

**“Implement Community Q&A in our Next.js 14 App Router + Tailwind storefront: public `/q` hub and SSR `/q/[slug]` pages with QAPage JSON-LD; product-page Q&A widget via `GET /products/:id/questions`; protected `/q/ask` form; verified mechanics answer via `POST /questions/:id/answers` with badge UI; authors accept/close answers; wire `qa.answer_posted` notifications to `/q/[slug]`. Use the API envelope and types in docs/nextjs-community-qa-prompt.md and docs/sample-payloads.md.”**
