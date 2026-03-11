# Two-Seed Autoplay Algorithm (Concept)

## Goal
Build an autoplay queue that combines recommendations from two seed songs while staying adaptive to skips and listens.

## Core Idea
Model autoplay as four stages:
1. Retrieval: gather candidate songs from each seed.
2. Fusion: score each candidate by how well it matches both seeds.
3. Sequencing: order songs with flow and anti-repetition constraints.
4. Feedback: update weights from user behavior and re-rank periodically.

## Inputs
- Seed A and Seed B (`videoId`).
- Recommendation lists for each seed (top `N`).
- Lightweight metadata per candidate (artist, duration, genre/mood tags if available).
- Session feedback events (impress, start, skip, complete).

## Stage 1: Candidate Retrieval
- Fetch top `N` candidates from seed A.
- Fetch top `N` candidates from seed B.
- Union by `videoId` (deduplicate).
- Track provenance:
  - `rankA` (or null if not in A list)
  - `rankB` (or null if not in B list)
  - `inBoth` flag

## Stage 2: Candidate Fusion (Scoring)
Use a blended score per candidate:

`score = baseRel + overlapBonus + noveltyBonus - repetitionPenalty - skipPenalty`

Where:
- `baseRel = wA * relA + wB * relB`
- `relA`, `relB`: rank-based relevance from A/B.
- `wA`, `wB`: seed weights (default `0.5 / 0.5`).

### Rank Has Direct Impact
Yes: lower rank index means higher value.
- Recommendation `#1` should score higher than `#7`.
- Typical mapping (Reciprocal Rank):
  - `rel = 1 / (k + rank + 1)` with `k` around `20-80`.
  - If rank is missing in one seed list, that side contributes `0`.
- Example with `k=60`:
  - rank `0` -> `~0.0164`
  - rank `6` -> `~0.0149`
  - rank `20` -> `~0.0123`

This keeps rank influence smooth and avoids over-dominating by only top 1-2 items.

## Detailed Term Definitions
### overlapBonus
Purpose: reward candidates that both seeds agree on.

Recommended form:
- `overlapBonus = betaOverlap * I(inBoth)`
- Optional stronger variant:
  - `overlapBonus = betaOverlap * sqrt(relA * relB)` if in both, else `0`

Notes:
- `I(inBoth)` is `1` if candidate appears in both lists, else `0`.
- Start with small values so overlap helps but does not dominate.
- Typical range: `betaOverlap = 0.02 .. 0.10`

### noveltyBonus
Purpose: avoid stale queue behavior and inject controlled freshness.

Recommended components:
- `artistNovelty`: `1` if artist not seen in last `W` tracks, else `0`.
- `seedBalanceNovelty`: boost items from the underrepresented seed.
- `exploration`: tiny random jitter for tie-breaking.

Example:
- `noveltyBonus = betaArtistNovelty * artistNovelty + betaSeedBalance * seedBalanceGap + betaExplore * rand(0..1)`

Typical ranges:
- `betaArtistNovelty = 0.00 .. 0.06`
- `betaSeedBalance = 0.00 .. 0.04`
- `betaExplore = 0.00 .. 0.01`

If you want lots of same-artist output, keep `betaArtistNovelty` near `0`.

### repetitionPenalty
Purpose: prevent undesirable repetition patterns.

Important for your preference:
- If you expect same-artist runs (for example Linkin Park seeds), set artist repetition penalty very low or zero.
- Keep only strict duplicate protection (same track replay too soon).

Suggested decomposition:
- `repetitionPenalty = lambdaTrackRepeat * trackRepeat + lambdaArtistRepeat * artistRepeat + lambdaNearDup * nearDuplicate`

Where:
- `trackRepeat`: `1` if same `videoId` appears in recent window, else `0`.
- `artistRepeat`: `1` if same artist appears in recent `K`, else `0`.
- `nearDuplicate`: similarity score for alternate versions/live edits of same song.

Recommended defaults for your use case:
- `lambdaTrackRepeat = 0.15` (keep this)
- `lambdaArtistRepeat = 0.00 .. 0.02` (almost disabled)
- `lambdaNearDup = 0.04`

### skipPenalty
Purpose: respond to user dislikes fast within current session.

