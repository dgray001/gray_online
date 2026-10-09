v0.9: Risq beta version
 x: Revamp summary report (fix colors, can see "events" on map, can interact with events i.e. replay them or see what happened precisely)
 y: Settings setup (including alt win conditions, map selector, settings for maps like regions/resources/etc, etc)
 z: Alternate terrain images (mountains)

Small Risq Issues:
 - Controls
  => Need to be able to select multiple buildings
  => Units in control group(s) should show their number(s) when selected on map as little numbers on bottom right
  => Should show ctrl groups in bottom panel somehow with a way to clear them
  => If you have entire ctrl group selected exactly then assigning it a new number should clear previous number (but only if entire group exactly selected)
 - Orders
  => Need to find a better way to show attack zone/space on map (consider options)
  => Unit/building range needs to be shown better (maybe on hover explain what it is?)
 - Minimap
  => Minimap shouldn't show terrain beyond terrain type on anything other than default view mode and it should be more explicit in showing mil / resources in appropriate modes
  => Don't change unit dots based on zoom
  => Show resources on minimap
 - Economy
  => Buildings that can't build units shouldn't be counted as idle
  => Show idle units/buildings in bottom panel with ways to filter/turn off for certain groups
  => Default map sizes might be a bit too large
  => Wood gathering seems slow we need a way to speed it up (more techs!)
 - Combat
  => If enemy has massive unit stack there is no way to specify which ones to attack
  => Some way to set default stance for when units get built

Bigger Risq Issues/Ideas:
 - World risq map
 - Expanded level 2 tech tree
    => Magic and color damage
    => Walls
    => Stables and Archery range

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
