v0.9: Risq beta version
 e: Learnable AI pipeline
 f: Bottom bar in UI (summary report, view buttons, tech tree)
 g: Partial move pathfinding
 h: Different terrains
 i: Map scripting setup
 j: Regions and mercenaries
 k: Magic and color damage
 l: Expanded level 2 tech tree
 m: Settings setup (including alt win conditions)
 n: Military unit stances frontend
 - Gather points (frontend)
 - Can attack when garrisoned (configurable per-unit multiplier where future archers would have more)
 - Garrisoned frontend (can click units and show flag on building)

UI Improvements:
 - Revamp summary report
 - Fully customizable hotkeys
 - Message logic for top of screen (like for pop capped)
 - Pan from mouse on edge of screen doesn't really work
 - Right panel resource rows only show net amount, not spent/gaining/worker breakdown

Fiddlesticks Plans:
 - Revamp update dialog box
 - Update turn timer UX
 - Host can pause game and if host leaves then someone else takes over as host

Lobby Plans:
 - Loaders for client requests in lobby: room-create, room-join, room-leave, room-rename
 - Can chat with individual players
 - Upgraded chatbox => emoji selector, taunts, message id, turn off emoticon converter

Testing Plans:
 - Formalize the ad-hoc concurrency/reconnect repro scripts (currently one-off Node scripts written per-investigation) into a real backend test suite instead of relying on agents to improvise and run them each time

Bugs:
 - Player's cards can automatically resize => listener on game div resize
   - fullscreen mode need to recalc card horizontal line

v0.9: Database
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
