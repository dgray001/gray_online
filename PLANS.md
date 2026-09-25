v0.9: Risq beta version
 o: Regions
 p: Mercenaries
 r: Revamp summary report
 s: Customizable hotkeys

Small Risq Issues:
 - Zone ownership should just follow space ownership
 - Left panel: show live "workers X/Y" indicator on a selected gatherable building (predictedGathererCount already exists in risq.ts, just needs left_panel.ts wiring)
 - Send a moving unit's planned path (backend MoveIntent.path) to the frontend so it can be drawn on the map

 - Expanded level 2 tech tree
    => Magic and color damage
    => Stables and Archery range
    => Settings setup (including alt win conditions)

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
