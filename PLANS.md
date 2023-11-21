v0.4: Euchre game
 g: Frontend elements
 h: Frontend initial gameUpdate
 i: Frontend animations / styles (including mobile)
 j: Viewer updates stored in game base => backend sends viewer updates
 k: Proper viewer support for fiddlesticks
 l: Proper viewer support for euchre
 m: Abstract moves dialog
 n: Proper moves dialog for fiddlesticks and euchre
 o: Abstract players dialog in game base
 p: Frontend styles / UX cleanup
   => make dialog buttons bigger
 r: Lobby rooms can just refresh specific rooms instead of recreating entire els
 s: Game info button in game base (left of game updates button) => has all game specific settings and description, etc...
 t: Host can set description of room
 u: Can join room via link
 v: Can see version number of app in-game somehow

v0.5: More card games
 - Add 2 more card games
    => one should be poker => start with chips and winner is one with all chips => initially just 5 card draw
    => other should be egyptian rat crap with fun slap animations
    => abstract out the table / players / played card component and animations
 - Can launch games with AI (all games must handle)
 - Timed turns (or not) and can pause game if timed
 - AI takes over when player AFK
 - Room can be removed when all players leave for x amount of time
 - Can resign from game => AI takes over
 - Loaders for client requests in lobby: room-create, room-join, room-leave, room-rename
 - Player's cards can automatically resize => listener on game div resize
 - Upgraded chatbox => emoji selector, taunts, message id, turn off emoticon converter
 - Upgrade frontend animations for all games (add sound effects)

v0.6: Can launch risq
 - Hexagonal board on frontend that can be scrolled and translated
 - Basic turn logic
 - Create db tables and design rules
 - Initial backend logic
 - Initial frontend logic / animations / styles
 - Can make friends / personal DMs (all DMs and chatboxes saved)
 - Can see other people's shats

v0.7: Advanced risq
 - Can save games if everyone logged in and all players agree
 - Reporting => admin login can access admin page to see reports, etc.
 - Can report bugs / email admin / etc.
 - Advanced risq rules / variations

v0.8: Chess games
 - Various esoteric chess rules / basic chess rules as well

v0.9: Other esoteric games
 - Add 4 esoteric board games not found elsewhere on internet

Games to add:
 - Euchre
 - Poker(s) => save chips?
 - Risq
 - Chess with esoteric variations