v0.3: flesh out fiddlesticks game
 j: proper checks for max players/max viewers when joining lobby room => also can't join as player if game started => frontend message for room-join-failed
 k: alternative rules for fiddlesticks => abstract that section
 l: relaunch button in lobby to rejoin room
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
 - game launching message shouldn't come from anyone and should be gray for participants
 - frontend animations
 - chat updates have an id that ensures the correct order (can request missing)
 - add abstract dialog then add confirm dialog for confirming actions
 - when joining a lobby room determine if game is currently in progress => then check if joinee is a player => if not join as viewer
 - Viewer updates stored in game base => backend sends viewer updates
 - Add function lock => replace LobbyRoom::refresh_rooms_running with function lock
 - Resign button causes resign => AI takes over
 - When dc'ed the AI takes over temporarily => if dc'ed for too long the AI takes over
 - Can launch game with AI
