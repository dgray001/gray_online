v0.9: Risq beta version
 v: Backend risq tests
 w: Revamp summary report (fix colors, can see "events" on map, can interact with events i.e. replay them or see what happened precisely)
 x: Make v1.0 ai (current ones should be rename to 0.4, etc...)
 y: Settings setup (including alt win conditions, map selector, settings for maps like regions/resources/etc, etc)
 z: Alternate terrain images (mountains, swamps, shallows, water)

Small Risq Issues:
 - Game ending and defeat
  => Frontend didn't show game ended when I defeated all other players (investigate)
  => Also defeated should happen when you lose all units + all unit-producing buildings
 - Order behavior bugs
  => Gather point to zone seems to be auto-doing building ... that is bad look into it
  => Building attack unit orders need to go away when unit is out of range or dead
  => Confirm that priority will be ahead of distance in target selection
 - Selection, control groups, and navigation
  => Need to be able to select multiple buildings
  => Can double click control group to move to where it is
  => Hotkey to move to current selection (default should be space)
  => Units in control group(s) should show their number(s) when selected on map as little numbers on bottom right
  => Should show ctrl groups in bottom panel somehow with a way to clear them
  => If you have entire ctrl group selected exactly then assigning it a new number should clear previous number (but only if entire group exactly selected)
 - Idle management and command shortcuts
  => Buildings that can't build units shouldn't be counted as idle
  => Next idle should also go to buildings
  => Show idle units/buildings in bottom panel with ways to filter/turn off for certain groups
  => If holding ctrl when right click to build building then it should stay armed (true of any armed anything)
  => Hotkey to open window should also close it (tech tree, turn report)
 - Order presentation and pointer feedback
  => Orders list says "attack zone" / "attack space" (check what it says for attack unit/building)
  => Auto attacks show as "order"
  => Need to find a better way to show attack zone/space on map (consider options)
  => Setting gather to foundation doesn't draw arrow to foundation (but it works for resource and unit and built building)
  => Order arrows from a space in space view should originate from where the unit would be not just in the center of zone
  => Unit/building range needs to be shown better (maybe on hover explain what it is?)
  => Tooltips shouldn't be behind cursor they should ALWAYS be such that they don't overlap with where cursor is. Check logic on this
  => Sword (and other) cursors need to point to cursor point
 - Zoom, detail, and map modes
  => I can zoom out more on zone view
  => Should be able to zoom in more in general
  => Zooming in more should allow for more unit circles
  => When not in default view mode then draw detail threshholds can be a bit bigger imo
  => Minimap shouldn't show terrain beyond terrain type on anything other than default view mode and it should be more explicit in showing mil / resources in appropriate modes
  => Gold icon should be shown in region view mode
 - Gathering and production automation
  => After renewing a building the vils should auto-gather from it (similar to when they are done building)
  => In farms (or any gatherable building) you should be able to q up "auto-renew" any number of them and they should automatically work for ANY building *of the same building id*
  => Should be able to gather point inside building and shouldn't cost stamina (special case where unit is built directly inside building)
 - Combat targeting and defaults
  => If enemy has massive unit stack there is no way to specify which ones to attack
  => Some way to set default stance for when units get built
  => Units should heal when garrisoned
 - Game pacing and economy
  => Default map sizes might be a bit too large
  => Wood gathering seems slow we need a way to speed it up (more techs!)
 - Assets and combat aftermath
  => Actually use the images for 3 blacksmith techs
  => All buildings should have a "damaged" version when like 60% health and another when 20% health along with little fires on it to show damage
  => Rubble should be shown on map for recently killed buildings and bodies for recently killed units

Bigger Risq Issues/Ideas:
 - Tests should have their own configs for units/techs/buildings optionally to test things independent of actual configs
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
