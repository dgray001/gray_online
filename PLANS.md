v0.8: Playable Risq
 u: Space ownership (passive gold income)
 v: Views toggleable (terrain, ownership, buildings, villagers, military, resources ---- and add resources to summary view if not there)
 w: Research order (orders in left panel)
 x: Orders for groups (and general ui improvements)
 y: Show orders on map (like arc lines for movement)
 z: Attack order (combat implementation with color logic, etc)
 za: Summary report
 zb: Next idle thing button (idle icons, etc)
 zc: Design minimal playable game
 - Unit deletion
 - Ownership logic
 - Partial move pathfinding
 - Pop capped on UI

Future Risq Plans:
 - Make summary report animated and can view it after close (bottom bar)
 - Design basic building / unit / tech trees -> UX to show these (bottom bar)
 - Design scoring system
 - Settings setup (including alt win conditions)
 - Drag multiselect
 - Control groups
 - Different maps / terrain / etc

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
 - After reconnecting sometimes in backend the connection is nil (or delete_timer is not nil)
    => noticed when using client_id url param but not sure if that matters
    => connection is definitely not nil since update goes through from client to server just not vice versa
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
