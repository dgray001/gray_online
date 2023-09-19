
v0.2: launch game
 n: frontend abstract in-game chat
 o: abstract header (to exit, see players/viewers) in-game
 p: fiddlesticks frontend layout
 q: backend abstract game finish/exit
 r: game update messages for 
 s: can see scores / list of other players
 t: card images
 u: can bet
 v: players can see their dealt cards
 w: can play cards
 x: can score
 y: can see winner when game ends

v0.3: flesh out fiddlesticks game
 - can reconnect as anonymous user
 - chat emojis and taunts
 - mobile version of site
 - animations
 - alternative rules

v0.4: launch risq
 - can launch a simple game of risq

v0.5: save risq game
 - need fully functioning db and user profiles

P3/P4:
 - loader for client requests in lobby: room-create, room-join, room-leave, room-rename
 - client still leaves room if it fails => sync state
 - handle other failures by syncing state => show message
 - chatbox lets you know there's new messages
 - room selector component (can join as viewer)
 - backend (and frontend) prevents joining room when max players
 - remove all panics from backend
 - game launching message shouldn't come from anyone and should be gray for participants