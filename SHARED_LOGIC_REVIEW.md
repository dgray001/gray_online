# Shared Logic Review

Scope: backend `lobby/`, `game/` (base abstractions + `game_utils`), `util/`, `main.go`/`api.go`; frontend `scripts/`, `components/` shared elements (dwg_element, lobby, game shell, chatbox, dialogs, canvas/card utils). Individual game implementations excluded. No code was changed.

Severity legend: [CRIT] exploitable or crashes the server, [HIGH] real bug users will hit, [MED] bug in edge cases or leak, [LOW] smell/refactor.

---

## 1. Backend: concurrency model is fundamentally unsound

The design intent is actor-style (one goroutine per `Lobby.Run()` and per `LobbyRoom.run()`, communicating via channels), but shared state is mutated from at least five goroutine families with no synchronization: the lobby goroutine, each room goroutine, each client's `readMessages` goroutine, gin HTTP handler goroutines, and `time.Timer` callbacks. There are no mutexes anywhere in the backend. "Only one instance" does not help; Go maps are not goroutine-safe and concurrent map read/write is a fatal runtime panic that kills the whole process.

### [CRIT] HTTP handlers race the lobby goroutine on shared maps
`main.go` handlers (`/rooms/get`, `/users/get`, `/games/get`, `/games/get/:id`) call `lobby.GetRooms()`, `GetUsers()`, `GetGames()`, `GetGame()` which iterate/read `l.rooms`, `l.clients`, `l.games` while `Lobby.Run()` mutates them (`addClient`, `removeRoom`, etc.). Same for `game.ToFrontend(...)` reading live game state while room/game goroutines mutate it. Under any real concurrency this is a `fatal error: concurrent map iteration and map write` crash. Since the frontend polls `/rooms/get` and `/users/get` every 3.5s per client, the window is hit constantly.

### [CRIT] `readMessages` mutates lobby/room state directly
`lobby/client.go` handles `room-rename` and `room-update-description` by writing `room.room_name = message.Content` from the reader goroutine, then notifying the lobby goroutine. Also reads `room.host.client_id`, `room.gameStarted()`, `c.lobby_room.game`, and calls `ResendPlayerUpdate`/`ResendLastUpdate` (which read `player.update_list`) directly from the reader goroutine. All of these race with the lobby and room goroutines. The rename/description writes also bypass validation and happen even if the subsequent channel send fails.

### [CRIT] Timer callbacks bypass the actor loops
- `Lobby.removeClient` starts a 1h timer whose goroutine does `delete(l.clients, ...)` directly.
- `LobbyRoom.playerDisconnected` starts a 45s timer whose goroutine calls `r.lobby.removeRoom(r)` directly, which mutates `l.rooms` and iterates room membership.

Both mutate lobby state outside `Lobby.Run()`. `removeRoom` is also called from `LobbyRoom.run()`'s goroutine (via `addClient` -> `removeClient`), so `l.rooms` has three writers.

### [HIGH] Cross-goroutine calls between lobby and room actors
`Lobby.Run()` calls `data.room.kickClient/promotePlayer/setViewer/setPlayer` and `room.launchGame` synchronously, while `LobbyRoom.run()` mutates the same `r.players`/`r.viewers`/`r.game` maps from its own goroutine (`JoinRoom`, `PlayerConnected`). Two writers on room maps. The TODO in `lobby_room.go` ("move lobby room channels from lobby to lobby room") acknowledges this; it is not cosmetic, it is a data race.

### Recommendation
Either commit fully to the actor model (every mutation and read goes through exactly one owning goroutine; HTTP reads answered via request/response channels or by snapshotting), or drop the channels and put an `sync.RWMutex` on `Lobby` and one on each `LobbyRoom`/`GameBase`. The current hybrid is the worst of both. Run the test suite / dev server with `-race`; it will light up immediately.

---

## 2. Backend: channel and goroutine lifecycle bugs

### [HIGH] `break` inside `select` does not exit the loop
`lobby_room.go` `playerGameUpdates`: the `case game_over := <-player.FlushConnections:` arm ends with `break`, which only breaks the `select`, not the `for`. The flush mechanism therefore never terminates the old goroutine; it only exits when `c.deleted` happens to be observed. During reconnect, the old and new `playerGameUpdates` goroutines race to consume `player.Updates` until `old_client.deleted = true` lands, so updates can be delivered to the dead connection and silently lost. This is very likely the root cause of the PLANS.md bug "after reconnecting sometimes ... update goes through from client to server just not vice versa". Same latent issue in `writeMessages` (`break` in the ticker error arm), though there the loop recheck saves it.

