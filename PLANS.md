
v0.6: Playable Risq
 b: Collapsible right panel shows player resources (space for future minimap)
 c: Closable left panel
 d: Selecting building opens in left panel
 e: Selecting empty plot opens in left panel (empty building plot in left panel)
 f: Selecting unit opens them in left panel => group if 2+
 g: Can select unit from group or shift click to select of type
 h: Resource images / backend (each zone has one resource)
 i: Backend calculates player "score" and right panel shows it
 j: Orders abstraction
 k: UI to "finish" orders and right panel shows who is finished with orders
 l: Units and buildings have an array of orders they will complete in order
 m: Can see orders on left panel for building
 n: Can add to orders by creating a unit (villager)
 o: Can cancel creation of unit (villager)
 p: Movement backend logic
 q: Movement frontend logic
 r: Villager can create building
 s: Villager can gather resources
 t: Idle icon above buildings/units if idle
 u: Limit on number of units in zone
 v: Population limit / houses
 w: Barracks / soldier
 x: Backend creates summary report for each zone/space/player
 y: Frontend shows summart report
 z: Attack order => can win

 - Proper updates dialog for fiddlesticks and euchre
 - Can launch games with AI (all games must handle)
 - Timed turns (or not) and can pause game if timed
 - AI takes over when player AFK
 - Can resign from game => AI takes over
 - Player's cards can automatically resize => listener on game div resize
   - fullscreen mode need to recalc card horizontal line
 - Can chat with individual players
 - Loaders for client requests in lobby: room-create, room-join, room-leave, room-rename

v0.7: Advanced Risq
 - Can save games if everyone logged in and all players agree
 - Can launch a saved game if logged in
 - Risq is fully playable with custom settings
 - Upgraded chatbox => emoji selector, taunts, message id, turn off emoticon converter
 - Upgrade frontend animations for all games (add sound effects)

 - Setup db in prod and dev
 - Can create profile / login
 - Reporting => admin login can access admin page to see reports, etc.
 - Can report bugs / email admin / etc.
 - Advanced risq mechanics
    => various terrains / resources
    => various buildings
    => various units / can assign formations/etc. to control how they fight
 - Can make friends / personal DMs (all DMs and chatboxes saved)
 - Can see other people's stats

v0.8: More card games
 - Poker => save chips?
 - Egyptian rats crap
 - Abstract card table, card player, and trick cards / animations

v0.9: Add more esoteric games

Games to add:
 - Chess with esoteric variations
