v0.8: Playable Risq
 y: Show orders on map
 z: Attack order (combat implementation)
 za: Summary report
 zb: Design minimal playable tech tree
 UI Issues:
  - Moving to active unit when clicking next idle will take you in the black somewhere
  - Opening left panel with idle button on a vil automatically sets hover state of build housing to true (why is that idk)
  - Need to show pop capped in UI

Future Risq Plans:
 - Make summary report animated and can view it after close (bottom bar)
 - Design basic building / unit / tech trees -> UX to show these (bottom bar)
 - Design scoring system
 - Settings setup (including alt win conditions)
 - Drag multiselect
 - Control groups
 - Color logic for attacks
 - Different maps / terrain / etc
 - Ownership edge cases and "regions"
 - Mercenaries
 - Partial move pathfinding

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
 - Egyptian rat crap
 - Ben games?
