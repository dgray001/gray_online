package neural

import (
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/fiddlesticks"
	"math"
	"slices"
	"testing"
)

func TestContextFeaturesAndLegacyCompatibility(t *testing.T) {
	s := fiddlesticks.LearningState{Round: 4, Seat: 0, Bets: make([]uint8, 4), Tricks: make([]uint8, 4), Scores: []uint32{10, 15, 5, 2}, HasBet: make([]bool, 4), Voids: make([][4]bool, 4), Profiles: make([]fiddlesticks.PublicProfile, 4), PriorProfiles: make([]fiddlesticks.PublicProfile, 4), Progress: fiddlesticks.GameProgress{Index: 4, Total: 9, Remaining: 4, FuturePoints: 40, MaximumPoints: 100}}
	s.Voids[1][1] = true
	s.Profiles[1] = fiddlesticks.PublicProfile{Bids: 2, BidFraction: 1.2}
	s.PriorProfiles[2] = fiddlesticks.PublicProfile{Hands: 10, Exact: 8}
	s.Trump = game_utils.StandardCardStatic.CreateCard(4, 14)
	s.RoundHistory = []fiddlesticks.PublicAction{{Seat: 2, Suit: 1, Number: 5, Trick: 3, Order: 1}}
	input := Features(s, true)
	if len(input) != 740 || math.Abs(input[380]+1) > 1e-12 || input[381] != 0.625 || input[531] != 1 || input[574] != 0.6 || input[663] != 0.8 || input[728] != 0.5 || input[732] != 0.4 {
		t.Fatal("missing public context")
	}
	old := NewNetwork(LegacyInputs, 4, Outputs, 1)
	if !slices.Equal(FeaturesFor(old, s, true), input[:LegacyInputs]) {
		t.Fatal("legacy input layout changed")
	}
}
