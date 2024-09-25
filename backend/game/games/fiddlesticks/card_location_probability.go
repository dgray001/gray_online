package fiddlesticks

/** describes the probability that a card is in any possible location */
type CardLocationProbability struct {
	played bool
	hand   bool
	trump  bool
	kitty  float64
	player map[int]float64
}

func (p *CardLocationProbability) reset(players []*FiddlesticksPlayer, model_player int) {
	p.played = false
	p.hand = false
	p.trump = false
	p.kitty = 0
	for k := range p.player {
		delete(p.player, k)
	}
	for i, player := range players {
		if i == model_player {
			continue
		}
		p.player[player.player.Player_id] = 0
	}
}

func (p *CardLocationProbability) inHand() {
	p.hand = true
}

func (p *CardLocationProbability) isTrump() {
	p.trump = true
}

func (p *CardLocationProbability) unknown(prob_kitty float64, prob_hand float64) {
	p.kitty = prob_kitty
	for k := range p.player {
		p.player[k] = prob_hand
	}
}
