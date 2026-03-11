# Jabba The Bot — Playback Architecture

## Overview

The architecture is split into clearly separated layers, each owning a single concern.
No layer knows more about its neighbours than the interface it talks to.

---

## The Full Diagram

```
  User Command (TS chat or HTTP)
      │
      ├── TS Command Handler
      └── HTTP Handler
              │
              │  both talk to
              ▼
      ┌───────────────┐
      │  MusicService │  ← single facade over the entire playback stack
      └───────┬───────┘
              │
              ▼
┌─────────────────────────────────┐
│           Resolver              │
│  ┌──────────────────────────┐   │   Track {
│  │  AppleMusicResolver      │   │       Title    string
│  │  ISRC, canonical metadata│   │       Artist   string
│  └──────────────────────────┘   │       Duration time.Duration
│  ┌──────────────────────────┐   │       ISRC     string  // set by AppleMusic
│  │  YouTubeResolver         │   │       VideoID  string  // set by YouTube
│  │  VideoID, fallback       │   │   }
│  └──────────────────────────┘   │
└────────────────┬────────────────┘
                 │ enqueues resolved Track
                 ▼
┌────────────────────────────────────────────────────────┐
│                       Queue                            │
│               ([]Track, mutex, cond)                   │
│                                                        │
│  emits QueueEvents: RunningLow / Drained / Updated     │
└──────┬─────────────────────┬──────────────────┬────────┘
       │                     │                  │
       │ TrackProvider        │ subscribes       │ subscribes
       │ Next(ctx) *Track     │                  │
       │                      ▼                  ▼
       │            ┌──────────────┐   ┌──────────────────┐
       │            │  Autoplay    │   │    Prefetcher    │
       │            │  Controller  │   │  (cache warming) │
       │            │              │   │                  │
       │            │ seeded by    │   │ watches depth    │
       │            │ NowPlaying   │   │ calls            │
       │            │ event        │   │ provider.Prefetch │
       │            │ pushes Track │   │ on next N tracks │
       │            │ back to Queue│   └──────────────────┘
       │            └──────────────┘
       │
       │                            ┌────────────────────────────┐
       │                            │    AudioSourceProvider     │
       │                            │                            │
       │                            │    YouTubeProvider         │
       │                            │    Open(ctx, track)        │
       │                            │                            │
       │                            │  priority chain:           │
       │                            │  1. VideoID → direct       │
       │                            │  2. ISRC → YT search       │
       │                            │  3. Title+Artist → search  │
       │                            └────────────┬───────────────┘
       │                                         │ AudioSource
       ▼                                         │ NextFrame() ([]byte, error)
┌─────────────┐                                  │ FrameDuration() time.Duration
│   Engine    │◄─────────────────────────────────┘
│             │
│  loop       │  wraps source with InterruptibleStream (lives in engine pkg)
│  state      │  ┌─────────────────────────────────────────────┐
│  ctrl chans │  │ InterruptibleStream                         │
│             │  │   source    AudioSource                     │
└──────┬──────┘  │   stopCh    chan struct{}                    │
       │         │   pauseCh   chan struct{}                    │
       │         │                                             │
       │         │ ProvideFrame() — selects on stop/pause/source│
       │         │ FrameDuration() — delegates to source       │
       │         └─────────────────────────────────────────────┘
       │
       │  AudioSink interface
       │  Consume(OpusFrameSource) error
       │
       ├──────────────────────────────────────────┐
       │                                          │
       ▼                                          ▼
┌─────────────────┐                     ┌──────────────────┐
│   TS Adapter    │                     │  Discord Adapter │
│                 │                     │                  │
│ internally      │                     │ passthrough      │
│ composes        │                     │ no transcode     │
│ BitrateAdapter  │                     │                  │
│ (126k → 96k)    │                     └────────┬─────────┘
│ applies gain    │                              │
│ on raw PCM      │                              │ Consume(OpusFrameSource)
└────────┬────────┘                              ▼
         │                               pkg/discord Client
         │ Consume(OpusFrameSource)
         │ calls client.SendOpusStream()
         ▼
      pkg/teamspeak Client
      SendOpusStreamWithCodec()
      owns the timing loop
      uses source.FrameDuration() — no hardcoded 20ms
```

---

## Core Interfaces

