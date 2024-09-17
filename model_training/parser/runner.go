package parser

import (
	"fmt"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func (i *Input) Run() error {
	fmt.Println("")
	fmt.Println("Begin Run")
	for _, g := range i.games {
		e := g.run()
		if e != nil {
			return e
		}
	}
	return nil
}

func (i *InputGame) run() error {
	fmt.Println("")
	fmt.Println("Begin Game")
	f := i.initialize()
	go startGame(f)
	<-f.GetBase().GameEndedChannel
	// TODO: stop the start game loop
	fmt.Println("End Game")
	return nil
}

func (i *InputGame) initialize() *fiddlesticks.GameFiddlesticks {
	game_base := &game.GameBase{
		GameEndedChannel: make(chan string),
	}
	f := fiddlesticks.InitializeGame(game_base)
	f.SetSettings(i.min_round, i.max_round, i.round_points, i.trick_points)
	for _, p := range i.players {
		f.AddInternalAiPlayer(p.ai_model_id)
	}
	return f
}

func startGame(f *fiddlesticks.GameFiddlesticks) {
	f.GetBase().StartGame()
	f.StartAiGame()
	for {
		if f.ExecuteAiTurn() {
			break
		}
	}
}
