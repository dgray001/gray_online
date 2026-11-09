package fiddlesticks

import "testing"

func TestProgressIncludesRepeatedRounds(t *testing.T) {
	f := InitializeGame(nil)
	f.min_round, f.max_round, f.iterations = 1, 3, 2
	hands := []uint8{1, 1, 2, 2, 3, 3, 2, 2, 1, 1}
	for index, hand := range hands {
		f.round = hand
		f.rounds_increasing = index < 6
		f.iteration = uint32(index % 2)
		p := f.progress()
		future := 0.0
		for _, next := range hands[index+1:] {
			future += 10 + float64(next)
		}
		if p.Index != index || p.Total != 10 || p.Remaining != 9-index || p.FuturePoints != future {
			t.Fatal(index, p, future)
		}
	}
}
