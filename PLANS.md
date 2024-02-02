
v0.6: Visible Risq

NOTES:
 - each zone holds resource OR building (if there's a resource must clear it build)

 f: Can click space to see in right panel info about the space
 g: Can click zone to see in right panel info about that zone
 h: Zoom in more and can see zone info
 i: Closable left panel
 j: Can select building in zone (including empty plot)
 k: Can select unit(s) in zone
 l: Can select unit from unit group in right panel (or shift click to filter by type)
 m: Zones have resources that are selectable
 n: Backend calculates player score and frontend shows it
 o: Frontend has scrollable area for orders (below scores)
 p: Split units in zone by villager and military
 q: Can toggle between views: building/villager/military/resource
 r: Orders abstraction (backend game update of "add order") => each order has unique id
 s: Backend game update of "finish orders" with frontend button (UI to show) => confirm dialog box
 t: Backend orders abstraction on each unit and building
 u: Left panel shows unit / building orders in the order the will complete
 v: Unit and building return list of possible orders based on game state => send to frontend
 w: Frontend can see/add building orders for making unit => population limit
 x: Frontend shows summary report skeleton each turn (backend creates from zone/space/player)
 y: Can cancel existing orders (as part of orders turn)

v0.7: Playable Risq
 o: Can cancel creation of unit (villager)
 p: Movement backend logic
 q: Movement frontend logic
 r: Villager can create building
 s: Villager can gather resources
 t: Idle icon above buildings/units if idle
 u: Limit on number of units in zone
 v: Population limit / houses
 w: Barracks / soldier
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

v0.8: Advanced Risq
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