Related: `reconnectClient` sends `player.FlushConnections <- false` and then immediately starts the new goroutine, which begins with a non-blocking drain of `FlushConnections`. The new goroutine can eat the flush intended for the old one.

### [HIGH] `AddUpdate` can block the game loop forever
`Player.Updates`/`AiUpdates` are buffered at 2 (`UPDATE_CHANNEL_SIZE`). `AddUpdate` does a blocking send. If a player's consumer goroutine is gone (client deleted after the 1h timer, or never started because the game start was missed), the third broadcast blocks `Game_BroadcastUpdate`, which blocks `LobbyRoom.run()`, which deadlocks the entire room: no more actions, joins, or chat. Same for `GameBase.AddViewerUpdate` (`ViewerUpdates` buffer 12, plus per-viewer sends to possibly unconsumed `viewer.Updates` buffered at 24). `ResendPlayerUpdate` can likewise block a client's `readMessages` goroutine.

### [HIGH] Send-on-closed-channel panics
`Client.close()` sets `c.closed = true` then `close(c.send_message)` with no synchronization against senders. `Client.send()` checks `c.closed` non-atomically (TOCTOU), and several paths skip `send()` entirely and write `c.send_message <- ...` directly: `Lobby.addClient`, `Lobby.Run`'s `room-launch-failed` reply, `lobby_room.addClient` failure replies, and both `playerGameUpdates`/`viewerGameUpdates`. Any of these racing `close()` panics the process. The `validDebug` guards reduce the window but do not close it.

### [MED] `gameBaseUpdates` goroutines leak forever
The loop condition is `if game_base == nil` on a local that never becomes nil. After `GameEndedChannel` fires it sets `r.game = nil` and then loops forever draining `ViewerUpdates`. One permanently-leaked goroutine per game ever played. `viewerGameUpdates` similarly only exits on `c.deleted` (1h timer), and its single-case `select` wrapper is pointless.

### [MED] `Lobby.games` map is never cleaned
Games are added in `Run()` on launch and never deleted, so every finished game's full state (players, update history) is retained for the life of the process, and `/api/lobby/games/get/:id` serves stale ended games.

### [MED] `writeMessages` abuses `deleted` on transient write errors
On a `WriteJSON`/ping error it sets `c.deleted = true`. `Lobby.removeClient` early-returns for deleted clients ("don't want to remove reconnected client"), so a client whose write failed is never removed from `l.clients` or its room, never broadcast as left, and gets no delete timer: a permanent ghost. `deleted` is doing double duty as "connection is dead" and "identity was replaced"; those need to be separate flags.

### [MED] `PlayerConnected` race with game end
`LobbyRoom.run()`'s `PlayerConnected` arm does `r.game.GetBase()` unguarded. The reader goroutine's nil check happens earlier on a different goroutine; if the game ends (gameBaseUpdates sets `r.game = nil`) between check and use, this nil-derefs and kills the room goroutine.

---

## 3. Backend: plain logic bugs

### [HIGH] Nil dereference in room broadcast error path
`lobby_room.go` `broadcastMessage`:
```go
if client == nil || !client.validDebug(true) {
    fmt.Fprintln(os.Stderr, "Room failed to send message to player", client.client_id)
```
When `client == nil`, the log line dereferences nil and panics. Duplicated in both the players and viewers loops.

### [HIGH] `StandardDeck.ShuffleDiscardPile` is broken (game_utils/card_util.go)
```go
new_draw_pile[v] = d.discard_pile[i]   // i indexes the combined range
```
For `i >= len(draw_pile)` the index into `discard_pile` should be `i - len(d.draw_pile)`; as written it panics (index out of range) whenever the draw pile is non-empty. It also never clears `d.discard_pile`, so the shuffled-in cards would be duplicated. Currently unused by any game, which is why it hasn't blown up.

