# Next.js Frontend Prompt — Auto-Store (Minimalist, Modern)

Use this document as the **single source of truth** to build a **modern, minimalist, aesthetically pleasing** web frontend that consumes the Auto-Store backend API. The app should feel calm, focused, and intentional—not generic or cluttered.

---

## 1. Tech Stack & Setup

- **Framework:** Next.js 14+ with **App Router** (not Pages Router).
- **Styling:** **Tailwind CSS** only. No component libraries (no MUI, Chakra, etc.); build a custom, cohesive look with Tailwind utilities and a small set of reusable components you define.
- **Language:** TypeScript throughout.
- **API base URL:** Configurable via environment variable (e.g. `NEXT_PUBLIC_API_URL`). Default for local dev: `http://localhost:8080` (or the port your backend uses). API prefix: `/api/v1` (full base = `NEXT_PUBLIC_API_URL/api/v1`).
- **Data fetching:** Use **fetch** or a thin client (e.g. a single `api.ts` that wraps fetch with base URL, auth header, and response parsing). Prefer **Server Components** where possible for initial data; use **Client Components** and optional **React Query / SWR** for mutations and client-side refetch.

---

## 2. Design Direction (Minimalist & Modern)

- **Aesthetic:** Clean, minimal, plenty of whitespace. Avoid visual noise: no heavy gradients, no busy patterns, no redundant borders. Prefer subtle hierarchy (typography + spacing) over decoration.
- **Typography:** Choose **one clear, readable font** for body (e.g. **Inter**, **Geist**, **DM Sans**, or **Source Sans 3**) and optionally a **distinctive but restrained** font for headings (e.g. **Instrument Sans**, **Outfit**, or **Sora**). Avoid overused pairings (e.g. Inter + nothing else everywhere).
- **Color:** Use a **limited palette**. For example: a near-black or dark gray for text (`#0f172a` or similar), a light gray for secondary text, a very light background for cards/sections (`#f8fafc` or `slate-50`), and **one accent color** used sparingly (links, primary buttons, focus states). Prefer a single, intentional accent (e.g. a muted blue, teal, or indigo) rather than multiple competing colors.
- **Borders & shadows:** Prefer **thin borders** (`border border-slate-200`) or **very soft shadows** (`shadow-sm`) for cards and inputs. Avoid thick borders and heavy drop shadows.
- **Layout:** Generous padding and max-width containers (e.g. `max-w-6xl mx-auto`) so content breathes. Consistent spacing scale (Tailwind’s default or a custom scale).
- **Components:** Buttons, inputs, cards, and modals should share the same design language: same border radius (e.g. `rounded-lg`), same focus ring style, same transition for hover/active. Keep forms simple: labels above inputs, clear error state below the field.
- **Accessibility:** Sufficient contrast, focus-visible rings, semantic HTML (headings, landmarks, labels). No information conveyed by color alone.

---

## 3. Backend API Contract

### 3.1 Response envelope

All JSON responses follow this shape (unless noted, e.g. health):

**Success with data:**
```json
{ "success": true, "data": { ... } }
```

