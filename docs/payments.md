# Paystack payments

Storefront and installation bookings can be paid via [Paystack](https://paystack.com/docs/api/).

## Configuration

Set in `.env` (see `.env.example`):

| Variable | Description |
|----------|-------------|
| `PAYSTACK_SECRET_KEY` | Secret key (server-side). Required to enable payments. |
| `PAYSTACK_PUBLIC_KEY` | Public key (returned to clients for inline JS). |
| `PAYSTACK_WEBHOOK_SECRET` | Webhook signing secret from Paystack dashboard. |
| `PAYSTACK_CALLBACK_URL` | Redirect URL after hosted checkout (e.g. frontend verify page). |
| `PAYSTACK_CURRENCY` | ISO currency, default `NGN`. Amounts are sent in minor units (kobo/cents). |

Payments are disabled until `PAYSTACK_SECRET_KEY` is set.

## Checkout flow

1. Create an **order** (`POST /api/v1/orders`) or **installation booking** (`POST /api/v1/installation/bookings`) — `payment_status` stays `pending`.
2. Initialize Paystack:
   - Orders: `POST /api/v1/orders/:id/pay`
   - Bookings: `POST /api/v1/installation/bookings/:id/pay`
3. Response includes `authorization_url`, `access_code`, `reference`, `public_key`, `amount`, `currency`.
4. Redirect the customer to `authorization_url` (or use Paystack Inline with `public_key` + `access_code`).
5. After payment, verify on your frontend by calling `GET /api/v1/payments/verify?reference=...` (authenticated), or rely on the webhook.

## Webhook

Register in Paystack dashboard:

```
POST https://your-api.example.com/webhooks/paystack
```

Events handled: `charge.success`, `refund.processed`, `refund.failed`. Signature header: `x-paystack-signature` (HMAC-SHA512 of raw body with `PAYSTACK_WEBHOOK_SECRET`).

On success:

- **Orders:** `payment_status` → `paid`, `status` → `confirmed` if still `pending`
- **Bookings:** `payment_status` → `paid`, `status` → `confirmed` if `pending_payment`

## Metadata

Each transaction includes Paystack metadata:

- `entity_type`: `order` | `booking`
- `entity_id`: UUID
- `user_id`: payer UUID

## Mechanic split payouts

Installation bookings split payment between the platform and the assigned mechanic. See [mechanic-payouts.md](./mechanic-payouts.md).

## Refunds

Refund a **paid Paystack** order or booking (full or partial). Paystack queues the refund; status is updated locally immediately and confirmed via `refund.processed` webhook.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/orders/:id/refund` | Customer | Refund own order (not shipped/delivered) |
| POST | `/api/v1/installation/bookings/:id/refund` | Customer | Refund own booking (not completed) |
| POST | `/api/v1/admin/orders/:id/refund` | Admin | Refund any eligible order |
| POST | `/api/v1/admin/installation/bookings/:id/refund` | Admin | Refund any eligible booking |

Optional body:

```json
{
  "amount": 50.00,
  "customer_note": "Changed mind",
  "merchant_note": "Approved by support"
}
```

Omit `amount` for a full refund.

**Cancel with auto-refund:** `PATCH /api/v1/installation/bookings/:id/cancel` automatically initiates a Paystack refund when the booking was paid via Paystack.

Orders must be `payment_status: paid` with `payment_method: paystack`. Shipped/delivered orders cannot be refunded via API.

## Not yet implemented

- Dedicated admin payment reports
