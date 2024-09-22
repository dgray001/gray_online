package runner

import (
	"fmt"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func (i *InputGame) run() error {
	fmt.Println("")
	fmt.Println("Begin Game")
	f := i.initialize()
	go startGame(f, i.debug)
	<-f.GetBase().GameEndedChannel
	// TODO: stop the start game loop
	fmt.Println("End Game")
	max_score, scores := f.GetGameResults()
	fmt.Println("Max Score:", max_score)
	for i, s := range scores {
		fmt.Println("AI player", i, "score:", s)
	}
	return nil
}

func (i *InputGame) initialize() *fiddlesticks.GameFiddlesticks {
	game_base := &game.GameBase{
		GameEndedChannel: make(chan string),
	}
	f := fiddlesticks.InitializeGame(game_base)
	for _, p := range i.players {
		f.AddInternalAiPlayer(p.ai_model_id, p.model_input)
	}
	f.SetSettings(i.min_round, i.max_round, i.round_points, i.trick_points, i.iterations)
	return f
}

func startGame(f *fiddlesticks.GameFiddlesticks, debug bool) {
	f.GetBase().StartGame()
	f.StartAiGame(false)
	for {
		if f.ExecuteAiTurn(debug) {
			break
		}
	}
}
