Small Risq Issues:
 - Unsubmit orders I lost my previous orders
 - Game keeps on disconnecting in risq while I play idk why (need automatic reconnect to socket)
 - Resources spent this turn don't show while orders submitted and waiting
 - Autoattacks need to rechoose target when higher priority target comes in range ?
 - After submitting orders I should be able to highlight say build buttons and see how much something costs
 - Sometimes when I zoom out there are some zoom levels that hide cursor and panels for some reason

Bigger Risq Issues/Ideas:
 - Utilize events on map
 - Sim-only fast path: build the AI snapshot directly from engine state
 - Expanded level 2 tech tree
    => Magic and color damage
    => Walls
    => Stables and Archery range
    => Wood gathering seems slow we need a way to speed it up (more techs!)

Fiddlesticks Plans:
 - Update turn timer UX
 - Host can pause game and if host leaves then someone else takes over as host

Lobby Plans:
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
