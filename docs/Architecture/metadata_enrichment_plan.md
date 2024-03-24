# Metadata Enrichment Plan

## Goal

Add artist/album/cover metadata enrichment for tracks without increasing playback startup latency.

## Principles

1. Playback must never block on metadata fetch.
2. Metadata fetching is backend-only (provider-specific logic stays server-side).
3. UI updates should be event-driven (SSE) when metadata arrives.
4. Aggressive prefetch is allowed, but controlled via rate limiting, dedupe, and caching.

## Target Behavior

1. A track can start playing immediately with minimal fields (`title`, `videoId`, `url`).
2. Rich metadata (`artist`, `album`, `coverUrl`, etc.) is attached asynchronously.
3. As soon as metadata is available, backend emits events and UI updates live.
4. Metadata survives restarts via persistent cache.

## Current Baseline (Already Implemented)

1. SQLite bootstrap (open, pragmas, migrations) is in place.
2. `track_metadata` schema migration exists.
3. Runtime wiring in `cmd/jabba` initializes SQLite.
4. Metadata store scaffold exists (`GetByVideoID`, `Upsert`, `Touch`).
5. Cover art derivation is now backend-driven and exposed via API.

## Relevant Files Map

### Composition / Runtime

1. `cmd/jabba/main.go` - composition root, currently opens SQLite and creates store.
2. `cmd/teamspeak/main.go` - legacy runtime path (keep behavior consistent if needed).

### SQLite Platform

1. `internal/platform/storage/sqlite/sqlite.go` - DB open + PRAGMA setup + migration runner.
2. `internal/platform/storage/sqlite/migrations.go` - migration registry and `track_metadata` table.

### Metadata Persistence (Adapter)

1. `internal/adapters/outbound/state/sqlite_metadata_store.go` - metadata store adapter scaffold.

### Eventing / SSE

1. `internal/core/app/dto/event.go` - app event types.
2. `internal/core/app/ports/outbound.go` - event bus ports.
3. `internal/adapters/outbound/eventbus/inproc_bus.go` - in-proc event bus.
4. `internal/adapters/inbound/http/event_bridge.go` - app events -> SSE bridge.
5. `internal/adapters/inbound/http/sse.go` - SSE endpoint fanout.

### Playback / Provider Path (Metadata Hook Points)

1. `internal/music/manager.go` - playback lifecycle, now playing updates, queue updates, progress loop.
2. `internal/music/provider/youtube.go` - provider integration and best place for metadata fetch adapter usage.
3. `pkg/youtube/music/` - Innertube request/response parsing layer.

### HTTP Views / DTOs / Frontend Consumption

1. `internal/core/app/ports/inbound.go` - API view structs (`NowPlayingView`, `QueueItemView`).
2. `internal/adapters/inbound/http/legacy_bridge.go` - maps manager snapshots to API views.
3. `web/src/lib/types.ts` - frontend API typings.
4. `web/src/routes/home-data.ts` - runtime query/SSE consumption.
5. `web/src/routes/home.tsx` and `web/src/routes/now-playing.tsx` - now-playing UI and controls.

## Explicit Next Action Plan (Start Here)

### Step 1: Define Metadata Contracts

1. Add metadata DTO + ports:
   - `internal/core/app/dto/metadata.go`
   - `internal/core/app/ports/outbound.go` (or dedicated metadata port file)
2. Define:
   - `TrackMetadata`
   - `MetadataProvider`
   - `MetadataStore`
   - `MetadataService`

Acceptance:

1. Code compiles with interfaces only (no runtime behavior change).

### Step 2: Implement Metadata Service (Memory + SQLite + Provider)

1. Add `internal/core/app/service/metadata_service.go`.
2. Read path:
   - memory cache -> sqlite -> provider fetch (async on miss/stale).
3. Write path:
   - provider response -> sqlite upsert -> memory cache.
4. Add in-flight dedupe per `videoID`.

Acceptance:

1. `Get(videoID)` returns quickly and does not block playback on cache miss.

### Step 3: Add Provider Metadata Fetch

1. Add Innertube player metadata adapter in:
   - `internal/adapters/outbound/music/youtube_metadata_provider.go`
2. Parse and normalize:
   - title, artist, album, cover URL, duration.
3. Add rate limiting and timeout.

Acceptance:

1. Metadata fetch works for a single known `videoID`.
2. 429/5xx are retried with backoff and do not crash runtime.

### Step 4: Hook Metadata Service into Playback + Queue Lifecycle

1. In `internal/music/manager.go`:
   - request metadata for now playing when track is selected/started.
   - prefetch metadata for next queue items.
2. Keep all calls async/non-blocking.

Acceptance:

1. Playback startup latency unchanged.
2. Metadata starts appearing shortly after play/enqueue.

### Step 5: Publish Metadata Events

1. Add `metadata_updated` to app events in `internal/core/app/dto/event.go`.
2. Publish on successful metadata upsert.
3. SSE bridge should forward it without extra changes (generic event forwarding already exists).

Acceptance:

1. `GET /api/events` emits `metadata_updated` payloads.

### Step 6: Extend API Views and UI

1. Extend:
   - `internal/core/app/ports/inbound.go` with `artist`, `album`.
   - `internal/adapters/inbound/http/legacy_bridge.go` to populate these fields.
2. Update frontend:
   - `web/src/lib/types.ts`
   - `web/src/routes/home-data.ts`
   - `web/src/routes/now-playing.tsx`
   - queue UI components once added.

Acceptance:

1. Now playing displays artist/album when available.
2. UI updates after metadata events (no page reload).

### Step 7: Aggressive Prefetch + Guardrails

1. Trigger prefetch on:
   - search results (top N),
   - enqueue,
   - queue reorder/change,
   - autoplay recommendations.
