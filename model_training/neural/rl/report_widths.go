package rl

import (
	"fmt"
	"strings"
)

func writeWidths(text *strings.Builder, rows []WidthResult) {
	fmt.Fprint(text, "## Width comparison\n\nEach width selects its recipe using mean validation across three seeds. Held-out results use the fixed seed-1 representative of that recipe.\n\n| Hidden | Parameters | Validation mean | Seed SD | Full-arc core exact | Core upper-half exact | Core win share | Core zero bids | Historical exact |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range rows {
		fmt.Fprintf(text, "| %d | %d | %.2f%% | %.2f pp | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% |\n", r.Hidden, r.Parameters, 100*r.Validation, 100*r.SeedSD, 100*r.CoreExact, 100*r.CoreUpper, 100*r.CoreWin, 100*r.ZeroBids, 100*r.Historical)
	}
	fmt.Fprint(text, "\n")
}
