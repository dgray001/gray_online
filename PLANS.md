
v0.2: launch game
 j: room host can launch game => fullscreen game component
 k: backend abstract game launch / finish / exit
 l: abstract messaging and header (to exit) in-game
 m: abstract game update messages somehow
 n: game update messages for fiddlesticks
 o: finish needed api for fiddlesticks
 p-q: finish initial frontend for fiddlesticks

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