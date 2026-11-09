package goldset

func foundationContractCases() []FoundationCase {
	c := fixtureCard
	result := []FoundationCase{}
	for _, bid := range []uint8{0, 1, 2} {
		name := []string{"need-none", "need-one", "need-two"}[bid]
		x := foundationCase(name, "play-card", 4, 3, []Card{c(3, 2), c(3, 7), c(2, 2)}, c(3, 11))
		x.Bids = []uint8{0, 1, 0, bid}
		if bid == 2 {
			x.Bids = []uint8{0, 0, 1, 2}
		}
		x.Plays = []ScriptedCard{{0, c(4, 13)}, {1, c(4, 3)}, {2, c(4, 5)}}
		result = append(result, x)
	}
	x := foundationCase("overbid-pressure", "play-card", 4, 3, []Card{c(3, 2), c(3, 7), c(2, 2)}, c(3, 11))
	x.Bids = []uint8{2, 2, 0, 0}
	x.Plays = result[0].Plays
	result = append(result, x)
	for _, bid := range []uint8{0, 1} {
		name := "already-over-contract"
		if bid == 1 {
			name = "contract-met"
		}
		x := foundationCase(name, "play-card", 4, 3, []Card{c(3, 13), c(4, 8), c(4, 12)}, c(3, 9))
		x.Bids = []uint8{0, 2, 0, bid}
		x.Plays = []ScriptedCard{{0, c(2, 4)}, {1, c(2, 8)}, {2, c(2, 14)}, {3, c(3, 13)}}
		result = append(result, x)
	}
	return result
}
