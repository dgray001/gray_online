 - After reconnecting sometimes in backend the connection is nil (or delete_timer is not nil)
    => noticed when using client_id url param but not sure if that matters
    => connection is definitely not nil since update goes through from client to server just not vice versa
 
 - Fiddlesticks updates grouped by round and trick, includes trick winner and turn points
 - Proper updates dialog for euchre
 - Timed turns (or not) and can pause game if timed
 - AI takes over when player AFK
 - Can resign from game => AI takes over
 - Loaders for client requests in lobby: room-create, room-join, room-leave, room-rename
 - Can chat with individual players
 - Upgraded chatbox => emoji selector, taunts, message id, turn off emoticon converter
 - Upgrade frontend animations for all games (add sound effects)
 - Setup db in prod and dev
 - Can create profile / login

v0.7: Playable Risq
 u: Abstract scrollbar
 v: Scrollable area for orders for buildings / units => add order list in backend and data on frontend
 w: Can see all new orders in right panel with button to "finish orders" and can see who is finished
 x: Order to build unit in building => update works => population limit
 y: Frontend shows summary report skeleton each turn (backend creates from zone/space/player)
 z Can toggle between views: building/villager/military/resource
 r: Orders abstraction (backend game update of "add order") => each order has unique id
 u: Left panel shows unit / building orders in the order the will complete
 v: Unit and building return list of possible orders based on game state => send to frontend
 y: Can cancel existing orders (as part of orders turn)
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

 - Can select all units in zone / space
 - Can launch games with AI (all games must handle)
 - Player's cards can automatically resize => listener on game div resize
   - fullscreen mode need to recalc card horizontal line

v0.8: Advanced Risq
 - Can save games if everyone logged in and all players agree
 - Can launch a saved game if logged in
 - Risq is fully playable with custom settings

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
