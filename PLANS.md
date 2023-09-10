
v0.2: launch game
 g: room host can choose game and max players and max viewers
 h: room host can make someone a viewer in room
 i: room selector component (can join as viewer)
 j: room host can launch game => fullscreen game component
 k: backend abstract game launch / finish / exit
 l: abstract messaging and header (to exit) in-game
 m: abstract game update messages somehow
 n: game update messages for fiddlesticks
 o: finish needed api for fiddlesticks
 p-q: finish initial frontend for fiddlesticks

v0.3: risk outline
 - can launch a simple game of risq


P3/P4:
 - loader for client requests in lobby: room-create, room-join, room-leave, room-rename
 - client still leaves room if it fails => sync state
 - handle other failures by syncing state => show message
 - chatbox lets you know there's new messages