# Jabba The Bot - Hexagonal Restructure Plan

## Goal

Move from multiple patched entrypoints to one clean application with:

1. a stable domain core
2. explicit ports (interfaces)
3. replaceable adapters (Discord, TeamSpeak, Web API/UI)
4. infrastructure modules isolated from business logic

## Target Principles

1. Domain/Application code must not import Discord/TeamSpeak/YT/ffmpeg packages.
2. All external integrations are adapters behind ports.
3. Runtime composition happens only in `cmd/jabba`.
4. One command model for all controllers: chat, slash command, HTTP.
5. One event model for all observers: chat feedback, logs, SSE, metrics.

## Proposed Target Structure

```text
cmd/
  jabba/
    main.go                # single runtime composition root

internal/
  core/
    domain/
      playback/
        types.go           # Track, QueueItem, PlaybackState, Events
        errors.go
      queue/
        queue.go           # pure queue rules
    app/
      ports/
        inbound.go         # command/query interfaces used by controllers
        outbound.go        # provider/player/state/event ports
      service/
        music_service.go   # use-case orchestration (play, enqueue, skip...)
        autoplay.go
        prefetch.go
      dto/
        command.go
        query.go
        event.go

  adapters/
    inbound/
      discord/
        command_router.go  # maps slash/chat -> app commands
      teamspeak/
        command_router.go  # maps ts chat -> app commands
      http/
        server.go
        handlers_commands.go
        handlers_queries.go
        sse.go
    outbound/
      playback/
        discord_player.go
        teamspeak_player.go
      music/
        youtube_provider.go
      eventbus/
        inproc_bus.go
      state/
        memory_store.go

  platform/
    config/
      config.go
    logging/
      logger.go
    lifecycle/
      rungroup.go

pkg/
  teamspeak/               # keep protocol/transport implementation
  discord/                 # keep gateway/voice transport implementation
  audio/                   # keep demuxers/buffers
  ebml/
  youtube/music/
```

## Port Design (Core Contracts)

### Inbound ports (used by adapters)

1. `MusicCommands`
   - `Enqueue(query string, source RequestSource) (QueuePosition, error)`
   - `Skip() error`
   - `Pause() error`
   - `Resume() error`
   - `Stop(clearQueue bool) error`
   - `Remove(itemID string) error`
   - `Move(itemID string, toIndex int) error`
2. `MusicQueries`
   - `GetState() PlaybackState`
   - `GetQueue() []QueueItemView`
   - `GetNowPlaying() *NowPlayingView`

### Outbound ports (implemented by infrastructure/adapters)

1. `TrackProvider`
   - `Search(ctx, query) ([]TrackCandidate, error)`
   - `Recommend(ctx, seedID, limit) ([]TrackCandidate, error)`
   - `OpenStream(ctx, trackID) (TimedAudioStream, StreamMeta, error)`
   - `Prefetch(ctx, trackID) error`
2. `AudioPlayer`
   - `Play(stream TimedAudioStream) error`
   - `Stop() error` (optional if stream interruption is in app layer)
3. `StateStore`
   - `Load() (AppState, error)`
   - `Save(AppState) error`
4. `EventPublisher`
   - `Publish(AppEvent)`

## Runtime Composition (Single Program)

`cmd/jabba/main.go` should:

1. Load config/env
2. Build shared core service (`MusicService`)
3. Build adapters:
   - TeamSpeak inbound + outbound player
   - Discord inbound + outbound player
   - HTTP inbound (commands/queries/SSE)
   - YouTube outbound provider
4. Register event subscriptions (chat notifications, SSE fanout, logs)
5. Run all components under one lifecycle group with context cancellation

## Mapping Current Code -> Target Modules

1. `internal/music/manager.go`
   - split into:
   - `internal/core/app/service/music_service.go`
   - `internal/core/app/service/autoplay.go`
   - `internal/core/app/service/prefetch.go`
2. `internal/music/provider/youtube.go`
   - move to `internal/adapters/outbound/music/youtube_provider.go`
   - keep `pkg/youtube/music` client as low-level dependency
3. `internal/music/output/teamspeak.go`
   - move to `internal/adapters/outbound/playback/teamspeak_player.go`
4. `internal/music/output/discord.go`
   - move to `internal/adapters/outbound/playback/discord_player.go`