```go
// Track is a pure value type — resolved metadata only, no streams, no behaviour.
// Not all fields are populated — depends on which Resolver produced it.
// YouTubeProvider.Open() handles missing fields via its priority chain.
type Track struct {
    Title    string
    Artist   string
    Duration time.Duration

    // Set by whichever Resolver produced this Track.
    // YouTubeResolver    → sets VideoID
    // AppleMusicResolver → sets ISRC (+ canonical Title/Artist)
    ISRC    string
    VideoID string
}

// Resolver turns a raw user query into a resolved Track.
type Resolver interface {
    Resolve(ctx context.Context, query string) (Track, error)
}

// TrackProvider is what the Engine sees. Queue implements this.
type TrackProvider interface {
    Next(ctx context.Context) (*Track, error) // blocks until available or ctx cancelled
}

// AudioSourceProvider opens a resolved Track into a playable stream.
type AudioSourceProvider interface {
    Open(ctx context.Context, track Track) (AudioSource, error)
}

// AudioSource is what comes out of the provider.
type AudioSource interface {
    NextFrame() ([]byte, error)
    FrameDuration() time.Duration
    Seek(pos time.Duration) error
    Position() time.Duration
    Duration() time.Duration
    Close() error
}

// OpusFrameSource is the seam between Engine and Adapters.
// This interface already exists in pkg/teamspeak — add FrameDuration() to it.
type OpusFrameSource interface {
    ProvideFrame() ([]byte, error)
    FrameDuration() time.Duration  // source declares its own timing — no hardcoded 20ms
}

// AudioSink is what the Engine hands frames to.
type AudioSink interface {
    Consume(source OpusFrameSource) error
}
```

---

## Package Layout

```
pkg/
├── teamspeak/              — already built, minimal changes needed
│   └── client.go           — connection, Run() event loop, RegisterMessageHandlerTyped,
│                             SendOpusStream (owns timing loop)
│                             CHANGE: opusFrameInterval removed, use source.FrameDuration()
│                             CHANGE: OpusFrameSource gains FrameDuration() method
│
└── discord/                — already built
    └── client.go

internal/
├── music/
│   ├── track.go              — Track struct, QueueEvent constants
│   ├── service.go            — MusicService facade (see below)
│   │
│   ├── resolver/
│   │   ├── resolver.go       — Resolver interface
│   │   ├── apple.go          — AppleMusicResolver  (sets ISRC + canonical metadata)
│   │   └── youtube.go        — YouTubeResolver     (sets VideoID, fallback)
│   │
│   ├── queue/
│   │   ├── queue.go          — Queue, implements TrackProvider
│   │   └── events.go         — QueueEvent, subscription mechanism
│   │
│   ├── autoplay/
│   │   └── controller.go     — AutoplayController, listens to NowPlayingEvent
│   │
│   ├── prefetch/
│   │   └── prefetcher.go     — Prefetcher, listens to QueueUpdated
│   │
│   ├── provider/
│   │   ├── provider.go       — AudioSourceProvider interface
│   │   └── youtube.go        — YouTubeProvider (priority chain + yt-dlp)
│   │
│   └── playback/
│       ├── engine.go         — Engine: loop, state, ctrl channels
│       ├── interruptible.go  — InterruptibleStream (engine pkg, not teamspeak pkg)
│       └── events.go         — NowPlayingEvent, TrackEndedEvent, ProgressEvent
│
├── adapters/
│   ├── ts/
│   │   └── adapter.go        — TSAdapter: composes BitrateAdapter, calls pkg/teamspeak
│   ├── discord/
│   │   └── adapter.go        — DiscordAdapter: passthrough, calls pkg/discord
│   └── bitrate/
│       └── adapter.go        — BitrateAdapter: decode → gain → re-encode
│
├── commands/
│   ├── ts/
│   │   ├── handler.go        — RegisterMessageHandlerTyped on pkg/teamspeak Client
│   │   └── router.go         — maps "!play", "!skip" etc. to MusicService calls
│   └── http/
│       ├── server.go         — sets up router, injects MusicService
│       └── handlers/
│           ├── playback.go   — POST /play, /skip, /stop, /pause, /resume
│           ├── queue.go      — GET /queue, DELETE /queue
│           └── nowplaying.go — GET /nowplaying, GET /progress (SSE)
```

---

## MusicService — The Shared Facade

