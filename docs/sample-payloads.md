# API Sample Payloads

Base URL: `http://localhost:8089/api/v1`

Use `Authorization: Bearer <access_token>` for protected endpoints.

---

## Authentication

### POST /auth/register

**Request body:**
```json
{
  "email": "customer@example.com",
  "password": "SecurePass123",
  "first_name": "John",
  "last_name": "Doe",
  "phone": "+1 555-123-4567"
}
```

Minimal (phone optional):
```json
{
  "email": "customer@example.com",
  "password": "SecurePass123"
}
```

---

### POST /auth/login

**Request body:**
```json
{
  "email": "customer@example.com",
  "password": "SecurePass123"
}
```

---

### POST /auth/refresh

**Request body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

---

### POST /auth/forgot-password

**Request body:**
```json
{
  "email": "customer@example.com"
}
```

---

### POST /auth/reset-password

**Request body:**
```json
{
  "token": "reset-token-from-email",
  "new_password": "NewSecurePass456"
}
```

---

### POST /auth/verify-email

**Request body:**
```json
{
  "token": "verification-token-from-email"
}
```

---

### POST /auth/logout

**Headers:** `Authorization: Bearer <access_token>`

No body.

---

## Products

### GET /products

**Query (optional):**
```
?page=1&limit=20
```

---

### GET /products/search

**Query (all optional):**
```
?q=brake+pads
&category=brakes
&tags=ceramic,performance
&make=toyota
&model=camry
&year=2015-2020
&minPrice=50
&maxPrice=200
&condition=new
&brand=ACDelco
&sort=price_asc
&page=1
&limit=20
```

Example URL:
```
GET /products/search?q=brake%20pads&category=brakes&minPrice=50&maxPrice=200&condition=new&sort=price_asc&page=1&limit=20
```

---

### GET /products/:id

**Path:** `id` = product UUID

Example: `GET /products/550e8400-e29b-41d4-a716-446655440001`

---

### GET /products/:id/compatibility

**Path:** `id` = product UUID

Example: `GET /products/550e8400-e29b-41d4-a716-446655440001/compatibility`

---

### POST /products (Admin/Vendor)

**Headers:** `Authorization: Bearer <access_token>`

**Request body:**
```json
{
  "sku": "BP-001-CER",
  "name": "Ceramic Brake Pads Set",
  "description": "High-performance ceramic brake pads for reduced dust and noise.",
  "brand": "ACDelco",
  "manufacturer_part_number": "18A1234",
  "price": 89.99,
  "cost_price": 45.00,
  "stock_quantity": 100,
  "weight": 2.5,
  "dimensions": "8x4x2 in",
  "condition": "new",
  "warranty_months": 24,
  "category_ids": ["550e8400-e29b-41d4-a716-446655440010"],
  "tag_ids": ["550e8400-e29b-41d4-a716-446655440020"]
}
```

Minimal:
```json
{
  "sku": "BP-002",
  "name": "Basic Brake Pads",
  "price": 49.99,
  "stock_quantity": 50,
  "category_ids": [],
  "tag_ids": []
}
```

`condition` must be one of: `new`, `refurbished`, `used`.

---

### PUT /products/:id (Admin/Vendor)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = product UUID

**Request body (all fields optional):**
```json
{
  "name": "Ceramic Brake Pads Set (Updated)",
  "description": "Updated description.",
  "brand": "ACDelco",
  "manufacturer_part_number": "18A1234",
  "price": 79.99,
  "cost_price": 40.00,
  "stock_quantity": 80,
  "weight": 2.5,
  "dimensions": "8x4x2 in",
  "condition": "new",
  "warranty_months": 24,
  "category_ids": ["550e8400-e29b-41d4-a716-446655440010"],
  "tag_ids": ["550e8400-e29b-41d4-a716-446655440020"]
}
```

**Optional `images`:** If the JSON includes an `images` key, it **replaces** all existing product images (same shape as POST `/products/:id/images`). Omit `images` entirely to leave images unchanged. Use `"images": []` to remove all images.

```json
{
  "name": "Updated name only",
  "images": [
    {
      "url": "https://cdn.example.com/p1.jpg",
      "alt_text": "Main",
      "display_order": 0,
      "is_primary": true
    }
  ]
}
```

---

### POST /products/:id/images (Admin/Vendor)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = product UUID

**Request body:** JSON with an array of image objects. Each image is referenced by **URL** (the API does not accept file uploads; upload files to S3/Cloudinary/etc. first, then pass the resulting URLs here).

