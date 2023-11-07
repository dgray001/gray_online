0.3: flesh out fiddlesticks game
 - Need card hand component to send play_card event
 - Add betting input to fiddlesticks component
 - Refactor the fiddlesticks_player component to not handle betting/playing but just have info
 v: Desktop fiddlesticks layout/styles
 w: Mobile fiddlesticks styles
 x: AI can take over when player AFK
 y: Resign button causes resign => AI takes over permanently
 z: Small fixes:
   => Sometimes updates don't apply properly when refreshing page
   => Need to send ping in lobby for launching game

P3/P4:
 - Basic players dialog in game base
 - Viewer support

v0.4: Euchre game
 - New game backend logic
 - New frontend component
 - Proper viewer logic at every step
 - Viewer updates stored in game base => backend sends viewer updates
 - Can launch games with AI
 - Can pause games
 - Frontend animations for fiddlesticks and euchre
 - Upgrade moves dialog and players dialog in game base
 - Loaders for client requests in lobby: room-create, room-join, room-leave, room-rename

v0.5: More card games
 - Add 4 more card games with settings / etc.
 - Styles and animations for all of them
 - Current game state gets saved in db
 - Can login and save user profile

v0.6: Can launch risq
 - Hexagonal board on frontend that can be scrolled and translated
 - Basic turn logic
 - Create db tables and design rules
 - Initial backend logic
 - Initial frontend logic / animations / styles
 - Can make friends / personal DMs (all DMs and chatboxes saved)
 - Can see other people's shats
 - Upgraded chatbox => emoji selector, taunts, message id, turn off emoticon converter

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