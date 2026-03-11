# Jabba The Bot — Cover Metadata Flow (Sketch)

## Goal

Decouple cover-art delivery from raw third-party URLs:

1. Client receives `videoID` (and normal track metadata), not external cover URL.
2. Backend owns cover URL resolution and caching.
3. Cover image is served from Jabba endpoint (`/api/covers/{videoID}`), backed by metadata + cache.

This keeps frontend simple, avoids CORS/canvas taint issues, and centralizes rate-limit handling.

---

## High-Level Flow

```text
Search Resolver / Metadata Enricher
        │
        │ upsert metadata (videoID -> coverURL + title + artist + ...)
        ▼
MetadataStore (SQLite) + CoverCache (disk)
        │
        ├── /api/queue, /api/now-playing:
        │      returns videoID (+ metadata), no raw cover URL required
        │
        └── /api/covers/{videoID}:
               resolve metadata by videoID
               load cached image OR fetch upstream once
               return image bytes + cache headers
```

---

## API Shape (Proposed)

### Playback/Queue payloads

- Keep current fields, but treat `videoID` as canonical media key.
- `coverUrl` becomes optional or removed from these payloads over time.

Example:

```json
{
  "title": "Cut the Bridge",
  "videoID": "_f9b0NB5o4E",
  "position": 135,
  "duration": 228
}
```

### Cover endpoint

`GET /api/covers/{videoID}`

- Looks up metadata by `videoID`.
- Uses stored `cover_url` as upstream source.
- Serves cached image bytes.
- Returns `404` if no metadata / no cover known.

Optional:

- `GET /api/covers/{videoID}?size=small|medium|large` if you want normalized sizes later.

---

## Storage Model

Reuse `track_metadata` as source of truth for cover URL.

Current table already contains:

- `video_id` (PK)
- `cover_url`
- `title`, `artist`, `album`, `duration_sec`
- `fetched_at`, `last_accessed_at`, `source_version`

Optional extension for stronger cover management:

- `cover_etag` TEXT
- `cover_last_checked_at` DATETIME
- `cover_fetch_status` TEXT (`ok`, `rate_limited`, `missing`, `error`)

No client-visible external URL is required once this is wired.

---

## Components

### 1. Metadata Writer

Where metadata enters system:

- Search results
- Resolver resolution
- Provider enrichment

Responsibility:

- `Upsert(videoID, title, artist, coverURL, ...)` into `MetadataStore`.

### 2. Cover Service (new app service)

Interface sketch:

```go
type CoverService interface {
    GetCover(ctx context.Context, videoID string) (body []byte, contentType string, etag string, err error)
}
```

Responsibility:

- Lookup metadata by `videoID`.
- If cached image exists and fresh, serve it.
- Otherwise fetch upstream via allowlisted hosts, cache to disk, return image.

### 3. HTTP Cover Handler

`GET /api/covers/{videoID}` only.

No raw upstream URL query param needed for normal clients.

### 4. Cover Cache

Disk path example:

- `library/cache/covers/{sha256(videoID + sourceURL)}.img`

Stores:

- bytes + content-type sidecar (or tiny metadata index).

---

## Package Placement (aligned with current architecture)

```text
internal/
├── core/app/
│   ├── service/
│   │   └── cover_service.go        // use-case: videoID -> image bytes
│   └── ports/
│       ├── inbound.go              // CoverQueries interface
│       └── outbound.go             // MetadataStore + CoverCache ports
│
├── adapters/
│   ├── outbound/state/
│   │   └── sqlite_metadata_store.go
│   ├── outbound/cache/
│   │   └── file_cover_cache.go
│   └── http/
│       └── handlers_cover.go       // GET /api/covers/{videoID}
```

If you keep current non-hexagonal structure short-term, same boundaries still apply logically.

---

## Request Lifecycle Example

`Now Playing UI` asks for current track:

1. `GET /api/now-playing` -> `{ videoID: "_f9b0NB5o4E", title: "...", ... }`
2. UI renders image from `/api/covers/_f9b0NB5o4E`
3. Cover handler:
   - metadata lookup by `videoID`
   - cover cache lookup
   - optional upstream fetch (only backend does this)
   - returns image with `Cache-Control` + `ETag`

Result:

- UI never needs googleusercontent/ytimg URLs.
- FastAverageColor can safely run against same-origin `/api/covers/{videoID}`.

---

## Rollout Plan

1. Keep existing fields and add `/api/covers/{videoID}`.
2. Update frontend to prefer `/api/covers/{videoID}` for hero/dock/search.
3. Continue storing/updating `cover_url` in metadata store.
4. After stable period, de-emphasize or remove raw `coverUrl` from response DTOs.

---

## Tradeoffs

Pros:

- Strong separation: client does not know external media host details.
- Centralized retries/rate-limit/backoff.
- Consistent same-origin image behavior for rendering and color extraction.
- Better observability (cache hit rate, upstream failures).

Cons:

- Extra backend responsibility and storage.
- Need metadata freshness policy for changed artwork URLs.
- More moving parts than direct URL passthrough.

---

## Recommendation

Use `videoID` as the canonical frontend media key and move cover resolution to a dedicated `CoverService` backed by `SQLiteMetadataStore` + file cache. Keep your current proxy endpoint as transitional compatibility, then converge to `/api/covers/{videoID}` as the only cover path.

