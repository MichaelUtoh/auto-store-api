# Auto-Store API — Endpoints

Base URL: `http://localhost:8089` (or `PORT` from env)  
API prefix: `/api/v1`  
Protected routes: send `Authorization: Bearer <access_token>`.

---

## General

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | No | Health check. Returns `{"status":"ok"}`. |
| GET | `/docs` | No | Redirects to Swagger UI. |
| GET | `/docs/*any` | No | Swagger UI / OpenAPI doc. |

---

## Authentication (`/api/v1/auth`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register` | No | Register a new user. |
| POST | `/api/v1/auth/login` | No | Login; returns access_token, refresh_token, user. |
| POST | `/api/v1/auth/forgot-password` | No | Request password reset email. |
| POST | `/api/v1/auth/reset-password` | No | Reset password with token. |
| POST | `/api/v1/auth/verify-email` | No | Verify email with token. |
| POST | `/api/v1/auth/refresh` | No | Refresh access token using refresh_token. |
| POST | `/api/v1/auth/logout` | Yes | Logout (invalidate session). |

---

## Products (`/api/v1/products`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/products` | No | List products (paginated; query: page, limit, category, search, **min**, **max** price; if both min and max are set, max must be **greater than** min). |
| GET | `/api/v1/products/search` | No | Search products (query: q, category, tags, make, model, year, minPrice, maxPrice, condition, brand, sort, page, limit). |
| GET | `/api/v1/products/:id` | No | Get product by ID. |
| GET | `/api/v1/products/:id/compatibility` | No | Get vehicle compatibility for product. |
| GET | `/api/v1/products/:id/reviews` | No | List reviews for product (paginated). |
| POST | `/api/v1/products/:id/reviews` | Yes | Create a review for product. |
| POST | `/api/v1/products` | Admin/Vendor | Create product. |
| POST | `/api/v1/products/batch` | Admin/Vendor | Create multiple products. |
| PUT | `/api/v1/products/:id` | Admin/Vendor | Update product. |
| POST | `/api/v1/products/:id/images` | Admin/Vendor | Add images to product. |
| DELETE | `/api/v1/products/:id/images/:imageId` | Admin/Vendor | Delete one image (`imageId` = product_images row UUID). |
| POST | `/api/v1/products/:id/compatibility` | Admin/Vendor | Add vehicle compatibilities. |
| DELETE | `/api/v1/products/:id` | Admin | Delete product. |

---

## Categories (`/api/v1/categories`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/categories` | No | List all categories. |
| GET | `/api/v1/categories/:id` | No | Get category by ID. |
| GET | `/api/v1/categories/:id/products` | No | List products in category (paginated). |
| POST | `/api/v1/categories` | Admin | Create category. |
| PUT | `/api/v1/categories/:id` | Admin | Update category. |
| DELETE | `/api/v1/categories/:id` | Admin | Delete category. |

---

## Cart (`/api/v1/cart`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/cart` | Yes | Get current user's cart. |
| POST | `/api/v1/cart/items` | Yes | Add item to cart (body: product_id, quantity). |
| PUT | `/api/v1/cart/items/:id` | Yes | Update cart item quantity (:id = cart item UUID). |
| DELETE | `/api/v1/cart/items/:id` | Yes | Remove item from cart. |
| DELETE | `/api/v1/cart` | Yes | Clear entire cart. |

---

## Orders (`/api/v1/orders`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/orders` | Yes | Create order (body: shipping_address_id, billing_address_id, payment_method). |
| GET | `/api/v1/orders` | Yes | List current user's orders (paginated). |
| GET | `/api/v1/orders/:id` | Yes | Get order by ID. |
| PUT | `/api/v1/orders/:id/cancel` | Yes | Cancel order. |

---