### [MED] `DealCards` uint8 overflow
`int(players*cards)` multiplies in uint8 first, wrapping at 256 (e.g. 8 players x 32 cards -> 0 passes the size check, then deals nils). Convert before multiplying.

### [MED] Shared `*UpdateMessage` mutated per recipient
`Game_BroadcastUpdate` passes one `*UpdateMessage` to every player's `AddUpdate`, each of which writes `update.Id`. It only works because every list happens to have the same length; it is a data race across the per-player consumer goroutines and any divergence (a player added mid-game, failed appends) silently corrupts ids. The `gin.H` content map is also aliased by every recipient. Copy the message per recipient.

### [MED] `GameBase.EndGame`/`StartGame` warn but proceed
Both log "already ended/started" and then perform the transition anyway (`EndGame` re-sends on `GameEndedChannel`). Should early-return.

### [LOW] `game-resend-waiting-room` only handles the started case
If the game has not started it silently does nothing, so a client stuck in the waiting room re-polls forever with no reply (the frontend treats no response as fine, but a `game-waiting-room` refresh was presumably intended).

### [LOW] `room-join` and `room-join-player` in `client.go` are byte-identical
And the whole `readMessages` switch repeats the same room-id-parse / lookup / host-check / game-started-check block ~10 times. Extract `func (c *Client) resolveRoom(data string, failKind string) *LobbyRoom` and a host-guard helper; the switch shrinks by two thirds and future messages stop copy-pasting validation.

### [LOW] `GameSettings.Launchable` hardcodes `GameType > 4`
Must be edited every time a game is added (as `PLANS.md` intends). Compare against a `GameType_MAX` sentinel next to the enum.

### [LOW] Misc
- `failureResponseWiothResult` (api.go): typo, unused.
- `CreatBlankCard` (card_util.go): typo, and returns an invalid card by design; callers can't tell it apart from a real one without `Valid()`.
- "Persistant" (PersistantHistory, persistant_history) is misspelled across backend and frontend; it is now wire protocol, so fixing it is a coordinated change; worth doing before v0.9 freezes more clients against it.
- `PlayerAction.Client_id` is `int` while client ids are `uint64` everywhere else.
- `util/strings.go` parse helpers swallow errors and return 0/false, indistinguishable from legitimate values.
- `util.Shuffle` reimplements `rand.Shuffle`; `util/set.go` is a copy-pasted int-only set predating generics (you already have generics in `arrays.go`); `if P(v) == false` style.
- `util/math.go` `Pair`/`InvertPair` have acknowledged overflow TODOs; `invertCantorPair` relies on float sqrt and will break for large z. Fine for board coords, but the TODO should note the safe input range.
- Logging is unstructured `fmt.Println` everywhere, including full update payloads (private hands land in prod logs).
- Zero tests in the backend.

---

## 4. Security

Context: pre-database, no accounts. Still, several of these matter the moment this is deployed publicly (it is, per README).

### [CRIT] Stored XSS in chat
`chatbox.ts addChat`: `new_element.innerHTML = `<b>${sender}</b>${message.message}``. Message and sender are attacker-controlled (chat content, nickname). Any user can run script in every other client in the lobby/room/game. Nicknames come from a URL path parameter with no server-side validation at all. Use `textContent` / build nodes; the emoticon feature does not need HTML.

