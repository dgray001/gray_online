package goldset

func foundationChecks() []FoundationCase {
	c := fixtureCard
	result := []FoundationCase{
		foundationCase("one-card-A", "bet", 4, 3, []Card{c(1, 14)}, c(1, 5)),
		foundationCase("one-card-B", "bet", 5, 4, []Card{c(4, 13)}, c(4, 14)),
		foundationCase("one-card-C", "bet", 8, 7, []Card{c(2, 14)}, c(2, 9)),
		foundationCase("one-card-D", "bet", 4, 3, []Card{c(2, 2)}, c(1, 10)),
		foundationCase("one-card-E", "bet", 6, 5, []Card{c(4, 2)}, c(3, 11)),
		foundationCase("one-card-F", "bet", 8, 7, []Card{c(1, 2)}, c(2, 12)),
	}
	for i := range result {
		result[i].Check = "one-card"
	}
	a := foundationCase("discard-A", "play-card", 4, 3, []Card{c(3, 13), c(1, 2), c(2, 2)}, c(3, 5))
	a.Plays = []ScriptedCard{{0, c(4, 4)}, {1, c(3, 14)}, {2, c(2, 3)}}
	a.Check = "discard-top-trump"
	b := foundationCase("discard-B", "play-card", 5, 4, []Card{c(2, 13), c(4, 2), c(1, 2)}, c(2, 5))
	b.Plays = []ScriptedCard{{0, c(3, 4)}, {1, c(2, 14)}, {2, c(4, 3)}, {3, c(1, 3)}}
	b.Check = "discard-top-trump"
	return append(result, a, b)
}
