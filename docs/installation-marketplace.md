# Installation Marketplace

Connects customers who buy installation-eligible parts with **verified local mechanics** for quotes and bookings.

Prerequisites: [mechanics.md](./mechanics.md) (verified mechanic profiles).

**Next.js frontend:** [nextjs-installation-marketplace-prompt.md](./nextjs-installation-marketplace-prompt.md)

---

## Flow

1. Customer requests a quote (`POST /installation/quotes`) with vehicle info, service location (lat/lng), and either `order_id` or `product_ids`.
2. API matches verified mechanics within radius who offer the required job types.
3. Customer receives `quote.ready` notification and compares quote lines.
4. Customer books a slot (`POST /installation/bookings`) — creates booking with `pending_payment` (Paystack integration planned).
5. Mechanic updates status (`confirmed` → `en_route` → `in_progress` → `completed`).

---

## Product setup

Mark products as installable:

- `installation_eligible`: `true`
- `installation_job_type_id`: UUID from `GET /installation/job-types`

---

## API

### Public

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/installation/job-types` | Canonical install job catalog |

### Customer (authenticated)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/installation/quotes` | Request quotes from nearby mechanics |
| GET | `/api/v1/installation/quotes` | List own quotes |
| GET | `/api/v1/installation/quotes/:id` | Quote with mechanic lines |
| POST | `/api/v1/installation/bookings` | Book selected quote line + time |
| GET | `/api/v1/installation/bookings` | List own bookings |
| GET | `/api/v1/installation/bookings/:id` | Booking detail |
| PATCH | `/api/v1/installation/bookings/:id/cancel` | Cancel booking |

### Mechanic (verified profile required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/mechanic/installation/quotes` | Quote lines assigned to this mechanic |
| PATCH | `/api/v1/mechanic/installation/quotes/:id` | Update labor price / message |
| PUT | `/api/v1/mechanic/installation/services` | Set job types offered |
| GET | `/api/v1/mechanic/installation/bookings` | Mechanic's bookings |
| GET | `/api/v1/mechanic/installation/bookings/:id` | Booking detail |
| PATCH | `/api/v1/mechanic/installation/bookings/:id/status` | Update lifecycle status |

---

## Data model

| Table | Purpose |
|-------|---------|
| `installation_job_types` | Catalog (brake pads, oil change, …) |
| `mechanic_install_services` | Mechanic ↔ job type labor pricing |
| `installation_quotes` | Customer quote request |
| `installation_quote_items` | Products / job types on quote |
| `installation_quote_lines` | Per-mechanic offers |
| `installation_bookings` | Confirmed appointment |
| `booking_payments` | Payment provider refs (`manual` until Paystack) |

---

## Notifications

| Type | When |
|------|------|
| `quote.ready` | After quote lines generated |
| `booking.confirmed` | Customer + mechanic on booking |
| `mechanic.en_route` | Mechanic sets status `en_route` |

---

## Payments (planned — Paystack)

Not implemented yet. Bookings use `payment_status: pending` and `booking_payments.provider: manual`.

When added, expect:

- **Paystack** Initialize Transaction / Verify for customer labor + parts checkout
- Webhook (`charge.success`) to confirm booking and set `payment_status: paid`
- Optional **Paystack Subaccounts** or split rules for mechanic payouts (marketplace)
- Env: `PAYSTACK_SECRET_KEY`, `PAYSTACK_PUBLIC_KEY`, `PAYSTACK_WEBHOOK_SECRET` (see `.env.example`)

Until then, mechanics can mark bookings `confirmed` after off-platform payment if you allow that in ops.

## Not yet implemented

- Paystack checkout + webhooks + mechanic payouts
- Calendar / availability slots
- PostGIS (uses haversine on lat/lng)
- Admin dispute workflow
