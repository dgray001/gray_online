v0.8: Playable Risq
 q: Stamina and stamina costs shown on frontend
 r: Orders are human-readable on frontend
 s: Can cancel orders from left or right panel
 t: Backend recalculates vision each turn
 u: Can toggle between views UX
 v: Terrain view (default) and control view
 w: Buttons to toggle between summary or other overlay views (for each space)
 x: Add resources to summary view
 y: Implement other views (buildings, villagers, military, resources)
 z-zc: Left panel updates to handle all this including sending orders to all selected and selecting things directly from map
 zd-zg: Show orders on map from selected things (arc lines for movement, etc)
 zh-zz: Risq is playable (? maybe ?)

Risq Plans:
 - Summary report after each turn with animations (configurable)
 - Combat implementation with color logic, etc
 - Idle icons and button to go to next idle thing
 - Design basic building / unit / tech trees -> UX to show these??
 - Limitations on number of units in a space / zone?
 - Design scoring system
 - Logic around game ending

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
