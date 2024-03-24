# WebM Seek Refactor Plan (Low Latency + Full Seek)

## Goal

Keep current fast startup behavior (audio starts almost immediately) while adding true bidirectional seek for web UI and bot commands.

## Current State Summary

- Playback path is stream-first:
  - `yt-dlp` downloads media to stdout.
  - `ffmpeg` transcodes to Opus-in-WebM and writes to stdout.
  - stdout goes into `audio.AsyncBuffer`.
  - `webm.Demuxer` consumes an `io.Reader` and emits Opus frames.
- This gives great startup latency, but no random access because the source is not seekable.
- `webm.Demuxer` tracks media time (`currentTimestamp`) but not byte offsets or seek index.
- HTTP command surface does not expose `seek` yet.

## Architecture Decision

Use a **growing seekable cache file** as the playback source of truth.

- Writer: `yt-dlp -> ffmpeg -> append to .webm file`.
- Reader: demuxer reads from the same file via `io.ReadSeeker`.
- Seek is allowed whenever target bytes are already written.
- If target bytes are not yet present, return `buffering/loading` and resume when available.

This preserves low latency because playback can start as soon as the first clusters are written, without waiting for full download.

## What Is Missing In The Demuxer

1. Seek-capable input
- `NewDemuxer` currently accepts `io.Reader`.
- Must support `io.ReadSeeker` (or `io.ReadSeekCloser`) for random access.

2. Byte offset awareness
- Reader/demuxer currently does not expose absolute read offsets.
- Need current absolute file offset and segment-relative offset.

3. Time-to-offset index
- Need index entries while parsing:
  - `cluster timestamp` -> `cluster byte offset`.
  - Optional finer entries (keyframe/block boundaries) if needed later.
- Use parsed `Cues` when available.
- Fall back to runtime-built cluster index for live/growing files.

4. Seek API
- Add demuxer method:
  - `SeekToTime(target time.Duration) (SeekResult, error)`
- Responsibilities:
  - Locate nearest seekable offset <= target.
  - `Seek` underlying reader.
  - Reset parser state for continued demuxing.
  - Prime until first valid audio block after target.

5. Stream status for UI
- Report:
  - current position
  - known duration
  - buffered duration (how far written/indexed)
  - seekable range(s)
  - buffering state if target not yet available

## What WebM Contributes To Seeking

- `Cluster.Timestamp` + block timecode gives media timestamps.
- `SeekHead` maps to top-level elements (Info/Cues/etc), not directly to playback time.
- `Cues` are the proper time-based seek structure.
- For live/growing streams where complete cues may not exist yet, runtime indexing during demux is required.

## Pipeline Changes Needed

1. Replace async buffer as primary playback source
- Current: `ffmpeg stdout -> AsyncBuffer -> Demuxer(reader)`.
- New:
  - `ffmpeg stdout -> GrowingFileWriter(.webm + metadata state)`.
  - `Demuxer` reads from a `GrowingFileReader` (`Read + Seek`) over the same file.

2. Introduce a stream session object
- New per-track runtime object (example: `StreamSession`) holding:
  - file path + file handle(s)
  - bytes written
  - indexed seek points
  - duration/buffered duration
  - writer completion/error state
  - condition variable / channel for “new bytes available”

3. Add blocking reads aware of growth
- Reader should:
  - block briefly when EOF is reached but writer is still active
  - return EOF only when writer finished and no more bytes

4. Add seek command path
- Extend `MusicCommands` port with seek command (`Seek(positionSec or ratio)`).
- Implement in bridge and manager.
- Manager forwards seek to current stream session/demuxer.
- Return explicit errors for unsupported/unavailable states.

5. Add progress + seekability to query API
- Fill `NowPlayingView.PositionSec` and `DurationSec`.
- Add fields for `bufferedSec`, `seekableUntilSec`, `isBuffering`, `canSeek`.
- Emit SSE `progress` regularly and on seek state transitions.

## Suggested Incremental Delivery

### Phase 1: Observability and contracts
- Add domain interfaces for seekable timed stream:
  - `Position()`, `Duration()`, `SeekTo(time.Duration)`, `SeekableRange()`.
- Wire `NowPlaying` progress fields and SSE `progress`.
- No functional seek yet.

### Phase 2: Growing file transport
- Implement writer that appends ffmpeg output to cache file while exposing write progress.
- Implement growth-aware read/seek wrapper for playback.
- Switch demux source from async buffer to growing file reader.

### Phase 3: Demuxer seek primitives
- Add offset tracking + runtime index in WebM demuxer.
- Add `SeekToTime` and parser reset behavior.
- Validate seek correctness on cached complete files first.

### Phase 4: Live seek on in-progress streams
- Enable seeks within already buffered/indexed range.
- Return buffering status for targets beyond current write/index frontier.
- Auto-resume when target becomes available.

### Phase 5: API/UI command integration
- Add `/api/commands` type `"seek"` payload `{positionSec}`.
- UI slider:
  - enabled when `canSeek`.
  - loaded vs buffered bar.
  - loading state for future positions.

## Risks and Mitigations

- Incomplete metadata in live streams
  - Mitigation: runtime cluster index; do not depend only on `Cues`.
- Concurrency bugs between writer/reader/seek
  - Mitigation: single session owner + explicit locks/condvars + cancellation rules.
- Seek landing on non-decodable frame boundary
  - Mitigation: seek to nearest indexed cluster before target, then decode forward.
- Regression in startup latency
  - Mitigation: start playback after first valid cluster/block written; avoid full prebuffer.

## Acceptance Criteria

- Playback starts with latency comparable to current stream-first path.
- User can seek backward and forward on fully cached tracks.
- User can seek within buffered region of in-progress tracks.
- Out-of-buffer seeks show buffering state and recover automatically.
- API exposes stable `position`, `duration`, `buffered/seekable` info.
- No crashes/deadlocks on rapid repeated seek requests.

## Immediate Next Implementation Steps

1. Add seek-capable stream interface in `internal/music/provider/track.go`.
2. Create `StreamSession` abstraction in provider layer.
3. Swap provider output from `AsyncBuffer` to growing file writer+reader.
4. Extend WebM demuxer constructor to accept read-seek input and maintain seek index.
5. Add manager + port + HTTP command support for `seek`.
6. Add progress SSE payload and frontend slider states.

