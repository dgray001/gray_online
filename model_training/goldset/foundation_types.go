package goldset

type ScriptedCard struct {
	Seat int
	Card Card
}
type FoundationCase struct {
	Name        string
	Kind        string
	Players     int
	Seat        int
	Hand        []Card
	Trump       Card
	Bids        []uint8
	Plays       []ScriptedCard
	RoundPoints uint16
	TrickPoints uint16
	Check       string
}

func fixtureCard(suit uint8, number uint8) Card { return Card{Suit: suit, Number: number} }
func foundationCase(name string, kind string, players int, seat int, hand []Card, trump Card) FoundationCase {
	return FoundationCase{Name: name, Kind: kind, Players: players, Seat: seat, Hand: hand, Trump: trump, Bids: make([]uint8, players), RoundPoints: 10, TrickPoints: 1}
}
