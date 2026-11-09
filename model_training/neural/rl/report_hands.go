package rl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func handReport(output string, evaluations []Evaluation) error {
	var text strings.Builder
	fmt.Fprint(&text, "# Hand-size results on full-arc mixed-opponent games\n\nCore is 4–8 players. A dash means that hand size is impossible or absent for the group.\n\n")
	for _, e := range evaluations {
		fmt.Fprintf(&text, "## %s\n\n| Cards | All rounds | Exact bids | Core rounds | Core exact bids |\n|---|---:|---:|---:|---:|\n", e.Name)
		for hand := uint8(1); hand <= 25; hand++ {
			m, core := e.Hands[hand], e.CoreHands[hand]
			if m.Rounds == 0 {
				continue
			}
			rate := "-"
			if core.Rounds > 0 {
				rate = fmt.Sprintf("%.1f%%", 100*core.ExactRate)
			}
			fmt.Fprintf(&text, "| %d | %d | %.1f%% | %d | %s |\n", hand, m.Rounds, 100*m.ExactRate, core.Rounds, rate)
		}
		fmt.Fprint(&text, "\n")
	}
	return os.WriteFile(filepath.Join(output, "hand-results.md"), []byte(text.String()), 0600)
}
