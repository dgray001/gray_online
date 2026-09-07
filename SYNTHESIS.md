# Synthesis — shared/framework layer, remaining items

Remaining backlog after the first implementation pass. Completed items removed; this list is the
current state of work, not a historical record.

---

## Concurrency / reliability

1: Unlocked read of update history races room actor
`Client.readMessages` reads `update_list`/`viewer_update_list` unlocked while the room actor concurrently appends to them.
1) Route resend requests (`game-get-update`, `game-resend-last-update`) through the existing `RequestToFrontend`/`GameStateRequest` actor mechanism instead of reading directly.
2) Add a mutex around the slices, held by both readers and the actor's append path.

2: AddUpdate can block the room's actor goroutine forever
`AddUpdate`/`AddViewerUpdate` block on small fixed-size channels — a stalled or gone consumer freezes the entire room.
1) Non-blocking `select`/`default` send (mirroring `Client.send()`), drop-and-log on a full buffer.
2) `select` with a `time.After` timeout, treat a stuck consumer as disconnected and clean it up.

## Frontend bugs

3: MultiMap.set() leaves stale secondary-index entries
`set()` doesn't clean up a value's old entry in a keyname's map when the value is re-keyed under that dimension. (The headline "inverted keys()" claim was refuted — current code is correct there, no fix needed.)
Before inserting under a new key, look up and remove the value's prior entry in any changed key dimension (needs a reverse value→keys index), or document that callers must delete the old entry themselves first.

4: DwgElement has no reattach/cleanup lifecycle
No `disconnectedCallback`/reattach guard — detach+reattach leaves cached element refs pointing at stale DOM and duplicates global listeners (e.g. `game.ts`'s keyup handler).
Add `disconnectedCallback` that resets the `found_element` flags so `elementsParsed()` re-queries on reattach, and remove global listeners there too (paired with the same handler reference used to add them).

6 [PARTIAL]: GameType/kind literals duplicated with no shared source
Backend and frontend independently hand-maintain the `GameType` enum and message `kind` string literals. `Launchable()`'s hardcoded `> 4` bound is fixed (now bounds against `game.GameType_RISQ`/`game.GameType_TEST_GAME` directly, gated on the new `lobby.DEV`, set from `main.go`'s `DEV` at startup).
Remaining: the enum/kind literals are still hand-duplicated between Go and TS with no shared source. Generate the frontend enum/constants from the Go source at build time to eliminate drift entirely, or accept the manual duplication as-is.

7: Module-scoped state in game message handler
`message_handler.ts`'s `error_count`/`running_updates` are module-scoped, shared across every `DwgGame` instance rather than per-instance.
Move both into `DwgGame` instance fields (or a per-instance state object) passed into `handleMessage`/`handleGameUpdate`.

## Dead code / hygiene

8: Inconsistent wire-format stringification
`room_id` is sent raw while sibling fields are stringified; `game_type` is stringified on one endpoint but raw on another.
Pick one convention (stringify, matching the majority pattern in `toFrontendLocked`) and apply it consistently to `room_id` and `game_type`; update the corresponding TS types.
