v0.4: Euchre game
 l: Frontend initial gameUpdate
 m: Frontend animations / styles (including mobile)
 n: Viewer updates stored in game base => backend sends viewer updates
 o: Proper viewer support for fiddlesticks
 p: Proper viewer support for euchre
 q: Abstract moves dialog
 r: Proper moves dialog for fiddlesticks and euchre
 s: Abstract players dialog in game base
 t: Frontend styles / UX cleanup
   => make dialog buttons bigger
   => can play card with double click
 u: Lobby rooms can just refresh specific rooms instead of recreating entire els
 v: Game info button in game base (left of game updates button) => has all game specific settings and description, etc...
 w: Host can set description of room
 x: Can see version number of app in-game somehow

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