package behavior

import (
	"fmt"
	"strings"
)

func writeGroups(text *strings.Builder, pairs []Pair) {
	groups := map[string][]Pair{}
	for _, pair := range pairs {
		if group := comparisonGroup(pair); group != "" {
			groups[group] = append(groups[group], pair)
		}
	}
	fmt.Fprint(text, "## Comparisons changing one factor\n\nValues average pair distances across the same situation set.\n\n| Changed factor | Pairs | Bid MAE | Bid TV | Card TV | Top-card disagreement | Parameter RMS |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, name := range []string{"seed only", "initialization only", "learning rate only", "target type only", "entropy only", "architecture only"} {
		group, total := groups[name], Pair{}
		if len(group) == 0 {
			continue
		}
		weightCount := 0
		for _, p := range group {
			total.BidMAE += p.BidMAE
			total.BidTV += p.BidTV
			total.CardTV += p.CardTV
			total.CardDisagreement += p.CardDisagreement
			if p.ParameterComparable {
				total.ParameterRMS += p.ParameterRMS
				weightCount++
			}
		}
		n := float64(max(1, len(group)))
		if weightCount > 0 {
			total.ParameterComparable = true
			total.ParameterRMS /= float64(weightCount)
		}
		fmt.Fprintf(text, "| %s | %d | %.4f | %.4f | %.4f | %.1f%% | %s |\n", name, len(group), total.BidMAE/n, total.BidTV/n, total.CardTV/n, 100*total.CardDisagreement/n, weightText(total))
	}
	fmt.Fprint(text, "\n")
}
