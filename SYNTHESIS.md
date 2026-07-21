# Synthesis — shared/framework layer, remaining items

Remaining backlog after the first implementation pass. Completed items removed; this list is the
current state of work, not a historical record.

---

## Security

1: Chat XSS via unescaped innerHTML
`chatbox.ts` builds `innerHTML` directly from `sender`/`message.message` with no escaping — any client can inject a script that runs in every recipient's browser.
Build the chat line with DOM nodes/`textContent` instead of `innerHTML`, or HTML-escape both fields before interpolating.

2: Reconnect endpoint allows client_id hijack
`/reconnect/:nickname/:client_id` doesn't check the nickname matches the original client; disconnected clients stay hijackable for 1 hour; the "rejoin link" hands out the credential.
1) Validate nickname matches before rebinding — cheap, partial fix since nicknames aren't secret.
2) Issue an unguessable per-connection token at connect time, require it for reconnect instead of the raw client_id.
3) Accept as a known risk for a closed family deployment; fix only the nickname check and document the tradeoff.

3: Game-state endpoint trusts client-asserted identity
`POST /api/lobby/games/get/:id` trusts a body-supplied `client_id`/`viewer` with no check the caller owns it — discloses any player's private hand.
Require the request to authenticate as the connection it claims to be (reuse the websocket connection's own identity, or an unguessable per-connection token) instead of trusting the body.

4: Websocket origin check disabled
`CheckOrigin` on the websocket upgrader unconditionally returns `true`.
Check the `Origin` header against an allowlist of known frontend host(s), env-configurable for dev vs prod.

5: No auth required to join the lobby
Connecting only requires a bare nickname — possibly an accepted tradeoff for a private family tool.
1) Accept as-is; confirm intent and document as an accepted risk.
2) Add a shared invite-link token/passphrase requirement at connect.

## Concurrency / reliability

6: Unlocked read of update history races room actor
`Client.readMessages` reads `update_list`/`viewer_update_list` unlocked while the room actor concurrently appends to them.
1) Route resend requests (`game-get-update`, `game-resend-last-update`) through the existing `RequestToFrontend`/`GameStateRequest` actor mechanism instead of reading directly.
2) Add a mutex around the slices, held by both readers and the actor's append path.

7 [SKIPPED — deemed low risk]: GetGames/validLocked bypass actor-routing for game state
`Lobby.GetGames()`/`LobbyRoom.validLocked()` call `ToFrontend`/`Valid()` directly under `l.mu`, bypassing `RequestToFrontend`/`GameStateRequest` and racing the room actor's own mutations. Naively routing through `RequestToFrontend` while still holding `l.mu` would introduce a real deadlock (room actor could be blocked waiting on `l.mu` while `GetGames()` waits on the room's reply), so the real fix needs more care than a simple swap; `GetGames()` also appears unreachable from the current frontend, and `Valid()` is a no-op (`return true`) for the one in-scope game type. Owner judged not worth the complexity right now.

8: AddUpdate can block the room's actor goroutine forever
`AddUpdate`/`AddViewerUpdate` block on small fixed-size channels — a stalled or gone consumer freezes the entire room.
1) Non-blocking `select`/`default` send (mirroring `Client.send()`), drop-and-log on a full buffer.
2) `select` with a `time.After` timeout, treat a stuck consumer as disconnected and clean it up.

9: No rate limiting; message drops are silent
No rate limiting or resource caps exist anywhere; `Client.send()` silently drops on a full buffer with zero logging.
1) Minimal: log in the `default` branch of `Client.send()` so drops are at least visible; skip caps given <10 users.
2) Add basic caps (max rooms/clients, simple per-client rate limiter) if runaway load becomes an actual concern.

## Frontend bugs

10: Players dialog guard condition inverted
`players_dialog_player.ts`'s `parsedCallback` guard bails whenever `this.room_id` is truthy — backwards, so it fires on every normal call.
Change the condition to `!this.room_id`.

11: MultiMap.set() leaves stale secondary-index entries
`set()` doesn't clean up a value's old entry in a keyname's map when the value is re-keyed under that dimension. (The headline "inverted keys()" claim was refuted — current code is correct there, no fix needed.)
Before inserting under a new key, look up and remove the value's prior entry in any changed key dimension (needs a reverse value→keys index), or document that callers must delete the old entry themselves first.