5. `cmd/teamspeak/main.go` command handling
   - move chat parsing to `internal/adapters/inbound/teamspeak/command_router.go`
6. `cmd/main.go` + `cmd/handlers.go`
   - move slash parsing to `internal/adapters/inbound/discord/command_router.go`
7. `docs/Architecture/frontend.md` API model
   - implement under `internal/adapters/inbound/http/*`
8. keep `pkg/teamspeak/*`, `pkg/discord/*`, `pkg/audio/*`, `pkg/ebml/*`
   - treated as infrastructure libraries initially

## Migration Plan (Incremental, Safe)

## Phase 0 - Freeze and Guardrails

1. Keep current behavior as baseline.
2. Add smoke tests for:
   - enqueue/play/skip/stop
   - queue behavior
   - Teamspeak chat command routing (unit-level if possible)
3. Mark legacy files as deprecated in comments/docs.

Done when:

1. Current behavior is reproducible with tests or scripted manual checks.
2. No new feature work lands outside migration path.

## Phase 1 - Extract Core Contracts

1. Create `internal/core/domain` types and playback state model.
2. Create app ports in `internal/core/app/ports`.
3. Introduce `MusicService` with same public command surface as current manager.
4. Keep internals calling current provider/player adapters through new interfaces.

Done when:

1. Old `Manager` is replaced by `MusicService` in at least one runtime path.
2. Core layer imports no `pkg/teamspeak`, `pkg/discord`, or `os/exec`.

## Phase 2 - Adapterize Existing Integrations

1. Move YouTube provider into outbound adapter implementing `TrackProvider`.
2. Move Discord and TeamSpeak playback wrappers into outbound adapters implementing `AudioPlayer`.
3. Move Discord/TeamSpeak command parsing into inbound adapters using `MusicCommands`.

Done when:

1. Chat/slash code only translates transport payloads to app commands.
2. Playback logic only lives in core app service.

## Phase 3 - Single Composition Root

1. Add `cmd/jabba/main.go`.
2. Compose all adapters and run with shared context and lifecycle group.
3. Retire `cmd/main.go` and `cmd/teamspeak/main.go` after parity.

Done when:

1. One binary can run selected adapters via config flags.
2. Shutdown behavior is coordinated and graceful.

## Phase 4 - Web UI Backend (HTTP + SSE)

1. Implement endpoints from `frontend.md`:
   - `POST /api/commands`
   - `GET /api/state`
   - `GET /api/queue`
   - `GET /api/now-playing`
   - `GET /api/events` (SSE)
2. Connect HTTP handlers to `MusicCommands`/`MusicQueries`.
3. Publish internal app events to SSE subscribers.

Done when:

1. Web UI can fully control and observe playback state without direct Discord/TS coupling.

## Phase 5 - Cleanup and Hardening

1. Remove or archive legacy voice code:
   - `airhornMain.go`
   - `discordVoice.go`
   - `internal/voice/voice.go`
   - stale experimental handlers
2. Add integration tests for command->state->event flow.
3. Add configuration validation and structured startup logs.

Done when:

1. Repo has one obvious way to run the app.
2. Architecture boundaries are enforced by package dependencies.

## Recommended Initial Milestone (What to Build First)

Build this first slice end-to-end:

1. `MusicService` + ports + in-memory state
2. TeamSpeak command adapter calling `MusicService`
3. TeamSpeak player + YouTube provider as outbound adapters
4. `cmd/jabba/main.go` only with TeamSpeak enabled

This gives a production-relevant vertical slice before adding Discord and HTTP.

## Risks and Mitigations

1. Risk: Regression in playback timing/interruption.
   - Mitigation: keep current stream interruption logic intact during first extraction.
2. Risk: Concurrency issues in new state/event model.
   - Mitigation: single serialized command loop in app service + immutable event payloads.
3. Risk: Scope explosion (UI + Discord + TeamSpeak simultaneously).
   - Mitigation: migrate one adapter path at a time with parity checkpoints.

## Definition of Success

1. One binary runs bot + API.
2. Domain/application layer is transport-agnostic.
3. Discord/TeamSpeak/Web are just adapters.
4. New features (playlist import, persistence, multi-provider) require adding adapters, not rewriting core.