## User profile & addresses (`/api/v1/users`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/users/me` | Yes | Get current user profile. |
| PUT / PATCH | `/api/v1/users/me` | Yes | Update profile (`first_name` / `firstName`, `last_name` / `lastName`, `phone`). |
| GET | `/api/v1/users/me/addresses` | Yes | List user's addresses. |
| POST | `/api/v1/users/me/addresses` | Yes | Add address. |
| PUT | `/api/v1/users/me/addresses/:id` | Yes | Update address. |
| DELETE | `/api/v1/users/me/addresses/:id` | Yes | Delete address. |

---

## Mechanics (`/api/v1/mechanics`, `/api/v1/mechanic`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/mechanics` | No | List verified mechanics (query: page, limit). |
| GET | `/api/v1/mechanics/:id` | No | Get verified mechanic public profile by profile ID. |
| POST | `/api/v1/mechanic/apply` | Yes | Submit mechanic application (creates profile with status `pending`). |
| GET | `/api/v1/mechanic/profile` | Yes | Get current user's mechanic profile (includes documents). |
| PUT | `/api/v1/mechanic/profile` | Yes | Update own profile (allowed when status is `pending` or `verified`). |
| POST | `/api/v1/mechanic/documents` | Yes | Add verification document (body: document_type, url, file_name). |
| DELETE | `/api/v1/mechanic/documents/:id` | Yes | Remove a document from own profile. |

**Document types:** `license`, `insurance`, `certification`, `other`.

**Profile statuses:** `pending`, `verified`, `suspended`, `rejected`.

**Verified mechanic only** (installation marketplace):

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/mechanic/installation/quotes` | Quote lines for this mechanic |
| PATCH | `/api/v1/mechanic/installation/quotes/:id` | Update labor estimate |
| PUT | `/api/v1/mechanic/installation/services` | Job types offered |
| GET | `/api/v1/mechanic/installation/bookings` | Bookings |
| PATCH | `/api/v1/mechanic/installation/bookings/:id/status` | Status lifecycle |

See [installation-marketplace.md](./installation-marketplace.md).

---

## Installation marketplace (`/api/v1/installation`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/installation/job-types` | No | List install job catalog |
| POST | `/api/v1/installation/quotes` | Yes | Request installation quotes |
| GET | `/api/v1/installation/quotes` | Yes | List own quotes |
| GET | `/api/v1/installation/quotes/:id` | Yes | Quote with mechanic lines |
| POST | `/api/v1/installation/bookings` | Yes | Book selected line + time |
| GET | `/api/v1/installation/bookings` | Yes | List own bookings |
| GET | `/api/v1/installation/bookings/:id` | Yes | Booking detail |
| PATCH | `/api/v1/installation/bookings/:id/cancel` | Yes | Cancel booking |

---

## Notifications (`/api/v1/notifications`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/notifications` | Yes | List in-app notifications (query: page, limit, unread_only). |
| GET | `/api/v1/notifications/unread-count` | Yes | Unread in-app count for bell badge. |
| PATCH | `/api/v1/notifications/:id/read` | Yes | Mark one notification read. |
| PATCH | `/api/v1/notifications/read-all` | Yes | Mark all in-app notifications read. |
| GET | `/api/v1/users/me/notification-preferences` | Yes | Get channel preferences. |
| PUT | `/api/v1/users/me/notification-preferences` | Yes | Update preferences. |

Email delivery is async via Redis queue + `cmd/worker`. See [notifications.md](./notifications.md).

---

## Visual Part Finder (`/api/v1/diagrams`, `/api/v1/part-identification`)

