package rl

import (
	"fmt"
	"strings"
)

func writeArenaScores(text *strings.Builder, evaluations []Evaluation) {
	fmt.Fprint(text, "| Agent | Core mean points | Opponent mean points | Core score / max | Opponent score / max | Margin | Win share |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, e := range evaluations {
		m := selectionMetrics(e)
		fmt.Fprintf(text, "| %s | %.2f | %.2f | %.2f%% | %.2f%% | %+.2f pp | %.2f%% |\n", e.Name, m.Points, m.OpponentPoints, 100*m.Score, 100*m.OpponentScore, 100*m.Margin, 100*m.WinShare)
	}
	fmt.Fprint(text, "\n")
}
