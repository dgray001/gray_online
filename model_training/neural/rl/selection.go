package rl

func selectionMetrics(e Evaluation) Metrics {
	if m, exists := e.Groups["players=4-8"]; exists {
		return m
	}
	return e.All
}
