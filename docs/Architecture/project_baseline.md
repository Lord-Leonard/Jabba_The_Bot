# Jabba The Bot - Project Baseline (Code Map)

## Purpose

This document is a working baseline of the current repository so future refactors (single program + hexagonal architecture + web UI) can build on a shared map.

## Current High-Level State

- The repo contains two main runtime paths:
1. Discord bot runtime (`cmd/main.go`)
2. TeamSpeak bot runtime (`cmd/teamspeak/main.go`)
- The most advanced music orchestration is in `internal/music/manager.go`.
- Music source/provider is currently YouTube-only (`internal/music/provider/youtube.go` using `pkg/youtube/music` API client + `yt-dlp` + `ffmpeg`).
- Audio demuxing is custom and local (`pkg/audio/webm`, `pkg/audio/ogg`, `pkg/ebml`).
- TeamSpeak support is deep and custom (handshake, protocol, packet, crypto, session handling under `pkg/teamspeak/internal/...`).
- Discord has both:
1. A newer custom gateway/service flow (`internal/discord` + `pkg/discord/internal`)
2. Legacy experimental code (`discordVoice.go`, `airhornMain.go`, `internal/discord/handlers.go`, `internal/voice/voice.go`)

## Entrypoints

### 1) TeamSpeak entrypoint (music-first path)

- `cmd/teamspeak/main.go`
- Bootstraps TeamSpeak client, creates music manager, wires chat commands to playback control.
- Commands currently handled via chat:
1. `!ping`
2. `!play <query>` (currently also calls `manager.Stop()` before enqueue)
3. `!queue <query>`
4. `!skip`
5. `!stop`
- Updates TS nickname/description with now-playing metadata via `SetOnFirstBytes` callback.

### 2) Discord entrypoint

- `cmd/main.go`
- Initializes Discord bot, registers slash commands (`ping`, `play`), starts event loop.
- `cmd/handlers.go` resolves command option, joins user voice channel, creates per-session music manager, starts playback.

## Music Domain (Current Core)

### Manager

- `internal/music/manager.go`
- Responsibilities:
1. Queue management (`Enqueue`, `ClearQueue`, `QueueLength`)
2. Playback worker loop (`playbackLoop`)
3. Playback interruption (`Skip`, `Stop`) via interruptible stream wrapper
4. Autoplay recommendation queueing
5. Near-end prefetch and cache warm-up
- Internal queue item supports:
1. raw query
2. pre-resolved `Track`
3. autoplay marker
4. request timestamp

### Provider

- `internal/music/provider/youtube.go`
- `Search`: uses reverse-engineered YouTube Music endpoints (`pkg/youtube/music`)
- `Download`: runs `yt-dlp -> ffmpeg(libopus/webm) -> AsyncBuffer`
- Supports disk cache (`library/cache/webm` by default) with `.part` temp finalization
- Supports prefetch for recommended tracks
- First-byte callback used for startup latency tracking and user feedback

- `internal/music/provider/track.go`
- `Track.StartStream()` resolves stream by calling provider download then attaching custom webm demuxer.

### Output adapters

- `internal/music/output/teamspeak.go`: pushes opus frames to `teamspeak.Client.SendOpusStream`
- `internal/music/output/discord.go`: pushes opus frames to Discord voice session `PlayTest`

## Audio / Media Pipeline

- `pkg/audio/async_buffer.go`: unbounded producer-consumer in-memory buffer used by download pipeline.
- `pkg/audio/webm/demuxer.go`: custom EBML/WebM parser that yields opus frames and exposes `Duration()`/`Position()`.
- `pkg/audio/ogg/demuxer.go`: custom OGG parser with page/segment packetization.
- `pkg/ebml/reader.go`: low-level EBML reader used by webm demuxer.

## TeamSpeak Stack

### Public surface

