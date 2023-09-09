
v0.2: launch game
 h: game has no host => players and viewers
 i: backend abstract game logic => chatting/exiting/reconnecting
 j: frontend abstract game logic => chatting/exiting/reconnecting
 k: host can control max number viewers
 l: game enum selectable in game lobby
 m: lobby rooms shows room name, host name, and game
 n: show pings in lobby
 o: show pings in game room
 p: form created from custom config => each game has its own
 q: tic tac toe config
 r: tic tac toe backend
 s: tic tac toe frontend
 t-q: finish being able to launch and play tic tac toe

v0.3: risk outline
 - can launch a simple game of risq


P3/P4:
 - loader for client requests in lobby: room-create, room-join, room-leave, room-rename
 - client still leaves room if it fails => sync state
 - handle other failures by syncing state => show message