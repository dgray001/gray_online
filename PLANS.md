v0.3: flesh out fiddlesticks game
 d: frontend can fully restore state => can refresh page safely
 e: lobby connector will ask if you want to reconnect after it verifies that the stored id is valid
 f: frontend abstract game history modal
 g: handle lobby room failures by syncing state
 h: handle all other failures by syncing state
 i: upgrade chatbox styling for normal and in-game
 j: remove all panics from backend (no crashing server)
 k: room selector component in lobby list
 l: room selector gives info, lets you join as viewer
 m: proper checks for max players/max viewers when joining lobby room
 n: alternative rules for fiddlesticks => abstract that section
 o: relaunch button in lobby to rejoin room
 p: lobby properly resets when game is over
 q: loaders for client requests in lobby: room-create, room-join, room-leave, room-rename
 r: beginnings of simple chat emojis and taunts
 s: player's button in abstract game header => opens abstract dialog box
 t: chatbox lets you know there's new messages
 u-w: redo styling for desktop version of site
 x-y: create mobile version of site

v0.4: launch risq
 - can launch a simple game of risq

v0.5: save risq game
 - need fully functioning db and user profiles

P3/P4:
 - game launching message shouldn't come from anyone and should be gray for participants
 - frontend animations
 - chat updates have an id that ensures the correct order (can request missing)
 - add abstract dialog then add confirm dialog for confirming actions
 - when joining a lobby room determine if game is currently in progress => then check if joinee is a player => if not join as viewer
 - Viewer updates stored in game base => backend sends viewer updates
