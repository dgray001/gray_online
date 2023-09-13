package game

type GameBase struct {
	Game_id uint64
	// here the map keys are the lobby client ids
	Players      map[uint64]*Player
	Viewers      map[uint64]*Viewer
	game_started bool
	game_ended   bool
}

func CreateBaseGame(game_id uint64) *GameBase {
	return &GameBase{
		Game_id:      game_id,
		Players:      make(map[uint64]*Player),
		Viewers:      make(map[uint64]*Viewer),
		game_started: false,
		game_ended:   false,
	}
}

type Game interface {
	GetId() uint64
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
