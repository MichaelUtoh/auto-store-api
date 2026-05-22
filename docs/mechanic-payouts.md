# Mechanic payouts (Paystack subaccounts)

Verified mechanics receive their share of **installation booking** payments via Paystack [split payments](https://paystack.com/docs/payments/split-payments/).

## Split model

For each booking:

| Recipient | Amount |
|-----------|--------|
| Platform (main account) | `platform_fee` (10% of labor) — Paystack `transaction_charge` |
| Mechanic (subaccount) | `labor_total` + `parts_total` — remainder after platform fee |

Customer pays `total_amount` = labor + parts + platform fee.

Store **orders** are not split; only installation bookings use subaccounts.

## Mechanic setup

1. Mechanic is **verified** by admin.
2. Mechanic calls `GET /api/v1/payments/banks` to list bank codes (country from `PAYSTACK_BANK_COUNTRY` / currency).
3. Mechanic registers payout details:

```http
POST /api/v1/mechanic/payout
Authorization: Bearer <token>

{
  "bank_code": "058",
  "account_number": "0123456789"
}
```

Creates or updates a Paystack subaccount; stores `subaccount_code` on the mechanic profile.

4. `GET /api/v1/mechanic/payout` — returns configured status (last 4 digits of account, account name, subaccount code).

## Customer checkout

`POST /api/v1/installation/bookings/:id/pay` initializes Paystack with:

- `subaccount`: mechanic's `ACCT_xxx`
- `transaction_charge`: platform fee in minor units (kobo/cents)

If `PAYSTACK_REQUIRE_SPLIT_BOOKINGS=true` (default) and the mechanic has no subaccount, checkout returns **409** with `mechanic payout account is not configured`.

Set `PAYSTACK_REQUIRE_SPLIT_BOOKINGS=false` to allow checkout without split (full amount to platform) during rollout.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PAYSTACK_SPLIT_ENABLED` | `true` | Attach subaccount split on booking payments |
| `PAYSTACK_REQUIRE_SPLIT_BOOKINGS` | `true` | Block booking pay if mechanic has no subaccount |
| `PAYSTACK_BANK_COUNTRY` | from currency | `nigeria`, `ghana`, `kenya`, `south africa` for `/bank` list |

## Paystack fees

By default Paystack transaction fees are borne by the **main** (platform) account. To charge the mechanic instead, extend initialize with `bearer: subaccount` (not enabled by default).
