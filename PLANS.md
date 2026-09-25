v0.9: Risq beta version
 p: Revamp summary report
 q: Customizable hotkeys

Small Risq Issues:
 - Icons for interrupt current and attack back
 - Zone ownership should just follow space ownership
 - Left panel: show live "workers X/Y" indicator on a selected gatherable building (predictedGathererCount already exists in risq.ts, just needs left_panel.ts wiring)
 - Send a moving unit's planned path (backend MoveIntent.path) to the frontend so it can be drawn on the map

 - Expanded level 2 tech tree
    => Magic and color damage
    => Stables and Archery range
    => Settings setup (including alt win conditions)


## Refactoring

1. **Order validation/status/intent logic is split across 5 parallel switch statements per order
   type** (`validateFrontendOrder`, `receiveOrder`, `orderReceivable`, `orderStatus`, `tickIntent`),
   each re-deriving the same checks. Proposed fix: a `map[OrderType]unitOrderHandler` strategy table
   so each order type's full lifecycle lives in one place. **Not done, not attempted** — the riskiest
   remaining item, touches order-resolution directly with the largest surface area (5 functions × ~15
   order types); needs a dedicated go/no-go rather than folding into a batch pass.
2. **Subpackage reorg proposal — recommendation: don't do a full split.** The root package is one
   tangled object graph (units/buildings/orders/space/zone/player/vision) and three of its core
   interfaces (`Orderable`, `Attackable`, `IntentKind`) use **unexported methods** — Go forbids
   implementing an interface with unexported methods from another package, so splitting any of these
   into a subpackage means exporting everything, which is pure churn with no real encapsulation
   gained. Only one clean extraction candidate exists: `risq/mapgen` — turn the map-script DSL into a
   function producing a plain-data `Blueprint` (spaces/terrain/placements/regions), then materialize
   it back into `GameRisq` in root. Real risk to manage: RNG call order must stay byte-identical or
   seeded maps stop matching the sim determinism baselines — map generation is now seed-reproducible
   (fixed and confirmed), so this just needs to be preserved through the extraction, not fixed as
   part of it. Recommendation: extract `mapgen`, skip every other package boundary. The other pure
   file-move win (splitting `risq_map_steps.go`'s geometry/board-mutation code out) is already done
   (`risq_hex.go`/`risq_board.go`). Still open: `risq.go` itself still mixes the action dispatcher,
   behavior setters, gather points, turn resolution, cleanup, vision, and serialization — splitting
   it into `risq_actions.go` (the `execute*` handlers) + `risq_tick.go` (`resolveActiveOrders`,
   `cleanupDeleted`, `recalculateVision`) would be the same zero-risk, zero-rename kind of move.
3. **There are no `_test.go` files anywhere in risq.** The only safety net today is
   `go build`/`go vet` plus the sim determinism harness (`sim/inputs/determinism_check.json`). Worth
   discussing independent of the reorg question.


Fiddlesticks Plans:
 - Revamp update dialog box
 - Update turn timer UX
 - Host can pause game and if host leaves then someone else takes over as host
 - User settings (including trick-resolution animation durations) instead of hardcoded timings in fiddlesticks.ts

Game Frontend Plans:
 - Sync/refresh game state on in-game frontend errors thrown (the "TODO: try to sync data" scattered across message_handler.ts and each game's gameUpdate catch blocks). Updates are applied in guaranteed order by message_handler.ts, so if a game's gameUpdate throws while applying one, that's a real bug, not a transient network issue — the frontend should treat it as a signal to auto-refresh/resync game state (e.g. game.refreshGame()) instead of just logging and leaving the client silently desynced.

Lobby Plans:
 - Loaders for client requests in lobby: room-create, room-join, room-leave, room-rename
 - Can chat with individual players
 - Upgraded chatbox => emoji selector, taunts, message id, turn off emoticon converter

Testing Plans:
 - Formalize the ad-hoc concurrency/reconnect repro scripts (currently one-off Node scripts written per-investigation) into a real backend test suite instead of relying on agents to improvise and run them each time
 - Unify backend logging (stdout fmt.Println vs util.DebugLog vs stderr are used inconsistently across lobby/room/risq code with no clear rule for which tier a given log belongs in)

Refactor Plans:
 - Decomposition pass over risq frontend and backend (long functions and files that have grown past what one unit should own)
    => frontend: risq.ts mouseup() (~170 lines, 8 levels of nesting; zone-slot click body is the first extract, which also lets the ctrl-additive branch reuse handleUnitGridClick)
    => frontend: left_panel.ts (~2280 lines, 73 methods)
    => backend: risq_unit.go (931), risq.go (686), risq_map_steps.go (628)

v1.0: Database
 - Setup db in prod and dev
 - Can create profile / login
 - Can save games (handle ai players, people not logged in, etc)
 - Can launch a saved game if logged in
 - Risq is fully playable with custom settings
 - Reporting => admin login can access admin page to see error reports, etc.
 - Can report bugs / email admin / etc.
 - Advanced risq mechanics
    => various terrains / resources
    => various buildings
    => various units / can assign formations/etc. to control how they fight
 - Can make friends / personal DMs (all DMs and chatboxes saved)
 - Can see other people's stats

Games to add:
 - Chess with esoteric variations
 - Poker
 - Ben games?