Model skip memory with decay:
- For each skipped track `s`, store `time`, `artist`, and optional embedding/features.
- Penalty contribution from `s` to candidate `c`:
  - `p_s(c) = sim(c, s) * exp(-ageMinutes / tau)`

Then:
- `skipPenalty = lambdaSkip * sum(p_s(c))`

Similarity levels (in increasing strength):
- exact track match: `1.0`
- same artist: `0.5` (tunable)
- same cluster/embedding neighborhood: `0.2 .. 0.6`

Typical ranges:
- `lambdaSkip = 0.10 .. 0.40`
- `tau = 20 .. 120` minutes

If you want aggressive adaptation after skip:
- increase `lambdaSkip`
- increase same-artist similarity weight
- lower `tau` for faster decay reset

## Stage 3: Sequencing (Constraint-Aware Ordering)
Greedy build from highest score with configurable rules:
- Prevent exact `videoId` repeats in short window.
- Optionally penalize near-duplicate title/version collisions.
- Artist spacing is optional. For your preference, set to relaxed/off.
- Keep gentle mood/energy continuity only if desired.

If top candidate violates constraints, take next best valid candidate.

## Stage 4: Feedback Loop
After each playback event, update session state:
- `skip` increases penalty for that track and similar tracks.
- `complete` / long play increases affinity for that track family.
- Recompute queue every `M` events (or every `X` tracks) using latest state.

Important: treat feedback as session-local and short-horizon unless account-level persistence exists.

## Strict 20-Item Queue Mode
If you want autoplay to stay at exactly 20 items all the time, use this invariant:

`len(autoplayQueue) == 20`

### Runtime Behavior
1. User enqueues first song (or two anchor songs).
2. Controller generates autoplay tail and fills to exactly 20.
3. When one song starts playing (or is consumed from queue), queue length drops to 19.
4. Controller immediately appends exactly 1 new autoplay song, returning to 20.
5. Repeat forever.

This is per-track top-up (no watermark strategy).

### Important Reliability Detail
To preserve "always 20" during API latency/failures:
- Keep a hidden candidate buffer (for example 5-10 pre-scored songs).
- Refill buffer asynchronously in the background.
- If network call fails, consume from buffer first.

Without a buffer, temporary dips below 20 are unavoidable.

## Seed Update Policy (What Becomes a Seed?)
Do not blindly use the last autoplay song as the only new seed.

Use two seed classes:
- Anchor seeds: songs explicitly chosen by the user (strong, persistent weight).
- Promoted seeds: autoplay songs that received positive signal (complete/like), with smaller temporary weight.

### Recommended Policy
- Keep anchor seeds as the dominant signal (for example 70-90% combined weight).
- Allow promoted seeds only when positively validated.
- Remove promoted seeds quickly after skips.
- Manual user enqueue should immediately become a new anchor.

This keeps queue intent stable while still adapting.

## Event Handling Cheat Sheet
- `track consumed`: append 1 song immediately.
- `skip`: update skip penalties, then regenerate a small tail slice (for example next 3-5 autoplay slots).
- `complete`: small positive affinity update; no full rebuild needed.
- `manual enqueue`: insert requested song and re-anchor seed mix.
- `buffer low`: background fetch + score more candidates.

## Practical Defaults
- `N = 50` candidates per seed.
- `wA = 0.5`, `wB = 0.5` initially.
- `betaOverlap = 0.05`.
- `betaArtistNovelty = 0.01` (or `0.0` if you want artist streaks).
- `lambdaTrackRepeat = 0.15`.
- `lambdaArtistRepeat = 0.0`.
- `lambdaSkip = 0.25`.
- `autoplayTarget = 20`.
- `candidateBufferMin = 5`.
- `candidateBufferTarget = 10`.
- Re-rank every 3-5 tracks.

## Minimal Implementation Path
1. Implement rank-fusion + dedupe.
2. Add overlap bonus and exact-track repetition protection.
3. Add skip penalty from current session.
4. Add periodic refresh/re-rank.
5. Add optional novelty + mood/tempo continuity.

## Evaluation Signals
Track quality with:
- Skip rate in first 30 seconds.
- Consecutive same-artist occurrences (only if artist variety matters to you).
- Average play-through ratio.
- Queue churn after feedback.

If skip rate drops and play-through rises without collapsing diversity, the fusion is working.
