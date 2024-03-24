# TS3 Handshake Progress

## What now works
- Low-level Init0-Init4 handshake completes.
- High-level initivexpand2 flow completes (proof verified, clientek + clientinit sent).
- Server accepts identity (hashcash key offset computed) and returns initserver + channel data.

## Key fixes that made it work
- Packet IDs are monotonic across all client->server packets (no duplicate pid=1 for ack + clientek).
- clientek is encrypted with the bootstrap key/nonce (shared IV is applied after clientek).
- Curve25519 license/crypto uses the Java reference algorithm:
  - License chain deriveKey uses Edwards compressed points with sign bit flip.
  - Shared secret uses negated server key, scalar mult, Edwards encoding, then flip sign bit and sha512.
- clientinit uses:
  - client_version/client_version_sign/hwid that server accepts.
  - client_key_offset computed via hashcash level 8 using the omega from clientinitiv.
- clientinitiv uses the public key from the identity private key (not a standalone public.pem).

## Current state (log proof)
- initserver and channellist packets are received and decrypted.
- server sends periodic ping packets.

## Next steps (in order)
1) Pong handling
   - Reply to server ping packets with pong (unencrypted).
2) Fragment merge + decompression
   - Reassemble FR packets in order.
   - Decompress CP payloads with QuickLZ level 1.
3) Client ID update
   - Parse initserver and set client id for subsequent headers.

## Implementation notes / references
- Specs: pkg/ts/docs/ts3protocol.md section 3 and 4.
- Java reference:
  - pkg/ts/docs/inspiration/LocalClientHandlerConnecting.java
  - pkg/ts/docs/inspiration/Ts3Crypt.java
  - pkg/ts/docs/inspiration/License.java

