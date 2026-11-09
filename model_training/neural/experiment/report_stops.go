package experiment

import (
	"fmt"
	"strings"
)

func writeStops(text *strings.Builder, results []Result, options Options) {
	plateaus, capped := 0, 0
	for _, result := range results {
		if result.Stop == "validation plateau" {
			plateaus++
		} else {
			capped++
		}
	}
	fmt.Fprintf(text, "## Stopping and scope\n\n%d trials plateaued; %d reached the %d-epoch cap. Capped trials are not evidence of convergence.\n\n", plateaus, capped, options.MaxEpochs)
	fmt.Fprint(text, "Training imitates Theory4 decisions under different opponent policies. Game results measure actual play. Three training seeds provide an exploratory comparison, not a statistical significance claim. Matched deals are independent of move sampling. Tournament seats rotate across matched deals.\n\nsummary.json contains breakdowns by player count, scoring, hand limit, and opponents; trials.json contains per-epoch curves and separate bid/card validation and test losses. Game files preserve every evaluated score and seed.\n")
}
