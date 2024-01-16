v0.5: Can launch risq

 - Implement game info dialog => has game specific settings, description, etc...
 - Host can set description of room
 - Loaders for client requests in lobby: room-create, room-join, room-leave, room-rename

v0.6: Advanced risq

 g: Collapsible right panel (shows turn number)
 h: Collapsible right panel shows player resources (space for future minimap)
 i: Closable left panel
 j: Selecting building opens building in left panel
 k: Selecting resources opens them in left panel
 l: Selecting units opens them in left panel => is group of units if 2+
 m: Can select individual units from left panel
 n: More advanced visibility possibilities
 o: Show who controls a space with mixed control for current battles
 p: Right panel has all player's "scores"
 q: Right panel has whether players are finished with orders (UI to finish with orders)
 r: API to handle orders update
 s: Backend processes unit move orders
 t: Frontend processes turn outcome => summary report dialog
 u: Villager gather order
 v: Villager build order

 - Proper updates dialog for fiddlesticks and euchre
 - Can launch games with AI (all games must handle)
 - Timed turns (or not) and can pause game if timed
 - AI takes over when player AFK
 - Can resign from game => AI takes over
 - Player's cards can automatically resize => listener on game div resize
   - fullscreen mode need to recalc card horizontal line
 - Can chat with individual players

 - Setup db in prod and dev
 - Can create profile / login
 - Reporting => admin login can access admin page to see reports, etc.
 - Can report bugs / email admin / etc.
 - Advanced risq mechanics
    => various terrains / resources
    => various buildings
    => various units / can assign formations/etc. to control how they fight
 - Can make friends / personal DMs (all DMs and chatboxes saved)
 - Can see other people's stats

v0.7: Savable games
 - Can save games if everyone logged in and all players agree
 - Can launch a saved game if logged in
 - Risq is fully playable with custom settings
 - Upgraded chatbox => emoji selector, taunts, message id, turn off emoticon converter
 - Upgrade frontend animations for all games (add sound effects)

v0.8: More card games
 - Poker => save chips?
 - Egyptian rats crap
 - Abstract card table, card player, and trick cards / animations

v0.9: Add more esoteric games

Games to add:
 - Chess with esoteric variations
