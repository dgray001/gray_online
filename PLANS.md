
v0.2: launch game
 x: abstract game tracks game-update by order
 y: frontend abstract game history button

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
 - frontend animations
 - game updates have an id that ensures the correct order (can request missing)
 - chat updates have an id that ensures the correct order (can request missing)
 - add abstract dialog then add confirm dialog for confirming actions
 - when joining a lobby room determine if game is currently in progress => then check if joinee is a player => if not join as viewer
 - In lobby room have a relaunch button to rejoin room
 - In waiting room have a relaunch button
 - Player's button in abstract game header => opens abstract dialog box