See [part-finder.md](./part-finder.md).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/vehicle-systems` | No | List vehicle systems (brakes, suspension, …) |
| GET | `/api/v1/diagrams` | No | List diagrams (`make`, `model`, `year`, `system`, `page`, `limit`) |
| GET | `/api/v1/diagrams/:id` | No | Diagram detail (`?include_hotspots=true`) |
| GET | `/api/v1/diagrams/:id/hotspots` | No | Hotspots for diagram |
| GET | `/api/v1/diagrams/:id/hotspots/:hotspotId/products` | No | Products for hotspot (`?year=` optional) |
| POST | `/api/v1/part-identification` | Yes | AR/CV identify parts (multipart image + vehicle) |
| POST | `/api/v1/diagrams` | Admin/Vendor | Create diagram |
| PUT | `/api/v1/diagrams/:id` | Admin/Vendor | Update diagram |
| POST | `/api/v1/diagrams/:id/hotspots` | Admin/Vendor | Add hotspot |
| PUT | `/api/v1/diagrams/:id/hotspots/:hotspotId` | Admin/Vendor | Update hotspot |
| POST | `/api/v1/diagrams/:id/hotspots/:hotspotId/products` | Admin/Vendor | Link product to hotspot |
| DELETE | `/api/v1/diagrams/:id/hotspots/:hotspotId/products/:productId` | Admin/Vendor | Unlink product |
| DELETE | `/api/v1/diagrams/:id` | Admin | Delete diagram |
| DELETE | `/api/v1/diagrams/:id/hotspots/:hotspotId` | Admin | Delete hotspot |

---

## Community Q&A (`/api/v1/questions`)

See [community-qa.md](./community-qa.md).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/questions` | No | List questions (query: `q`, `product_id`, `category_id`, `make`, `model`, `year`, `status`, `page`, `limit`) |
| GET | `/api/v1/questions/:slug` | No | Question detail + answers |
| GET | `/api/v1/products/:id/questions` | No | Questions for a product |
| POST | `/api/v1/questions` | Yes | Ask a question |
| POST | `/api/v1/questions/:id/answers` | Verified mechanic | Post an answer |
| PATCH | `/api/v1/questions/:id/accept-answer/:answerId` | Yes | Accept answer (author) |
| PATCH | `/api/v1/questions/:id/close` | Yes / Admin | Close thread |

---

## Wishlist (`/api/v1/wishlist`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/wishlist` | Yes | Get user's wishlist. |
| POST | `/api/v1/wishlist` | Yes | Add product to wishlist (body: product_id). |
| DELETE | `/api/v1/wishlist/:productId` | Yes | Remove product from wishlist. |

---

## Admin (`/api/v1/admin`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/orders` | Admin | List all orders (query: page, limit, status). |
| PUT | `/api/v1/admin/orders/:id/status` | Admin | Update order status (body: status). |
| PUT | `/api/v1/admin/users/:id/role` | Admin | Update user role (body: role — ADMIN, VENDOR, CUSTOMER, MECHANIC). |
| GET | `/api/v1/admin/mechanics` | Admin | List mechanic profiles (query: status, page, limit). |
| PUT | `/api/v1/admin/mechanics/:userId/verify` | Admin | Verify mechanic; sets role to MECHANIC. |
| PUT | `/api/v1/admin/mechanics/:userId/suspend` | Admin | Suspend mechanic (optional body: reason). |
| PUT | `/api/v1/admin/mechanics/:userId/reject` | Admin | Reject application (body: reason required). |

---

## Summary

- **Public:** Health, docs, auth (except logout), products list/search/get/compatibility/reviews/questions GET, categories list/get/products, verified mechanics list/get, installation job types, community Q&A list/detail, vehicle systems, diagrams list/detail/hotspots/products.
- **Authenticated (any role):** Logout, cart, orders (create/list/get/cancel), profile, addresses, wishlist, notifications, notification preferences, create product review, ask/accept/close Q&A questions, part identification, mechanic apply/profile/documents, installation quotes/bookings.
- **Admin or Vendor:** Products create/batch/update, product images, product compatibility, diagrams and hotspots CRUD, hotspot product links.
- **Admin only:** Product delete, categories CRUD, admin orders list/status, admin user role update, mechanic verification workflow.
- **Mechanic (role MECHANIC, verified profile):** Installation quote responses, bookings, service catalog, Q&A answers.
