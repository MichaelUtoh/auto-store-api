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
| PUT | `/api/v1/admin/users/:id/role` | Admin | Update user role (body: role — ADMIN, VENDOR, CUSTOMER). |

---

## Summary

- **Public:** Health, docs, auth (except logout), products list/search/get/compatibility/reviews GET, categories list/get/products.
- **Authenticated (any role):** Logout, cart, orders (create/list/get/cancel), profile, addresses, wishlist, create product review.
- **Admin or Vendor:** Products create/batch/update, product images, product compatibility.
- **Admin only:** Product delete, categories CRUD, admin orders list/status, admin user role update.
