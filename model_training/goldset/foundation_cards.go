package goldset

func foundationCards() []FoundationCase {
	c := fixtureCard
	result := []FoundationCase{}
	for i, high := range []uint8{13, 6, 13} {
		name := []string{"discard-wide-gap", "discard-small-gap", "discard-without-trump-on-table"}[i]
		x := foundationCase(name, "play-card", 4, 3, []Card{c(4, 3), c(4, high), c(1, 2)}, c(2, 4))
		middle := c(2, 14)
		if i == 2 {
			middle = c(1, 5)
		}
		x.Plays = []ScriptedCard{{0, c(3, 4)}, {1, middle}, {2, c(3, 8)}}
		result = append(result, x)
	}
	return append(append(result, foundationRankCases()...), foundationContractCases()...)
}