12: DwgElement has no reattach/cleanup lifecycle
No `disconnectedCallback`/reattach guard — detach+reattach leaves cached element refs pointing at stale DOM and duplicates global listeners (e.g. `game.ts`'s keyup handler).
Add `disconnectedCallback` that resets the `found_element` flags so `elementsParsed()` re-queries on reattach, and remove global listeners there too (paired with the same handler reference used to add them).

13: setInterval timers never cleared
Several components' ping-refresh `setInterval`s are never stored/cleared; `refreshGame()` stacks a new interval on every call, including error-recovery retries.
Store each interval id on the instance, `clearInterval` before creating a new one, and clear in `disconnectedCallback` (once #12 adds it) for `lobby.ts`/`lobby_room.ts`/`lobby_users.ts`/`game.ts`.

14: NaN/-Infinity from unguarded parseInt/Math.max
`serverResponseToUser` (`lobby/data_models.ts`) can yield `NaN` since `??` doesn't catch it; `serverResponseToGame` (`game/data_models.ts`) can yield `-Infinity` from `Math.max()` on an empty array.
For the `NaN` case, check `Number.isNaN(...)` explicitly instead of relying on `??`. For the `-Infinity` case, guard with `updates.length ? Math.max(...) : <fallback>`.

15: clickButton never restores original text
`has_changed_text` is declared `const false` and never reassigned, so original button text is never restored after an async handler resolves with no explicit return.
Change to `let` and set it `true` on the path where the handler's return value signals a text change.

16: capitalize ignores custom word_split on rejoin
Splits on the caller-supplied `word_split` but always rejoins with a plain space, discarding non-space delimiters.
`.join(word_split)` instead of `.join(' ')`.

17: GameType/kind literals duplicated with no shared source
Backend and frontend independently hand-maintain the `GameType` enum and message `kind` string literals; `Launchable()` also hardcodes `> 4`.
1) Generate the frontend enum/constants from the Go source at build time — eliminates drift entirely.
2) Keep manual duplication but add a named Go constant (`MaxGameType`) referenced by `Launchable()`, with a sync-reminder comment on each side — cheaper, weaker guarantee.

18: Module-scoped state in game message handler
`message_handler.ts`'s `error_count`/`running_updates` are module-scoped, shared across every `DwgGame` instance rather than per-instance.
Move both into `DwgGame` instance fields (or a per-instance state object) passed into `handleMessage`/`handleGameUpdate`.

## Config / operational

19: Two unlinked DEV flags must be kept in sync manually
Backend and frontend each hardcode their own `DEV` flag; the frontend one failing silently exposes the TEST_GAME option in prod.
1) Derive frontend `DEV` from a build-time env var set by the same deploy pipeline that sets the backend flag — single source of truth.
2) Have the backend expose its `DEV` state to the frontend at connect time instead of a separate hardcoded copy.

20: Plaintext committed config sets a bad precedent
`environment_variables.go` is a plaintext, git-committed config array — harmless now, but a bad pattern to extend once v0.9 adds DB/login credentials.
Before v0.9 adds real secrets, move to env vars sourced from a non-committed `.env`/secret manager rather than extending this file's pattern.

21: Logging has no structure, levels, or alerting
All logging is unstructured `fmt.Println`/`fmt.Fprintln` with no severity levels and no aggregation/alerting.
Swap to a minimal structured logger (stdlib `log/slog`, no new dependency) with levels; rely on App Engine's existing stdout/stderr capture for aggregation given the low user count.

## Dead code / hygiene

22 [PARTIAL]: Stale/unimplemented TODOs in lobby_room.go
The "move lobby room channels" TODO described already-completed work — deleted. "Make someone else host" is still unimplemented — a host leaving an unstarted room still destroys it, evicting everyone else.
Remaining: promote another player/viewer (reuse `promotePlayer`) instead of calling `removeRoom` when others remain, or explicitly accept "leaving destroys the room" as intended and delete that TODO too.

23 [DEFERRED]: Blocking sends in playerGameUpdates/viewerGameUpdates
Reassessed: the blocking send here is likely deliberate, not a bug — it gives sequenced game updates a delivery guarantee via backpressure, unlike the fire-and-forget `Client.send()` used for pings/lobby broadcasts. Switching to non-blocking would trade a narrow race for silent game-update loss, which is worse. The real (much narrower) risk is a TOCTOU window between the `validDebug` check and the blocking send, where the client could go invalid in between and the goroutine blocks forever on a dead connection.
Bound the existing blocking send with a timeout (`select` with `time.After`) instead of making it non-blocking — preserves the delivery guarantee, eliminates the theoretical hang.

24: Inconsistent wire-format stringification
`room_id` is sent raw while sibling fields are stringified; `game_type` is stringified on one endpoint but raw on another.
Pick one convention (stringify, matching the majority pattern in `toFrontendLocked`) and apply it consistently to `room_id` and `game_type`; update the corresponding TS types.

25: Unvalidated settings pass-through with @ts-ignore
`serverResponseToGameSettings` passes `game_specific_settings` through with `@ts-ignore` — no runtime validation at a trust boundary.
1) Add runtime shape validation (hand-written or a small schema lib) per game type before assigning.
2) Leave as-is given the low-stakes trusted-server context, but replace the blanket `@ts-ignore` with a narrower cast plus a comment stating the accepted risk explicitly.

26: Duplicated room-join cases and repeated validation blocks
`room-join`/`room-join-player` are byte-identical; the room-id-parse/lookup/host-check/game-started-check block repeats near-verbatim across ~10 switch cases.
Merge `room-join`/`room-join-player` into one case. Extract the parse→lookup→host-check→game-started-check block into a shared helper called by each case.
