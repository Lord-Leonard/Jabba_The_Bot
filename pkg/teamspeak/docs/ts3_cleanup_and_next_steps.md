# TS3 Client Cleanup + Next Steps

## Status
- Handshake now fully succeeds and server sends initserver/channellist packets.
- Public API is clean: `ts.Connect`, `ts.Client.Run`, `ts.Client.RegisterCommand`.
- Low-level protocol/handshake code moved to internal packages.

## Current behavior
- Connection still drops after ~30s.
- Ping/pong is sent and logged, but server still disconnects.

## Root cause (likely)
- We still do NOT parse fragmented/compressed command packets (CP/FR).
- That means we never parse `initserver`, so we never learn/set `aclid` (client ID).
- Without CID, outgoing packets (pong/ack) use `cid=0` and are ignored, leading to disconnect.

## What was changed (high level)
- Created internal packages:
  - `pkg/ts/internal/handshake` (handshake/identity/init/crypto helpers)
  - `pkg/ts/internal/proto` (packet/crypto/command)
  - `pkg/ts/internal/client` (UDP transport + recv/send + per-type PID counters)
- Public API:
  - `pkg/ts/connect.go`: `Config`, `DefaultConfig`, `Connect` and `Client.Handshake`
  - `pkg/ts/client.go`: `Run`, `RegisterCommand`, `Command` (public)
- `cmd/teamspeak/main.go` now just calls `Connect` + `Run`.
- Pong now uses payload = ping PID, and ACKs are sent for commands.
- Packet ID counters are per packet type (spec requirement).

## Known issues
- No fragment merge (FR) or QuickLZ decompression (CP).
- No client ID update after initserver.
- No client-side ping sending (only pong).

## Next steps
1) Implement fragment merge for command packets (FR flag):
   - Buffer and reassemble split commands by PID sequence until last FR.
   - Only then attempt decrypt/parse.

2) Implement QuickLZ level 1 decompression for CP flag:
   - Use a library for now (GPL) if acceptable.
   - After decompress, feed to command parser.

3) Parse `initserver`:
   - Extract `aclid` (client ID).
   - Store in internal client and use in outgoing headers.

4) After CID is set, verify:
   - pongs are accepted,
   - connection stays alive.

## Decision needed
QuickLZ implementation:
- Fast path: use `dgryski/go-quicklz` (GPL)
- Clean path: implement a minimal QuickLZ level 1 decoder internally

## File map
- Public API: `pkg/ts/connect.go`, `pkg/ts/client.go`
- Internal protocol: `pkg/ts/internal/proto/*`
- Internal handshake: `pkg/ts/internal/handshake/*`
- Internal transport: `pkg/ts/internal/client/client.go`