Both the TS command handler and the HTTP handlers import this and nothing else
from the music stack.

```go
// internal/music/service.go

type MusicService struct {
    resolver Resolver
    queue    *queue.Queue
    engine   *playback.Engine
}

func (s *MusicService) Play(ctx context.Context, query string) error
    // Resolve(query) → Track → Queue.Enqueue → Engine picks up automatically

func (s *MusicService) Skip() error
func (s *MusicService) Stop() error
func (s *MusicService) Pause() error
func (s *MusicService) Resume() error
func (s *MusicService) Seek(pos time.Duration) error
func (s *MusicService) SetVolume(vol int) error

func (s *MusicService) NowPlaying() *NowPlayingSnapshot
func (s *MusicService) Queue() []QueueSnapshot
```

---

## pkg/teamspeak/client.go — What Changes

Your client is already well structured. Two small changes needed:

```go
// REMOVE this — timing is no longer the client's assumption
const opusFrameInterval = 20 * time.Millisecond

// UPDATE OpusFrameSource — add FrameDuration()
type OpusFrameSource interface {
    ProvideFrame() ([]byte, error)
    FrameDuration() time.Duration  // source declares its own frame size
}

// UPDATE SendOpusStreamWithCodec — use source timing instead of constant
func (c *Client) SendOpusStreamWithCodec(source OpusFrameSource, codec uint8) error {
    nextTick := time.Now()
    for {
        frame, err := source.ProvideFrame()
        if err != nil {
            if errors.Is(err, io.EOF) {
                return c.c.SendVoiceFrame(codec, nil)
            }
            return err
        }

        if err := c.c.SendVoiceFrame(codec, frame); err != nil {
            if isVoiceFrameTooLargeError(err) {
                // Keep this safety net during development.
                // Once BitrateAdapter is in place this should never trigger.
                slog.Warn("dropping oversized opus frame", "size", len(frame))
                nextTick = nextTick.Add(source.FrameDuration()) // ← changed
                // ... drift logic unchanged
                continue
            }
            return err
        }

        nextTick = nextTick.Add(source.FrameDuration()) // ← changed
        // ... drift logic unchanged
    }
}
```

Everything else in your client — `Run()`, `RegisterMessageHandlerTyped`,
`dispatchCommand`, `Send`, `UpdateNickname` — is already correct and unchanged.
`RegisterMessageHandlerTyped` is already the event subscription mechanism
the TS command handler will use.

---

## internal/commands/ts/handler.go

Uses `RegisterMessageHandlerTyped` from your existing client to subscribe
to chat messages. No music logic — only parsing, dispatch, and reply formatting.

```go
type Handler struct {
    service *music.MusicService
    client  *teamspeak.Client
}

func (h *Handler) Register() {
    teamspeak.RegisterMessageHandlerTyped(h.client, "notifytextmessage",
        func(c *teamspeak.Client, msg TextMessage) {
            if !strings.HasPrefix(msg.Text, "!") {
                return
            }
            h.router.Dispatch(msg.InvokerID, msg.TargetChannelID, msg.Text)
        },
    )
}

// router.go maps commands to funcs, each calls MusicService and replies via client.Send
// !play <query>  → service.Play()        → "▶ Queued: <title>"
// !skip          → service.Skip()        → "⏭ Skipped"
// !stop          → service.Stop()        → "⏹ Stopped"
// !pause         → service.Pause()       → "⏸ Paused"
// !resume        → service.Resume()      → "▶ Resumed"
// !queue         → service.Queue()       → formatted list
// !np            → service.NowPlaying()  → "▶ Now playing: ..."
// !seek <time>   → service.Seek()        → "⏩ Seeked to 1:23"
// !vol <0-100>   → service.SetVolume()   → "🔊 Volume: 80"
```

---

## internal/adapters/ts/adapter.go

Wraps the BitrateAdapter internally. Engine hands it an `OpusFrameSource`
and never knows what happens inside.

```go
type TSAdapter struct {
    client *teamspeak.Client
    gain   atomic.Int32
}

func (a *TSAdapter) Consume(source OpusFrameSource) error {
    adapted := bitrate.NewBitrateAdapter(source, 96000, &a.gain)
    return a.client.SendOpusStream(adapted)
}

func (a *TSAdapter) SetGain(g int) {
    a.gain.Store(int32(g))
}
```

