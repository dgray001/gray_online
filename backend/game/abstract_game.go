package game

type GameBase struct {
	Game_id      uint64
	Players      map[uint64]*Player
	Viewers      map[uint64]*Viewer
	game_started bool
	game_ended   bool
}

type Game interface {
	CreateGame(g *GameBase) Game
	StartGame()
}

func (g *GameBase) GameStarted() bool {
	return g.game_started
}

func (g *GameBase) StartGame() {
	if g.game_started {
		panic("Game already started")
	}
	g.game_started = true
}

func (g *GameBase) gameEnded() bool {
	return g.game_ended
}

func (g *GameBase) EndGame() {
	if g.game_ended {
		panic("Game already ended")
	}
	g.game_ended = true
}

type Player struct {
	player_id uint64
}

type Viewer struct {
	viewer_id uint64
}
