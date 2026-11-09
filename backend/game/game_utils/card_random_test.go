package game_utils

import (
	"math/rand"
	"testing"
)

func TestSeededDealsIgnoreActionRandomness(t *testing.T) {
	a, b := CreateStandardDeck(), CreateStandardDeck()
	x, y := rand.New(rand.NewSource(17)), rand.New(rand.NewSource(17))
	for round := 0; round < 5; round++ {
		a.ResetWithRandom(x)
		for i := 0; i < 100; i++ {
			rand.Float64()
		}
		b.ResetWithRandom(y)
		for i := 0; i < 52; i++ {
			if a.DrawCard().Hash() != b.DrawCard().Hash() {
				t.Fatal("deal changed")
			}
		}
	}
}
