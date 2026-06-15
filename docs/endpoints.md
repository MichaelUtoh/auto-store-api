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
| GET | `/api/v1/products` | No | List products (paginated; query: page, limit, category, search, **min**, **max** price, **sort**; if both min and max are set, max must be **greater than** min). |
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

Public product responses include `in_stock` (bool) and `stock_status` (`in_stock`, `low_stock`, or `out_of_stock`). Out-of-stock products remain visible but cannot be added to cart.

---

## Inventory (`/api/v1/admin/inventory`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/admin/inventory/low-stock` | Admin/Vendor | List products at or below their low-stock threshold. Vendors see only their products. |
| GET | `/api/v1/admin/inventory/products/:id/movements` | Admin/Vendor | Paginated stock movement history for a product. |
| PATCH | `/api/v1/admin/inventory/products/:id/stock` | Admin/Vendor | Adjust stock (body: `delta`, optional `reason`, `notes`). |
| PUT | `/api/v1/admin/inventory/products/:id/settings` | Admin/Vendor | Set `low_stock_threshold` for a product. |
| POST | `/api/v1/admin/inventory/bulk-threshold` | Admin | Bulk-set threshold for all products, a category, or specific product IDs. |

Low-stock alerts notify all admins and the product vendor (if assigned). Alerts fire once per threshold crossing until stock is restocked above the threshold.

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
| POST | `/api/v1/orders/:id/pay` | Yes | Initialize Paystack checkout (see [payments.md](./payments.md)). |
| POST | `/api/v1/orders/:id/refund` | Yes | Refund paid Paystack order. |
| PUT | `/api/v1/orders/:id/cancel` | Yes | Cancel order. |

---

## Payments

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/payments/verify` | Yes | Verify Paystack transaction (`?reference=`). |
| GET | `/api/v1/payments/banks` | Yes | List banks for mechanic payout setup. |
| POST | `/webhooks/paystack` | No | Paystack webhook (`x-paystack-signature`). |

See [payments.md](./payments.md).

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
| GET | `/api/v1/mechanic/payout` | Yes | Paystack subaccount / payout status. |
| POST | `/api/v1/mechanic/payout` | Yes | Register or update bank account for split payouts. |
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
| POST | `/api/v1/installation/bookings/:id/pay` | Yes | Initialize Paystack checkout |
| POST | `/api/v1/installation/bookings/:id/refund` | Yes | Refund paid Paystack booking |
| PATCH | `/api/v1/installation/bookings/:id/cancel` | Yes | Cancel booking (auto-refunds if paid via Paystack) |

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
| POST | `/api/v1/admin/orders/:id/refund` | Admin | Refund paid Paystack order. |
| POST | `/api/v1/admin/installation/bookings/:id/refund` | Admin | Refund paid Paystack booking. |
| PUT | `/api/v1/admin/users/:id/role` | Admin | Update user role (body: role — ADMIN, VENDOR, CUSTOMER, MECHANIC). |
| GET | `/api/v1/admin/mechanics` | Admin | List mechanic profiles (query: status, page, limit). |
| PUT | `/api/v1/admin/mechanics/:userId/verify` | Admin | Verify mechanic; sets role to MECHANIC. |
| PUT | `/api/v1/admin/mechanics/:userId/suspend` | Admin | Suspend mechanic (optional body: reason). |
| PUT | `/api/v1/admin/mechanics/:userId/reject` | Admin | Reject application (body: reason required). |

---

## Support chat

Real-time customer/guest ↔ admin chat. Full spec: [support-chat.md](./support-chat.md).

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/chat/guest-session` | No | Issue guest chat token (rate limited). |
| POST | `/api/v1/chat/guest-session/refresh` | Guest | Refresh guest token. |
| GET | `/api/v1/conversations/me` | User or guest | Current open conversation. |
| POST | `/api/v1/conversations` | User or guest | Get-or-create open conversation. |
| GET | `/api/v1/conversations/:id` | Owner or admin | Conversation detail. |
| GET | `/api/v1/conversations/:id/messages` | Owner or admin | Message history. |
| POST | `/api/v1/conversations/:id/messages` | Owner or admin | Send message (REST fallback). |
| PATCH | `/api/v1/conversations/:id` | Owner or admin | Close; guest email/name. |
| PATCH | `/api/v1/conversations/:id/read` | Owner or admin | Update read cursor. |
| POST | `/api/v1/conversations/link-guest` | User | Merge guest threads on login. |
| GET | `/api/v1/admin/conversations` | Admin | Support inbox. |
| GET | `/api/v1/admin/conversations/unread-count` | Admin | Inbox badge count. |
| GET | `/api/v1/ws/chat` | User, guest, or admin | WebSocket upgrade (`token` query). |

---

## Summary

- **Public:** Health, docs, auth (except logout), products list/search/get/compatibility/reviews/questions GET, categories list/get/products, verified mechanics list/get, installation job types, community Q&A list/detail, guest chat session (planned).
- **Authenticated (any role):** Logout, cart, orders (create/list/get/cancel), profile, addresses, wishlist, notifications, notification preferences, create product review, ask/accept/close Q&A questions, mechanic apply/profile/documents, installation quotes/bookings, link guest chat on login (`POST /conversations/link-guest`).
- **Flexible auth (user JWT or guest token):** Support chat conversations and messages, WebSocket `/ws/chat`.
- **Admin or Vendor:** Products create/batch/update, product images, product compatibility.
- **Admin only:** Product delete, categories CRUD, admin orders list/status, admin user role update, mechanic verification workflow, support chat inbox.
- **Mechanic (role MECHANIC, verified profile):** Installation quote responses, bookings, service catalog, Q&A answers.
