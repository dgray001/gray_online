package goldset

import "fmt"

type FoundationCheck struct {
	ItemID     string `json:"item_id"`
	Acceptable []int  `json:"acceptable_actions"`
	Proof      string `json:"proof"`
	Rule       string `json:"rule"`
}

func foundationKey(spec FoundationCase, item Item) (*FoundationCheck, error) {
	if spec.Check == "" {
		return nil, nil
	}
	if item.State.Progress.Total != 1 || item.State.RoundPoints == 0 || item.State.TrickPoints == 0 {
		return nil, fmt.Errorf("check does not establish the stated game objective")
	}
	for _, p := range item.State.Players {
		if p.Score != 0 || (p.Seat != item.State.Seat && (!p.HasBid || p.Bid != 0)) {
			return nil, fmt.Errorf("check has uncontrolled scores or bids")
		}
	}
	key := &FoundationCheck{ItemID: item.ID, Rule: spec.Check}
	if spec.Check == "one-card" {
		p := item.Analysis["trick_probabilities"].([]float64)
		if p[1] == 1 {
			key.Acceptable = []int{1}
			key.Proof = "The card is an unbeatable trump. An exact bid of one scores above every zero bidder and guarantees the game win."
		} else if p[0] == 1 {
			key.Acceptable = []int{0}
			key.Proof = "The lowest off-trump card cannot win when not leading. Bid zero guarantees a winning share; bid one loses to the other exact zero bidders."
		} else {
			return nil, fmt.Errorf("one-card check is not certain")
		}
	} else {
		own := item.State.Players[item.State.Seat]
		low, ace := 0, false
		for _, card := range item.State.Hand {
			if !card.Played && card.Suit != item.State.Trump.Suit && card.Number == 2 {
				low++
			}
		}
		for _, card := range item.State.Trick {
			if card.Suit == item.State.Trump.Suit && card.Number == 14 {
				ace = true
			}
		}
		if own.Bid != 0 || own.Tricks != 0 || low != 2 || !ace || item.State.HandSize != 3 {
			return nil, fmt.Errorf("discard check assumptions do not hold")
		}
		if len(item.State.Players) <= int(item.State.HandSize) {
			return nil, fmt.Errorf("pigeonhole condition missing")
		}
		for _, card := range item.State.Hand {
			if !card.Played && card.Suit == item.State.Trump.Suit && card.Number == 13 {
				key.Acceptable = []int{card.Action}
			}
		}
		if len(key.Acceptable) != 1 {
			return nil, fmt.Errorf("discard check lacks king")
		}
		key.Proof = "The current trick already contains the trump ace. Discarding the king now leaves only lowest off-trumps and guarantees zero tricks. Keeping it forces a later overtrick; because players outnumber tricks, another zero bidder must score and beat that failed contract."
	}
	return key, nil
}
