package parser

import (
	"fmt"
	"strconv"

	"fiddlesticks.live/utils"
)

type InputObject interface {
	AddLine(q []InputObject, k string, v string) []InputObject
}

type inputStatic struct{}

var InputStatic inputStatic

func (inputStatic) parseInt(s string) int {
	i, e := strconv.Atoi(s)
	if e != nil {
		return 0
	}
	return i
}

func (inputStatic) parseBool(s string) bool {
	trues := []string{"True", "true"}
	for _, t := range trues {
		if t == s {
			return true
		}
	}
	return false
}

type Input struct {
	games               []*InputGame
	benchmarks          []*InputBenchmark
	standard_benchmarks []*InputStandardBenchmark
}

func CreateInput() *Input {
	return &Input{
		games:               make([]*InputGame, 0),
		benchmarks:          make([]*InputBenchmark, 0),
		standard_benchmarks: make([]*InputStandardBenchmark, 0),
	}
}

func (i *Input) AddLine(q []InputObject, k string, v string) []InputObject {
	switch k {
	case "Game":
		game := createInputGame()
		i.games = append(i.games, game)
		q = append(q, game)
	case "Benchmark":
		benchmark := createInputBenchmark()
		i.benchmarks = append(i.benchmarks, benchmark)
		q = append(q, benchmark)
	case "StandardBenchmark":
		standard_benchmark := createInputStandardBenchmark()
		i.standard_benchmarks = append(i.standard_benchmarks, standard_benchmark)
		q = append(q, standard_benchmark)
	default:
		utils.PrintError("Unknown key for Input: %s", k)
	}
	return q
}

func (i *Input) Print() {
	fmt.Println("Model Run Plan")
	for _, g := range i.games {
		fmt.Println("")
		g.print()
	}
	for _, b := range i.benchmarks {
		fmt.Println("")
		b.print()
	}
	for _, b := range i.standard_benchmarks {
		fmt.Println("")
		b.print()
	}
}

func (i *Input) Validate() error {
	for _, g := range i.games {
		e := g.validate()
		if e != nil {
			return e
		}
	}
	for _, b := range i.benchmarks {
		e := b.validate()
		if e != nil {
			return e
		}
	}
	for _, b := range i.standard_benchmarks {
		e := b.validate()
		if e != nil {
			return e
		}
	}
	return nil
}

type InputGame struct {
	debug        bool
	min_round    uint8
	max_round    uint8
	trick_points uint16
	round_points uint16
	players      []*InputPlayer
}

func createInputGame() *InputGame {
	return &InputGame{
		round_points: 10,
		trick_points: 1,
		players:      make([]*InputPlayer, 0),
	}
}

func (i *InputGame) AddLine(q []InputObject, k string, v string) []InputObject {
	switch k {
	case "debug":
		i.debug = InputStatic.parseBool(v)
	case "min_round":
		i.min_round = uint8(InputStatic.parseInt(v))
	case "max_round":
		i.max_round = uint8(InputStatic.parseInt(v))
	case "trick_points":
		i.trick_points = uint16(InputStatic.parseInt(v))
	case "round_points":
		i.round_points = uint16(InputStatic.parseInt(v))
	case "Player":
		player := createInputPlayer()
		i.players = append(i.players, player)
		q = append(q, player)
	default:
		utils.PrintError("Unknown key for InputGame: %s", k)
	}
	return q
}

func (i *InputGame) print() {
	fmt.Println("Game:")
	fmt.Println("  Min Round:", i.min_round)
	fmt.Println("  Max Round:", i.max_round)
	fmt.Println("  Trick Points:", i.trick_points)
	fmt.Println("  Round Points:", i.round_points)
	for _, p := range i.players {
		p.print("  ")
	}
}

func (i *InputGame) validate() error {
	for _, p := range i.players {
		e := p.validate()
		if e != nil {
			return e
		}
	}
	return nil
}

type InputPlayer struct {
	ai_model_id uint8
}

func createInputPlayer() *InputPlayer {
	return &InputPlayer{}
}

func (i *InputPlayer) AddLine(q []InputObject, k string, v string) []InputObject {
	switch k {
	case "ai_model_id":
		i.ai_model_id = uint8(InputStatic.parseInt(v))
	default:
		utils.PrintError("Unknown key for InputPlayer: %s", k)
	}
	return q
}

func (i *InputPlayer) print(indent string) {
	fmt.Println(indent + "Player:")
	fmt.Println(indent+"  Model ID:", i.ai_model_id)
}

func (i *InputPlayer) validate() error {
	return nil
}

type InputBenchmark struct {
	iterations   uint32
	min_players  uint8
	max_players  uint8
	min_round    uint8
	max_round    uint8
	trick_points uint16
	round_points uint16
	ai_model_id  uint8
	output       string
}

func createInputBenchmark() *InputBenchmark {
	return &InputBenchmark{
		iterations:   1,
		min_players:  2,
		max_players:  8,
		min_round:    1,
		max_round:    12,
		trick_points: 1,
		round_points: 10,
		output:       "benchmark",
	}
}

func (i *InputBenchmark) AddLine(q []InputObject, k string, v string) []InputObject {
	switch k {
	case "iterations":
		i.iterations = uint32(InputStatic.parseInt(v))
	case "min_players":
		i.min_players = uint8(InputStatic.parseInt(v))
	case "max_players":
		i.max_players = uint8(InputStatic.parseInt(v))
	case "min_round":
		i.min_round = uint8(InputStatic.parseInt(v))
	case "max_round":
		i.max_round = uint8(InputStatic.parseInt(v))
	case "trick_points":
		i.trick_points = uint16(InputStatic.parseInt(v))
	case "round_points":
		i.round_points = uint16(InputStatic.parseInt(v))
	case "ai_model_id":
		i.ai_model_id = uint8(InputStatic.parseInt(v))
	case "output":
		i.output = v
	default:
		utils.PrintError("Unknown key for InputBenchmark: %s", k)
	}
	return q
}

func (i *InputBenchmark) print() {
	fmt.Println("Benchmark:")
	fmt.Println("  Iterations:", i.iterations)
	fmt.Println("  Min Players:", i.min_players)
	fmt.Println("  Max Players:", i.max_players)
	fmt.Println("  Min Round:", i.min_round)
	fmt.Println("  Max Round:", i.max_round)
	fmt.Println("  Trick Points:", i.trick_points)
	fmt.Println("  Round Points:", i.round_points)
	fmt.Println("  AI Model:", i.ai_model_id)
	fmt.Println("  Output:", i.output)
}

func (i *InputBenchmark) validate() error {
	return nil
}

type InputStandardBenchmark struct {
	iterations  uint32
	ai_model_id uint8
	output      string
}

func createInputStandardBenchmark() *InputStandardBenchmark {
	return &InputStandardBenchmark{
		iterations: 1,
		output:     "benchmark",
	}
}

func (i *InputStandardBenchmark) AddLine(q []InputObject, k string, v string) []InputObject {
	switch k {
	case "iterations":
		i.iterations = uint32(InputStatic.parseInt(v))
	case "ai_model_id":
		i.ai_model_id = uint8(InputStatic.parseInt(v))
	case "output":
		i.output = v
	default:
		utils.PrintError("Unknown key for InputStandardBenchmark: %s", k)
	}
	return q
}

func (i *InputStandardBenchmark) print() {
	fmt.Println("InputStandardBenchmark:")
	fmt.Println("  Iterations:", i.iterations)
	fmt.Println("  AI Model:", i.ai_model_id)
	fmt.Println("  Output:", i.output)
}

func (i *InputStandardBenchmark) validate() error {
	return nil
}