---

## internal/adapters/bitrate/adapter.go

Decode → apply gain on raw PCM → re-encode at target bitrate.
Gain is applied here for free since decode/encode is already happening.

```go
type BitrateAdapter struct {
    source  OpusFrameSource
    decoder *opus.Decoder
    encoder *opus.Encoder
    gain    *atomic.Int32
}

func (a *BitrateAdapter) ProvideFrame() ([]byte, error) {
    frame, err := a.source.ProvideFrame()
    if err != nil {
        return nil, err // EOF and stop signals propagate cleanly
    }

    pcm := make([]int16, 960*2) // 960 samples stereo @ 48kHz = 20ms
    if _, err = a.decoder.DecodeInt16(frame, pcm); err != nil {
        return nil, err
    }

    applyGain(pcm, a.gain.Load())

    return a.encoder.EncodeInt16(pcm)
}

func (a *BitrateAdapter) FrameDuration() time.Duration {
    return a.source.FrameDuration() // 20ms in, 20ms out
}
```

---

## Command Flow: `!play Never Gonna Give You Up` (from TS chat)

```
1.  User types "!play Never Gonna Give You Up" in TS channel chat
          │
          ▼
2.  pkg/teamspeak Client.Run() receives notifytextmessage
    dispatchCommand → RegisterMessageHandlerTyped handler fires
          │
          ▼
3.  internal/commands/ts Handler
    strips "!", routes to play handler func
          │
          ▼
4.  MusicService.Play(ctx, "Never Gonna Give You Up")
          │
          ▼
5.  AppleMusicResolver.Resolve(query)
    → Track{Title: "...", Artist: "Rick Astley", ISRC: "GBUM71400099", VideoID: ""}
          │
          ▼
6.  Queue.Enqueue(track)
    emits QueueUpdated
    Engine.Next() unblocks
    emits NowPlayingEvent → AutoplayController seeds on Rick Astley
          │
          ▼
7.  YouTubeProvider.Open(track)
    VideoID empty → ISRC search on YouTube → deterministic match
    yt-dlp → AudioSource (opus 126k, 20ms, 48kHz)
          │
          ▼
8.  Engine wraps in InterruptibleStream
    passes to TSAdapter.Consume()
          │
          ▼
9.  TSAdapter wraps in BitrateAdapter (internally)
    decode → apply gain on PCM → re-encode @ 96k
    calls pkg/teamspeak Client.SendOpusStream()
          │
          ▼
10. Timing loop ticks every source.FrameDuration() (20ms)
    TS packets fly
    TS command handler replies "▶ Now playing: Never Gonna Give You Up — Rick Astley"

    meanwhile:
    AutoplayController fetches recs based on Rick Astley seed
    pushes Tracks to Queue → emits QueueUpdated
    Prefetcher sees QueueUpdated → warms yt-dlp cache for next N tracks
          │
          ▼
11. io.EOF → track ends, Engine loops back to step 6
    next track already queued and prefetched → near-gapless
```

---

## What Owns What

| Concern                                    | Owner                                                 |
|--------------------------------------------|-------------------------------------------------------|
| Raw query → Track                          | Resolver (AppleMusic preferred, YouTube fallback)     |
| Track ordering                             | Queue                                                 |
| Recommendation seeding                     | AutoplayController (NowPlayingEvent)                  |
| Cache warming                              | Prefetcher (QueueUpdated)                             |
| Playback loop, state, skip/pause/stop/seek | Engine                                                |
| Interrupting a stream                      | InterruptibleStream (engine pkg)                      |
| Track → AudioSource                        | AudioSourceProvider (YouTubeProvider, priority chain) |
| Timing loop                                | pkg/teamspeak and pkg/discord clients                 |
| Bitrate adaptation + gain                  | BitrateAdapter (inside TSAdapter)                     |
| TS packet size constraints                 | TSAdapter (Engine never knows)                        |
| TS connection, Run() loop, event dispatch  | pkg/teamspeak/client.go (already built)               |
| Subscribing to TS chat events              | RegisterMessageHandlerTyped (already built)           |
| Parsing + routing TS commands              | internal/commands/ts/                                 |
| HTTP API surface                           | internal/commands/http/handlers/                      |
| Single entry point for all commands        | MusicService                                          |