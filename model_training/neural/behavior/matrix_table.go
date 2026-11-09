package behavior

import (
	"fmt"
	"html"
	"strings"
)

func matrixTable(text *strings.Builder, summary Summary, bids bool) {
	lookup, maximum := map[string]Pair{}, 0.0
	for _, pair := range summary.Pairs {
		lookup[pair.A+"|"+pair.B] = pair
		lookup[pair.B+"|"+pair.A] = pair
		value := pair.CardTV
		if bids {
			value = pair.BidMAE
		}
		maximum = max(maximum, value)
	}
	title := "Mean card-selection TV distance"
	if bids {
		title = "Mean absolute expected-bid difference"
	}
	fmt.Fprintf(text, "<h2>%s</h2><p>Darker cells show larger differences; scale: 0 to %.4f.</p><table><tr><th></th>", title, maximum)
	for _, name := range summary.Trials {
		fmt.Fprintf(text, "<th class=column>%s</th>", html.EscapeString(name))
	}
	fmt.Fprint(text, "</tr>")
	for _, row := range summary.Trials {
		fmt.Fprintf(text, "<tr><th>%s</th>", html.EscapeString(row))
		for _, column := range summary.Trials {
			matrixCell(text, lookup[row+"|"+column], bids, maximum)
		}
		fmt.Fprint(text, "</tr>")
	}
	fmt.Fprint(text, "</table>")
}