```json
{
  "images": [
    {
      "url": "https://your-bucket.s3.region.amazonaws.com/products/bp-001-main.jpg",
      "alt_text": "Ceramic brake pads front view",
      "display_order": 0,
      "is_primary": true
    },
    {
      "url": "https://your-bucket.s3.region.amazonaws.com/products/bp-001-side.jpg",
      "alt_text": "Side view",
      "display_order": 1,
      "is_primary": false
    }
  ]
}
```

- `url` (required): valid URL (e.g. from your storage bucket).
- `alt_text` (optional): max 255 chars.
- `display_order` (optional): integer; images are ordered by this, then by created_at.
- `is_primary` (optional): if `true`, this image is the main one; other images for the product are set to non-primary.

**Success (201):** `data` is the array of created `ProductImage` objects (id, product_id, url, alt_text, display_order, is_primary, created_at, updated_at).

---

### DELETE /products/:id/images/:imageId (Admin/Vendor)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = product UUID, `imageId` = **product image** UUID (the `id` field on the `ProductImage` row from GET product or POST images response — not the S3 URL).

**Example:** `DELETE /products/550e8400-e29b-41d4-a716-446655440001/images/660e8400-e29b-41d4-a716-446655440099`

No body.

**Success:** 204 No Content.

**404:** Image does not exist or does not belong to this product.

---

### DELETE /products/:id (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = product UUID

Example: `DELETE /products/550e8400-e29b-41d4-a716-446655440001`

No body.

---

## Categories

### GET /categories

No query or body.

---

### GET /categories/:id

**Path:** `id` = category UUID

Example: `GET /categories/550e8400-e29b-41d4-a716-446655440010`

---

### GET /categories/:id/products

**Path:** `id` = category UUID

**Query (optional):**
```
?page=1&limit=20
```

---

### POST /categories (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Request body:**
```json
{
  "parent_id": null,
  "name": "Brakes",
  "slug": "brakes",
  "description": "Brake pads, rotors, and related parts."
}
```

With parent (subcategory):
```json
{
  "parent_id": "550e8400-e29b-41d4-a716-446655440010",
  "name": "Brake Pads",
  "slug": "brake-pads",
  "description": "Brake pad sets."
}
```

---

### PUT /categories/:id (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = category UUID

**Request body (all fields optional):**
```json
{
  "name": "Brakes & Rotors",
  "slug": "brakes-rotors",
  "description": "Updated description.",
  "parent_id": null
}
```

---

### DELETE /categories/:id (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = category UUID

No body.

---

## Cart

### GET /cart

**Headers:** `Authorization: Bearer <access_token>`

No body.

---

### POST /cart/items

**Headers:** `Authorization: Bearer <access_token>`

**Request body:**
```json
{
  "product_id": "550e8400-e29b-41d4-a716-446655440001",
  "quantity": 2
}
```

---

### PUT /cart/items/:id

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = cart item UUID (not product ID)

**Request body:**
```json
{
  "quantity": 3
}
```

---

### DELETE /cart/items/:id

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = cart item UUID

No body.

---

### DELETE /cart

**Headers:** `Authorization: Bearer <access_token>`

Clears entire cart. No body.

---

## Orders

### POST /orders

**Headers:** `Authorization: Bearer <access_token>`

**Request body:**
```json
{
  "shipping_address_id": "550e8400-e29b-41d4-a716-446655440030",
  "billing_address_id": "550e8400-e29b-41d4-a716-446655440031",
  "payment_method": "credit_card"
}
```

---

### GET /orders

**Headers:** `Authorization: Bearer <access_token>`

**Query (optional):**
```
?page=1&limit=20
```

---

### GET /orders/:id

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = order UUID

Example: `GET /orders/550e8400-e29b-41d4-a716-446655440040`

---

### PUT /orders/:id/cancel

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = order UUID

No body.

---

### GET /admin/orders (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Query (optional):**
```
?page=1&limit=20&status=pending
```

`status` optional: filter by order status.

---

### PUT /admin/orders/:id/status (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = order UUID

**Request body:**
```json
{
  "status": "confirmed"
}
```

`status` must be one of: `pending`, `confirmed`, `processing`, `shipped`, `delivered`, `cancelled`.

---

### PUT /admin/users/:id/role (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = user UUID

**Request body:**
```json
{
  "role": "VENDOR"
}
```

`role` must be one of: `ADMIN`, `VENDOR`, `CUSTOMER`, `MECHANIC` (stored in caps).

