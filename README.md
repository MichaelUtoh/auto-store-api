# Auto-Store API

Production-ready RESTful API for an auto-parts e-commerce platform built with Go.

## Features

- **Authentication**: JWT-based auth with refresh tokens, email verification, password reset, RBAC (Admin, Vendor, Customer)
- **Products**: CRUD, advanced search (full-text, category, tags, vehicle compatibility, price range), reviews, compatibility
- **Categories**: Hierarchical categories, tree listing, products by category
- **Cart & Orders**: Cart management, checkout, order status (admin)
- **User Profile**: Profile, addresses, wishlist
- **Security**: Rate limiting, account lockout, CORS, security headers, request size limits
- **Infrastructure**: PostgreSQL (GORM), Redis (sessions/rate limit), config via env

## Tech Stack

- **Go** 1.21+
- **Gin** (HTTP router)
- **GORM** + PostgreSQL
- **Redis** (cache/sessions)
- **JWT** (golang-jwt/jwt/v5)
- **Zap** (logging)
- **Validator** (go-playground/validator)

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+

### Environment

```bash
cp .env.example .env
# Edit .env with your DB, Redis, JWT secret, etc.
```

### Run locally

```bash
# Install dependencies (requires network)
go mod download

# Run (ensure Postgres and Redis are up)
go run ./cmd/api
```

Server listens on `http://localhost:8089` (or `PORT` from env).

- **API docs (Swagger UI):** [http://localhost:8089/docs/index.html](http://localhost:8089/docs/index.html)

### Docker

```bash
docker-compose up -d
# API: http://localhost:8089
# Docs: http://localhost:8089/docs/index.html
# Postgres: localhost:5432, Redis: localhost:6379
```

The API container retries the database connection on startup (when `DB_HOST` is `postgres`), so it waits for Postgres to be ready. If Docker fails: ensure ports 8089, 5432, 6379 are free; run `docker-compose build --no-cache` then `docker-compose up -d`; check logs with `docker-compose logs -f api`.

### Makefile

- `make build` – build binary to `bin/api`
- `make run` – run API
- `make test` – run tests
- `make docker-up` / `make docker-down` – Docker Compose

## API Overview

| Area        | Endpoints |
|------------|-----------|
| **Auth**   | `POST /api/v1/auth/register`, `login`, `logout`, `refresh`, `forgot-password`, `reset-password`, `verify-email` |
| **Products** | `GET/POST/PUT/DELETE /api/v1/products`, `GET /api/v1/products/search`, `GET /api/v1/products/:id/compatibility`, `GET/POST /api/v1/products/:id/reviews` |
| **Categories** | `GET/POST/PUT/DELETE /api/v1/categories`, `GET /api/v1/categories/:id/products` |
| **Cart**   | `GET /api/v1/cart`, `POST /api/v1/cart/items`, `PUT/DELETE /api/v1/cart/items/:id`, `DELETE /api/v1/cart` |
| **Orders** | `POST/GET /api/v1/orders`, `GET /api/v1/orders/:id`, `PUT /api/v1/orders/:id/cancel`, Admin: `GET /api/v1/admin/orders`, `PUT /api/v1/admin/orders/:id/status` |
| **Users**  | `GET/PUT /api/v1/users/me`, `GET/POST/PUT/DELETE /api/v1/users/me/addresses` |
| **Wishlist** | `GET /api/v1/wishlist`, `POST /api/v1/wishlist`, `DELETE /api/v1/wishlist/:productId` |

Protected routes require header: `Authorization: Bearer <access_token>`.

**Sample payloads:** See [docs/sample-payloads.md](docs/sample-payloads.md) for request/response examples for all endpoints.

### Search example

```
GET /api/v1/products/search?q=brake+pads&category=brakes&tags=ceramic,performance&make=toyota&model=camry&year=2015-2020&minPrice=50&maxPrice=200&condition=new&sort=price_asc&page=1&limit=20
```

## Project Structure

```
auto-store-api/
├── cmd/api/main.go          # Entry point
├── internal/
│   ├── config/              # Env configuration
│   ├── database/            # GORM connect + AutoMigrate
│   ├── middleware/          # Auth, CORS, rate limit, security
│   ├── models/              # Domain models
│   ├── handlers/            # HTTP handlers + DTOs
│   ├── services/            # Business logic
│   ├── repositories/        # Data access
│   ├── validators/          # Custom validation
│   ├── utils/               # Response helpers
│   └── router/              # Route setup
├── pkg/
│   ├── auth/                # JWT
│   ├── cache/               # Redis client
│   └── logger/              # Zap logger
├── docs/                    # Swagger/OpenAPI
├── migrations/              # Optional SQL migrations
├── .env.example
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Configuration (env)

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP port | 8089 |
| `DB_*` | PostgreSQL connection | - |
| `REDIS_*` | Redis connection | - |
| `JWT_SECRET` | JWT signing key | - |
| `JWT_ACCESS_EXPIRY` | Access token TTL | 15m |
| `JWT_REFRESH_EXPIRY` | Refresh token TTL | 168h |
| `RATE_LIMIT_RPM` | General rate limit | 60 |
| `AUTH_RATE_LIMIT_RPM` | Auth rate limit | 10 |
| `ACCOUNT_LOCKOUT_ATTEMPTS` | Failed logins before lockout | 5 |
| `CORS_ORIGINS` | Allowed origins | * |

See `.env.example` for full list.

## Testing

```bash
go test -v ./...
```

## License

MIT
