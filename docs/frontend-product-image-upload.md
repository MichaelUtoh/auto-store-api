# Frontend prompt: product image upload (Auto-Store API)

Use this document to implement **admin/vendor product image upload** in a web or mobile client. The backend uses a **two-step flow**: upload files to object storage, then **attach returned URLs** to a product record.

---

## Who can upload

- **Step 1 (file upload):** `POST /api/v1/upload/images` — **Admin** or **Vendor** only (`Authorization: Bearer <access_token>`).
- **Step 2 (save on product):** `POST /api/v1/products/:productId/images` — same roles. `:productId` is the product **UUID**.

If the user is not Admin/Vendor, hide upload UI or show “insufficient permissions.”

---

## API response envelope

Successful JSON responses wrap payload in:

```json
{ "success": true, "data": { ... } }
```

Errors:

```json
{ "success": false, "error": "message" }
```

Parse `data` after checking `success === true`.

---

## Step 1 — Upload files to storage (multipart)

**Request**

- **Method:** `POST`
- **URL:** `{API_BASE}/api/v1/upload/images`  
  Example: `http://localhost:8089/api/v1/upload/images`
- **Headers:**
  - `Authorization: Bearer <access_token>` (required)
  - **Do not** set `Content-Type` manually when using `FormData` — the browser/client must set `multipart/form-data` with boundary automatically.
- **Body:** `multipart/form-data` with one or more files under the form field name **`file`** or **`files`** (both are accepted; repeat the same key for multiple files).

**Allowed types (default backend):** `image/jpeg`, `image/png`, `image/webp` (exact `Content-Type` on each part must match what the server allows).

**Max size (default):** 5,242,880 bytes (~5 MB) per file — configurable via backend `UPLOAD_MAX_SIZE`.

**Success — HTTP 201**

`data` shape:

```json
{
  "urls": [
    "https://your-storage.example.com/bucket/products/uuid.jpg",
    "https://..."
  ]
}
```

Order of `urls` matches the order files were processed.

**Failure examples**

| HTTP | Meaning |
|------|--------|
| 400 | No files, wrong form key, or disallowed MIME type (`error` may mention the type). |
| 413 | File too large. |
| 401 | Missing/invalid token. |
| 403 | Logged in but not Admin/Vendor. |
| 503 | S3 not configured on server (`S3_BUCKET` empty). |
| 500 | Upload/storage error (`error` may include provider message). |

---

## Step 2 — Attach URLs to the product (JSON)

After Step 1, persist images on the product so they appear in `GET /api/v1/products/:id`.

**Request**

- **Method:** `POST`
- **URL:** `{API_BASE}/api/v1/products/{productUuid}/images`
- **Headers:**
  - `Authorization: Bearer <access_token>`
  - `Content-Type: application/json`
- **Body:**

```json
{
  "images": [
    {
      "url": "https://...from-step-1...",
      "alt_text": "Front view",
      "display_order": 0,
      "is_primary": true
    },
    {
      "url": "https://...",
      "alt_text": "Side view",
      "display_order": 1,
      "is_primary": false
    }
  ]
}
```

| Field | Required | Notes |
|-------|----------|--------|
| `url` | Yes | Must be a valid URL string (from Step 1). |
| `alt_text` | No | Max 255 chars. |
| `display_order` | No | Integer; lower sorts first. |
| `is_primary` | No | If `true`, this image becomes primary; others for that product are cleared as primary. |

**Success — HTTP 201**

`data` is an array of saved image objects (ids, `product_id`, `url`, `alt_text`, `display_order`, `is_primary`, timestamps).

**Failure**

- **400** — invalid JSON, validation (`url` invalid), empty `images`.
- **404** — `productId` not a valid UUID or product does not exist.

---

## Recommended frontend flow

1. User selects one or more images ( `<input type="file" accept="image/jpeg,image/png,image/webp" multiple>` or drag-and-drop ).
2. Optionally validate **size** and **type** client-side before upload (match backend defaults to avoid wasted requests).
3. Build `FormData`, append each file as `file` (or `files`):
   - JavaScript: `formData.append("file", file)` for each file.
4. `POST` Step 1 with **only** `Authorization` + `FormData` (no `Content-Type` header override).
5. Read `data.urls` from the envelope.
6. Map URLs to your UI fields (`alt_text`, `display_order`, `is_primary`), then `POST` Step 2 with JSON body.
7. Refresh product detail or merge returned image records into local state.

**Optional UX**

- Progress: use `XMLHttpRequest` or `axios` `onUploadProgress` for Step 1 (multipart).
- Show per-file errors if Step 1 fails mid-batch (backend processes sequentially; first failure aborts the whole request — consider uploading one file per request if you need partial success).
- After Step 2, display images using the **returned** URLs from the API (or refetch product).

---

## Example: browser `fetch` (Step 1)

```javascript
async function uploadProductImages(accessToken, files) {
  const formData = new FormData();
  for (const file of files) {
    formData.append("file", file);
  }

  const res = await fetch(`${API_BASE}/api/v1/upload/images`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    body: formData,
  });

  const json = await res.json();
  if (!json.success) {
    throw new Error(json.error || res.statusText);
  }
  return json.data.urls; // string[]
}
```

## Example: attach to product (Step 2)

```javascript
async function attachImagesToProduct(accessToken, productId, urls, options = {}) {
  const images = urls.map((url, i) => ({
    url,
    alt_text: options.altTexts?.[i] ?? "",
    display_order: i,
    is_primary: i === (options.primaryIndex ?? 0),
  }));

  const res = await fetch(`${API_BASE}/api/v1/products/${productId}/images`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${accessToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ images }),
  });

  const json = await res.json();
  if (!json.success) {
    throw new Error(json.error || res.statusText);
  }
  return json.data;
}
```

---

## Next.js notes

- If Step 1 runs in the **browser**, call the API URL directly (or your Next.js rewrite proxy). Ensure CORS allows your origin (backend `CORS_ORIGINS`).
- Do **not** try to send `multipart/form-data` through a Server Action as a raw buffer unless you re-stream it; simplest path is **client component** + `fetch`/`axios` to the API.
- For Step 2, same-origin API route can proxy JSON if you prefer hiding the backend URL.

---

## One-line prompt for an AI assistant

**“Implement Admin/Vendor product image upload: (1) POST multipart to `/api/v1/upload/images` with form field `file` (repeat for multiple files) and `Authorization: Bearer` token; parse `{ success, data: { urls: string[] } }`. (2) POST JSON to `/api/v1/products/:productUuid/images` with body `{ images: [{ url, alt_text?, display_order?, is_primary? }] }`. Validate file type/size client-side to match image/jpeg, image/png, image/webp and ~5MB. Handle 401/403/413/503 and show errors from `error`.”**

---

## Backend references

- Upload handler: `internal/handlers/upload_handler.go`
- Add images to product: `internal/handlers/product_handler.go` (`AddImages`)
- Sample payloads: `docs/sample-payloads.md` (POST `/products/:id/images`)