- `pkg/teamspeak/connect.go`: config, client creation, init helpers
- `pkg/teamspeak/client.go`: run loop, typed message handlers, command send, opus stream send
- `pkg/teamspeak/messages.go`: typed command/message structs (`TextMessage`, `CommandError`, etc.)
- `pkg/teamspeak/lifecycle.go`: convenience start/run wrappers

### Protocol internals

- `pkg/teamspeak/internal/client/client.go`: UDP transport, handshake integration, packet sealing/opening, command/voice send/recv
- `pkg/teamspeak/internal/handshake/*`: TS3 crypto/handshake steps (hashcash, key flow, RSA puzzle, license logic)
- `pkg/teamspeak/internal/protocol/*`: packet, binary codec, command encoding/decoding, AEAD, voice framing, quicklz
- `pkg/teamspeak/internal/packet/*`: constructors for command/ack/voice/handshake packets
- `pkg/teamspeak/internal/session/session.go`: packet ID + generation tracking, handshake state container

### Supporting material

- `pkg/teamspeak/docs/*`: protocol notes, CSV/TOML maps, reverse engineering references, Java inspiration files

## Discord Stack

### Active custom implementation

- `internal/discord/bot.go`: event orchestration, slash command dispatch, voice-state tracking, session bootstrapping
- `internal/discord/commandContext.go`: interaction reply/edit/followup wrappers
- `pkg/discord/internal/discord.go`: gateway websocket, heartbeat/identify, command registration, interaction HTTP API, voice state update
- `pkg/discord/internal/voice/Session.go`: Discord voice websocket + UDP setup, encryption, speaking events, opus frame send loops

### Legacy / experimental

- `internal/discord/handlers.go`: older interaction flow (mostly commented)
- `internal/voice/voice.go`: older voice connection implementation (commented out)
- `discordVoice.go`, `airhornMain.go`: legacy examples / old code paths not integrated into main runtime

## YouTube Music API Layer

- `pkg/youtube/music/client.go`: endpoint and request client setup (`/search`, `/next`)
- `pkg/youtube/music/search.go`: search request build + parse
- `pkg/youtube/music/next.go`: recommendation/next request build + parse
- `pkg/youtube/music/parse.go`: response parsing logic
- `pkg/youtube/music/types.go`: payload + response structs
- Test coverage exists for parsing and API methods under `pkg/youtube/music/*_test.go`

## Architecture Notes / Docs

- `docs/Architecture/frontend.md`: strong draft for command/query/events model + SSE-powered web UI
- `docs/Architecture/backend.md`: currently empty
- `docs/Architecture/WebsocketVoice.excalidraw`: voice flow diagram asset

## Repository Areas By Intent

### Runtime app code

- `cmd/*`
- `internal/*`
- `pkg/*`

### Development artifacts and fixtures

- `requests/*`, `internal/requests/*`: Bruno request collections / API experiments
- `docs/events/*.json`: captured Discord event payloads
- `library/cache/*`: cached media artifacts
- `scripts/*`: build/deploy helpers
- top-level binaries and audio files: local artifacts (`bot`, `dist/bot`, `output.wav`, etc.)

## Observed Seams For Refactor

1. Multiple entrypoints and duplicate voice implementations create drift.
2. Music manager is concrete on provider type (`*YouTubeProvider`) instead of domain interface.
3. Bot/platform logic and transport details are mixed with orchestration concerns.
4. State changes and command handling are spread across chat handlers and manager internals; no single app service boundary yet.
5. Web API/SSE architecture exists as plan (`frontend.md`) but has no backend implementation yet.

## Suggested Next Step For Restructure

Use this repository map as the input for phase 1 extraction:

1. Define domain ports (`PlayerOutput`, `TrackProvider`, `QueueStore`, `EventBus`).
2. Move `internal/music/manager.go` toward application service with explicit command API.
3. Treat Discord and TeamSpeak as adapters.
4. Add HTTP+SSE adapter for web UI control/state.
5. Keep current provider + demuxers as infrastructure adapters behind interfaces.

