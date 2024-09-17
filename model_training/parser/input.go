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

type Input struct {
	games []*InputGame
}

func CreateInput() *Input {
	return &Input{
		games: make([]*InputGame, 0),
	}
}

func (i *Input) AddLine(q []InputObject, k string, v string) []InputObject {
	switch k {
	case "Game":
		game := createInputGame()
		i.games = append(i.games, game)
		q = append(q, game)
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
}

func (i *Input) Validate() error {
	for _, g := range i.games {
		e := g.validate()
		if e != nil {
			return e
		}
	}
	return nil
}

type InputGame struct {
	min_round    uint8
	max_round    uint8
	trick_points uint16
	round_points uint16
	players      []*InputPlayer
}

func createInputGame() *InputGame {
	return &InputGame{
		players: make([]*InputPlayer, 0),
	}
}

func (i *InputGame) AddLine(q []InputObject, k string, v string) []InputObject {
	switch k {
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