---

## User Profile

### GET /users/me

**Headers:** `Authorization: Bearer <access_token>`

No body.

**Response (when user has a mechanic application):** `data` may include `mechanic_profile`:

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "mechanic@example.com",
    "first_name": "Alex",
    "last_name": "Rivera",
    "role": "MECHANIC",
    "phone": "+1 555-222-3333",
    "email_verified": true,
    "created_at": "2026-05-01T12:00:00Z",
    "mechanic_profile": {
      "id": "660e8400-e29b-41d4-a716-446655440010",
      "status": "verified",
      "business_name": "Bay Area Brakes",
      "is_verified": true
    }
  }
}
```

`mechanic_profile` is omitted when the user has not applied. After apply (before admin verify), `role` may still be `CUSTOMER` with `status`: `pending` and `is_verified`: `false`.

---

### PUT /users/me

**Headers:** `Authorization: Bearer <access_token>`

**Request body (all fields optional):**
```json
{
  "first_name": "Jane",
  "last_name": "Doe",
  "phone": "+1 555-987-6543"
}
```

---

## Addresses

### GET /users/me/addresses

**Headers:** `Authorization: Bearer <access_token>`

No body.

---

### POST /users/me/addresses

**Headers:** `Authorization: Bearer <access_token>`

**Request body:**
```json
{
  "type": "shipping",
  "street": "123 Main St",
  "city": "Detroit",
  "state": "MI",
  "postal_code": "48201",
  "country": "USA",
  "is_default": true
}
```

`type` must be `shipping` or `billing`.

---

### PUT /users/me/addresses/:id

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = address UUID

**Request body (all fields optional):**
```json
{
  "street": "456 Oak Ave",
  "city": "Detroit",
  "state": "MI",
  "postal_code": "48202",
  "country": "USA",
  "is_default": false
}
```

---

### DELETE /users/me/addresses/:id

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = address UUID

No body.

---

## Wishlist

### GET /wishlist

**Headers:** `Authorization: Bearer <access_token>`

No body.

---

### POST /wishlist

**Headers:** `Authorization: Bearer <access_token>`

**Request body:**
```json
{
  "product_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

---

### DELETE /wishlist/:productId

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `productId` = product UUID

Example: `DELETE /wishlist/550e8400-e29b-41d4-a716-446655440001`

No body.

---

## Mechanics

Mechanic identity supports a verification workflow: users **apply**, admins **verify / suspend / reject**, and only **verified** profiles appear on public listings.

See also: [mechanics.md](./mechanics.md) for roles, statuses, and flow overview.

### Roles and profile status

| User `role` | Set when |
|-------------|----------|
| `CUSTOMER` | Default registration; restored on **reject** if user was `MECHANIC` |
| `MECHANIC` | Admin **verify**, or admin assigns via `PUT /admin/users/:id/role` |

| Profile `status` | Meaning |
|------------------|---------|
| `pending` | Application submitted; awaiting admin review |
| `verified` | Approved; listed publicly; user role set to `MECHANIC` |
| `suspended` | Temporarily blocked (optional `reason` stored server-side) |
| `rejected` | Application denied (`reason` required on reject) |

**Document types** (for verification uploads): `license`, `insurance`, `certification`, `other`.

Upload files first via `POST /upload/images` (Admin/Vendor) or your S3 flow, then pass the returned URL in document payloads.

---

### GET /mechanics (public)

List verified mechanics only.

**Query (optional):**
```
?page=1&limit=20
```

**Response (paginated):**
```json
{
  "success": true,
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440010",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "business_name": "Bay Area Brakes",
      "bio": "ASE-certified brake and suspension work.",
      "phone": "+1 555-222-3333",
      "street": "100 Industrial Way",
      "city": "San Jose",
      "state": "CA",
      "postal_code": "95112",
      "country": "US",
      "latitude": 37.3382,
      "longitude": -121.8863,
      "service_radius_km": 40,
      "status": "verified",
      "rating_avg": 0,
      "rating_count": 0,
      "verified_at": "2026-05-10T14:00:00Z",
      "created_at": "2026-05-08T10:00:00Z",
      "updated_at": "2026-05-10T14:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

Public responses do **not** include `documents`.

---

### GET /mechanics/:id (public)

**Path:** `id` = mechanic **profile** UUID (not user UUID).

Returns 404 if profile is not `verified`.

Same shape as a single item in the list above.

---

### POST /mechanic/apply

**Headers:** `Authorization: Bearer <access_token>`

Submit a mechanic application. One profile per user. Does **not** change `role` until admin verifies.

**Request body:**
```json
{
  "business_name": "Bay Area Brakes",
  "bio": "ASE-certified, 10+ years. Brakes, suspension, steering.",
  "phone": "+1 555-222-3333",
  "street": "100 Industrial Way",
  "city": "San Jose",
  "state": "CA",
  "postal_code": "95112",
  "country": "US",
  "latitude": 37.3382,
  "longitude": -121.8863,
  "service_radius_km": 40,
  "documents": [
    {
      "document_type": "license",
      "url": "https://cdn.example.com/uploads/ase-cert.pdf",
      "file_name": "ase-cert.pdf"
    },
    {
      "document_type": "insurance",
      "url": "https://cdn.example.com/uploads/liability.pdf",
      "file_name": "liability-insurance.pdf"
    }
  ]
}
```

| Field | Required | Notes |
|-------|----------|-------|
| `business_name` | Yes | Max 200 chars |
| `city`, `state`, `postal_code` | Yes | |
| `bio`, `street`, `phone`, `country` | No | `country` defaults to `US` if omitted |
| `latitude`, `longitude` | No | For future geo matching |
| `service_radius_km` | No | 1–500; defaults to 25 |
| `documents` | No | Each needs `document_type`, `url` (valid URL) |

**Response:** `201` with full profile in `data` (includes `documents`, `status`: `pending`).

**Errors:** `409` if profile already exists for this user.

---

### GET /mechanic/profile

**Headers:** `Authorization: Bearer <access_token>`

Returns the authenticated user's mechanic profile including documents.

**Response:** `200` — same profile shape as apply response.

**Errors:** `404` if user has not applied.

---

### PUT /mechanic/profile

**Headers:** `Authorization: Bearer <access_token>`

Update own profile. Allowed only when `status` is `pending` or `verified`.

**Request body (all fields optional):**
```json
{
  "business_name": "Bay Area Brakes & Suspension",
  "bio": "Updated description.",
  "phone": "+1 555-222-4444",
  "street": "200 Industrial Way",
  "city": "San Jose",
  "state": "CA",
  "postal_code": "95113",
  "country": "US",
  "latitude": 37.34,
  "longitude": -121.89,
  "service_radius_km": 50
}
```

**Errors:** `400` if status is `suspended` or `rejected`.

---

### POST /mechanic/documents

**Headers:** `Authorization: Bearer <access_token>`

Add a verification document to an existing profile.

**Request body:**
```json
{
  "document_type": "certification",
  "url": "https://cdn.example.com/uploads/ase-master.pdf",
  "file_name": "ase-master.pdf"
}
```

**Response:** `201` — document object in `data`:
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440020",
    "document_type": "certification",
    "url": "https://cdn.example.com/uploads/ase-master.pdf",
    "file_name": "ase-master.pdf",
    "status": "pending",
    "created_at": "2026-05-08T11:00:00Z"
  }
}
```

---

### DELETE /mechanic/documents/:id

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = mechanic document UUID

No body. **Response:** `204`.

---

### GET /admin/mechanics (Admin)

**Headers:** `Authorization: Bearer <access_token>`

List mechanic profiles for admin review.

**Query (optional):**
```
?status=pending&page=1&limit=20
```

`status`: `pending`, `verified`, `suspended`, or `rejected`.

**Response:** Paginated list; each item includes `documents` (unlike public list).

---

### PUT /admin/mechanics/:userId/verify (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `userId` = applicant's **user** UUID (not profile UUID)

No body required.

**Effect:** Sets profile `status` to `verified`, sets `verified_at`, assigns user `role` to `MECHANIC`.

**Response:** `200` — updated profile in `data`.

---

### PUT /admin/mechanics/:userId/suspend (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `userId` = user UUID

**Request body (optional):**
```json
{
  "reason": "Insurance document expired."
}
```

**Effect:** Sets profile `status` to `suspended`. User keeps `MECHANIC` role.

---

### PUT /admin/mechanics/:userId/reject (Admin)

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `userId` = user UUID

**Request body:**
```json
{
  "reason": "Unable to verify business license."
}
```

`reason` is **required**.

**Effect:** Sets profile `status` to `rejected`. If user `role` was `MECHANIC`, reverts to `CUSTOMER`.

---

## Installation marketplace

See [installation-marketplace.md](./installation-marketplace.md). Products must have `installation_eligible: true` and `installation_job_type_id` set.

### POST /installation/quotes

**Headers:** `Authorization: Bearer <access_token>`

```json
{
  "product_ids": ["550e8400-e29b-41d4-a716-446655440010"],
  "vehicle_make": "Toyota",
  "vehicle_model": "Camry",
  "vehicle_year": 2018,
  "service_street": "123 Main St",
  "service_city": "San Jose",
  "service_state": "CA",
  "service_postal_code": "95112",
  "latitude": 37.3382,
  "longitude": -121.8863,
  "notes": "Front brake pads from recent order",
  "search_radius_km": 50
}
```

Alternatively pass `"order_id": "<uuid>"` instead of `product_ids`.

**Response (201):** quote with `lines[]` (per-mechanic labor estimates, `distance_km`, expires in 24h).

### POST /installation/bookings

```json
{
  "quote_id": "660e8400-e29b-41d4-a716-446655440100",
  "quote_line_id": "770e8400-e29b-41d4-a716-446655440101",
  "scheduled_at": "2026-05-25T14:00:00Z"
}
```

**Response (201):** booking with `status`: `pending_payment`, totals (`labor_total`, `parts_total`, `platform_fee`, `total_amount`). Paystack checkout is not wired yet — payment stays `pending` until integrated.

### PATCH /mechanic/installation/bookings/:id/status (verified mechanic)

```json
{ "status": "en_route" }
```

Valid: `confirmed`, `en_route`, `in_progress`, `completed`, `cancelled`.

---

## Notifications

In-app feed for the Next.js bell + async email via worker. See [notifications.md](./notifications.md).

### GET /notifications

**Headers:** `Authorization: Bearer <access_token>`

**Query (optional):**
```
?page=1&limit=20&unread_only=true
```

Returns **in-app** channel only.

**Response (paginated):**
```json
{
  "success": true,
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440030",
      "type": "mechanic.verified",
      "channel": "in_app",
      "title": "Mechanic profile verified",
      "body": "Your mechanic profile for Bay Area Brakes is now verified...",
      "payload": {
        "href": "/mechanic/profile"
      },
      "read_at": null,
      "created_at": "2026-05-21T10:00:00Z"
    }
  ],
  "meta": { "page": 1, "limit": 20, "total": 1, "total_pages": 1 }
}
```

Use `payload.href` with Next.js `<Link href={...}>`.

---

### GET /notifications/unread-count

**Headers:** `Authorization: Bearer <access_token>`

**Response:**
```json
{
  "success": true,
  "data": { "count": 3 }
}
```

---

### PATCH /notifications/:id/read

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = notification UUID

**Response:** `204`

---

### PATCH /notifications/read-all

**Headers:** `Authorization: Bearer <access_token>`

**Response:** `204`

---

### GET /users/me/notification-preferences

**Headers:** `Authorization: Bearer <access_token>`

**Response:**
```json
{
  "success": true,
  "data": {
    "email_enabled": true,
    "sms_enabled": false,
    "push_enabled": false,
    "in_app_enabled": true
  }
}
```

---

### PUT /users/me/notification-preferences

**Headers:** `Authorization: Bearer <access_token>`

**Request body (all optional):**
```json
{
  "email_enabled": true,
  "sms_enabled": false,
  "push_enabled": false,
  "in_app_enabled": true
}
```

---

## Reviews

### GET /products/:id/reviews

**Path:** `id` = product UUID

**Query (optional):**
```
?page=1&limit=20
```

---

### POST /products/:id/reviews

**Headers:** `Authorization: Bearer <access_token>`

**Path:** `id` = product UUID

**Request body:**
```json
{
  "rating": 5,
  "title": "Great brake pads",
  "comment": "Quiet and effective. Fits my 2018 Camry perfectly."
}
```

`rating` required, 1–5. `title` and `comment` optional.

---

## Community Q&A

See [community-qa.md](./community-qa.md).

### POST /questions

**Headers:** `Authorization: Bearer <access_token>`

**Request body (link to product, category, or vehicle):**
```json
{
  "title": "Will these pads fit a 2018 Camry LE?",
  "body": "I see compatibility listed for 2015-2020 but want to confirm for the LE trim.",
  "product_id": "550e8400-e29b-41d4-a716-446655440010"
}
```

Or vehicle context:
```json
{
  "title": "Best ceramic pads for daily driving?",
  "body": "Mostly city commuting, occasional highway. Low dust preferred.",
  "make": "Toyota",
  "model": "Camry",
  "year": 2018
}
```

**Response (201):** `data` includes `id`, `slug`, `status` (`open`), `answers` (empty).

---

### GET /questions

**Query (optional):**
```
?q=brake+pads&product_id=<uuid>&make=toyota&model=camry&year=2018&page=1&limit=20
```

---

### GET /questions/:slug

**Path:** `slug` from create response (e.g. `will-these-pads-fit-a-2018-camry-le`)

Increments `view_count`. Returns full `body` and `answers` array.

---

### GET /products/:id/questions

**Path:** `id` = product UUID. Same list shape as `GET /questions?product_id=...`.

---

### POST /questions/:id/answers

**Headers:** `Authorization: Bearer <access_token>` (verified mechanic)

**Request body:**
```json
{
  "body": "Yes — these pads fit 2018 Camry LE with the standard brake package. Torque spec 25 ft-lb on caliper bolts."
}
```

**Response (201):** Answer with `is_verified_mechanic: true`. Question author receives `qa.answer_posted` notification.

---

### PATCH /questions/:id/accept-answer/:answerId

**Headers:** `Authorization: Bearer <access_token>` (question author)

Marks answer accepted and sets question `status` to `answered`.

---

### PATCH /questions/:id/close

**Headers:** `Authorization: Bearer <access_token>` (author or admin)

Sets `status` to `closed`; thread no longer appears in public list.

---

## Visual Part Finder

See [part-finder.md](./part-finder.md).

### GET /vehicle-systems

No auth.

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "code": "brakes",
      "name": "Brakes",
      "description": "Brake pads, rotors, calipers, lines",
      "display_order": 1
    }
  ]
}
```

---

### GET /diagrams

**Query:** `make`, `model`, `year`, `system` (vehicle system code), `page`, `limit`

**Response (200):** Paginated list of diagrams with nested `vehicle_system`.

---

### GET /diagrams/:id

**Query:** `include_hotspots=true` to embed hotspots.

---

### GET /diagrams/:id/hotspots/:hotspotId/products

**Query:** `year` (optional; defaults to diagram year range)

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "sku": "BP-CAMRY-F",
      "name": "Front Brake Pads — Ceramic",
      "brand": "StopTech",
      "manufacturer_part_number": "ST-1234",
      "price": 89.99,
      "condition": "new",
      "stock_quantity": 42,
      "primary_image_url": "https://cdn.example.com/products/bp-camry.jpg"
    }
  ]
}
```

