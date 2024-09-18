package parser

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"fiddlesticks.live/utils"
	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
)

func (i *Input) Run() error {
	time_start := time.Now().UTC()
	fmt.Println("")
	fmt.Println("Begin Run")
	for _, g := range i.games {
		e := g.run()
		if e != nil {
			return e
		}
	}
	for _, b := range i.benchmarks {
		e := b.run()
		if e != nil {
			return e
		}
	}
	for _, b := range i.standard_benchmarks {
		e := b.run()
		if e != nil {
			return e
		}
	}
	time_dif := time.Now().UTC().Sub(time_start)
	fmt.Println("\nEnd Run")
	fmt.Println("Total Time:", time_dif.String())
	return nil
}

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
		fmt.Println("AI player", i, "score", s)
	}
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

func startGame(f *fiddlesticks.GameFiddlesticks, debug bool) {
	f.GetBase().StartGame()
	f.StartAiGame(false)
	for {
		if f.ExecuteAiTurn(debug) {
			break
		}
	}
}

func (i *InputBenchmark) run() error {
	fmt.Println("")
	time_start := time.Now().UTC()
	fmt.Println("Begin Benchmark")
	data := make([]*[]string, i.max_players-i.min_players+1)
	player_test := &InputPlayer{
		ai_model_id: i.ai_model_id,
	}
	player_random := &InputPlayer{
		ai_model_id: 0,
	}
	file_string := ""
	for r := i.min_round; r <= i.max_round; r++ {
		file_string += " R" + strconv.FormatInt(int64(r), 10)
	}
	file_string += "\n"
	for p := range data {
		players := make([]*InputPlayer, i.min_players+uint8(p))
		for p_i := range players {
			if p_i == 0 {
				players[p_i] = player_test
			} else {
				players[p_i] = player_random
			}
		}
		player_row := make([]string, i.max_round-i.min_round+1)
		for r := range player_row {
			round := i.min_round + uint8(r)
			if int(round)*len(players) > 51 {
				player_row[r] = "-"
				continue
			}
			input_game := &InputGame{
				debug:        false,
				min_round:    round,
				max_round:    round,
				trick_points: i.trick_points,
				round_points: i.round_points,
				players:      players,
			}
			score := float64(0)
			for j := uint32(0); j < i.iterations; j++ {
				f := input_game.initialize()
				go startGame(f, false)
				<-f.GetBase().GameEndedChannel
				max_score, scores := f.GetGameResults()
				score += float64(scores[0]) / float64(max_score)
			}
			player_row[r] = strconv.FormatFloat(score/float64(i.iterations), 'f', 3, 64)
		}
		data[p] = &player_row
		row_string := "P" + strconv.FormatInt(int64(len(players)), 10) + " " + strings.Join(player_row, " ")
		file_string += row_string + "\n"
	}
	file_data := []byte(file_string)
	err := os.WriteFile("outputs/"+i.output+".txt", file_data, 0644)
	utils.ErrorCheck(err, "Error writing file")
	fmt.Println("End Benchmark")
	time_dif := time.Now().UTC().Sub(time_start)
	fmt.Println(time_dif.String())
	return nil
}

func (i *InputStandardBenchmark) run() error {
	benchmark := &InputBenchmark{
		iterations:  i.iterations,
		min_players: 2,
		max_players: 8,
		min_round:   1,
		max_round:   12,
		ai_model_id: i.ai_model_id,
	}
	output_string := fmt.Sprintf("benchmark_%s_", i.output)
	benchmark.round_points = 1
	benchmark.trick_points = 0
	benchmark.output = output_string + "round1"
	benchmark.run()
	benchmark.round_points = 0
	benchmark.trick_points = 1
	benchmark.output = output_string + "trick1"
	benchmark.run()
	benchmark.round_points = 10
	benchmark.trick_points = 1
	benchmark.output = output_string + "standard"
	benchmark.run()
	benchmark.round_points = 10
	benchmark.trick_points = 3
	benchmark.output = output_string + "modern"
	benchmark.run()
	return nil
}
