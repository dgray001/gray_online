package runner

import (
	"fmt"
	"math"
	"math/rand"
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
	if g.seeded {
		rand.Seed(g.seed)
	}
	file_strings := make([]string, len(g.players))
	header_strings := []string{"Step", "Temperature", "Score"}
	for i := range file_strings {
		file_header_strings := append(header_strings, g.players[i].modelOutputHeaders()...)
		file_strings[i] = strings.Join(file_header_strings, " ")
	}
	debug_lines := []string{strings.Join(append([]string{"Step", "Temperature", "Player", "Child", "Winner", "Score"}, g.players[0].modelOutputHeaders()...), " ")}
	for k, player := range g.players {
		start_row := append([]string{"0", "-", strconv.Itoa(k), "-", "-", "-"}, player.modelOutput()...)
		debug_lines = append(debug_lines, strings.Join(start_row, " "))
	}
	step := 0
	for _, e := range g.episodes {
		t := float64(e.start_temp)
		td := float64(e.end_temp-e.start_temp) / float64(e.steps)
		for range e.steps {
			step++
			input_players := make([]*InputPlayer, 0)
			for _, player := range g.players {
				input_players = append(input_players, player.mutate(t, player.childrenAt(step))...)
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
				children := player.childrenAt(step)
				best_score := scores[i]
				best := i
				for j := i + 1; j < i+children; j++ {
					if scores[j] > best_score {
						best_score = scores[j]
						best = j
					}
				}
				if g.debug {
					for j := i; j < i+children; j++ {
						child := &InputGeneticAlgorithmPlayer{ai_model_id: player.ai_model_id, start: input_players[j].model_input}
						child_score := float64(scores[j]) / float64(max_score)
						child_row := []string{strconv.Itoa(step), strconv.FormatFloat(t, 'f', 3, 64), strconv.Itoa(k),
							strconv.Itoa(int(j - i)), strconv.FormatBool(j == best), strconv.FormatFloat(child_score, 'f', 5, 64)}
						debug_lines = append(debug_lines, strings.Join(append(child_row, child.modelOutput()...), " "))
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
				i += children
			}
			t += td
			if g.debug {
				err := os.WriteFile("outputs/genetic_algorithms/"+g.output+"_debug.log", []byte(strings.Join(debug_lines, "\n")+"\n"), 0644)
				utils.ErrorCheck(err, "Error writing debug file")
			}
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

func (p *InputGeneticAlgorithmPlayer) mutateParam(key string, min, max float64, t float64) string {
	start := InputStatic.parseFloat(p.start[key])
	return mutateFloat(start, min, max, p.tempFor(key, t))
}

func (p *InputGeneticAlgorithmPlayer) mutate(t float64, children uint8) []*InputPlayer {
	model_inputs := make([]map[string]string, children)
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
		for i := range model_inputs {
			mi := make(map[string]string)
			for k, v := range p.start {
				mi[k] = v
			}
			mi["win_vs_all_trump_factor"] = p.mutateParam("win_vs_all_trump_factor", 0.2, 2, t)
			mi["win_vs_all_nontrump_factor"] = p.mutateParam("win_vs_all_nontrump_factor", 0.2, 2, t)
			mi["min_win_weight_factor"] = p.mutateParam("min_win_weight_factor", 0, 1, t)
			mi["bet_scale"] = p.mutateParam("bet_scale", 0.5, 1.5, t)
			mi["aggressive_factor"] = p.mutateParam("aggressive_factor", 0.2, 5, t)
			mi["late_seat_bonus"] = p.mutateParam("late_seat_bonus", 0, 2, t)
			mi["trump_hold_factor"] = p.mutateParam("trump_hold_factor", 0, 1, t)
			mi["void_exploit"] = p.mutateParam("void_exploit", 0, 2, t)
			model_inputs[i] = mi
		}
	case 4: // theory model 4 (Model 3 + dials + bid inference)
		for i := range model_inputs {
			mi := make(map[string]string)
			for k, v := range p.start {
				mi[k] = v
			}
			mi["win_vs_all_trump_factor"] = p.mutateParam("win_vs_all_trump_factor", 0.2, 2, t)
			mi["win_vs_all_nontrump_factor"] = p.mutateParam("win_vs_all_nontrump_factor", 0.2, 2, t)
			mi["min_win_weight_factor"] = p.mutateParam("min_win_weight_factor", 0, 1, t)
			mi["bet_scale"] = p.mutateParam("bet_scale", 0.5, 1.5, t)
			mi["aggressive_factor"] = p.mutateParam("aggressive_factor", 0.2, 5, t)
			mi["late_seat_bonus"] = p.mutateParam("late_seat_bonus", 0, 2, t)
			mi["trump_hold_factor"] = p.mutateParam("trump_hold_factor", 0, 1, t)
			mi["void_exploit"] = p.mutateParam("void_exploit", 0, 2, t)
			mi["bet_optimism"] = p.mutateParam("bet_optimism", -1.0, 1.0, t)
			mi["trick_chasing"] = p.mutateParam("trick_chasing", 0.0, 2.0, t)
			mi["trump_hoarding"] = p.mutateParam("trump_hoarding", 0.0, 2.0, t)
			mi["void_hunting"] = p.mutateParam("void_hunting", 0.0, 2.0, t)
			mi["vindictiveness"] = p.mutateParam("vindictiveness", 0.0, 3.0, t)
			mi["bid_trust"] = p.mutateParam("bid_trust", 0.0, 2.0, t)
			model_inputs[i] = mi
		}
	case 5: // theory model 4a (Contract planner + Model 3 beliefs)
		for i := range model_inputs {
			mi := make(map[string]string)
			for k, v := range p.start {
				mi[k] = v
			}
			mi["win_vs_all_trump_factor"] = p.mutateParam("win_vs_all_trump_factor", 0.2, 2, t)
			mi["win_vs_all_nontrump_factor"] = p.mutateParam("win_vs_all_nontrump_factor", 0.2, 2, t)
			mi["min_win_weight_factor"] = p.mutateParam("min_win_weight_factor", 0, 1, t)
			mi["bet_scale"] = p.mutateParam("bet_scale", 0.5, 1.5, t)
			mi["bet_optimism"] = p.mutateParam("bet_optimism", -1.0, 1.0, t)
			mi["planner_temperature"] = p.mutateParam("planner_temperature", 0.05, 1.0, t)
			mi["overtrick_aversion"] = p.mutateParam("overtrick_aversion", 0.1, 3.0, t)
			mi["undertrick_aversion"] = p.mutateParam("undertrick_aversion", 0.1, 3.0, t)
			mi["trump_hoarding"] = p.mutateParam("trump_hoarding", 0.0, 2.0, t)
			mi["void_hunting"] = p.mutateParam("void_hunting", 0.0, 2.0, t)
			mi["vindictiveness"] = p.mutateParam("vindictiveness", 0.0, 3.0, t)
			model_inputs[i] = mi
		}
	case 6: // theory model 4b (Contract planner + bid-conditioned beliefs)
		for i := range model_inputs {
			mi := make(map[string]string)
			for k, v := range p.start {
				mi[k] = v
			}
			mi["win_vs_all_trump_factor"] = p.mutateParam("win_vs_all_trump_factor", 0.2, 2, t)
			mi["win_vs_all_nontrump_factor"] = p.mutateParam("win_vs_all_nontrump_factor", 0.2, 2, t)
			mi["min_win_weight_factor"] = p.mutateParam("min_win_weight_factor", 0, 1, t)
			mi["bet_scale"] = p.mutateParam("bet_scale", 0.5, 1.5, t)
			mi["bet_optimism"] = p.mutateParam("bet_optimism", -1.0, 1.0, t)
			mi["planner_temperature"] = p.mutateParam("planner_temperature", 0.05, 1.0, t)
			mi["overtrick_aversion"] = p.mutateParam("overtrick_aversion", 0.1, 3.0, t)
			mi["undertrick_aversion"] = p.mutateParam("undertrick_aversion", 0.1, 3.0, t)
			mi["trump_hoarding"] = p.mutateParam("trump_hoarding", 0.0, 2.0, t)
			mi["void_hunting"] = p.mutateParam("void_hunting", 0.0, 2.0, t)
			mi["vindictiveness"] = p.mutateParam("vindictiveness", 0.0, 3.0, t)
			mi["bid_trust"] = p.mutateParam("bid_trust", 0.0, 2.0, t)
			model_inputs[i] = mi
		}
	case 7: // theory model 4c (Monte Carlo particle filter + planner)
		for i := range model_inputs {
			mi := make(map[string]string)
			for k, v := range p.start {
				mi[k] = v
			}
			mi["bet_optimism"] = p.mutateParam("bet_optimism", -1.0, 1.0, t)
			mi["planner_temperature"] = p.mutateParam("planner_temperature", 0.05, 1.0, t)
			mi["overtrick_aversion"] = p.mutateParam("overtrick_aversion", 0.1, 3.0, t)
			mi["undertrick_aversion"] = p.mutateParam("undertrick_aversion", 0.1, 3.0, t)
			mi["paranoia"] = p.mutateParam("paranoia", 0.0, 1.0, t)
			mi["bid_trust"] = p.mutateParam("bid_trust", 0.0, 2.0, t)
			mi["vindictiveness"] = p.mutateParam("vindictiveness", 0.0, 3.0, t)
			model_inputs[i] = mi
		}
	default:
		utils.PrintError("Unknown AI model id for genetic algorithm: %d", p.ai_model_id)
	}
	players := make([]*InputPlayer, children)
	for i := range children {
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
	case 3: // theory model 3
		return []string{"Win vs All Trump", "Win vs All Non-Trump", "Min Win Weight", "Bet Scale", "Aggressive Factor", "Late Seat Bonus", "Trump Hold", "Void Exploit"}
	case 4: // theory model 4
		return []string{"Win vs All Trump", "Win vs All Non-Trump", "Min Win Weight", "Bet Scale", "Aggressive Factor", "Late Seat Bonus", "Trump Hold", "Void Exploit", "Bet Optimism", "Trick Chasing", "Trump Hoarding", "Void Hunting", "Vindictiveness", "Bid Trust"}
	case 5: // theory model 4a
		return []string{"Win vs All Trump", "Win vs All Non-Trump", "Min Win Weight", "Bet Scale", "Bet Optimism", "Planner Temperature", "Overtrick Aversion", "Undertrick Aversion", "Trump Hoarding", "Void Hunting", "Vindictiveness"}
	case 6: // theory model 4b
		return []string{"Win vs All Trump", "Win vs All Non-Trump", "Min Win Weight", "Bet Scale", "Bet Optimism", "Planner Temperature", "Overtrick Aversion", "Undertrick Aversion", "Trump Hoarding", "Void Hunting", "Vindictiveness", "Bid Trust"}
	case 7: // theory model 4c
		return []string{"Bet Optimism", "Planner Temperature", "Overtrick Aversion", "Undertrick Aversion", "Paranoia", "Bid Trust", "Vindictiveness"}
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
	case 3: // theory model 3
		return []string{p.start["win_vs_all_trump_factor"], p.start["win_vs_all_nontrump_factor"], p.start["min_win_weight_factor"], p.start["bet_scale"], p.start["aggressive_factor"], p.start["late_seat_bonus"], p.start["trump_hold_factor"], p.start["void_exploit"]}
	case 4: // theory model 4
		return []string{p.start["win_vs_all_trump_factor"], p.start["win_vs_all_nontrump_factor"], p.start["min_win_weight_factor"], p.start["bet_scale"], p.start["aggressive_factor"], p.start["late_seat_bonus"], p.start["trump_hold_factor"], p.start["void_exploit"], p.start["bet_optimism"], p.start["trick_chasing"], p.start["trump_hoarding"], p.start["void_hunting"], p.start["vindictiveness"], p.start["bid_trust"]}
	case 5: // theory model 4a
		return []string{p.start["win_vs_all_trump_factor"], p.start["win_vs_all_nontrump_factor"], p.start["min_win_weight_factor"], p.start["bet_scale"], p.start["bet_optimism"], p.start["planner_temperature"], p.start["overtrick_aversion"], p.start["undertrick_aversion"], p.start["trump_hoarding"], p.start["void_hunting"], p.start["vindictiveness"]}
	case 6: // theory model 4b
		return []string{p.start["win_vs_all_trump_factor"], p.start["win_vs_all_nontrump_factor"], p.start["min_win_weight_factor"], p.start["bet_scale"], p.start["bet_optimism"], p.start["planner_temperature"], p.start["overtrick_aversion"], p.start["undertrick_aversion"], p.start["trump_hoarding"], p.start["void_hunting"], p.start["vindictiveness"], p.start["bid_trust"]}
	case 7: // theory model 4c
		return []string{p.start["bet_optimism"], p.start["planner_temperature"], p.start["overtrick_aversion"], p.start["undertrick_aversion"], p.start["paranoia"], p.start["bid_trust"], p.start["vindictiveness"]}
	default:
		utils.PrintError("Unknown AI model id for genetic algorithm: %d", p.ai_model_id)
	}
	return []string{}
}
