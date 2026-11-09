package rl

import (
	"fmt"
	"strings"
)

func writeWidthScores(text *strings.Builder, rows []WidthResult) {
	fmt.Fprint(text, "## Score comparison on full-arc 4–8-player games\n\nMean points are raw points per game. Normalized score is the mean of points divided by that game's maximum possible player score. Opponent score averages the other players under the same normalization. Margin is their difference in percentage points.\n\n| Hidden | Mean points | Score / max | Opponent score / max | Score margin | Win share | Historical mean points | Historical score / max |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range rows {
		fmt.Fprintf(text, "| %d | %.2f | %.2f%% | %.2f%% | %+.2f pp | %.2f%% | %.2f | %.2f%% |\n", r.Hidden, r.CorePoints, 100*r.CoreScore, 100*r.CoreOpponentScore, 100*r.CoreMargin, 100*r.CoreWin, r.HistoricalPoints, 100*r.HistoricalScore)
	}
	fmt.Fprint(text, "\n")
}
