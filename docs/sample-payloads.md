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

`role` must be one of: `ADMIN`, `VENDOR`, `CUSTOMER` (stored in caps).

---

## User Profile

### GET /users/me

**Headers:** `Authorization: Bearer <access_token>`

No body.

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

## Health

### GET /health

No auth. No body.

**Response:** `{"status":"ok"}`
