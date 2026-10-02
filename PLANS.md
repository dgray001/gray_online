v0.9: Risq beta version
 v: Backend risq tests
 w: Revamp summary report (fix colors, can see "events" on map, can interact with events i.e. replay them or see what happened precisely)
 x: Make v1.0 ai (current ones should be rename to 0.4, etc...)
 y: Settings setup (including alt win conditions, map selector, settings for maps like regions/resources/etc, etc)
 z: Alternate terrain images (mountains, swamps, shallows, water)

 Small risq issues:
  - Build orders aren't counting toward tmp predicted resource spend in frontend --- or like some are and some aren't idk why
  - Trees (and stone) should change image as they are depleted (and deer and berries)
  - Filters for right panel order list (analyze/design first)

 - World risq map
 - Expanded level 2 tech tree
    => Magic and color damage
    => Stables and Archery range
 - Send a final state/report update before EndGame when elimination ends a risq game (no final start-turn goes out, so clients keep a stale eliminated=false)
 - AI config parse failures should return structured errors (config/action index) instead of falling back to NoopModel, which loses the real parse error

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