---

### POST /part-identification

**Headers:** `Authorization: Bearer <access_token>`

**Content-Type:** `multipart/form-data`

| Field | Value |
|-------|--------|
| `image` | Image file |
| `make` | `toyota` |
| `model` | `camry` |
| `year` | `2018` |
| `system` | `brakes` (optional) |
| `labels` | `["brake pad","caliper"]` or comma-separated (optional) |

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "image_url": "https://cdn.example.com/part-identification/abc.jpg",
    "diagram_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "candidates": [
      {
        "part_name": "Front Brake Pad",
        "confidence": 0.9,
        "hotspot_id": "6ba7b811-9dad-11d1-80b4-00c04fd430c8",
        "diagram_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
        "product_ids": ["550e8400-e29b-41d4-a716-446655440001"]
      }
    ]
  }
}
```

Requires S3 configured (`S3_BUCKET`). Without labels, pass `system` for broader hotspot suggestions.

---

### POST /diagrams (Admin/Vendor)

**Request body:**
```json
{
  "vehicle_system_code": "brakes",
  "title": "2018 Toyota Camry — Front Brakes",
  "make": "Toyota",
  "model": "Camry",
  "year_start": 2018,
  "year_end": 2020,
  "image_url": "https://cdn.example.com/diagrams/camry-brakes.png",
  "svg_overlay_url": "",
  "image_width": 1200,
  "image_height": 800,
  "is_published": true
}
```

---

### POST /diagrams/:id/hotspots

**Request body:**
```json
{
  "label": "Front Brake Pad",
  "oem_part_number": "ST-1234",
  "x": 12.5,
  "y": 34.0,
  "width": 18.0,
  "height": 12.0,
  "display_order": 1
}
```

Coordinates are percentages (0–100) of the diagram image.

---

### POST /diagrams/:id/hotspots/:hotspotId/products

**Request body:**
```json
{
  "product_id": "550e8400-e29b-41d4-a716-446655440001",
  "match_type": "primary"
}
```

---

## Health

### GET /health

No auth. No body.

**Response:** `{"status":"ok"}`
