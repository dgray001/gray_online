
v0.2: launch game
 f: lobby room can launch game => specify game type
 g: lobby can handle game chats/exiting/reconnecting
 h: somehow abstract game update messages
 j: frontend abstract game logic => chatting/exiting/reconnecting
 k: host can control max number viewers
 l: game enum selectable in game lobby
 m: lobby room selector component
 p: form created from custom config => each game has its own
 q-z: finish being able to launch and play fiddlesticks

v0.3: risk outline
 - can launch a simple game of risq


P3/P4:
 - loader for client requests in lobby: room-create, room-join, room-leave, room-rename
 - client still leaves room if it fails => sync state
 - handle other failures by syncing state => show message