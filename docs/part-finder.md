# Visual Part Finder (2D diagrams + AR identification)

Interactive exploded-view diagrams let shoppers click a component and jump to matching catalog SKUs. **AR part identification** accepts a camera image plus vehicle context and returns ranked part guesses (MVP uses label taxonomy + hotspot matching; plug in a CV service later).

---

## Phase A: 2D diagrams (implemented)

### Data model

| Table | Purpose |
|-------|---------|
| `vehicle_systems` | Canonical systems: `brakes`, `suspension`, `engine`, … |
| `diagrams` | Exploded image per make/model/year range + system |
| `diagram_hotspots` | Click regions (`x`, `y`, `width`, `height` as **% of image**) |
| `hotspot_products` | Explicit hotspot → product links |
| `part_label_taxonomies` | CV label → hotspot label patterns (AR matching) |
| `part_identifications` | Audit log of identification requests |

Hotspot coordinates use **0–100 percent** of image width/height so the frontend can scale responsively.

### Product matching

`GET /diagrams/:id/hotspots/:hotspotId/products` returns:

1. Products linked in `hotspot_products`
2. Products where `manufacturer_part_number` matches hotspot `oem_part_number` and `vehicle_compatibilities` match the diagram vehicle (optional `year` query overrides diagram default)

Results are merged and deduplicated.

---

## Public API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/vehicle-systems` | List systems for filters |
| GET | `/api/v1/diagrams` | List diagrams (`make`, `model`, `year`, `system`, `page`, `limit`) |
| GET | `/api/v1/diagrams/:id` | Diagram detail (`?include_hotspots=true`) |
| GET | `/api/v1/diagrams/:id/hotspots` | Hotspots for a diagram |
| GET | `/api/v1/diagrams/:id/hotspots/:hotspotId/products` | Matched products (`?year=` optional) |

---

## Authenticated API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/part-identification` | AR/CV identification (multipart) |

**Multipart fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `image` or `file` | Yes | Photo (JPEG/PNG/WebP) |
| `make` | Yes | Vehicle make |
| `model` | Yes | Vehicle model |
| `year` | Yes | Model year |
| `system` | No | Vehicle system code (`brakes`, …) |
| `labels` | No | JSON array or comma-separated CV labels |

Image is stored in S3 under `part-identification/`. Response includes `candidates[]` with `part_name`, `confidence`, `hotspot_id`, `diagram_id`, `product_ids`.

**MVP matching:** labels → taxonomy patterns → hotspot labels → catalog products. Without `labels`, `system` returns hotspots from the best matching diagram at lower confidence.

---

## Admin / vendor API

Same auth as product catalog (`ADMIN` or `VENDOR`):

| Method | Path |
|--------|------|
| POST | `/api/v1/diagrams` |
| PUT | `/api/v1/diagrams/:id` |
| POST | `/api/v1/diagrams/:id/hotspots` |
| PUT | `/api/v1/diagrams/:id/hotspots/:hotspotId` |
| POST | `/api/v1/diagrams/:id/hotspots/:hotspotId/products` |
| DELETE | `/api/v1/diagrams/:id/hotspots/:hotspotId/products/:productId` |

**Admin only:**

| Method | Path |
|--------|------|
| DELETE | `/api/v1/diagrams/:id` |
| DELETE | `/api/v1/diagrams/:id/hotspots/:hotspotId` |

Upload diagram images via `POST /api/v1/upload/images`, then reference URLs in diagram create/update.

---

## Phase B: AR / ML (next steps)

1. Add a worker or external service that calls your CV model and writes labels before matching.
2. Replace or augment taxonomy matching with embedding similarity.
3. Flutter client: camera → `POST /part-identification` → product/cart/install flows.

See [sample-payloads.md](./sample-payloads.md#visual-part-finder) for request/response examples.