2. Add limits:
   - global token bucket,
   - max concurrent metadata workers,
   - backoff/cooldown on repeated 429.

Acceptance:

1. Metadata coverage improves without API meltdown.
2. Request throughput remains bounded.

## Resume Checklist (When Picking Up Later)

1. Run: `go test ./cmd/jabba ./internal/platform/storage/sqlite ./internal/adapters/outbound/state`
2. Confirm DB file and migration: `library/state/jabba.db` contains `track_metadata`.
3. Start at "Explicit Next Action Plan / Step 1" and implement in order.

## Architecture

### Core Concepts

1. `TrackMetadata` model
   - `videoId` (primary key)
   - `title`
   - `artist`
   - `album`
   - `coverUrl`
   - `durationSec` (optional)
   - `provider`
   - `fetchedAt`
   - `lastAccessedAt`
   - `sourceVersion` (optional, for schema/provider evolution)

2. `MetadataService` (application service)
   - `Get(videoID)` returns best known metadata fast (memory -> sqlite).
   - `Prefetch(videoIDs)` schedules async fetch with dedupe/rate limiting.
   - `Refresh(videoID)` forces background refresh.
   - Publishes `metadata_updated` events when new data is persisted.

3. `MetadataProvider` (outbound port)
   - YouTube implementation uses Innertube player call.
   - Returns normalized `TrackMetadata`.

4. Two-layer cache
   - L1: in-memory cache (hot, low latency).
   - L2: SQLite table (persistent cache).

## Data Flow

### Play Path (Non-blocking)

1. Track selected for playback.
2. Playback starts immediately (`StartStream`, `PlayOpus`).
3. Manager requests metadata from `MetadataService.Get(videoID)`:
   - If present in cache, attach immediately.
   - If missing/stale, service schedules async fetch and returns current best data.
4. When async fetch completes:
   - persist to sqlite
   - update memory cache
   - emit `metadata_updated` event
   - emit compatibility events (`now_playing` / `queue_updated`) if needed

### Prefetch Triggers

1. Search results: prefetch top N IDs (e.g. 5).
2. Enqueue: prefetch enqueued ID immediately.
3. Queue changes: prefetch next M upcoming IDs (e.g. 10).
4. Autoplay recommendations: prefetch recommendation batch IDs.

## Event Model

### New Event

`metadata_updated`

Payload:

1. `videoId`
2. `title`
3. `artist`
4. `album`
5. `coverUrl`
6. `durationSec` (optional)
7. `fetchedAt`

### Compatibility

Keep existing events:

1. `now_playing`
2. `queue_updated`

During migration, emit both:

1. `metadata_updated` for precise updates
2. existing events so current UI keeps working

## Rate Limiting and Safety

1. Global token bucket for metadata provider requests.
   - Initial recommendation: `1 req/sec`, burst `2`.
2. Per-video dedupe (singleflight behavior).
3. Backoff retries for transient errors.
4. 429/5xx tracking with temporary slowdown (mini circuit breaker).
5. Hard timeout per request (e.g. 4-6s).

## SQLite Design

Table: `track_metadata`

Columns:

1. `video_id TEXT PRIMARY KEY`
2. `provider TEXT NOT NULL`
3. `title TEXT`
4. `artist TEXT`
5. `album TEXT`
6. `cover_url TEXT`
7. `duration_sec INTEGER`
8. `fetched_at DATETIME NOT NULL`
9. `last_accessed_at DATETIME NOT NULL`
10. `source_version INTEGER NOT NULL DEFAULT 1`

Indexes:

1. `idx_track_metadata_last_accessed_at`
2. `idx_track_metadata_fetched_at`

Maintenance:

1. TTL refresh check (e.g. refresh if older than 7-30 days).
2. Periodic cleanup job for cold rows.

## API / DTO Changes

Extend queue and now-playing view models with:

1. `artist`
2. `album`
3. `coverUrl` (already in progress)

Optional future:

1. `metadataStatus` (`missing`, `loading`, `ready`, `stale`)

## Implementation Phases

### Phase 1: Foundation

1. Add metadata DTO + ports (`MetadataService`, `MetadataProvider`, `MetadataStore`).
2. Add in-memory cache + singleflight dedupe.
3. Add new `metadata_updated` app event type.

Done when:

1. Metadata can be fetched and cached in memory.
2. Event emitted on successful fetch.

### Phase 2: Persistent Cache

1. Add sqlite metadata store adapter.
2. Wire service read path: memory -> sqlite -> provider fetch.
3. Add TTL and access-time updates.

Done when:

1. Metadata survives process restart.
2. Cache hit path avoids provider request.

### Phase 3: Aggressive Prefetch

1. Hook prefetch triggers (search/enqueue/queue/autoplay).
2. Add rate limiter and backoff logic.
3. Add metrics/logging for request volume and errors.

Done when:

1. Metadata appears quickly for queued and now-playing tracks.
2. Provider request rate remains bounded.

### Phase 4: UI Integration

1. Display `artist`/`album` in now-playing and queue.
2. Subscribe to `metadata_updated` SSE event.
3. Keep fallback behavior for missing metadata.

Done when:

1. UI updates automatically as metadata arrives.
2. No blocking spinner on playback startup.

## Risks

1. API limit pressure from aggressive prefetch.
   - Mitigate with strict throttling + dedupe + TTL.
2. Inconsistent metadata shape from provider responses.
   - Normalize in provider adapter and version schema.
3. Increased complexity in event fanout.
   - Keep one app event model; adapters only forward.

## Success Criteria

1. Playback start latency unchanged.
2. Artist/album/cover appear quickly for now playing and queue.
3. Metadata persists across restart.
4. SSE-driven UI updates without manual refresh.
