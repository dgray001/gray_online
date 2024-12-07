package fiddlesticks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dgray001/gray_online/game/game_utils"
)

/** describes the probability that a card is in any possible location */
type CardLocationProbability struct {
	card    *game_utils.StandardCard
	played  bool
	hand    bool
	trump   bool
	unknown bool
	kitty   float64
	player  map[int]float64
}

func (p *CardLocationProbability) reset(players []*FiddlesticksPlayer, model_player int) {
	p.played = false
	p.hand = false
	p.trump = false
	p.unknown = false
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

func (p *CardLocationProbability) isUnknown(prob_kitty float64, prob_hand float64) {
	p.unknown = true
	p.kitty = prob_kitty
	for k := range p.player {
		p.player[k] = prob_hand
	}
}

func (p *CardLocationProbability) cardPlayed() {
	p.played = true
	p.hand = false
	p.unknown = false
	p.kitty = 0
	for k := range p.player {
		p.player[k] = 0
	}
}

func (p *CardLocationProbability) printString() string {
	players := make([]string, 0, len(p.player))
	for _, v := range p.player {
		players = append(players, strconv.FormatFloat(v, 'f', 3, 64))
	}
	return fmt.Sprintf("%s: %t, %t, %t, %.3f, (%s)", p.card.GetShortName(), p.played, p.hand, p.trump, p.kitty, strings.Join(players, ", "))
}
