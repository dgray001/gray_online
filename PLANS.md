v0.3: flesh out fiddlesticks game
 p: Universal font sizes, button, and form field styles
 q: Mobile lobby page
 r: Mobile lobby component
 s: Mobile lobby chatbox component
 t: Mobile lobby users component
 u: Mobile lobby room component
 v-w: Desktop fiddlesticks layout/styles
 x: Mobile game styles
 y-z: Mobile fiddlesticks styles

v0.4: launch risq
 - can launch a simple game of risq

v0.5: save risq game
 - need fully functioning db and user profiles

P3/P4:
 - Player's dialog box in DwgGame
 - Loaders for client requests in lobby: room-create, room-join, room-leave, room-rename
 - Frontend animations
 - Chat updates have an id that ensures the correct order (can request missing)
 - Add abstract dialog then add confirm dialog for confirming actions
 - When joining a lobby room determine if game is currently in progress => then check if joinee is a player => if not join as viewer
 - Viewer updates stored in game base => backend sends viewer updates
 - Resign button causes resign => AI takes over
 - When dc'ed the AI takes over temporarily => if dc'ed for too long the AI takes over
 - Can launch game with AI
 - Proper viewer logic at every step
 - Chat emoji selector
 - Chat taunts
 - Can turn off emoticon converter in chat
