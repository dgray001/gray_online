package rl

func exactCount(episode Episode) int {
	count := 0
	for _, outcome := range episode.Outcomes {
		if outcome.Bid == outcome.Tricks {
			count++
		}
	}
	return count
}
