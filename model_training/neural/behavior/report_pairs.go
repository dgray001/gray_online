package behavior

import (
	"fmt"
	"strings"
)

func writePairs(text *strings.Builder, title string, pairs []Pair) {
	fmt.Fprintf(text, "## %s\n\n| A | B | Bid MAE | Bid TV | Top-bid disagreement | Card TV | Top-card disagreement | Parameter RMS |\n|---|---|---:|---:|---:|---:|---:|---:|\n", title)
	for _, p := range pairs {
		fmt.Fprintf(text, "| %s | %s | %.4f | %.4f | %.1f%% | %.4f | %.1f%% | %s |\n", p.A, p.B, p.BidMAE, p.BidTV, 100*p.BidDisagreement, p.CardTV, 100*p.CardDisagreement, weightText(p))
	}
	fmt.Fprint(text, "\n")
}
