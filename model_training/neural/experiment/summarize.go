package experiment

import (
	"fmt"
	"slices"
)

func summarize(rows []GameResult) []Evaluation {
	groups := map[string]*Evaluation{}
	for _, row := range rows {
		keys := []string{"all", fmt.Sprintf("players=%d", row.Settings.Players), fmt.Sprintf("scoring=%d:%d", row.Settings.RoundPoints, row.Settings.TrickPoints), "opponents=" + row.Opponents, fmt.Sprintf("rounds=%d", row.Settings.MaxRound)}
		for _, key := range keys {
			id := row.Agent + "|" + key
			if groups[id] == nil {
				groups[id] = &Evaluation{Agent: row.Agent, Group: key}
			}
			m := groups[id]
			m.Games++
			m.WinShare += row.WinShare
			m.WinAboveChance += row.WinShare - row.Chance
			m.Score += row.Score
			m.Margin += row.Margin
		}
	}
	result := []Evaluation{}
	for _, m := range groups {
		m.WinShare /= float64(m.Games)
		m.WinAboveChance /= float64(m.Games)
		m.Score /= float64(m.Games)
		m.Margin /= float64(m.Games)
		result = append(result, *m)
	}
	slices.SortFunc(result, func(a Evaluation, b Evaluation) int {
		if a.Agent < b.Agent {
			return -1
		}
		if a.Agent > b.Agent {
			return 1
		}
		if a.Group < b.Group {
			return -1
		}
		if a.Group > b.Group {
			return 1
		}
		return 0
	})
	return result
}
