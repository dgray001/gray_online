Bigger Risq Issues/Ideas:
 - Utilize events on map
 - For v0.8 ai I'd want it to be tech-tree agnostic at least to some extent maybe? Or at least stat-agnostic so it reads current stats ... it also needs to be support many different maps
 - Sim-only fast path: build the AI snapshot directly from engine state
 - Expanded level 2 tech tree
    => Magic and color damage
    => Walls
    => Stables and Archery range
    => Wood gathering seems slow we need a way to speed it up (more techs!)

Fiddlesticks Plans:
 - Host can pause game and if host leaves then someone else takes over as host

Lobby Plans:
 - Can chat with individual players
 - Upgraded chatbox => emoji selector, taunts, message id, turn off emoticon converter

Testing Plans:
 - Formalize the ad-hoc concurrency/reconnect repro scripts (currently one-off Node scripts written per-investigation) into a real backend test suite instead of relying on agents to improvise and run them each time

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
