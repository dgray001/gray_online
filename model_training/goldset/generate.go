package goldset

import (
	"fmt"
	"math/rand"
)

func Generate(q GenerateRequest) ([]Item, error) {
	if q.Count < 1 || q.Count > 100 || q.MinHand < 1 || q.MaxHand < q.MinHand || q.MaxHand > 25 || q.RoundPoints > 99 || q.TrickPoints > 99 || q.RoundPoints+q.TrickPoints == 0 || (q.Kind != "both" && q.Kind != "bet" && q.Kind != "play-card") {
		return nil, fmt.Errorf("invalid count, hand range, phase, or scoring")
	}
	players := []int{}
	for p := 2; p <= 10; p++ {
		if (q.Players == p || (q.Players == 0 && p >= 4 && p <= 8)) && int(q.MinHand) <= 51/p {
			players = append(players, p)
		}
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("no legal player count for this hand range")
	}
	bids, cards := q.Count, 0
	if q.Kind == "play-card" {
		bids, cards = 0, q.Count
	}
	if q.Kind == "both" {
		bids, cards = (q.Count+1)/2, q.Count/2
	}
	pool := map[string]*reservoir{"bet": {Limit: bids, Random: rand.New(rand.NewSource(q.Seed))}, "play-card": {Limit: cards, Random: rand.New(rand.NewSource(q.Seed + 1))}}
	if cards > 0 && q.MaxHand == 1 {
		return nil, fmt.Errorf("one-card hands have no card choice; select bidding or a larger hand")
	}
	for gameIndex := 0; gameIndex < 100; gameIndex++ {
		generateGame(q, players[gameIndex%len(players)], gameIndex+1, pool)
		if gameIndex+1 >= len(players)*2 && len(pool["bet"].Items) == bids && len(pool["play-card"].Items) == cards {
			break
		}
	}
	result := append(pool["bet"].Items, pool["play-card"].Items...)
	if len(result) < q.Count {
		return nil, fmt.Errorf("only %d meaningful choices found; widen the hand range", len(result))
	}
	return result, nil
}
