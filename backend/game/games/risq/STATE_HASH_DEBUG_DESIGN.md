# State-hash debug logging

Replaces manual JSON-diffing of the per-player (fog-filtered) AI dump for
determinism debugging. Engine-level, not per-player, not AI-specific.

## What gets hashed

One canonical snapshot of all engine-truth state, opt-in fields:

- **Units**: internal_id, player_id, unit_id, location (space+zone), health,
  stamina, garrisoned_in, deleted
- **Buildings**: internal_id, player_id, building_id, location, health,
  stamina_remaining, deleted, production_queue contents
- **Spaces**: coordinate, ownership, terrain_id
- **Zones**: coordinate, resource (id/category/amount), building id if any
- **Players**: resources, researched_techs, population support/limit
- **Vision**: visibility level per (space, player)

Not included: order internal_ids (assigned by a race between AI goroutines,
never compared anywhere in game logic, so their value is meaningless noise),
timestamps, anything display-only.

## Float handling

Round every float to a fixed precision (reuse `gatherRoundingPlaces`, 4)
before hashing, so legitimate floating-point summation-order noise can't
produce a false mismatch.

## When it's logged

- Once per **turn**: after `resolveActiveOrders` finishes resolving that
  turn.
- Once per **tick**: at the end of each tick's execute + resolveHealthDelta
  pass, inside the resolution loop.

One line each via `util.DebugLog.Printf`, e.g.:
`state_hash turn=%d tick=%d hash=%s`

## How the hash is computed

Sort every collection by a stable key (internal_id, coordinate) before
serializing, so iteration order (the exact thing being debugged) can't affect
the hash. Serialize to a canonical byte form, hash with FNV or truncated
SHA-256.

## Workflow

1. Compare the sequence of per-turn hashes between two runs of the same seed.
2. First mismatched turn hash -> that's the divergent turn.
3. Compare per-tick hashes within that turn -> first mismatched tick.
4. Read the existing tick-tagged event logs (combat hits, foundation started,
   construction complete, unit created, moving unit) for that exact tick in
   both runs. The diff between those lines is the direct explanation.

No JSON diffing, no per-player fog filtering, no guessing which fields to
compare - just "do the hashes match," and if not, "where's the first tick
that differs."

## Tooling impact

`find_divergence.py` is replaced with a much simpler tool: parse `state_hash`
lines, find the first mismatched turn, then the first mismatched tick within
it, then print both games' tick-tagged event lines at that tick side by side.
The AI-view "new turn data" dump stays for AI-behavior debugging, but is no
longer used for determinism comparison - it's fog-filtered and incomplete by
construction, which is what caused the last few rounds of chasing missing
fields (foundations, spaces/vision) one at a time.
