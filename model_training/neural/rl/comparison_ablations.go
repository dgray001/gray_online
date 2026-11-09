package rl

func comparisonAblations(agents []benchmarkAgent, summary savedSummary) []benchmarkAgent {
	uniform := initialNetwork(trial("zero", 0.002, 0, 1))
	agents = append(agents, benchmarkAgent{Name: "Uniform policy", Model: 4, Network: uniform}, benchmarkAgent{Name: "Always zero + random cards", Model: 4, Network: uniform, ZeroBid: true, RandomCards: true})
	best := summary.Widths[0]
	for _, row := range summary.Widths {
		if row.Hidden > 0 && row.Validation > best.Validation {
			best = row
		}
	}
	name := ""
	for _, candidate := range agents {
		if candidate.Network != nil && len(candidate.Network.HiddenBias) == best.Hidden {
			name = candidate.Name
			break
		}
	}
	for _, candidate := range agents {
		if candidate.Name == name {
			candidate.Name += " + random cards"
			candidate.RandomCards = true
			agents = append(agents, candidate)
			break
		}
	}
	return agents
}
