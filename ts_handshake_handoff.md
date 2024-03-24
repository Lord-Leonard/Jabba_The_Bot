# TeamSpeak Handshake Handoff

## Issue Summary
- Low-level handshake (Init0–Init4) works.
- High-level crypto handshake stalls after `clientek` + `clientinit`.
- Server stops responding (no `initserver`, no `error`), leading to timeout.
- Likely root cause: incorrect SharedIV / clientek derivation in `initivexpand2` path (Curve25519/Ed25519 encoding mismatch or license chain math).

## Current Behavior
**Expected:** after `initivexpand2` → send `clientek` → send `clientinit` → receive encrypted `initserver` or `error`.
**Actual:** server sends `initivexpand2`, we send `clientek` and `clientinit`, then silence.

## Key Suspicion
- Using `filippo.io/edwards25519` and treating license public keys/root as Edwards-compressed points.
- TS3 docs say “compressed Curve25519 EC point,” which likely means Montgomery u-coordinate.
- Attempts:
  - Montgomery conversion broke with “sqrt failed” (license pubkeys not mapping to Edwards).
  - `BytesMontgomery()` for shared secret/clientek didn’t fix it.
  - Currently reverted to Edwards compressed bytes for both shared secret input and clientek.

## Files Involved
- `pkg/ts/handshake2.go` — license parsing, shared IV derivation, clientek.
- `cmd/teamspeak/main.go` — handshake flow, `clientinit` send.
- `pkg/ts/crypto.go` — EAX, key/nonce derive.
- `pkg/ts/command.go` — TS3 escaping.

## Current State of `handshake2.go`
- License root and block public keys decoded with `Point.SetBytes` (Edwards).
- `sharedData = sharedPoint.Bytes()` (Edwards compressed).
- `clientEK = clientPub.Bytes()` (Edwards compressed).
- This is the current experiment and still times out.

## Last Known Good Changes (kept)
- Encrypted Ack for server commands (stops repeated `initivexpand2`).
- Send `clientinit` immediately after `clientek` with escaped fields.
- Added `BuildEncryptedPacketWithKeyNonce` for bootstrap-phase ACKs.

## Suggested Next Steps
1) Add runtime toggle between Edwards vs Montgomery encoding for:
   - license pubkey interpretation
   - shared secret bytes
   - `clientek` bytes
   so you can test combinations without rebuilding.
2) Add minimal logging of computed SharedIV (first few bytes only) to compare with a known client.
3) Implement or import a Curve25519 Montgomery library and strictly follow the doc’s definition for license key conversion.
4) Verify if `client_version_sign`/identity/hashcash must be computed; server may reject `clientinit` silently.

## Log Pattern
- `initivexpand2 ...`
- `sent clientek`
- `sent clientinit`
- silence → timeout
