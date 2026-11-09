package fiddlesticks

type GameProgress struct {
	Index         int
	Total         int
	Remaining     int
	FuturePoints  float64
	MaximumPoints float64
}

func (f *GameFiddlesticks) progress() GameProgress {
	low, high, hand, repeats := int(f.min_round), int(f.max_round), int(f.round), int(f.iterations)
	path := hand - low
	if !f.rounds_increasing {
		path = high - low + high - hand
	}
	total := (2*(high-low) + 1) * repeats
	index := path*repeats + int(f.iteration)
	future := (hand*(hand-1) - low*(low-1)) / 2
	if f.rounds_increasing {
		future = (high*(high+1) - hand*(hand+1) + high*(high-1) - low*(low-1)) / 2
	}
	cards := max(0, repeats-int(f.iteration)-1)*hand + repeats*future
	remaining := max(0, total-index-1)
	return GameProgress{Index: index, Total: total, Remaining: remaining, FuturePoints: float64(f.round_points)*float64(remaining) + float64(f.trick_points)*float64(max(0, cards)), MaximumPoints: float64(f.round_points)*float64(total) + float64(f.trick_points)*float64(repeats*(high*high-low*(low-1)))}
}