**Success with paginated list:**
```json
{
  "success": true,
  "data": [ ... ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

**Error:**
```json
{ "success": false, "error": "human-readable message" }
```

**Validation error:**
```json
{
  "success": false,
  "error": "validation failed",
  "errors": [
    { "field": "email", "message": "invalid email format" }
  ]
}
```

Auth middleware may return **401** with body like `{ "error": "missing or invalid authorization header" }` or `{ "error": "invalid or expired token" }`. Treat as “not authenticated”; attempt token refresh or redirect to login.

### 3.2 Authentication

- **Register:** `POST /api/v1/auth/register` — Body: `email`, `password` (required, min 8), `first_name`, `last_name`, `phone` (optional). Success 201: `data` = user object (no tokens); then user logs in.
- **Login:** `POST /api/v1/auth/login` — Body: `email`, `password`. Success 200: `data` has `access_token`, `refresh_token`, `expires_at`, `user`.
- **Refresh:** `POST /api/v1/auth/refresh` — Body: `refresh_token`. Success 200: same shape as login.
- **Logout:** `POST /api/v1/auth/logout` — Header: `Authorization: Bearer <access_token>`. Success 204.
- **Forgot password:** `POST /api/v1/auth/forgot-password` — Body: `email`.
- **Reset password:** `POST /api/v1/auth/reset-password` — Body: `token`, `new_password` (min 8).
- **Verify email:** `POST /api/v1/auth/verify-email` — Body: `token`.

**User object (from login/refresh/me):** `id`, `email`, `first_name`, `last_name`, `role` (ADMIN | VENDOR | CUSTOMER), `phone`, `email_verified`, `created_at`.

**Frontend auth:** Store `access_token` and `refresh_token` (e.g. httpOnly cookie via API route, or secure client storage if no backend proxy). Send `Authorization: Bearer <access_token>` on every request to protected endpoints. On 401: try refresh with `refresh_token`; if refresh succeeds, retry the request and persist new tokens; if refresh fails, clear tokens and redirect to login. Optionally use `expires_at` to refresh proactively.

---

## 4. API Endpoints Summary

Use this for building the client. All paths are relative to base `NEXT_PUBLIC_API_URL/api/v1`. Protected = require `Authorization: Bearer <access_token>`.

| Area | Method | Path | Auth | Notes |
|------|--------|------|------|--------|
| Health | GET | `/health` | No | Base URL without `/api/v1`. Returns `{ "status": "ok" }`. |
| Register | POST | `/auth/register` | No | |
| Login | POST | `/auth/login` | No | |
| Refresh | POST | `/auth/refresh` | No | |
| Logout | POST | `/auth/logout` | Yes | |
| Forgot pwd | POST | `/auth/forgot-password` | No | |
| Reset pwd | POST | `/auth/reset-password` | No | |
| Verify email | POST | `/auth/verify-email` | No | |
| Profile | GET / PUT | `/users/me` | Yes | |
| Addresses | GET / POST / PUT / DELETE | `/users/me/addresses` (/:id for PUT/DELETE) | Yes | |
| Products list | GET | `/products` | No | Query: page, limit |
| Products search | GET | `/products/search` | No | Query: q, category, tags, make, model, year, minPrice, maxPrice, condition, brand, sort, page, limit |
| Product by ID | GET | `/products/:id` | No | |
| Compatibility | GET | `/products/:id/compatibility` | No | |
| Product reviews | GET | `/products/:id/reviews` | No | Query: page, limit |
| Create review | POST | `/products/:id/reviews` | Yes | Body: rating (1–5), title?, comment? |
| Categories | GET | `/categories`, `/categories/:id`, `/categories/:id/products` | No | |
| Cart | GET | `/cart` | Yes | |
| Cart add | POST | `/cart/items` | Yes | Body: product_id, quantity |
| Cart update | PUT | `/cart/items/:id` | Yes | :id = cart item UUID. Body: quantity |
| Cart remove | DELETE | `/cart/items/:id` | Yes | |
| Cart clear | DELETE | `/cart` | Yes | |
| Orders | POST / GET | `/orders`, `/orders/:id` | Yes | Create body: shipping_address_id, billing_address_id, payment_method |
| Cancel order | PUT | `/orders/:id/cancel` | Yes | |
| Wishlist | GET / POST | `/wishlist` | Yes | POST body: product_id |
| Wishlist remove | DELETE | `/wishlist/:productId` | Yes | |
| Admin orders | GET / PUT | `/admin/orders`, `/admin/orders/:id/status` | Admin | |
| Admin user role | PUT | `/admin/users/:id/role` | Admin | Body: role (ADMIN/VENDOR/CUSTOMER) |
| Products (admin/vendor) | POST / PUT / etc. | `/products`, `/products/:id`, ... | Admin/Vendor | Create/update products, images, compatibility |
| Categories (admin) | POST / PUT / DELETE | `/categories`, `/categories/:id` | Admin | |

---

## 5. Data Models (TypeScript)

Define types that match the API (use `snake_case` in JSON; you can map to `camelCase` in TS if desired). Key entities:

- **User:** id, email, first_name, last_name, role, phone, email_verified, created_at
- **Product:** id, sku, name, description, brand, manufacturer_part_number, price, cost_price, stock_quantity, weight, dimensions, condition (new | refurbished | used), warranty_months, created_at, updated_at, categories?, tags?, images?, compatibilities?, reviews?
- **ProductImage:** id, product_id, url, alt_text, display_order, is_primary
- **Category:** id, parent_id, name, slug, description, level, parent?, children?, products?
- **Cart:** array of items with id, product_id, quantity, product?, etc. (exact shape from GET /cart)
- **Order:** id, order_number, status, subtotal, tax, shipping_cost, total, order_items?, shipping_address?, billing_address?, etc.
- **Address:** id, type (shipping | billing), street, city, state, postal_code, country, is_default
- **Review:** id, product_id, user_id, rating, title, comment, verified_purchase?, created_at, user?

Exact field names and nested structures: see backend `internal/models/` and `internal/handlers/dto/dto.go`, or `docs/sample-payloads.md`.

---

## 6. Pages & Features to Build

- **Public**
  - **Home:** Hero or simple value prop, featured or recent products (from GET /products), clear CTAs to browse and login/register.
  - **Products:** List (GET /products or /products/search) with optional filters (category, price range, condition, search query), pagination, clean product cards (image, name, price, condition).
  - **Product detail:** GET /products/:id — images, description, price, stock, compatibility (GET /products/:id/compatibility), reviews (GET /products/:id/reviews). If logged in: add to cart, add to wishlist, submit review form (POST /products/:id/reviews).
  - **Categories:** GET /categories — list categories; category detail with GET /categories/:id/products.
  - **Login / Register:** Forms with validation; show API validation errors (field-level from `errors[]`). After login, redirect and set auth state.
  - **Forgot password / Reset password / Verify email:** Simple forms and success messages (reset/verify may receive token via query param).

- **Protected (any logged-in user)**
  - **Profile:** GET /users/me, PUT /users/me — view and edit name, phone.
  - **Addresses:** List (GET), add (POST), edit (PUT), delete (DELETE) with type (shipping/billing) and default.
  - **Cart:** GET /cart — list items; add (POST /cart/items), update quantity (PUT /cart/items/:id), remove (DELETE), clear (DELETE /cart). Minimal, clear layout.
  - **Checkout:** Select shipping/billing addresses (from user addresses), payment method (e.g. text field for now), POST /orders. Redirect to order confirmation.
  - **Orders:** List (GET /orders) and detail (GET /orders/:id), cancel (PUT /orders/:id/cancel) when allowed.
  - **Wishlist:** GET /wishlist — list; add from product page; remove (DELETE /wishlist/:productId).

- **Optional (Admin / Vendor)**
  - **Admin:** List all orders, update order status; list users, update user role. Product and category CRUD (create/edit/delete products, categories) if you want a full dashboard.

Use **middleware or layout checks** to protect routes: redirect unauthenticated users to login for cart, profile, orders, wishlist, checkout.

---

## 7. Implementation Notes

- **API client:** Single module that builds full URL from `NEXT_PUBLIC_API_URL` + path, sets `Content-Type: application/json`, attaches `Authorization: Bearer <token>` when token exists, and parses response: check `success`; on success return `data` (and `meta` for paginated); on failure throw or return error message and `errors` for validation. Handle 401 with refresh logic as above.
- **Auth state:** React Context, Zustand, or similar to hold user + tokens (or “session”) and expose login/logout/refresh. Persist tokens (cookie or storage) and rehydrate on load.
- **Forms:** Controlled components, validate before submit; display backend `errors[].field` and `errors[].message` next to the corresponding inputs.
- **Loading & errors:** Use consistent loading UIs (skeleton or spinner) and error boundaries or inline error messages; avoid blank screens.
- **SEO:** Use Next.js metadata and semantic HTML for product and category pages.

---

## 8. References

- **Backend endpoints:** `docs/endpoints.md` in this repo.
- **Request/response examples:** `docs/sample-payloads.md`.
- **Backend DTOs/models:** `internal/handlers/dto/dto.go`, `internal/models/*.go`.
- **OpenAPI:** Backend serves Swagger at `/docs` (e.g. `http://localhost:8080/docs`).

---

## 9. One-line summary for an AI or developer

**“Build a modern, minimalist Next.js 14+ (App Router) + Tailwind CSS frontend for the Auto-Store API: clean typography, one accent color, generous whitespace, no component library. Implement auth (login/register/refresh), products (list/search/detail/reviews), categories, cart, checkout, orders, profile, addresses, wishlist. Use the response envelope (success/data/error/errors/meta), Bearer token with refresh on 401, and the endpoint list in docs/endpoints.md. Follow the design and API contract in docs/nextjs-frontend-prompt.md.”**

Use this prompt (and the referenced docs) so the frontend stays aligned with the backend and achieves a modern, minimalist, aesthetically pleasing UI.
