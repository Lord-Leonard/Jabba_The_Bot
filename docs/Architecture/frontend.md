# TeamSpeak Music Bot — Control API & UI Architecture

## Goals

* Provide a lightweight web UI to control the TeamSpeak music bot.
* Support real-time queue and playback updates.
* Keep implementation simple (single user / small trusted group).
* Allow development with Vite dev server and production static hosting.
* Avoid unnecessary infrastructure.

---

## Core Architectural Model

The system follows a **command + state + events** model.

### Commands

Commands represent **user intent** and are sent via HTTP.

Examples:

* pause
* resume
* skip
* enqueue track
* remove queue item
* reorder queue
* seek
* set volume

Commands are **not guaranteed to succeed**.
They return an acknowledgement and optional immediate state snapshot.

### Events

Events represent **facts that already happened**.

Examples:

* playback state changed
* queue updated
* track started
* progress update
* error occurred

Events are broadcast to all connected UI clients using **Server-Sent Events (SSE)**.

### Queries

Queries retrieve the current state and are idempotent.

Examples:

* get queue
* get now playing
* get playback state

---

## Why This Separation Exists

* Commands express intent and may be rejected.
* Events express confirmed state transitions.
* Queries provide synchronous state reads.

This prevents mixing "requested action" with "actual result".

---

## High-Level Components

### Bot Core

Responsible for:

* TeamSpeak interaction
* playback
* queue
* state machine
* emitting internal events

---

### HTTP API Layer

Responsible for:

* accepting commands
* serving state queries
* streaming events (SSE)
* serving static UI in production

Must **not** contain playback logic.
Thats where the [Music Manager](/internal/music/manager.go) comes in

---

### UI (React + Vite)

Responsible for:

* rendering state snapshot
* sending commands
* subscribing to SSE updates
* optimistic UI updates (optional)

---

## Playback State Machine

Suggested states:

* `stopped`
* `loading`
* `playing`
* `paused`

Command validity:

| Command | Valid States    |
| ------- | --------------- |
| pause   | playing         |
| resume  | paused          |
| skip    | playing, paused |
| stop    | playing, paused |
| enqueue | any             |

Invalid commands return `409 Conflict`. - Sure?

---

## HTTP API Design

### Commands

```
POST /api/commands
```

Example request:

```json
{
  "type": "pause",
  "payload": {},
  "clientRequestId": "uuid"
}
```

Response (accepted):

```json
{
  "accepted": true,
  "commandId": "cmd_123"
}
```

Response (rejected):

```json
{
  "accepted": false,
  "error": {
    "code": "NOT_PLAYING",
    "message": "Nothing is playing"
  }
}
```

`clientRequestId` enables idempotency.

---

### Queries

```
GET /api/state
GET /api/queue
GET /api/history
GET /api/now-playing
```

---

### Events (SSE)

```
GET /api/events
```

The first message must be a full snapshot. - Not sure if I like this. as there are queries for the current state...

Example:

```
event: snapshot
data: { state, queue, nowPlaying }
```

Incremental events:

```
event: playback_state_changed
event: queue_updated
event: now_playing
event: progress
event: error
```

Events should contain enough data to update UI without an additional fetch.

---

## Concurrency Model (Go)

Recommended pattern:

* command channel
* event fan-out broadcaster
* immutable state struct protected by RWMutex

Flow:

1. HTTP handler pushes command into command channel
2. playback loop processes command
3. playback loop mutates state
4. playback loop emits event
5. SSE broadcaster forwards event to clients

This prevents race conditions and state corruption.

---

## Development Setup

### Processes

* Go bot + API → `localhost:8080`
* Vite dev server → `localhost:5173`

### Vite Proxy

```ts
server: {
  proxy: {
    "/api": "http://localhost:8080"
  }
}
```

UI always uses relative paths.

---

## Production Setup

Two options:

### Option A (recommended)

Go binary serves:

* `/api`
* `/events`
* static UI (`dist/`)

UI is embedded or copied to server.

### Option B

Static files served separately by web server.

---

## UI Integration Strategy

### Startup

1. Fetch `/state`
2. Connect SSE
3. Apply incremental updates

### Command Flow

1. Send command
2. Apply optimistic update (optional)
3. Wait for SSE confirmation

---

## Future Extensions

* multi-user liked songs
* persistent queue storage
* reconnect-safe SSE with last-event-id
* volume normalization
* playback history
* playlist import

---

## Non-Goals

* authentication
* distributed scaling
* multi-node playback coordination

---

## Implementation Notes

* keep event payloads self-contained
* log all state transitions
* keep playback state changes serialized
* ensure SSE clients are cleaned up on disconnect
* consider heartbeat events to keep connections alive

---

## Summary

The system is intentionally simple:

* HTTP commands for intent
* SSE for synchronization
* state machine driven playback loop
* minimal API surface
* Vite proxy during development
* static serving in production

This keeps the bot responsive, easy to reason about, and trivial to extend.
