package goldset

func foundationBids() []FoundationCase {
	c := fixtureCard
	a := foundationCase("protected-ace", "bet", 4, 0, []Card{c(4, 4), c(4, 14)}, c(2, 10))
	b := foundationCase("unprotected-highs", "bet", 4, 0, []Card{c(1, 13), c(4, 14)}, c(2, 10))
	d := foundationCase("protected-eight", "bet", 8, 0, []Card{c(4, 4), c(4, 14)}, c(2, 10))
	e := foundationCase("low-trump-bid-evidence", "bet", 4, 3, []Card{c(2, 5)}, c(2, 11))
	e.Bids = []uint8{0, 0, 1, 0}
	f := foundationCase("low-trump-all-zero", "bet", 4, 3, []Card{c(2, 5)}, c(2, 11))
	g := foundationCase("weak-off-trump-standard-points", "bet", 4, 3, []Card{c(2, 5)}, c(1, 10))
	h := foundationCase("weak-off-trump-trick-points", "bet", 4, 3, []Card{c(2, 5)}, c(1, 10))
	h.RoundPoints, h.TrickPoints = 0, 5
	i := foundationCase("position-overbid-pressure", "bet", 4, 3, []Card{c(3, 13), c(3, 10), c(2, 4)}, c(3, 9))
	i.Bids = []uint8{3, 0, 0, 0}
	j := foundationCase("position-orphan-pressure", "bet", 4, 3, []Card{c(3, 13), c(3, 10), c(2, 4)}, c(3, 9))
	return []FoundationCase{a, b, d, e, f, g, h, i, j}
}
