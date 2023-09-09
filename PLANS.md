v0.1: basic lobby functionality
 - using apis for join/leave/etc instead of socket, so is a security hole
 - rename rooms
 - see players in room
 - host can kick players in room
 - host can reassign host

v0.2: launch game
 - backend connects to mysql
 - can launch games (new connections => get routed into game hub)
 - implement tic tac toe
 - abstract in-game logic => like chatting/exiting/etc

v0.3: risk outline
 - can launch a simple game of risq