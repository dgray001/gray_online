package runner

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"fiddlesticks.live/utils"
	"github.com/dgray001/gray_online/util"
)

func (g *InputGeneticAlgorithm) run() error {
	fmt.Println("")
	time_start := time.Now().UTC()
	fmt.Println("Begin Genetic Algorithm")
	file_strings := make([]string, len(g.players))
	header_strings := []string{"Step", "Temperature", "Score"}
	for i := range file_strings {
		file_header_strings := append(header_strings, g.players[i].modelOutputHeaders()...)
		file_strings[i] = strings.Join(file_header_strings, " ")
	}
	step := 0
	for _, e := range g.episodes {
		t := float64(e.start_temp)
		td := float64(e.end_temp-e.start_temp) / float64(e.steps)
		for range e.steps {
			step++
			input_players := make([]*InputPlayer, 0)
			for _, player := range g.players {
				input_players = append(input_players, player.mutate(t)...)
			}
			input_game := &InputGame{
				debug:        false,
				min_round:    g.min_round,
				max_round:    g.max_round,
				trick_points: g.trick_points,
				round_points: g.round_points,
				players:      input_players,
				iterations:   g.iterations,
			}
			f := input_game.initialize()
			go startGame(f, false)
			<-f.GetBase().GameEndedChannel
			max_score, scores := f.GetGameResults()
			i := uint8(0)
			for k, player := range g.players {
				best_score := scores[i]
				best := i
				for j := i + 1; j < i+player.children; j++ {
					if scores[j] > best_score {
						best_score = scores[j]
						best = j
					}
				}
				player.start = input_players[best].model_input
				score := float64(best_score) / float64(max_score)
				row_strings := make([]string, 3)
				row_strings[0] = strconv.Itoa(step)
				row_strings[1] = strconv.FormatFloat(t, 'f', 3, 64)
				row_strings[2] = strconv.FormatFloat(score, 'f', 3, 64)
				row_strings = append(row_strings, player.modelOutput()...)
				file_strings[k] += "\n" + strings.Join(row_strings, " ")
				i += player.children
			}
			t += td
		}
	}
	for i, s := range file_strings {
		file_data := []byte(s)
		err := os.WriteFile("outputs/genetic_algorithms/"+g.output+"_"+g.players[i].output+".txt", file_data, 0644)
		utils.ErrorCheck(err, "Error writing file")
	}
	fmt.Println("End Genetic Algorithm")
	time_dif := time.Now().UTC().Sub(time_start)
	fmt.Println(time_dif.String())
	return nil
}

func (p *InputGeneticAlgorithmPlayer) mutate(t float64) []*InputPlayer {
	model_inputs := make([]map[string]string, p.children)
	switch p.ai_model_id {
	case 0: // random
		for i := range model_inputs {
			model_inputs[i] = make(map[string]string)
		}
	case 1: // theory model 1
		aggressive_factor := InputStatic.parseFloat(p.start["aggressive_factor"])
		for i := range model_inputs {
			mi := make(map[string]string)
			mi["aggressive_factor"] = mutateFloat(aggressive_factor, 0.05, 20, t)
			model_inputs[i] = mi
		}
	case 2: // theory model 2
		trump_min := InputStatic.parseFloat(p.start["trump_min"])
		non_trump_min := InputStatic.parseInt(p.start["non_trump_min"])
		non_trump_max := InputStatic.parseFloat(p.start["non_trump_max"])
		for i := range model_inputs {
			mi := make(map[string]string)
			mi["trump_min"] = mutateFloat(trump_min, 0, 1, t)
			mi["non_trump_min"] = mutateInt(non_trump_min, 2, 14, t)
			mi["non_trump_max"] = mutateFloat(non_trump_max, 0, 1, t)
			model_inputs[i] = mi
		}
	case 3: // theory model 3
	default:
		utils.PrintError("Unknown AI model id for genetic algorithm: %d", p.ai_model_id)
	}
	players := make([]*InputPlayer, p.children)
	for i := range p.children {
		players[i] = &InputPlayer{
			ai_model_id: p.ai_model_id,
			model_input: model_inputs[i],
		}
	}
	return players
}

func mutateFloat(start float64, min float64, max float64, t float64) string {
	max_change := math.Max(0.001*(max-min)*t, (t/100)*0.8*start)
	f := start + util.RandomFloat(-max_change, max_change)
	if f < min {
		f = min
	} else if f > max {
		f = max
	}
	return strconv.FormatFloat(f, 'f', 3, 64)
}

func mutateInt(start int, min int, max int, t float64) string {
	max_change := (t / 100) * 0.5 * float64(max-min)
	f := float64(start) + util.RandomFloat(-max_change, max_change)
	i := int(f)
	if util.RandomChance(math.Mod(f, 1)) {
		i++
	}
	if i < min {
		i = min
	} else if i > max {
		i = max
	}
	return strconv.Itoa(i)
}

func (p *InputGeneticAlgorithmPlayer) modelOutputHeaders() []string {
	switch p.ai_model_id {
	case 0: // random
		return []string{}
	case 1: // theory model 1
		return []string{"Aggressive Factor"}
	case 2: // theory model 2
		return []string{"Trump Base Value", "Non-Trump Min Value", "Non-Trump Ace Value"}
	default:
		utils.PrintError("Unknown AI model id for genetic algorithm: %d", p.ai_model_id)
	}
	return []string{}
}

func (p *InputGeneticAlgorithmPlayer) modelOutput() []string {
	switch p.ai_model_id {
	case 0: // random
		return []string{}
	case 1: // theory model 1
		return []string{p.start["aggressive_factor"]}
	case 2: // theory model 2
		return []string{p.start["trump_min"], p.start["non_trump_min"], p.start["non_trump_max"]}
	default:
		utils.PrintError("Unknown AI model id for genetic algorithm: %d", p.ai_model_id)
	}
	return []string{}
}
