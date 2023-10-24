v0.3: flesh out fiddlesticks game
 l: rejoin room bugs:
   => show trump card
   => remove trump card at end of game
   => can't seem to receive updates (check update applying logic)
   => When rejoining seem to receive empty udpate ?? what's that about?
 m: lobby properly resets when game is over
 n: loaders for client requests in lobby: room-create, room-join, room-leave, room-rename
 o: beginnings of simple chat emojis and taunts
 p: player's button in abstract game header => opens abstract dialog box
 q-u: redo styling for desktop version of site
 v-z: create mobile version of site

v0.4: launch risq
 - can launch a simple game of risq

v0.5: save risq game
 - need fully functioning db and user profiles

P3/P4:
 - frontend animations
 - chat updates have an id that ensures the correct order (can request missing)
 - add abstract dialog then add confirm dialog for confirming actions
 - when joining a lobby room determine if game is currently in progress => then check if joinee is a player => if not join as viewer
 - Viewer updates stored in game base => backend sends viewer updates
 - Resign button causes resign => AI takes over
 - When dc'ed the AI takes over temporarily => if dc'ed for too long the AI takes over
 - Can launch game with AI
