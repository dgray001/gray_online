package goldset

func foundationRankCases() []FoundationCase {
	c := fixtureCard
	a := foundationCase("seen-intermediate-rank", "play-card", 4, 3, []Card{c(1, 11), c(1, 9), c(3, 2)}, c(2, 3))
	a.Plays = []ScriptedCard{{0, c(2, 5)}, {1, c(1, 10)}, {2, c(2, 14)}, {3, c(3, 2)}, {2, c(1, 12)}}
	b := foundationCase("unseen-intermediate-rank", "play-card", 4, 3, []Card{c(1, 11), c(1, 9), c(3, 2)}, c(2, 3))
	b.Plays = []ScriptedCard{{0, c(2, 5)}, {1, c(4, 10)}, {2, c(2, 14)}, {3, c(3, 2)}, {2, c(1, 12)}}
	d := foundationCase("adjacent-top-ranks", "play-card", 4, 3, []Card{c(1, 13), c(1, 14), c(3, 2)}, c(2, 3))
	d.Plays = []ScriptedCard{{0, c(2, 5)}, {1, c(4, 10)}, {2, c(2, 14)}, {3, c(3, 2)}, {2, c(1, 2)}}
	e := foundationCase("gap-at-top-ranks", "play-card", 4, 3, []Card{c(1, 12), c(1, 14), c(3, 2)}, c(2, 3))
	e.Plays = d.Plays
	return []FoundationCase{a, b, d, e}
}