### [CRIT] Private game state disclosure via the game fetch API
`POST /api/lobby/games/get/:id` trusts the body: any caller can pass `viewer: "true"` (gets `player_actions`, all `viewer_updates`, every player's update history) or any other player's `client_id` (gets that player's private updates, i.e. their hand in fiddlesticks). Game ids are small sequential ints. This is a complete cheating vector. The server must derive identity from the connection/session, not the request body.

### [HIGH] Session takeover via predictable client ids
`/api/lobby/reconnect/:nickname/:client_id` (and nickname `!!previous!!`) attaches the caller to any existing client whose connection is currently down, with no secret. Client ids are sequential and are deliberately shared in rejoin links (`players_dialog_player.ts` builds `?room_id=X&client_id=Y`). Anyone can enumerate ids and hijack disconnected players' seats. Reconnect needs an unguessable token issued at connect time.

### [HIGH] Sender spoofing / server impersonation in chat
For `lobby-chat`, `room-chat`, `game-chat` the backend broadcasts `message.Sender` verbatim as supplied by the client. A client can set `sender` to `client-<other-id>` (impersonate another user and bypass the self-exclusion), or `room-X-Y-!!server!!` which the frontend renders with the reserved server name (`room_chat_sender_split[3]` path in lobby message_handler). `SERVER_CHAT_NAME` is only blocked client-side. The server should construct the sender string itself from `c.client_id`.

### [MED] `CheckOrigin: func(...) bool { return true }`
The websocket upgrader accepts any origin, enabling cross-site WebSocket hijacking from arbitrary web pages (combined with the reconnect issue above, a drive-by page can act as a victim). Check `Origin` against the site host in prod.

### [MED] No server-side input validation or rate limiting
Nickname (length/charset), room name, room description, chat messages, and `GameSpecificSettings` (arbitrary JSON stored and re-broadcast to everyone) are unvalidated; the only bound is the 5120-byte read limit. No rate limiting on chat or room actions; every 4th pong broadcasts a ping-update to the whole lobby (O(clients^2) traffic).

---

## 5. Frontend: bugs

### [HIGH] `MultiMap.keys()` condition is inverted (scripts/multi_map.ts)
```ts
public keys(keyname: string): IterableIterator<K> {
  if (this.maps.has(keyname)) { return [].values(); }   // returns empty when it EXISTS
  return this.maps.get(keyname)!.keys();                // derefs undefined otherwise
```
Returns empty for valid keynames and throws for invalid ones. Also: `set()` under a new key value does not remove the old key entry, so secondary indexes accumulate stale entries (see card_hand below), and `_size` silently tracks only the first keyname.

### [HIGH] `players_dialog_player.ts` guard is inverted
```ts
if (!this.player || !this.lobby_player || this.room_id) {
  console.error('Must set data before attaching to dom');
  return;
}
```
`this.room_id` should be `!this.room_id`, so any correctly-configured row (room_id >= 1) errors out and never renders ping/nickname/rejoin link. Additionally `!this.lobby_player` rejects the legitimate disconnected-player case that `setData` explicitly allows (`lobby_player: LobbyUser | undefined`) and that the code below handles.

### [HIGH] `DwgCardHand.playCard` desyncs the i_dom index
After deleting the played card it decrements `other_card.i_dom` and updates the CSS var, but never re-`set()`s the cards in the MultiMap, so the `i_dom` index still maps old positions. Subsequent drag reordering (`cards.get('i_dom', ...)` in `dragCard`) operates on the wrong neighbors. Combined with the MultiMap stale-key behavior above, the index degrades further each play.

### [HIGH] `DwgButton.mousemove` missing return (canvas_components/button/button.ts)
```ts
if (this.disabled) {
  this.hovering = false;
  false;          // no-op; should be `return false;`
}
```
Disabled buttons keep computing hover state and firing `hovered()`/`unhovered()` callbacks.

### [MED] `clickButton` never restores button text (scripts/util.ts)
`const has_changed_text = false;` is never set to true, so after an async handler completes with no return value the original label is never restored; buttons that used `loading_text` stay stuck on it.

### [MED] `capitalize` ignores its `word_split` on join (scripts/util.ts)
Splits on `word_split` but always joins with `' '`, silently rewriting delimiters for any non-space split.

### [MED] `serverResponseToUser` produces `room_id: NaN` (lobby/data_models.ts)
`parseInt(x ?? '') ?? undefined`: `??` does not catch NaN. Downstream truthiness checks happen to work, but any `=== undefined` comparison or serialization is wrong. Should be `|| undefined`. Same pattern risk in `serverResponseToGame`: `Math.max(...updates.map(...))` on an empty updates array yields `-Infinity` for `highest_received_update_id`.

### [MED] `DwgGame.refreshGame` timing hole and interval leak
- If `game_el.initialize()` resolves after the outer `game_initialized` check, the inner callback sends `game-connected` and sets `launched = true`, but `refreshGame`/`launchGame` return false, so `page_home` never hides the lobby even though the game is live. Two racing "who finishes last" flags is the wrong shape; await initialize (it is already a promise) instead of the callback dance.
- On the success path a new `ping_interval` is created on every (re)launch without clearing the previous one; rejoining stacks intervals and `exitGame` only clears the last.

### [MED] Global listener leaks
- `game.ts parsedCallback`: `document.addEventListener('keyup', ...)` toggles the game chatbox on any Enter keypress anywhere in the app (including while the game is hidden and while typing in lobby inputs), and is never removed.
- `canvas_board.ts`: `setInterval(..., 20)` render loop is never cleared (runs at 50fps forever after the board is removed; a second `initialize` stacks a second loop and a second set of listeners), `document.body` keydown/keyup listeners are never removed, and `resize_observer` is never disconnected.
- `card_hand.ts`: four `document.body`/`documentElement` listeners per instance, never removed.
- `lobby_users.ts`, `lobby_room.ts`, `lobby.ts`: `setInterval`s created in `parsedCallback` are never cleared.

These matter because custom elements here are created/destroyed across game sessions. Use `AbortController` signals (the pattern already exists in `game.ts` for socket listeners) and `disconnectedCallback`.

### [MED] `DwgCardHand.setCards` re-entry race
`createCardEl` appends via `setTimeout((1 + i_dom) * animation_time)`. A second `setCards` call clears the container but pending timeouts from the first call still fire, appending stale card elements and writing them into the fresh map. Track and cancel pending timers on reset.

### [MED] `DwgListbox` constructor monkey-patch loses `this`
```ts
const scroll_callback = config.scrollbar.scrollCallback;
config.scrollbar.scrollCallback = (value) => { if (scroll_callback) scroll_callback(value); ... };
```
The captured method is invoked unbound, so any subclass `scrollCallback` using `this` gets undefined. Needs `.call(config.scrollbar, value)`. Also `check_mousemove_on_next_draw` is never reset to false, so after the first scroll a synthetic mousemove runs on every draw tick.

### [LOW] `cardToIcon` HTML ends up in `img.alt`
`card_hand.ts createCardEl`: `img.alt = cardToIcon(data.card)` uses the default `render_html = true`, so the alt attribute contains a `<span style=...>` blob. Pass `false`.

### [LOW] Lobby message_handler fallthrough block
`'room-join-failed'/'room-refresh-failed'/'room-leave-failed'` call `lobby.leaveRoom()` and then fall through (uncommented) into the generic error-dialog cases. Kicking the user out of their room because one `room-refresh` failed is aggressive, and the silent fallthrough will bite the next editor. The same function also declares `const`s inside un-braced `case`s throughout (no-case-declarations).

### [LOW] Module-level mutable state in game message_handler
`error_count` and `running_updates` are module globals, shared across any current/future `DwgGame` instances and never reset on relaunch, so errors from a previous session count toward the next session's `MAX_ERROR_COUNT`.

---

## 6. Frontend: smells and refactors

- **`DwgElement` parse-by-polling**: `until()` polls every 300ms, so nested component readiness adds up to 300ms latency per level and burns timers. `connectedCallback` also re-runs full init (wiping innerHTML/state) if an element is ever re-attached. Consider `queueMicrotask`/`requestAnimationFrame` polling or explicit child-ready events; and the `@ts-ignore` property-injection in `elementsParsed` trades all type safety for the string-name convention (a `Proxy`-free alternative: pass a typed record of refs).
- **`customElements.define` guards are inconsistent**: only `game.ts` guards with `customElements.get`. Since each game bundle is loaded as a separate script and shared modules are duplicated per bundle, an unguarded `define` in a shared component will throw the second time a different game bundle loads. Guard uniformly (or better, split shared chunks in webpack).
- **`serverResponseToGameSettings`** uses `@ts-ignore: trust the backend`; `game_settings.ToFrontend` on the backend stringifies every number, forcing `parseInt` scattered across the frontend. Send numbers as numbers and delete both hacks.
- **GameType enums duplicated** (backend `game_type.go`, frontend `data_models.ts`) plus the hardcoded `> 4` check; three places to update per new game. At minimum add a comment linking them; ideally generate one from the other.
- **DEV flags in three places** (backend `main.go`, frontend `util.ts`, `version.js` for deploys) that must be flipped in lockstep by hand per README. Use build-time injection (webpack DefinePlugin / go build tags or env).
- **`apiToUrl` appends a trailing slash** that gin's registered routes don't have, so every API call takes a 30x redirect round trip. Drop the slash.
- **`atangent` (scripts/math.ts)** claims "correct atan in [0, 2pi)" but actually returns the angle mirrored (clockwise convention). If that is intentional for screen-space y-down, the doc comment should say so; as documented it is wrong.
- **`clampNumber`**: `isNaN(undefined)` is already true, so the `=== undefined/null` arms are dead; signature says `number` anyway.
- **`removeLinebreaks` (scripts/util.ts)** contains two leftover debug `console.log`s that run on every call.
- **`ColorRGB.getString`** emits legacy 4-arg `rgb()`; use `rgba()` or the modern slash syntax.
- **`canvas_board.setView`** clamps view to `[0, board_size * scale]`; the max should subtract the viewport size, so the board can currently be scrolled entirely off-screen. Wheel zoom is also not anchored at the cursor (`setScale` scales the view from the origin), which shows as drift when zooming.
- **`canvas_board.setMaxScale`**: `if (!!scale_ratio)` is always true for any valid ratio; the else branch is unreachable except NaN.
- **`drawNgon`** logs an error for n < 3 but draws anyway; **`getFittedFont`** divides by `measureText` width without a zero check (empty string -> Infinity, currently benign).
- **`lobby_rooms.scss`** styles `> #user-container`, but the HTML has `#room-container`; the room-list styling is dead CSS.
- **Duplicated ping-icon logic** in `room_user.ts`, `lobby_user.ts`, `players_dialog_player.ts` (same 4-branch threshold ladder three times); extract a helper.
- **Duplicated card-name logic** backend `card_util.go` vs frontend `card_util.ts` is unavoidable across languages, but the two disagree on fallback strings ('Error' vs 'error' vs icons); harmless today, but worth a shared spec comment.
- **`isGameMessage`/`isLobbyMessage`** use `!prefixes.every(p => !kind.startsWith(...))`; `some` says the same thing without the double negative. Both handlers stay attached to the socket simultaneously while in-game by design; a comment at the two `isXMessage` filters would make that intent explicit.
- **`createLock` queue recursion**: each queued fn awaits the next, building a promise chain proportional to queue depth; fine at current scale, but `while (queue.length)` is simpler and flat.
- **`DwgGame` waiting-room rebuild** and `DwgLobbyRoom.setRoom` fully rebuild DOM on every refresh (every 3.5s poll for the room panel); noticeable churn once rooms are busy, and it defeats the `:hover` button states mid-refresh.

---

## 7. Protocol / architecture observations

- **Polling everywhere**: the lobby REST-polls full room and user lists every 3.5s per client even though a websocket is open and already carries all mutation events. The poll is effectively masking lost-message bugs (see section 2). Fixing delivery and dropping the poll removes a lot of load and code (`refreshed` bookkeeping in lobby_rooms/lobby_users).
- **The update-id resync dance** (`handleGameUpdate`) re-requests missed updates from the server even when they are already sitting in `game_base.updates`; the local map plus one "request gaps only" path would simplify it. `running_updates` as a module-level bool is the fragile part (noted above).
- **Waiting-room stall**: a game only starts when every player connects (`GameBase.PlayerConnected`); if one player closes the tab after launch, the room hangs forever (host can't relaunch: "Game already launched", and the 45s empty-room timer only arms after a game has started disconnect flow). Needs a launch timeout or host cancel.
- **Host leaving an unstarted room deletes it** (TODO acknowledged in code and PLANS.md); combined with `lobby-left` handling on the frontend this is the documented "host leaves" feature gap.
- **`GameBase.player_updates` grows unboundedly** for long games (every action retained); fine until v0.9 persistence, then it becomes the natural event log; worth noting it is currently also served in full to any "viewer" (see security).

---

## 8. Suggested priority order

1. XSS in chatbox, game-fetch state disclosure, sender spoofing (small, contained fixes).
2. Concurrency: pick actor-or-mutex and apply it to Lobby/LobbyRoom/GameBase; run `-race` in dev.
3. `playerGameUpdates` flush/`break` bug + blocking `AddUpdate` (fixes the known reconnect bug in PLANS.md).
4. Frontend inverted conditions (`MultiMap.keys`, `players_dialog_player`, `DwgButton.mousemove`) and `playCard` index desync.
5. Reconnect token + origin check before wider deployment.
6. Leak cleanup (intervals/listeners/goroutines/games map).
